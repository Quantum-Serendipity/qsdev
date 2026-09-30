package security

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"path/filepath"
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

// handle scans every lock file under the project (or the explicit
// manifest_path) against OSV.dev and reports findings at or above the severity
// threshold, together with the scan state of each lock file found.
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
	case len(scan.results) == 0:
		return s.nothingScanned(scan), nil
	}
	return scanReport(scan, threshold), nil
}

// nothingScanned is the not_configured result of a scan that queried OSV for
// no lock file. A located-but-unscannable lock file is reported distinctly from
// "no lock file" so a caller does not misread it as a clean (zero-vuln) scan.
func (s *securityScanner) nothingScanned(scan scanOutcome) *spi.ToolResult {
	missed := scan.notScanned()
	if len(missed) == 0 {
		return toolutil.NotConfigured("no lock file with pinned dependencies found",
			map[string]any{
				"project_root":      s.projectRoot,
				"lock_files_status": scan.statuses,
				"remediation":       "generate a lock file (" + strings.Join(vulnscan.SupportedLockFileNames(), ", ") + ") then re-run the scan",
			})
	}
	reason := "lock file present but not scannable; scan did not run"
	if slices.ContainsFunc(missed, func(st lockFileStatus) bool { return st.State == stateParseError }) {
		reason = "lock file present but could not be parsed; scan did not run"
	}
	return toolutil.NotConfigured(reason,
		map[string]any{
			"path":                 missed[0].Path,
			"error":                missed[0].Error,
			"lock_files_status":    scan.statuses,
			"unscanned_lock_files": scan.unscanned(),
			"scannable_formats":    vulnscan.SupportedLockFileNames(),
			"hint":                 "the dependency scan did NOT complete — treat as unknown, not vulnerability-free",
		})
}

// scanReport renders a scan that queried OSV for at least one lock file.
// coverage is "complete" only when every lock file found was scanned or had
// nothing pinned in it; otherwise it is "partial" and the text names what was
// not scanned.
func scanReport(scan scanOutcome, threshold string) *spi.ToolResult {
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
	vulns := filterVulns(dedupeVulns(all), threshold)
	missed := scan.notScanned()
	coverage := "complete"
	if len(missed) > 0 {
		coverage = "partial"
	}
	structured := map[string]any{
		"lock_files":          lockFiles,
		"lock_files_status":   scan.statuses,
		"ecosystems":          slices.Sorted(maps.Keys(ecosystems)),
		"coverage":            coverage,
		"dependencies":        dependencies,
		"severity_threshold":  threshold,
		"vulnerabilities":     vulns,
		"vulnerability_count": len(vulns),
	}
	text := fmt.Sprintf("security_scan: %d dependencies scanned from %d lock file(s); %d vulnerabilities at or above %q",
		dependencies, len(lockFiles), len(vulns), threshold)
	if unpinned := scan.withState(stateNoPinnedDeps); len(unpinned) > 0 {
		text += "; no pinned dependencies in: " + describeStatuses(unpinned, false)
	}
	if len(missed) > 0 {
		structured["unscanned_lock_files"] = scan.unscanned()
		text += "; PARTIAL coverage — not scanned: " + describeStatuses(missed, true)
	}
	return toolutil.Result(text, structured)
}

// describeStatuses lists lock files by path, with their state when withState.
func describeStatuses(statuses []lockFileStatus, withState bool) string {
	parts := make([]string, len(statuses))
	for i, st := range statuses {
		parts[i] = st.Path
		if withState {
			parts[i] += " (" + strings.ReplaceAll(st.State, "_", " ") + ")"
		}
	}
	return strings.Join(parts, ", ")
}

// dedupeVulns drops repeats of the same advisory on the same package version,
// which arise when several lock files of one ecosystem pin the same package.
func dedupeVulns(vulns []vulnscan.Vulnerability) []vulnscan.Vulnerability {
	type key struct{ id, eco, pkg, ver string }
	seen := make(map[key]bool, len(vulns))
	out := vulns[:0:0]
	for _, v := range vulns {
		k := key{v.ID, v.Ecosystem, v.Package, v.Version}
		if !seen[k] {
			seen[k] = true
			out = append(out, v)
		}
	}
	return out
}

// errManifestEscapesRoot is returned by scan when a caller-supplied
// manifest_path resolves outside the project root; handle degrades it to a
// not_configured result rather than reading the out-of-tree file.
var errManifestEscapesRoot = errors.New("manifest_path escapes the project root")

