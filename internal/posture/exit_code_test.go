package posture

import (
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
		name     string
		critical int
		high     int
		moderate int
		want     bool
	}{
		{"no vulns", 0, 0, 0, false},
		{"moderate present", 0, 0, 1, true},
		{"high present", 0, 1, 0, true},
		{"critical present", 1, 0, 0, true},
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
		name string
		low  int
		want bool
	}{
		{"no vulns", 0, false},
		{"low present", 1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := &PostureReport{
				Dependencies: DependencyHealth{
					Totals: VulnSeverityCounts{Low: tt.low},
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

func TestShouldExitNonZero_UnknownLevel(t *testing.T) {
	// Unknown levels default to "high" behavior.
	report := &PostureReport{
		Dependencies: DependencyHealth{
			Totals: VulnSeverityCounts{High: 1},
		},
		Conformance: ConformanceResult{
			Baseline: ConformanceLevel{Pass: true},
		},
	}
	if !ShouldExitNonZero(report, "unknown") {
		t.Error("unknown level should default to 'high' behavior")
	}
}
