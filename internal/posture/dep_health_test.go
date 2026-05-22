package posture

import (
	"math"
	"testing"
)

func TestComputeDepScore_ZeroVulns(t *testing.T) {
	ecosystems := []EcosystemStatus{
		{Name: "go", Detected: true, LockFile: "valid"},
		{Name: "npm", Detected: true, LockFile: "valid"},
	}
	result := ComputeDepScore(ecosystems)
	if result.Score != 100.0 {
		t.Errorf("zero vulns: got %f, want 100.0", result.Score)
	}
	if result.Totals.Critical != 0 || result.Totals.High != 0 || result.Totals.Moderate != 0 || result.Totals.Low != 0 {
		t.Errorf("totals should be zero: %+v", result.Totals)
	}
}

func TestComputeDepScore_FourCriticals(t *testing.T) {
	ecosystems := []EcosystemStatus{
		{Name: "go", Detected: true, LockFile: "valid", VulnCounts: VulnSeverityCounts{Critical: 4}},
	}
	result := ComputeDepScore(ecosystems)
	// 100 - 4*25 = 0
	if result.Score != 0.0 {
		t.Errorf("4 criticals: got %f, want 0.0", result.Score)
	}
}

func TestComputeDepScore_FloorAtZero(t *testing.T) {
	ecosystems := []EcosystemStatus{
		{Name: "go", Detected: true, LockFile: "valid", VulnCounts: VulnSeverityCounts{Critical: 10}},
	}
	result := ComputeDepScore(ecosystems)
	// 100 - 10*25 = -150, floor at 0
	if result.Score != 0.0 {
		t.Errorf("floor at zero: got %f, want 0.0", result.Score)
	}
}

func TestComputeDepScore_MixedSeverity(t *testing.T) {
	ecosystems := []EcosystemStatus{
		{Name: "go", Detected: true, LockFile: "valid", VulnCounts: VulnSeverityCounts{
			Critical: 1, // -25
			High:     2, // -20
			Moderate: 3, // -9
			Low:      5, // -5
		}},
	}
	result := ComputeDepScore(ecosystems)
	// 100 - 25 - 20 - 9 - 5 = 41
	want := 41.0
	if math.Abs(result.Score-want) > 0.001 {
		t.Errorf("mixed severity: got %f, want %f", result.Score, want)
	}
	if result.Totals.Critical != 1 || result.Totals.High != 2 || result.Totals.Moderate != 3 || result.Totals.Low != 5 {
		t.Errorf("totals mismatch: %+v", result.Totals)
	}
}

func TestComputeDepScore_MissingLockFile(t *testing.T) {
	ecosystems := []EcosystemStatus{
		{Name: "go", Detected: true, LockFile: "missing"},
	}
	result := ComputeDepScore(ecosystems)
	// 100 - 15 = 85
	if result.Score != 85.0 {
		t.Errorf("missing lock file: got %f, want 85.0", result.Score)
	}
}

func TestComputeDepScore_MultipleMissingLockFiles(t *testing.T) {
	ecosystems := []EcosystemStatus{
		{Name: "go", Detected: true, LockFile: "missing"},
		{Name: "npm", Detected: true, LockFile: "missing"},
	}
	result := ComputeDepScore(ecosystems)
	// 100 - 15 - 15 = 70
	if result.Score != 70.0 {
		t.Errorf("two missing lock files: got %f, want 70.0", result.Score)
	}
}

func TestComputeDepScore_MissingLockAndVulns(t *testing.T) {
	ecosystems := []EcosystemStatus{
		{Name: "go", Detected: true, LockFile: "missing", VulnCounts: VulnSeverityCounts{High: 1}},
	}
	result := ComputeDepScore(ecosystems)
	// 100 - 15 - 10 = 75
	if result.Score != 75.0 {
		t.Errorf("missing lock + vulns: got %f, want 75.0", result.Score)
	}
}

func TestComputeDepScore_EmptyEcosystems(t *testing.T) {
	result := ComputeDepScore(nil)
	if result.Score != 100.0 {
		t.Errorf("empty: got %f, want 100.0", result.Score)
	}
}

func TestComputeDepScore_MultipleEcosystemsTotals(t *testing.T) {
	ecosystems := []EcosystemStatus{
		{Name: "go", Detected: true, LockFile: "valid", VulnCounts: VulnSeverityCounts{Critical: 1, High: 1}},
		{Name: "npm", Detected: true, LockFile: "valid", VulnCounts: VulnSeverityCounts{High: 1, Moderate: 2}},
	}
	result := ComputeDepScore(ecosystems)
	// Totals: critical=1, high=2, moderate=2
	// 100 - 25 - 20 - 6 = 49
	if result.Totals.Critical != 1 {
		t.Errorf("totals critical: got %d, want 1", result.Totals.Critical)
	}
	if result.Totals.High != 2 {
		t.Errorf("totals high: got %d, want 2", result.Totals.High)
	}
	if result.Totals.Moderate != 2 {
		t.Errorf("totals moderate: got %d, want 2", result.Totals.Moderate)
	}
	want := 49.0
	if math.Abs(result.Score-want) > 0.001 {
		t.Errorf("multi-ecosystem: got %f, want %f", result.Score, want)
	}
}

func TestComputeDepScore_OnlyLow(t *testing.T) {
	ecosystems := []EcosystemStatus{
		{Name: "go", Detected: true, LockFile: "valid", VulnCounts: VulnSeverityCounts{Low: 10}},
	}
	result := ComputeDepScore(ecosystems)
	// 100 - 10 = 90
	if result.Score != 90.0 {
		t.Errorf("only low: got %f, want 90.0", result.Score)
	}
}

func TestComputeDepScore_NALockFileNotPenalized(t *testing.T) {
	ecosystems := []EcosystemStatus{
		{Name: "go", Detected: true, LockFile: "go.sum"},
		{Name: "shell", Detected: true, LockFile: "n/a"},
	}
	result := ComputeDepScore(ecosystems)
	if result.Score != 100.0 {
		t.Errorf("n/a lock file should not deduct: got %f, want 100.0", result.Score)
	}
}

func TestComputeDepScore_EcosystemsPreserved(t *testing.T) {
	ecosystems := []EcosystemStatus{
		{Name: "go", Detected: true, LockFile: "valid"},
		{Name: "npm", Detected: true, LockFile: "missing"},
	}
	result := ComputeDepScore(ecosystems)
	if len(result.Ecosystems) != 2 {
		t.Errorf("ecosystems count: got %d, want 2", len(result.Ecosystems))
	}
	if result.Ecosystems[0].Name != "go" || result.Ecosystems[1].Name != "npm" {
		t.Errorf("ecosystems order not preserved")
	}
}
