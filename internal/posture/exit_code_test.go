package posture

import (
	"errors"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/posture/drift"
)

func TestShouldExitNonZero_None(t *testing.T) {
	report := &PostureReport{
		Dependencies: DependencyHealth{Totals: VulnSeverityCounts{Critical: 100}},
	}
	if ShouldExitNonZero(report, "none") {
		t.Error("audit-level 'none' should never return true")
	}
}

func TestShouldExitNonZero_Critical(t *testing.T) {
	tests := []struct {
		name     string
		critical int
		high     int
		want     bool
	}{
		{"no vulns", 0, 0, false},
		{"critical present", 1, 0, true},
		{"high only", 0, 5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := &PostureReport{
				Dependencies: DependencyHealth{
					Totals: VulnSeverityCounts{Critical: tt.critical, High: tt.high},
				},
				Conformance: ConformanceResult{
					Baseline: ConformanceLevel{Pass: true},
				},
			}
			got := ShouldExitNonZero(report, "critical")
			if got != tt.want {
				t.Errorf("ShouldExitNonZero = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldExitNonZero_High(t *testing.T) {
	tests := []struct {
		name         string
		critical     int
		high         int
		baselinePass bool
		want         bool
	}{
		{"no issues", 0, 0, true, false},
		{"critical", 1, 0, true, true},
		{"high", 0, 3, true, true},
		{"baseline fail", 0, 0, false, true},
		{"all bad", 1, 2, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := &PostureReport{
				Dependencies: DependencyHealth{
					Totals: VulnSeverityCounts{Critical: tt.critical, High: tt.high},
				},
				Conformance: ConformanceResult{
					Baseline: ConformanceLevel{Pass: tt.baselinePass},
				},
			}
			got := ShouldExitNonZero(report, "high")
			if got != tt.want {
				t.Errorf("ShouldExitNonZero = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldExitNonZero_Moderate(t *testing.T) {
	tests := []struct {
		name         string
		critical     int
		high         int
		moderate     int
		baselinePass bool
		want         bool
	}{
		{"no vulns", 0, 0, 0, true, false},
		{"moderate present", 0, 0, 1, true, true},
		{"high present", 0, 1, 0, true, true},
		{"critical present", 1, 0, 0, true, true},
		// moderate is stricter than high, so it keeps high's conformance gate.
		{"baseline fail", 0, 0, 0, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := &PostureReport{
				Dependencies: DependencyHealth{
					Totals: VulnSeverityCounts{
						Critical: tt.critical,
						High:     tt.high,
						Moderate: tt.moderate,
					},
				},
				Conformance: ConformanceResult{
					Baseline: ConformanceLevel{Pass: tt.baselinePass},
				},
			}
			got := ShouldExitNonZero(report, "moderate")
			if got != tt.want {
				t.Errorf("ShouldExitNonZero = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldExitNonZero_Low(t *testing.T) {
	tests := []struct {
		name         string
		low          int
		baselinePass bool
		want         bool
	}{
		{"no vulns", 0, true, false},
		{"low present", 1, true, true},
		// low is stricter than high, so it keeps high's conformance gate.
		{"baseline fail", 0, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := &PostureReport{
				Dependencies: DependencyHealth{
					Totals: VulnSeverityCounts{Low: tt.low},
				},
				Conformance: ConformanceResult{
					Baseline: ConformanceLevel{Pass: tt.baselinePass},
				},
			}
			got := ShouldExitNonZero(report, "low")
			if got != tt.want {
				t.Errorf("ShouldExitNonZero = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldExitNonZero_Info(t *testing.T) {
	t.Run("clean report", func(t *testing.T) {
		report := &PostureReport{
			Conformance: ConformanceResult{
				Baseline: ConformanceLevel{Pass: true},
			},
			Defense: DefenseCoverage{
				Layers: []DefenseLayer{
					{Status: LayerEnabled},
				},
			},
			Config: ConfigHealth{
				Files: []ConfigFileInfo{
					{State: "current"},
				},
			},
		}
		if ShouldExitNonZero(report, "info") {
			t.Error("clean report should not exit non-zero at info level")
		}
	})

	t.Run("drift findings", func(t *testing.T) {
		report := &PostureReport{
			Conformance: ConformanceResult{
				Baseline: ConformanceLevel{Pass: true},
			},
			Drift: drift.Report{TotalFindings: 1},
		}
		if !ShouldExitNonZero(report, "info") {
			t.Error("report with drift should exit non-zero at info level")
		}
	})

	t.Run("disabled layer", func(t *testing.T) {
		report := &PostureReport{
			Conformance: ConformanceResult{
				Baseline: ConformanceLevel{Pass: true},
			},
			Defense: DefenseCoverage{
				Layers: []DefenseLayer{
					{Status: LayerDisabled},
				},
			},
		}
		if !ShouldExitNonZero(report, "info") {
			t.Error("report with disabled layer should exit non-zero at info level")
		}
	})

	t.Run("info vulns only", func(t *testing.T) {
		report := &PostureReport{
			Conformance: ConformanceResult{
				Baseline: ConformanceLevel{Pass: true},
			},
			Dependencies: DependencyHealth{
				Totals: VulnSeverityCounts{Info: 5},
			},
		}
		if !ShouldExitNonZero(report, "info") {
			t.Error("report with info vulns should exit non-zero at info level")
		}
	})
}

func TestShouldExitNonZero_Any(t *testing.T) {
	// "any" should behave like "info".
	report := &PostureReport{
		Conformance: ConformanceResult{
			Baseline: ConformanceLevel{Pass: true},
		},
		Drift: drift.Report{TotalFindings: 1},
	}
	if !ShouldExitNonZero(report, "any") {
		t.Error("'any' should exit non-zero when there are drift findings")
	}
}

// TestShouldExitNonZero_Uncertifiable confirms the gate still fails closed on any
// non-certifiable dependency result — a failed scan or an unresolved-severity
// vuln — at every gating level, and never for "none". This is the exit-gate half
// of the altitude fix, now routed through DependencyHealth.Certifiable().
func TestShouldExitNonZero_Uncertifiable(t *testing.T) {
	cases := []struct {
		name string
		deps DependencyHealth
	}{
		{"scan failed", DependencyHealth{ScanFailed: true}},
		{"unresolved severity", DependencyHealth{Scanned: true, Totals: VulnSeverityCounts{Unknown: 1}}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			report := &PostureReport{
				Dependencies: tc.deps,
				Conformance:  ConformanceResult{Baseline: ConformanceLevel{Pass: true}},
			}
			for _, lvl := range []string{"critical", "high", "moderate", "low", "info", "any"} {
				if !ShouldExitNonZero(report, lvl) {
					t.Errorf("ShouldExitNonZero(%q) = false, want true for a non-certifiable result", lvl)
				}
			}
			if ShouldExitNonZero(report, "none") {
				t.Error("ShouldExitNonZero(none) must stay false even for a non-certifiable result")
			}
		})
	}
}

func TestShouldExitNonZero_UnknownLevel(t *testing.T) {
	// Unknown levels fail closed, even for a clean report: a gate that cannot
	// tell what it enforces must not pass.
	report := &PostureReport{
		Dependencies: DependencyHealth{Scanned: true},
		Conformance: ConformanceResult{
			Baseline: ConformanceLevel{Pass: true},
		},
	}
	if !ShouldExitNonZero(report, "unknown") {
		t.Error("unknown level should fail closed")
	}
}

// TestShouldExitNonZero_MediumAlias guards against "medium" (the spelling
// `check --audit-level` uses) silently loosening the gate to "high".
func TestShouldExitNonZero_MediumAlias(t *testing.T) {
	t.Parallel()
	report := &PostureReport{
		Dependencies: DependencyHealth{Scanned: true, Totals: VulnSeverityCounts{Moderate: 3}},
		Conformance:  ConformanceResult{Baseline: ConformanceLevel{Pass: true}},
	}
	for _, level := range []string{"medium", "MEDIUM", "moderate"} {
		if !ShouldExitNonZero(report, level) {
			t.Errorf("ShouldExitNonZero(%q) = false with moderate vulnerabilities, want true", level)
		}
	}
}

func TestParseAuditLevel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{in: "none", want: "none"},
		{in: "info", want: "info"},
		{in: "any", want: "info"},
		{in: "low", want: "low"},
		{in: "moderate", want: "moderate"},
		{in: "medium", want: "moderate"},
		{in: " High ", want: "high"},
		{in: "critical", want: "critical"},
		{in: "", wantErr: true},
		{in: "hgih", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			got, err := ParseAuditLevel(tt.in)
			if tt.wantErr {
				if !errors.Is(err, ErrUnknownAuditLevel) {
					t.Errorf("ParseAuditLevel(%q) error = %v, want ErrUnknownAuditLevel", tt.in, err)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Errorf("ParseAuditLevel(%q) = %q, %v; want %q", tt.in, got, err, tt.want)
			}
		})
	}
}

// TestShouldExitNonZero_CustomConformance: a failing project policy
// (.qsdev-policy.yaml) gates the exit code at the same levels as baseline
// conformance; a project without one, or whose policy passes, is unaffected.
func TestShouldExitNonZero_CustomConformance(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		custom *ConformanceLevel
		level  string
		want   bool
	}{
		{"no policy high", nil, "high", false},
		{"passing policy high", &ConformanceLevel{Pass: true}, "high", false},
		{"failing policy high", &ConformanceLevel{Pass: false}, "high", true},
		{"failing policy moderate", &ConformanceLevel{Pass: false}, "moderate", true},
		{"failing policy low", &ConformanceLevel{Pass: false}, "low", true},
		{"failing policy info", &ConformanceLevel{Pass: false}, "info", true},
		{"passing policy low", &ConformanceLevel{Pass: true}, "low", false},
		{"failing policy critical", &ConformanceLevel{Pass: false}, "critical", false},
		{"failing policy none", &ConformanceLevel{Pass: false}, "none", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			report := &PostureReport{
				Conformance: ConformanceResult{
					Baseline: ConformanceLevel{Pass: true},
					Custom:   tt.custom,
				},
			}
			if got := ShouldExitNonZero(report, tt.level); got != tt.want {
				t.Errorf("ShouldExitNonZero(%q) = %v, want %v", tt.level, got, tt.want)
			}
		})
	}
}

// TestShouldExitNonZero_UnknownBaseline pins that a baseline reported unknown
// because the dependencies were not scanned is not treated as a failure (exit
// codes without --scan are unchanged), while a failed baseline still fails.
func TestShouldExitNonZero_UnknownBaseline(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		baseline ConformanceLevel
		custom   *ConformanceLevel
		want     bool
	}{
		{"unknown baseline", ConformanceLevel{Status: CheckUnknown}, nil, false},
		{"failed baseline", ConformanceLevel{Status: CheckFail}, nil, true},
		{"unknown baseline, failed custom", ConformanceLevel{Status: CheckUnknown}, &ConformanceLevel{Status: CheckFail}, true},
		{"legacy failed baseline", ConformanceLevel{}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			report := &PostureReport{
				Dependencies: DependencyHealth{Status: DepUnscanned},
				Conformance:  ConformanceResult{Baseline: tt.baseline, Custom: tt.custom},
			}
			for _, lvl := range []string{"high", "moderate", "low"} {
				if got := ShouldExitNonZero(report, lvl); got != tt.want {
					t.Errorf("ShouldExitNonZero(%q) = %v, want %v", lvl, got, tt.want)
				}
			}
		})
	}
}
