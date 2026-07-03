package posture

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/vulnscan"
	"github.com/Quantum-Serendipity/qsdev/internal/vulnscan/vulnscantest"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// failingOSVServer returns a server that answers every request with HTTP 500,
// so vulnscan.ScanFile reports a scan failure (as when OSV.dev is unreachable).
func failingOSVServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// writeGoSum writes a minimal single-module go.sum into dir.
func writeGoSum(t *testing.T, dir string) {
	t.Helper()
	body := "example.com/mod v1.0.0 h1:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa=\n" +
		"example.com/mod v1.0.0/go.mod h1:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb=\n"
	if err := os.WriteFile(filepath.Join(dir, "go.sum"), []byte(body), 0o644); err != nil {
		t.Fatalf("write go.sum: %v", err)
	}
}

// TestBuildEcosystemStatuses_FreshScanCriticalDrivesExit is the core seam test:
// a fresh scan against a lock file with a known critical advisory must populate
// VulnCounts, so ComputeDepScore surfaces the critical in Totals and
// ShouldExitNonZero fires at the "critical" audit level.
func TestBuildEcosystemStatuses_FreshScanCriticalDrivesExit(t *testing.T) {
	dir := t.TempDir()
	writeGoSum(t, dir)

	// The stub reports a single critical advisory for the first queried
	// dependency (the sole go.sum module).
	srv := vulnscantest.NewServer(t,
		map[int][]string{0: {"GHSA-CRIT-001"}},
		map[string]string{"GHSA-CRIT-001": "CRITICAL"},
	)
	scanner := &vulnscan.Scanner{BaseURL: srv.URL, HTTPClient: srv.Client()}
	detected := types.DetectedProject{Ecosystems: map[string]bool{ecosystem.NameGo: true}}

	ecos := buildEcosystemStatuses(detected, dir, scanner)

	var goStatus *EcosystemStatus
	for i := range ecos {
		if ecos[i].Name == ecosystem.NameGo {
			goStatus = &ecos[i]
		}
	}
	if goStatus == nil {
		t.Fatal("go ecosystem status not built")
	}
	if goStatus.LockFile != "go.sum" {
		t.Errorf("LockFile = %q, want go.sum", goStatus.LockFile)
	}
	if !goStatus.Scanned {
		t.Error("Scanned should be true after a fresh scan")
	}
	if goStatus.LastScan == nil {
		t.Error("LastScan should be set after a fresh scan")
	}
	if goStatus.VulnCounts.Critical != 1 {
		t.Fatalf("VulnCounts.Critical = %d, want 1", goStatus.VulnCounts.Critical)
	}

	dep := ComputeDepScore(ecos)
	dep.Scanned = true
	report := &PostureReport{
		Dependencies: dep,
		Conformance:  ConformanceResult{Baseline: ConformanceLevel{Pass: true}},
	}
	if report.Dependencies.Totals.Critical == 0 {
		t.Fatalf("Dependencies.Totals.Critical = 0, want > 0")
	}
	if !ShouldExitNonZero(report, "critical") {
		t.Error("ShouldExitNonZero(critical) = false, want true with a scanned critical advisory")
	}
}

// TestBuildEcosystemStatuses_NoScannerLeavesUnscanned proves that without a
// scanner (no --scan) the ecosystem is not marked scanned and its counts stay
// zero — so a later reader knows the result is "unknown", not "clean".
func TestBuildEcosystemStatuses_NoScannerLeavesUnscanned(t *testing.T) {
	dir := t.TempDir()
	writeGoSum(t, dir)
	detected := types.DetectedProject{Ecosystems: map[string]bool{ecosystem.NameGo: true}}

	ecos := buildEcosystemStatuses(detected, dir, nil)
	if len(ecos) != 1 {
		t.Fatalf("got %d ecosystems, want 1", len(ecos))
	}
	if ecos[0].Scanned {
		t.Error("Scanned should be false when no scanner is provided")
	}
	if ecos[0].VulnCounts.Total() != 0 {
		t.Errorf("VulnCounts.Total = %d, want 0", ecos[0].VulnCounts.Total())
	}
	if ecos[0].LockFile != "go.sum" {
		t.Errorf("LockFile = %q, want go.sum", ecos[0].LockFile)
	}
}

