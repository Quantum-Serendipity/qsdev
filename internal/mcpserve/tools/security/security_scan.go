package security

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
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

// handle extracts dependencies from the project's lock files (one per
// ecosystem, or the explicit manifest_path) and queries OSV.dev for known
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

	scan, err := s.scan(ctx, req.Arguments)
	switch {
	case errors.Is(err, errManifestEscapesRoot):
		return toolutil.NotConfigured("manifest_path escapes the project root",
			map[string]any{"manifest_path": toolutil.StringArgOr(req.Arguments, "manifest_path", ""), "project_root": s.projectRoot}), nil
	case err != nil:
		return toolutil.ErrorResult("OSV.dev vulnerability query failed",
			map[string]any{"error": err.Error(), "lock_file": scan.failedLockFile}), nil
	case len(scan.results) == 0 && len(scan.unscanned) > 0:
		// Every lock file was located but none could be parsed (truncated,
		// tampered, or an unsupported format). Distinguish this from "no lock
		// file" below so a caller does not misread a broken lock file as a clean
		// (zero-vuln) scan.
		return toolutil.NotConfigured("lock file present but could not be parsed; scan did not run",
			map[string]any{
				"path":                 scan.unscanned[0].Path,
				"error":                scan.unscanned[0].Error,
				"unscanned_lock_files": scan.unscanned,
				"hint":                 "the dependency scan did NOT complete — treat as unknown, not vulnerability-free",
			}), nil
	case len(scan.results) == 0:
		return toolutil.NotConfigured("no lock file with pinned dependencies found",
			map[string]any{
				"project_root": s.projectRoot,
				"remediation":  "generate a lock file (go.sum, package-lock.json, Cargo.lock, poetry.lock, uv.lock, or requirements.txt) then re-run the scan",
			}), nil
	}

	var (
		all          []vulnscan.Vulnerability
		lockFiles    []string
		dependencies int
	)
	ecosystems := make(map[string]bool)
	for _, res := range scan.results {
		all = append(all, res.Vulnerabilities...)
		lockFiles = append(lockFiles, res.LockFile)
		dependencies += res.Dependencies
		ecosystems[res.Ecosystem] = true
	}
	vulns := filterVulns(all, threshold)
	coverage := "complete"
	if len(scan.unscanned) > 0 {
		coverage = "partial"
	}
	structured := map[string]any{
		"lock_files":          lockFiles,
		"ecosystems":          slices.Sorted(maps.Keys(ecosystems)),
		"coverage":            coverage,
		"dependencies":        dependencies,
		"severity_threshold":  threshold,
		"vulnerabilities":     vulns,
		"vulnerability_count": len(vulns),
	}
	text := fmt.Sprintf("security_scan: %d dependencies scanned from %d lock file(s); %d vulnerabilities at or above %q",
		dependencies, len(lockFiles), len(vulns), threshold)
	if len(scan.unscanned) > 0 {
		structured["unscanned_lock_files"] = scan.unscanned
		text += fmt.Sprintf("; PARTIAL coverage: %d lock file(s) could not be parsed and were not scanned", len(scan.unscanned))
	}
	return toolutil.Result(text, structured), nil
}

// errManifestEscapesRoot is returned by scan when a caller-supplied
// manifest_path resolves outside the project root; handle degrades it to a
// not_configured result rather than reading the out-of-tree file.
var errManifestEscapesRoot = errors.New("manifest_path escapes the project root")

// unscannedLockFile reports a located lock file the scan could not read.
type unscannedLockFile struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}

// scanOutcome is what a scan covered: one vulnscan.Result per scanned lock
// file, and every located lock file that could not be parsed. A lock file with
// no pinned dependencies contributes neither.
type scanOutcome struct {
	results   []*vulnscan.Result
	unscanned []unscannedLockFile
	// failedLockFile names the lock file whose OSV query failed, if any.
	failedLockFile string
}

// scan resolves the lock files either from an explicit manifest_path argument
// or by auto-detecting every ecosystem's preferred lock file under the project
// root (so a polyglot project is scanned in full rather than only its first
// ecosystem), and scans each through vulnscan.Scanner.ScanFile, which owns the
// deadline, chunking, validation and per-package FixedIn. A file that fails to
// parse (or is not a supported lock format) is recorded as unscanned rather
// than aborting; an OSV query failure aborts the scan.
func (s *securityScanner) scan(ctx context.Context, args map[string]any) (scanOutcome, error) {
	var paths []string
	if mp, ok := toolutil.StringArg(args, "manifest_path"); ok && mp != "" {
		// Confine a caller-supplied manifest_path to the project root. Without
		// this the tool would open (and report dependency coordinates from) any
		// lock-file-named path on the host (path traversal).
		resolved, ok := toolutil.ConfineToRoot(s.projectRoot, mp)
		if !ok {
			return scanOutcome{}, errManifestEscapesRoot
		}
		if _, ok := vulnscan.LockFileForPath(resolved); !ok {
			return scanOutcome{unscanned: []unscannedLockFile{{Path: resolved, Error: fmt.Sprintf("unsupported lock file %q", mp)}}}, nil
		}
		paths = []string{resolved}
	} else {
		for _, d := range vulnscan.DetectLockFiles(s.projectRoot) {
			paths = append(paths, d.Path)
		}
	}

	var out scanOutcome
	for _, path := range paths {
		res, err := s.scanner.ScanFile(ctx, path)
		switch {
		case errors.Is(err, vulnscan.ErrLockParse):
			out.unscanned = append(out.unscanned, unscannedLockFile{Path: path, Error: err.Error()})
		case errors.Is(err, vulnscan.ErrNoPinnedDeps):
			// Nothing pinned to scan in this file.
		case err != nil:
			out.failedLockFile = path
			return out, err
		case res != nil:
			out.results = append(out.results, res)
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
