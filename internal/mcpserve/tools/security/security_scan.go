package security

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools/toolutil"
	"github.com/Quantum-Serendipity/qsdev/internal/vulnscan"
)

// securityScanner queries OSV.dev for vulnerabilities affecting the project's
// pinned dependencies, extracted from its lock files. Lock-file parsing, the OSV
// client and the scan pipeline (including its deadline) live in
// internal/vulnscan; this type layers the MCP tool contract (manifest-path
// confinement, severity-threshold filtering, and structured graceful
// degradation) on top of vulnscan.Scanner.ScanFile.
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
	// Validate through the SAME shared mapping the CLI scan and scanned advisories
	// use, rather than a parallel accept-list that could drift from it. A ranked
	// floor ("low"/"moderate"/"high"/"critical", with "medium" an alias of
	// "moderate") normalizes to one of those buckets; anything unrecognized
	// normalizes to "info", and an empty value to "unknown" — both rejected here.
	threshold := vulnscan.NormalizeSeverity(rawThreshold)
	if threshold == "info" || threshold == "unknown" {
		return toolutil.NotConfigured("invalid severity_threshold",
			map[string]any{"got": rawThreshold, "allowed": []string{"low", "moderate", "high", "critical", "medium (alias of moderate)"}}), nil
	}

	res, lockPath, err := s.scan(ctx, req.Arguments)
	switch {
	case errors.Is(err, errManifestEscapesRoot):
		return toolutil.NotConfigured("manifest_path escapes the project root",
			map[string]any{"manifest_path": lockPath, "project_root": s.projectRoot}), nil
	case errors.Is(err, vulnscan.ErrLockParse), errors.Is(err, errUnsupportedLockFile):
		// The lock file was located but could not be parsed (truncated, tampered,
		// or an unsupported format). Distinguish this from "no lock file" below so
		// a caller does not misread a broken lock file as a clean (zero-vuln) scan.
		return toolutil.NotConfigured("lock file present but could not be parsed; scan did not run",
			map[string]any{
				"path":  lockPath,
				"error": err.Error(),
				"hint":  "the dependency scan did NOT complete — treat as unknown, not vulnerability-free",
			}), nil
	case errors.Is(err, vulnscan.ErrNoPinnedDeps), err == nil && res == nil:
		return toolutil.NotConfigured("no lock file with pinned dependencies found",
			map[string]any{
				"project_root": s.projectRoot,
				"remediation":  "generate a lock file (go.sum, package-lock.json, Cargo.lock, poetry.lock, uv.lock, or requirements.txt) then re-run the scan",
			}), nil
	case err != nil:
		return toolutil.ErrorResult("OSV.dev vulnerability query failed",
			map[string]any{"error": err.Error(), "lock_file": lockPath}), nil
	}

	vulns := filterVulns(res.Vulnerabilities, threshold)
	structured := map[string]any{
		"lock_file":           res.LockFile,
		"ecosystem":           res.Ecosystem,
		"dependencies":        res.Dependencies,
		"severity_threshold":  threshold,
		"vulnerabilities":     vulns,
		"vulnerability_count": len(vulns),
	}
	text := fmt.Sprintf("security_scan: %d dependencies scanned; %d vulnerabilities at or above %q",
		res.Dependencies, len(vulns), threshold)
	return toolutil.Result(text, structured), nil
}

// errManifestEscapesRoot is returned by scan when a caller-supplied
// manifest_path resolves outside the project root; handle degrades it to a
// not_configured result rather than reading the out-of-tree file.
var errManifestEscapesRoot = errors.New("manifest_path escapes the project root")

// errUnsupportedLockFile is returned by scan when a caller-supplied
// manifest_path names a file that is not a lock format the scanner can read.
var errUnsupportedLockFile = errors.New("unsupported lock file")

// scan resolves the lock file either from an explicit manifest_path argument or
// by auto-detecting one under the project root, and scans it. It returns the
// lock file path it resolved (for reporting) alongside the scan outcome; a nil
// Result with a nil error means no lock file was found.
func (s *securityScanner) scan(ctx context.Context, args map[string]any) (*vulnscan.Result, string, error) {
	if mp, ok := toolutil.StringArg(args, "manifest_path"); ok && mp != "" {
		// Confine a caller-supplied manifest_path to the project root. Without
		// this the tool would open (and report dependency coordinates from) any
		// lock-file-named path on the host (path traversal).
		resolved, ok := toolutil.ConfineToRoot(s.projectRoot, mp)
		if !ok {
			return nil, mp, errManifestEscapesRoot
		}
		if _, ok := vulnscan.LockFileForPath(resolved); !ok {
			return nil, resolved, fmt.Errorf("%w %q", errUnsupportedLockFile, mp)
		}
		res, err := s.scanner.ScanFile(ctx, resolved)
		return res, resolved, err
	}
	_, path, ok := vulnscan.DetectLockFile(s.projectRoot)
	if !ok {
		return nil, "", nil
	}
	res, err := s.scanner.ScanFile(ctx, path)
	return res, path, err
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

// filterVulns applies the severity threshold to the scanner's (already
// deterministically ordered) findings and renders them in the tool's report
// shape. A detail record that failed to fetch (or was dropped by the fetch cap)
// carries the "unknown" severity — always above the floor, so the vuln is
// reported rather than silently dropped or given a fabricated severity.
func filterVulns(vulns []vulnscan.Vulnerability, threshold string) []vulnReport {
	var reports []vulnReport
	for _, v := range vulns {
		if !vulnscan.SeverityAtOrAbove(v.Severity, threshold) {
			continue // below the requested floor ("unknown" always qualifies)
		}
		reports = append(reports, vulnReport{
			ID:          v.ID,
			Package:     v.Package,
			Version:     v.Version,
			Ecosystem:   v.Ecosystem,
			Severity:    v.Severity,
			FixedIn:     v.FixedIn,
			AdvisoryURL: v.AdvisoryURL,
			Summary:     v.Summary,
		})
	}
	return reports
}
