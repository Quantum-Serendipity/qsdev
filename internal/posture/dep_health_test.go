package posture

import (
	"math"
	"testing"
)

// scanned returns a detected ecosystem whose lock file was scanned with the
// given vulnerability counts.
func scanned(name string, counts VulnSeverityCounts) EcosystemStatus {
	return EcosystemStatus{Name: name, Detected: true, LockFile: "valid", Scanned: true, VulnCounts: counts}
}

func TestComputeDepScore_Score(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		ecos []EcosystemStatus
		want float64
	}{
		{"zero vulns", []EcosystemStatus{scanned("go", VulnSeverityCounts{}), scanned("npm", VulnSeverityCounts{})}, 100},
		{"four criticals", []EcosystemStatus{scanned("go", VulnSeverityCounts{Critical: 4})}, 0},
		{"floor at zero", []EcosystemStatus{scanned("go", VulnSeverityCounts{Critical: 10})}, 0},
		// 100 - 25 - 20 - 9 - 5 = 41
		{"mixed severity", []EcosystemStatus{scanned("go", VulnSeverityCounts{Critical: 1, High: 2, Moderate: 3, Low: 5})}, 41},
		// 100 - 25 - 20 - 6 = 49
		{"multiple ecosystems", []EcosystemStatus{
			scanned("go", VulnSeverityCounts{Critical: 1, High: 1}),
			scanned("npm", VulnSeverityCounts{High: 1, Moderate: 2}),
		}, 49},
		{"only low", []EcosystemStatus{scanned("go", VulnSeverityCounts{Low: 10})}, 90},
		{"unknown severity at high rate", []EcosystemStatus{scanned("go", VulnSeverityCounts{Unknown: 1})}, 100 - DeductUnknown},
		{"missing lock file", []EcosystemStatus{
			scanned("go", VulnSeverityCounts{}),
			{Name: "npm", Detected: true, LockFile: "missing"},
		}, 100 - DeductMissingLock},
		{"two missing lock files", []EcosystemStatus{
			scanned("go", VulnSeverityCounts{}),
			{Name: "npm", Detected: true, LockFile: "missing"},
			{Name: "python", Detected: true, LockFile: "missing"},
		}, 100 - 2*DeductMissingLock},
		// 100 - 15 - 10 = 75
		{"missing lock and vulns", []EcosystemStatus{
			scanned("go", VulnSeverityCounts{High: 1}),
			{Name: "npm", Detected: true, LockFile: "missing"},
		}, 75},
		{"n/a lock file not penalized", []EcosystemStatus{
			scanned("go", VulnSeverityCounts{}),
			{Name: "shell", Detected: true, LockFile: "n/a"},
		}, 100},
		{"no ecosystems", nil, 100},
		{"only undetected ecosystems", []EcosystemStatus{{Name: "go", LockFile: "missing"}}, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ComputeDepScore(tt.ecos).Score
			if got == nil {
				t.Fatalf("score = nil, want %.1f", tt.want)
			}
			if math.Abs(*got-tt.want) > 0.001 {
				t.Errorf("score = %.1f, want %.1f", *got, tt.want)
			}
		})
	}
}

// TestComputeDepScore_UnscannedIsUnknown pins F328: dependencies that were
// never scanned have unknown health, so they score nil — never a clean 100 —
// and report the unscanned status.
func TestComputeDepScore_UnscannedIsUnknown(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		ecos []EcosystemStatus
	}{
		{"no scan requested", []EcosystemStatus{
			{Name: "go", Detected: true, LockFile: "go.sum"},
			{Name: "npm", Detected: true, LockFile: "package-lock.json"},
		}},
		{"only missing lock files", []EcosystemStatus{{Name: "go", Detected: true, LockFile: "missing"}}},
		{"lock format without OSV coverage", []EcosystemStatus{{Name: "shell", Detected: true, LockFile: "n/a"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ComputeDepScore(tt.ecos)
			if got.Score != nil {
				t.Errorf("score = %.1f, want nil (unknown)", *got.Score)
			}
			if got.Status != DepUnscanned || got.Scanned || got.ScanFailed {
				t.Errorf("status = %q scanned=%v scanFailed=%v, want %q false false",
					got.Status, got.Scanned, got.ScanFailed, DepUnscanned)
			}
		})
	}
}

