package security

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools/toolutil"
)

// defaultOSVBaseURL is the OSV.dev REST API root. It is a struct field on the
// scanner so tests can point it at a local httptest server.
const defaultOSVBaseURL = "https://api.osv.dev"

// scanHTTPTimeout bounds the whole OSV interaction (batch query plus detail
// fetches) so a slow or unreachable OSV endpoint cannot stall the tool past its
// external-API budget.
const scanHTTPTimeout = 20 * time.Second

// maxVulnDetailFetches caps how many unique vulnerability records the scanner
// fetches full details for, bounding fan-out on a heavily-vulnerable project.
const maxVulnDetailFetches = 200

// severityRank orders the threshold levels so a vuln can be compared against the
// requested floor. Unknown severities are always reported (rank -1 < any floor).
var severityRank = map[string]int{
	"low": 0, "medium": 1, "high": 2, "critical": 3,
}

// securityScanner queries OSV.dev for vulnerabilities affecting the project's
// pinned dependencies, extracted from its lock files.
type securityScanner struct {
	projectRoot string
	baseURL     string
	httpClient  *http.Client
}

func newSecurityScanner(projectRoot string) *securityScanner {
	return &securityScanner{
		projectRoot: projectRoot,
		baseURL:     defaultOSVBaseURL,
		httpClient:  &http.Client{Timeout: scanHTTPTimeout},
	}
}

// handle extracts dependencies from a lock file and queries OSV.dev for known
// vulnerabilities, filtering to those at or above the severity threshold.
func (s *securityScanner) handle(ctx context.Context, _ *spi.ToolCallContext, req *spi.ToolRequest) (*spi.ToolResult, error) {
	threshold := strings.ToLower(toolutil.StringArgOr(req.Arguments, "severity_threshold", "medium"))
	if _, ok := severityRank[threshold]; !ok {
		return toolutil.NotConfigured("invalid severity_threshold",
			map[string]any{"got": threshold, "allowed": []string{"low", "medium", "high", "critical"}}), nil
	}

	pkgs, lockPath, err := s.collectDeps(req.Arguments)
	if err != nil {
		return toolutil.NotConfigured("could not read lock file",
			map[string]any{"path": lockPath, "error": err.Error()}), nil
	}
	if len(pkgs) == 0 {
		return toolutil.NotConfigured("no lock file with pinned dependencies found",
			map[string]any{
				"project_root": s.projectRoot,
				"remediation":  "generate a lock file (go.sum, package-lock.json, Cargo.lock, poetry.lock, uv.lock, or requirements.txt) then re-run the scan",
			}), nil
	}

	ctx, cancel := context.WithTimeout(ctx, scanHTTPTimeout)
	defer cancel()

	idsByQuery, err := s.queryBatch(ctx, pkgs)
	if err != nil {
		return toolutil.ErrorResult("OSV.dev batch query failed",
			map[string]any{"error": err.Error(), "dependencies": len(pkgs)}), nil
	}

	vulns := s.resolveVulns(ctx, pkgs, idsByQuery, threshold)
	structured := map[string]any{
		"lock_file":           lockPath,
		"ecosystem":           pkgs[0].Ecosystem,
		"dependencies":        len(pkgs),
		"severity_threshold":  threshold,
		"vulnerabilities":     vulns,
		"vulnerability_count": len(vulns),
	}
	text := fmt.Sprintf("security_scan: %d dependencies scanned; %d vulnerabilities at or above %q",
		len(pkgs), len(vulns), threshold)
	return toolutil.Result(text, structured), nil
}

// collectDeps resolves the dependency set either from an explicit manifest_path
// argument or by auto-detecting a lock file under the project root.
func (s *securityScanner) collectDeps(args map[string]any) ([]osvPackage, string, error) {
	if mp, ok := toolutil.StringArg(args, "manifest_path"); ok && mp != "" {
		lf, ok := lockFileForPath(mp)
		if !ok {
			return nil, mp, fmt.Errorf("unsupported lock file %q", mp)
		}
		pkgs, err := lf.parse(mp)
		return pkgs, mp, err
	}
	lf, path, ok := detectLockFile(s.projectRoot)
	if !ok {
		return nil, "", nil
	}
	pkgs, err := lf.parse(path)
	return pkgs, path, err
}

// osvBatchRequest / osvBatchResponse mirror the OSV.dev /v1/querybatch contract.
type osvBatchRequest struct {
	Queries []osvQuery `json:"queries"`
}

type osvQuery struct {
	Package osvQueryPackage `json:"package"`
	Version string          `json:"version"`
}

type osvQueryPackage struct {
	Name      string `json:"name"`
	Ecosystem string `json:"ecosystem"`
}

type osvBatchResponse struct {
	Results []struct {
		Vulns []struct {
			ID string `json:"id"`
		} `json:"vulns"`
	} `json:"results"`
}

// queryBatch POSTs the dependency set to /v1/querybatch and returns, per query
// index, the list of vulnerability ids OSV reported. The slice is index-aligned
// with pkgs so callers can map a vuln back to the package that triggered it.
func (s *securityScanner) queryBatch(ctx context.Context, pkgs []osvPackage) ([][]string, error) {
	reqBody := osvBatchRequest{Queries: make([]osvQuery, len(pkgs))}
	for i, p := range pkgs {
		reqBody.Queries[i] = osvQuery{
			Package: osvQueryPackage{Name: p.Name, Ecosystem: p.Ecosystem},
			Version: p.Version,
		}
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling OSV batch query: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/v1/querybatch", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("building OSV request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("calling OSV: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OSV returned status %d", resp.StatusCode)
	}

	var parsed osvBatchResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decoding OSV response: %w", err)
	}

	out := make([][]string, len(pkgs))
	for i := range pkgs {
		if i >= len(parsed.Results) {
			break
		}
		for _, v := range parsed.Results[i].Vulns {
			out[i] = append(out[i], v.ID)
		}
	}
	return out, nil
}

