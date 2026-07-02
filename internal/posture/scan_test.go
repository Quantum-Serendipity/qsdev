package posture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/vulnscan"
	"github.com/Quantum-Serendipity/qsdev/internal/vulnscan/vulnscantest"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

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
