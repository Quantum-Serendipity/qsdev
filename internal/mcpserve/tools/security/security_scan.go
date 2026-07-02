package security

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools/toolutil"
	"github.com/Quantum-Serendipity/qsdev/internal/vulnscan"
)

// scanHTTPTimeout bounds the whole OSV interaction (batch query plus detail
// fetches) so a slow or unreachable OSV endpoint cannot stall the tool past its
// external-API budget.
const scanHTTPTimeout = 20 * time.Second

// severityRank orders the threshold levels so a vuln can be compared against the
// requested floor. Unknown severities are always reported (rank -1 < any floor).
var severityRank = map[string]int{
	"low": 0, "medium": 1, "high": 2, "critical": 3,
}

// securityScanner queries OSV.dev for vulnerabilities affecting the project's
// pinned dependencies, extracted from its lock files. Lock-file parsing and the
// OSV client live in internal/vulnscan; this type layers the MCP tool contract
// (manifest-path confinement, severity-threshold filtering, and structured
// graceful degradation) on top of that single implementation.
type securityScanner struct {
	projectRoot string
	scanner     *vulnscan.Scanner
}

func newSecurityScanner(projectRoot string) *securityScanner {
	return &securityScanner{
		projectRoot: projectRoot,
		scanner:     vulnscan.New(),
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
	if errors.Is(err, errManifestEscapesRoot) {
		return toolutil.NotConfigured("manifest_path escapes the project root",
			map[string]any{"manifest_path": lockPath, "project_root": s.projectRoot}), nil
	}
	if err != nil {
		// The lock file was located but could not be parsed (truncated, tampered,
		// or an unsupported format). Distinguish this from "no lock file" below so
		// a caller does not misread a broken lock file as a clean (zero-vuln) scan.
		return toolutil.NotConfigured("lock file present but could not be parsed; scan did not run",
			map[string]any{
				"path":  lockPath,
				"error": err.Error(),
				"hint":  "the dependency scan did NOT complete — treat as unknown, not vulnerability-free",
			}), nil
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

	idsByQuery, err := s.scanner.QueryBatch(ctx, pkgs)
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

// errManifestEscapesRoot is returned by collectDeps when a caller-supplied
// manifest_path resolves outside the project root; handle degrades it to a
// not_configured result rather than reading the out-of-tree file.
var errManifestEscapesRoot = errors.New("manifest_path escapes the project root")

// collectDeps resolves the dependency set either from an explicit manifest_path
// argument or by auto-detecting a lock file under the project root.
func (s *securityScanner) collectDeps(args map[string]any) ([]vulnscan.Package, string, error) {
	if mp, ok := toolutil.StringArg(args, "manifest_path"); ok && mp != "" {
		// Confine a caller-supplied manifest_path to the project root. Without
		// this the tool would open (and report dependency coordinates from) any
		// lock-file-named path on the host (path traversal).
		resolved, ok := toolutil.ConfineToRoot(s.projectRoot, mp)
		if !ok {
			return nil, mp, errManifestEscapesRoot
		}
		lf, ok := vulnscan.LockFileForPath(resolved)
		if !ok {
			return nil, resolved, fmt.Errorf("unsupported lock file %q", mp)
		}
		pkgs, err := lf.Parse(resolved)
		return pkgs, resolved, err
	}
	lf, path, ok := vulnscan.DetectLockFile(s.projectRoot)
	if !ok {
		return nil, "", nil
	}
	pkgs, err := lf.Parse(path)
	return pkgs, path, err
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

// resolveVulns fetches details for each unique vulnerability id, maps it back to
// the originating package, applies the severity threshold, and returns the
// filtered, deterministically-ordered report list. A detail record that failed
// to fetch (or was dropped by the fetch cap) carries an empty severity label,
// which maps to "unknown" — always above the floor, so the vuln is reported
// rather than silently dropped or given a fabricated severity.
func (s *securityScanner) resolveVulns(ctx context.Context, pkgs []vulnscan.Package, idsByQuery [][]string, threshold string) []vulnReport {
	floor := severityRank[threshold]
	details := s.scanner.FetchDetails(ctx, idsByQuery)

	var reports []vulnReport
	for i, ids := range idsByQuery {
		for _, id := range ids {
			d := details[id]
			sev := vulnSeverity(d.SeverityLabel)
			if rank, ok := severityRank[sev]; ok && rank < floor {
				continue // known and below the requested floor
			}
			reports = append(reports, vulnReport{
				ID:          id,
				Package:     pkgs[i].Name,
				Version:     pkgs[i].Version,
				Ecosystem:   pkgs[i].Ecosystem,
				Severity:    sev,
				FixedIn:     d.FixedIn,
				AdvisoryURL: d.AdvisoryURL,
				Summary:     d.Summary,
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

// vulnSeverity resolves a coarse severity label from the OSV record's raw
// database_specific.severity string, falling back to "unknown" for absent or
// unrecognized labels so they are never filtered out by the threshold.
func vulnSeverity(label string) string {
	if s := strings.ToLower(strings.TrimSpace(label)); s != "" {
		if _, ok := severityRank[s]; ok {
			return s
		}
	}
	return "unknown"
}