// TestEvaluateBaseline_UnscannedDoesNotClaimClean asserts the honest-reporting
// rule: with no scan, the no-critical-vulns check must NOT read "no critical
// vulnerabilities" — it must state that dependencies were not scanned.
func TestEvaluateBaseline_UnscannedDoesNotClaimClean(t *testing.T) {
	deps := DependencyHealth{
		Scanned: false,
		Ecosystems: []EcosystemStatus{
			{Name: "go", Detected: true, LockFile: "go.sum"},
		},
	}
	checks := evaluateBaseline(DefenseCoverage{}, deps, types.GeneratedState{})

	reason := reasonFor(t, checks, CheckNoCriticalVulns)
	if reason == "no critical vulnerabilities" {
		t.Error("unscanned report must not claim 'no critical vulnerabilities'")
	}
	if !strings.Contains(reason, "not scanned") {
		t.Errorf("reason = %q, want it to state dependencies were not scanned", reason)
	}

	// Sanity: once scanned with zero criticals, the clean reason returns.
	deps.Scanned = true
	scanned := evaluateBaseline(DefenseCoverage{}, deps, types.GeneratedState{})
	if got := reasonFor(t, scanned, CheckNoCriticalVulns); got != "no critical vulnerabilities" {
		t.Errorf("scanned+clean reason = %q, want 'no critical vulnerabilities'", got)
	}
}