// vulnReport is one reported vulnerability tied to the package that triggered it.
type vulnReport struct {
	ID          string `json:"id"`
	Package     string `json:"package"`
	Version     string `json:"version"`
	Ecosystem   string `json:"ecosystem"`
	Severity    string `json:"severity"`
	FixedIn     string `json:"fixed_in,omitempty"`
	AdvisoryURL string `json:"advisory_url,omitempty"`
	Summary     string `json:"summary,omitempty"`
}

// osvVuln models the subset of an OSV /v1/vulns/{id} record we surface.
type osvVuln struct {
	ID       string `json:"id"`
	Summary  string `json:"summary"`
	Details  string `json:"details"`
	Severity []struct {
		Type  string `json:"type"`
		Score string `json:"score"`
	} `json:"severity"`
	Affected []struct {
		Ranges []struct {
			Events []struct {
				Fixed string `json:"fixed"`
			} `json:"events"`
		} `json:"ranges"`
	} `json:"affected"`
	References []struct {
		Type string `json:"type"`
		URL  string `json:"url"`
	} `json:"references"`
	DatabaseSpecific struct {
		Severity string `json:"severity"`
	} `json:"database_specific"`
}

// resolveVulns fetches details for each unique vulnerability id, maps it back to
// the originating package, applies the severity threshold, and returns the
// filtered, deterministically-ordered report list.
func (s *securityScanner) resolveVulns(ctx context.Context, pkgs []osvPackage, idsByQuery [][]string, threshold string) []vulnReport {
	floor := severityRank[threshold]
	details := s.fetchDetails(ctx, idsByQuery)

	var reports []vulnReport
	for i, ids := range idsByQuery {
		for _, id := range ids {
			v := details[id]
			sev := vulnSeverity(v)
			if rank, ok := severityRank[sev]; ok && rank < floor {
				continue // known and below the requested floor
			}
			reports = append(reports, vulnReport{
				ID:          id,
				Package:     pkgs[i].Name,
				Version:     pkgs[i].Version,
				Ecosystem:   pkgs[i].Ecosystem,
				Severity:    sev,
				FixedIn:     fixedVersion(v),
				AdvisoryURL: advisoryURL(v),
				Summary:     v.Summary,
			})
		}
	}
	sort.Slice(reports, func(a, b int) bool {
		if reports[a].Package != reports[b].Package {
			return reports[a].Package < reports[b].Package
		}
		return reports[a].ID < reports[b].ID
	})
	return reports
}

// fetchDetails retrieves the OSV record for every unique vulnerability id (up to
// maxVulnDetailFetches), returning a map keyed by id. A failed or missing fetch
// leaves a zero-value record so the vuln is still reported with unknown severity.
func (s *securityScanner) fetchDetails(ctx context.Context, idsByQuery [][]string) map[string]osvVuln {
	unique := make(map[string]struct{})
	for _, ids := range idsByQuery {
		for _, id := range ids {
			unique[id] = struct{}{}
		}
	}

	details := make(map[string]osvVuln, len(unique))
	fetched := 0
	for id := range unique {
		if fetched >= maxVulnDetailFetches {
			break
		}
		fetched++
		if v, err := s.fetchVuln(ctx, id); err == nil {
			details[id] = v
		} else {
			details[id] = osvVuln{ID: id}
		}
	}
	return details
}

// fetchVuln GETs a single /v1/vulns/{id} record.
func (s *securityScanner) fetchVuln(ctx context.Context, id string) (osvVuln, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+"/v1/vulns/"+id, nil)
	if err != nil {
		return osvVuln{}, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return osvVuln{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return osvVuln{}, fmt.Errorf("OSV vuln %s status %d", id, resp.StatusCode)
	}
	var v osvVuln
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return osvVuln{}, err
	}
	return v, nil
}

// vulnSeverity resolves a coarse severity label from the OSV record, preferring
// the database_specific.severity string and falling back to "unknown".
func vulnSeverity(v osvVuln) string {
	if s := strings.ToLower(strings.TrimSpace(v.DatabaseSpecific.Severity)); s != "" {
		if _, ok := severityRank[s]; ok {
			return s
		}
	}
	return "unknown"
}

// fixedVersion returns the first "fixed" event found across the affected ranges.
func fixedVersion(v osvVuln) string {
	for _, a := range v.Affected {
		for _, r := range a.Ranges {
			for _, e := range r.Events {
				if e.Fixed != "" {
					return e.Fixed
				}
			}
		}
	}
	return ""
}

// advisoryURL returns the canonical advisory link, preferring a reference typed
// ADVISORY and falling back to the first reference.
func advisoryURL(v osvVuln) string {
	for _, r := range v.References {
		if strings.EqualFold(r.Type, "ADVISORY") {
			return r.URL
		}
	}
	if len(v.References) > 0 {
		return v.References[0].URL
	}
	return ""
}