// Lock file scan states. scanned and no_pinned_deps count toward complete
// coverage; parse_error and unsupported_format make a scan partial.
const (
	stateScanned      = "scanned"
	stateNoPinnedDeps = "no_pinned_deps"
	stateParseError   = "parse_error"
	stateUnsupported  = "unsupported_format"
)

// lockFileStatus is the scan state of one lock file found in the project.
// Path is relative to the project root, in slash form.
type lockFileStatus struct {
	Path      string `json:"path"`
	Ecosystem string `json:"ecosystem"`
	State     string `json:"state"`
	Error     string `json:"error,omitempty"`

	abs string // full path, for the back-compat unscanned_lock_files field
}

// unscannedLockFile reports a located lock file the scan could not read. It is
// kept for callers of the unscanned_lock_files field; lock_files_status
// supersedes it.
type unscannedLockFile struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}

// scanOutcome is what a scan covered: one vulnscan.Result per scanned lock
// file, and the state of every lock file found.
type scanOutcome struct {
	results  []*vulnscan.Result
	statuses []lockFileStatus
	// failedLockFile names the lock file whose OSV query failed, if any.
	failedLockFile string
}

// withState returns the statuses in the given state.
func (o scanOutcome) withState(state string) []lockFileStatus {
	var out []lockFileStatus
	for _, st := range o.statuses {
		if st.State == state {
			out = append(out, st)
		}
	}
	return out
}

// notScanned returns the lock files whose dependencies were not checked.
func (o scanOutcome) notScanned() []lockFileStatus {
	return slices.Concat(o.withState(stateParseError), o.withState(stateUnsupported))
}

// unscanned renders notScanned in the back-compat unscanned_lock_files shape.
func (o scanOutcome) unscanned() []unscannedLockFile {
	missed := o.notScanned()
	out := make([]unscannedLockFile, len(missed))
	for i, st := range missed {
		out[i] = unscannedLockFile{Path: st.abs, Error: st.Error}
	}
	return out
}

// scan resolves the lock files, either the explicit manifest_path argument or
// every catalog lock file vulnscan.EnumerateLockFiles finds under the project
// root, and scans each supported one through vulnscan.Scanner.ScanFile, which
// owns the deadline, chunking, validation and per-package FixedIn. Every file
// gets a status; only an OSV query failure aborts the scan.
func (s *securityScanner) scan(ctx context.Context, args map[string]any) (scanOutcome, error) {
	var targets []vulnscan.LockFileStatus
	if mp, ok := toolutil.StringArg(args, "manifest_path"); ok && mp != "" {
		// Confine a caller-supplied manifest_path to the project root. Without
		// this the tool would open (and report dependency coordinates from) any
		// lock-file-named path on the host (path traversal).
		resolved, ok := toolutil.ConfineToRoot(s.projectRoot, mp)
		if !ok {
			return scanOutcome{}, errManifestEscapesRoot
		}
		// An explicit path limits the scan, and its coverage, to that file.
		targets = []vulnscan.LockFileStatus{vulnscan.DescribeLockFile(resolved)}
	} else {
		targets = vulnscan.EnumerateLockFiles(s.projectRoot)
	}

	var out scanOutcome
	for _, target := range targets {
		st := lockFileStatus{Path: s.relPath(target.Path), Ecosystem: target.CatalogEcosystem, abs: target.Path}
		res, err := s.scanTarget(ctx, target)
		switch {
		case errors.Is(err, vulnscan.ErrLockParse):
			st.State, st.Error = stateParseError, err.Error()
		case errors.Is(err, vulnscan.ErrNoPinnedDeps):
			st.State = stateNoPinnedDeps
		case err != nil:
			out.failedLockFile = target.Path
			return out, err
		case res == nil:
			st.State, st.Error = stateUnsupported, "no vulnerability scanner for this lock file format"
		default:
			st.State = stateScanned
			out.results = append(out.results, res)
		}
		out.statuses = append(out.statuses, st)
	}
	return out, nil
}

// scanTarget scans one supported lock file; an unsupported one yields (nil, nil).
func (s *securityScanner) scanTarget(ctx context.Context, target vulnscan.LockFileStatus) (*vulnscan.Result, error) {
	if !target.Supported {
		return nil, nil
	}
	return s.scanner.ScanFile(ctx, target.Path)
}

// relPath renders path relative to the project root in slash form, or as given
// when it is not under the root.
func (s *securityScanner) relPath(path string) string {
	rel, err := filepath.Rel(s.projectRoot, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
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
