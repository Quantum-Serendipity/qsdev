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

// acceptedThresholds is the set of severity_threshold values a caller may pass.
// "medium" is accepted as an alias for "moderate" (both normalize to "moderate"
// via vulnscan.NormalizeSeverity) so the tool's floor and a scanned advisory's
// severity are ranked by the SAME shared mapping the CLI scan uses.
var acceptedThresholds = map[string]bool{
	"low": true, "medium": true, "moderate": true, "high": true, "critical": true,
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
	rawThreshold := strings.ToLower(toolutil.StringArgOr(req.Arguments, "severity_threshold", "medium"))
	if !acceptedThresholds[rawThreshold] {
		return toolutil.NotConfigured("invalid severity_threshold",
			map[string]any{"got": rawThreshold, "allowed": []string{"low", "moderate", "high", "critical", "medium (alias of moderate)"}}), nil
	}
	// Normalize through the shared mapping so the floor is in the same vocabulary
	// as scanned advisories ("medium" -> "moderate").
	threshold := vulnscan.NormalizeSeverity(rawThreshold)

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
	details := s.scanner.FetchDetails(ctx, idsByQuery)

	var reports []vulnReport
	for i, ids := range idsByQuery {
		for _, id := range ids {
			d := details[id]
			sev := vulnscan.NormalizeSeverity(d.SeverityLabel)
			if !vulnscan.SeverityAtOrAbove(sev, threshold) {
				continue // below the requested floor ("unknown" always qualifies)
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