func TestComputeDepScore_Status(t *testing.T) {
	t.Parallel()
	failed := EcosystemStatus{Name: "go", Detected: true, LockFile: "go.sum", ScanError: true}
	tests := []struct {
		name           string
		ecos           []EcosystemStatus
		want           DepScanStatus
		wantScanned    bool
		wantScanFailed bool
	}{
		{"none detected", []EcosystemStatus{{Name: "go"}}, DepNotApplicable, false, false},
		{"unscanned", []EcosystemStatus{{Name: "go", Detected: true, LockFile: "go.sum"}}, DepUnscanned, false, false},
		{"scanned", []EcosystemStatus{scanned("go", VulnSeverityCounts{})}, DepScanned, true, false},
		{"scanned with coverage gap", []EcosystemStatus{
			scanned("go", VulnSeverityCounts{}),
			{Name: "shell", Detected: true, LockFile: "n/a"},
		}, DepScanned, true, false},
		{"scan error", []EcosystemStatus{failed}, DepScanFailed, false, true},
		{"scan error beside a clean scan", []EcosystemStatus{scanned("npm", VulnSeverityCounts{}), failed}, DepScanFailed, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ComputeDepScore(tt.ecos)
			if got.Status != tt.want || got.Scanned != tt.wantScanned || got.ScanFailed != tt.wantScanFailed {
				t.Errorf("status = %q scanned=%v scanFailed=%v, want %q %v %v",
					got.Status, got.Scanned, got.ScanFailed, tt.want, tt.wantScanned, tt.wantScanFailed)
			}
			if got.ScanStatus() != tt.want {
				t.Errorf("ScanStatus() = %q, want %q", got.ScanStatus(), tt.want)
			}
		})
	}
}

// TestComputeDepScore_ScanErrorIsNotClean pins that an ecosystem whose scan
// errored does not score as a clean 100 on the strength of its zero counts.
func TestComputeDepScore_ScanErrorIsNotClean(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		ecos []EcosystemStatus
		want float64
	}{
		{
			name: "one failed ecosystem",
			ecos: []EcosystemStatus{{Name: "go", Detected: true, LockFile: "go.sum", ScanError: true}},
			want: 100 - DeductScanError,
		},
		{
			name: "failed and clean ecosystems",
			ecos: []EcosystemStatus{
				{Name: "go", Detected: true, LockFile: "go.sum", ScanError: true},
				{Name: "npm", Detected: true, LockFile: "package-lock.json", Scanned: true},
			},
			want: 100 - DeductScanError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ComputeDepScore(tt.ecos).Score
			if got == nil || *got != tt.want {
				t.Errorf("score = %v, want %.1f", got, tt.want)
			}
		})
	}
}

// TestDependencyHealth_ScanStatusLegacy pins that a report written before the
// status field existed derives its status from the legacy flags.
func TestDependencyHealth_ScanStatusLegacy(t *testing.T) {
	t.Parallel()
	detected := []EcosystemStatus{{Name: "go", Detected: true, LockFile: "go.sum"}}
	tests := []struct {
		name string
		deps DependencyHealth
		want DepScanStatus
	}{
		{"explicit status wins", DependencyHealth{Status: DepUnscanned, Scanned: true}, DepUnscanned},
		{"scan failed", DependencyHealth{ScanFailed: true, Ecosystems: detected}, DepScanFailed},
		{"scanned", DependencyHealth{Scanned: true, Ecosystems: detected}, DepScanned},
		{"unscanned", DependencyHealth{Ecosystems: detected}, DepUnscanned},
		{"no ecosystems", DependencyHealth{}, DepNotApplicable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.deps.ScanStatus(); got != tt.want {
				t.Errorf("ScanStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestComputeDepScore_EcosystemsPreserved(t *testing.T) {
	t.Parallel()
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
