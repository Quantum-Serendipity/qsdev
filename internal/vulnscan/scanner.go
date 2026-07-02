// Package vulnscan scans a project's pinned dependencies for known
// vulnerabilities by querying the OSV.dev database. It extracts dependency
// coordinates from lock files (go.sum, package-lock.json, Cargo.lock,
// poetry.lock, uv.lock, Pipfile.lock, requirements.txt), batches them to
// OSV.dev, and aggregates the advisories by severity.
//
// The OSV interaction is a self-contained copy of the logic in the mcpserve
// security_scan tool so that callers such as internal/posture can run a real
// dependency scan without importing the MCP tool tree. The two copies are
// intentionally duplicated for now; consolidating them is a follow-up cleanup.
package vulnscan

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"
)

// DefaultOSVBaseURL is the OSV.dev REST API root. It is the default value of
// Scanner.BaseURL; tests point BaseURL at a local httptest server instead.
const DefaultOSVBaseURL = "https://api.osv.dev"

// scanHTTPTimeout bounds the whole OSV interaction (batch query plus detail
// fetches) so a slow or unreachable OSV endpoint cannot stall the scan.
const scanHTTPTimeout = 20 * time.Second

// maxVulnDetailFetches caps how many unique vulnerability records the scanner
// fetches full details for, bounding fan-out on a heavily-vulnerable project.
const maxVulnDetailFetches = 200

// SeverityCounts aggregates vulnerabilities by normalized severity. An advisory
// whose severity cannot be resolved to one of the named buckets is counted as
// Info so it remains visible in Total without being overstated as critical.
type SeverityCounts struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Moderate int `json:"moderate"`
	Low      int `json:"low"`
	Info     int `json:"info"`
}

// Total returns the sum of all severity buckets.
func (c SeverityCounts) Total() int {
	return c.Critical + c.High + c.Moderate + c.Low + c.Info
}

// Vulnerability is one advisory tied to the package that triggered it.
type Vulnerability struct {
	ID          string `json:"id"`
	Package     string `json:"package"`
	Version     string `json:"version"`
	Ecosystem   string `json:"ecosystem"`
	Severity    string `json:"severity"` // normalized: critical|high|moderate|low|info
	FixedIn     string `json:"fixedIn,omitempty"`
	AdvisoryURL string `json:"advisoryUrl,omitempty"`
	Summary     string `json:"summary,omitempty"`
}

// Result is the outcome of scanning a single lock file.
type Result struct {
	LockFile        string          `json:"lockFile"`
	Ecosystem       string          `json:"ecosystem"`
	Dependencies    int             `json:"dependencies"`
	Vulnerabilities []Vulnerability `json:"vulnerabilities"`
	Counts          SeverityCounts  `json:"counts"`
}

// Scanner queries OSV.dev for vulnerabilities affecting pinned dependencies.
// BaseURL and HTTPClient are exported so callers (and tests) can point the
// scanner at an alternate endpoint or supply a custom client; the zero value is
// not usable — construct one with New.
type Scanner struct {
	BaseURL    string
	HTTPClient *http.Client
}

// New returns a Scanner configured against the public OSV.dev endpoint with a
// bounded HTTP timeout.
func New() *Scanner {
	return &Scanner{
		BaseURL:    DefaultOSVBaseURL,
		HTTPClient: &http.Client{Timeout: scanHTTPTimeout},
	}
}

// client returns the configured HTTP client, defaulting to a bounded one so a
// zero-value or partially-constructed Scanner never dereferences a nil client.
func (s *Scanner) client() *http.Client {
	if s.HTTPClient != nil {
		return s.HTTPClient
	}
	return &http.Client{Timeout: scanHTTPTimeout}
}

// baseURL returns the configured OSV base URL, defaulting to the public
// endpoint.
func (s *Scanner) baseURL() string {
	if s.BaseURL != "" {
		return s.BaseURL
	}
	return DefaultOSVBaseURL
}

// ScanProject auto-detects the first supported lock file under projectRoot and
// scans it. It returns (nil, nil) when no supported lock file is present.
func (s *Scanner) ScanProject(ctx context.Context, projectRoot string) (*Result, error) {
	lf, path, ok := detectLockFile(projectRoot)
	if !ok {
		return nil, nil
	}
	return s.scan(ctx, lf, path)
}

// ScanFile scans a single lock file at lockPath, resolving the parser and OSV
// ecosystem from the file's base name. It returns (nil, nil) when the file is
// not a lock format with OSV coverage — the caller then knows the ecosystem was
// not scanned rather than confirmed vulnerability-free.
func (s *Scanner) ScanFile(ctx context.Context, lockPath string) (*Result, error) {
	lf, ok := lockFileForPath(lockPath)
	if !ok {
		return nil, nil
	}
	return s.scan(ctx, lf, lockPath)
}