// TestAssess_FreshScanSetsScannedFlag proves that AssessOptions.FreshScan is now
// honored end-to-end: Assess marks Dependencies as scanned and stamps LastScan,
// and without FreshScan the conformance report states the scan was not run.
func TestAssess_FreshScanSetsScannedFlag(t *testing.T) {
	// Serial: this test substitutes the package-level scanner factory.
	orig := newVulnScanner
	t.Cleanup(func() { newVulnScanner = orig })
	srv := vulnscantest.NewServer(t, nil, nil) // reports no vulnerabilities
	newVulnScanner = func() *vulnscan.Scanner {
		return &vulnscan.Scanner{BaseURL: srv.URL, HTTPClient: srv.Client()}
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".qsdev.yaml"), []byte("version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	scanned, err := Assess(dir, AssessOptions{FreshScan: true})
	if err != nil {
		t.Fatalf("Assess(FreshScan): %v", err)
	}
	if !scanned.Dependencies.Scanned {
		t.Error("Dependencies.Scanned should be true when FreshScan is set")
	}
	if scanned.Dependencies.LastScan == nil {
		t.Error("Dependencies.LastScan should be stamped when FreshScan is set")
	}

	unscanned, err := Assess(dir, AssessOptions{FreshScan: false})
	if err != nil {
		t.Fatalf("Assess(no scan): %v", err)
	}
	if unscanned.Dependencies.Scanned {
		t.Error("Dependencies.Scanned should be false when FreshScan is not set")
	}
	reason := reasonFor(t, unscanned.Conformance.Baseline.Checks, CheckNoCriticalVulns)
	if !strings.Contains(reason, "not scanned") {
		t.Errorf("unscanned conformance reason = %q, want a 'not scanned' statement", reason)
	}
}

// TestBuildEcosystemStatuses_ScanFailureMarksScanError proves a scan that errors
// (OSV unreachable) is recorded as a failure, not silently left "unscanned with
// zero counts" — the seam that C1 depends on.
func TestBuildEcosystemStatuses_ScanFailureMarksScanError(t *testing.T) {
	dir := t.TempDir()
	writeGoSum(t, dir)
	srv := failingOSVServer(t)
	scanner := &vulnscan.Scanner{BaseURL: srv.URL, HTTPClient: srv.Client()}
	detected := types.DetectedProject{Ecosystems: map[string]bool{ecosystem.NameGo: true}}

	ecos := buildEcosystemStatuses(detected, dir, scanner)

	var goStatus *EcosystemStatus
	for i := range ecos {
		if ecos[i].Name == ecosystem.NameGo {
			goStatus = &ecos[i]
		}
	}
	if goStatus == nil {
		t.Fatal("go ecosystem status not built")
	}
	if !goStatus.ScanError {
		t.Error("ScanError should be true when the OSV query fails")
	}
	if goStatus.Scanned {
		t.Error("Scanned must stay false on a failed scan")
	}
	if goStatus.VulnCounts.Total() != 0 {
		t.Errorf("VulnCounts.Total = %d, want 0 on a failed scan", goStatus.VulnCounts.Total())
	}
}

// TestAssess_FreshScanFailureFailsClosed is the C1 regression: when --scan is
// requested but the scan errors, the report must NOT present a clean bill of
// health. It marks ScanFailed, the exit gate fails closed at every gating level,
// and conformance does not certify "no critical vulnerabilities".
func TestAssess_FreshScanFailureFailsClosed(t *testing.T) {
	orig := newVulnScanner
	t.Cleanup(func() { newVulnScanner = orig })
	srv := failingOSVServer(t)
	newVulnScanner = func() *vulnscan.Scanner {
		return &vulnscan.Scanner{BaseURL: srv.URL, HTTPClient: srv.Client()}
	}

	dir := t.TempDir()
	writeGoSum(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "go.mod"),
		[]byte("module example.com/x\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".qsdev.yaml"), []byte("version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	report, err := Assess(dir, AssessOptions{FreshScan: true})
	if err != nil {
		t.Fatalf("Assess(FreshScan): %v", err)
	}

	if !report.Dependencies.ScanFailed {
		t.Error("Dependencies.ScanFailed should be true when the requested scan errored")
	}
	if report.Dependencies.Scanned {
		t.Error("Dependencies.Scanned must be false when the scan failed")
	}

	// Fail closed at every vuln-gating level; never for 'none'.
	for _, lvl := range []string{"critical", "high", "moderate", "low", "info"} {
		if !ShouldExitNonZero(report, lvl) {
			t.Errorf("ShouldExitNonZero(%q) = false, want true on a failed scan", lvl)
		}
	}
	if ShouldExitNonZero(report, "none") {
		t.Error("ShouldExitNonZero(none) must stay false even on a failed scan")
	}

	// Conformance must not certify clean, and must say so.
	reason := reasonFor(t, report.Conformance.Baseline.Checks, CheckNoCriticalVulns)
	if !strings.Contains(reason, "scan failed") {
		t.Errorf("conformance reason = %q, want it to state the scan failed", reason)
	}
	for _, c := range report.Conformance.Baseline.Checks {
		if c.Name == CheckNoCriticalVulns && c.Pass {
			t.Error("CheckNoCriticalVulns must not pass when the scan failed")
		}
	}
}

// TestBuildEcosystemStatuses_UnknownSeverityFailsGate is the M2 regression: a
// vulnerability whose severity could not be resolved (empty label — what a
// failed or truncated detail fetch leaves) must count as Unknown, fail the exit
// gate, and drop the dependency score — never fold to Info and pass silently.
func TestBuildEcosystemStatuses_UnknownSeverityFailsGate(t *testing.T) {
	dir := t.TempDir()
	writeGoSum(t, dir)
	// The advisory id is reported by the batch query but carries no severity in
	// its detail record (absent from the severities map).
	srv := vulnscantest.NewServer(t,
		map[int][]string{0: {"GHSA-UNK-001"}},
		nil,
	)
	scanner := &vulnscan.Scanner{BaseURL: srv.URL, HTTPClient: srv.Client()}
	detected := types.DetectedProject{Ecosystems: map[string]bool{ecosystem.NameGo: true}}

	ecos := buildEcosystemStatuses(detected, dir, scanner)
	dep := ComputeDepScore(ecos)
	dep.Scanned = true

	if dep.Totals.Unknown != 1 {
		t.Fatalf("Totals.Unknown = %d, want 1", dep.Totals.Unknown)
	}
	if dep.Totals.Info != 0 {
		t.Errorf("Totals.Info = %d, want 0 (unknown severity must not fold to Info)", dep.Totals.Info)
	}
	if dep.Score >= 100 {
		t.Errorf("Score = %.0f, want < 100 for an unknown-severity vuln", dep.Score)
	}

	report := &PostureReport{
		Dependencies: dep,
		Conformance:  ConformanceResult{Baseline: ConformanceLevel{Pass: true}},
	}
	for _, lvl := range []string{"critical", "high", "moderate", "low"} {
		if !ShouldExitNonZero(report, lvl) {
			t.Errorf("ShouldExitNonZero(%q) = false, want true for an unknown-severity vuln", lvl)
		}
	}
}

// reasonFor returns the reason of the named conformance check.
func reasonFor(t *testing.T, checks []ConformanceCheck, name CheckName) string {
	t.Helper()
	for _, c := range checks {
		if c.Name == name {
			return c.Reason
		}
	}
	t.Fatalf("conformance check %q not found", name)
	return ""
}