// scan parses the lock file, queries OSV for the extracted dependencies, and
// aggregates the advisories into a Result.
func (s *Scanner) scan(ctx context.Context, lf lockFile, path string) (*Result, error) {
	pkgs, err := lf.parse(path)
	if err != nil {
		return nil, fmt.Errorf("parsing lock file %q: %w", path, err)
	}
	res := &Result{
		LockFile:     path,
		Ecosystem:    lf.ecosystem,
		Dependencies: len(pkgs),
	}
	if len(pkgs) == 0 {
		return res, nil
	}

	ctx, cancel := context.WithTimeout(ctx, scanHTTPTimeout)
	defer cancel()

	idsByQuery, err := s.queryBatch(ctx, pkgs)
	if err != nil {
		return nil, fmt.Errorf("querying OSV: %w", err)
	}

	res.Vulnerabilities = s.resolveVulns(ctx, pkgs, idsByQuery)
	for _, v := range res.Vulnerabilities {
		switch v.Severity {
		case "critical":
			res.Counts.Critical++
		case "high":
			res.Counts.High++
		case "moderate":
			res.Counts.Moderate++
		case "low":
			res.Counts.Low++
		default:
			res.Counts.Info++
		}
	}
	return res, nil
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
// index, the vulnerability ids OSV reported. The slice is index-aligned with
// pkgs so callers can map a vuln back to the package that triggered it.
func (s *Scanner) queryBatch(ctx context.Context, pkgs []Package) ([][]string, error) {
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

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL()+"/v1/querybatch", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("building OSV request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.client().Do(httpReq)
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
// the originating package, normalizes its severity, and returns the report list
// in deterministic order.
func (s *Scanner) resolveVulns(ctx context.Context, pkgs []Package, idsByQuery [][]string) []Vulnerability {
	details := s.fetchDetails(ctx, idsByQuery)

	var reports []Vulnerability
	for i, ids := range idsByQuery {
		for _, id := range ids {
			v := details[id]
			reports = append(reports, Vulnerability{
				ID:          id,
				Package:     pkgs[i].Name,
				Version:     pkgs[i].Version,
				Ecosystem:   pkgs[i].Ecosystem,
				Severity:    normalizeSeverity(v),
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
// maxVulnDetailFetches), returning a map keyed by id. Ids are sorted before
// truncation so the fetched subset is deterministic. A failed fetch or an id
// dropped by truncation leaves a zero-value record, which normalizeSeverity maps
// to "info" — the vuln is still reported, never silently dropped.
func (s *Scanner) fetchDetails(ctx context.Context, idsByQuery [][]string) map[string]osvVuln {
	unique := make(map[string]struct{})
	for _, ids := range idsByQuery {
		for _, id := range ids {
			unique[id] = struct{}{}
		}
	}

	ids := make([]string, 0, len(unique))
	for id := range unique {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	if len(ids) > maxVulnDetailFetches {
		slog.Warn("vulnscan: vulnerability detail fetch truncated; dropped vulns reported with unknown severity",
			"unique_vulns", len(ids), "fetch_limit", maxVulnDetailFetches)
		ids = ids[:maxVulnDetailFetches]
	}

	details := make(map[string]osvVuln, len(ids))
	for _, id := range ids {
		if v, err := s.fetchVuln(ctx, id); err == nil {
			details[id] = v
		} else {
			details[id] = osvVuln{ID: id}
		}
	}
	return details
}

// fetchVuln GETs a single /v1/vulns/{id} record.
func (s *Scanner) fetchVuln(ctx context.Context, id string) (osvVuln, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL()+"/v1/vulns/"+id, nil)
	if err != nil {
		return osvVuln{}, err
	}
	resp, err := s.client().Do(req)
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

// normalizeSeverity resolves a coarse severity bucket from the OSV record's
// database_specific.severity label. "medium" is folded into "moderate" (OSV
// GHSA advisories use MODERATE while some sources emit MEDIUM). An absent or
// unrecognized label yields "info" so the vuln stays visible without being
// overstated.
func normalizeSeverity(v osvVuln) string {
	switch strings.ToLower(strings.TrimSpace(v.DatabaseSpecific.Severity)) {
	case "critical":
		return "critical"
	case "high":
		return "high"
	case "moderate", "medium":
		return "moderate"
	case "low":
		return "low"
	default:
		return "info"
	}
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
