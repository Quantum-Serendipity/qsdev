package posture

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/posture/drift"
)

func TestVulnSeverityCountsTotal(t *testing.T) {
	tests := []struct {
		name   string
		counts VulnSeverityCounts
		want   int
	}{
		{
			name:   "all zeros",
			counts: VulnSeverityCounts{},
			want:   0,
		},
		{
			name: "all ones",
			counts: VulnSeverityCounts{
				Critical: 1,
				High:     1,
				Moderate: 1,
				Low:      1,
				Info:     1,
			},
			want: 5,
		},
		{
			name: "mixed values",
			counts: VulnSeverityCounts{
				Critical: 3,
				High:     7,
				Moderate: 12,
				Low:      25,
				Info:     100,
			},
			want: 147,
		},
		{
			name: "only critical",
			counts: VulnSeverityCounts{
				Critical: 42,
			},
			want: 42,
		},
		{
			name: "only info",
			counts: VulnSeverityCounts{
				Info: 99,
			},
			want: 99,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.counts.Total()
			if got != tt.want {
				t.Errorf("VulnSeverityCounts.Total() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestLayerStatusStringValues(t *testing.T) {
	tests := []struct {
		status LayerStatus
		want   string
	}{
		{LayerEnabled, "enabled"},
		{LayerPartial, "partial"},
		{LayerDisabled, "disabled"},
		{LayerNotApplicable, "not-applicable"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := string(tt.status)
			if got != tt.want {
				t.Errorf("LayerStatus = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLayerWeightStringValues(t *testing.T) {
	tests := []struct {
		weight LayerWeight
		want   string
	}{
		{WeightCritical, "critical"},
		{WeightHigh, "high"},
		{WeightMedium, "medium"},
		{WeightLow, "low"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := string(tt.weight)
			if got != tt.want {
				t.Errorf("LayerWeight = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDriftSeverityStringValues(t *testing.T) {
	tests := []struct {
		severity drift.Severity
		want     string
	}{
		{drift.Critical, "critical"},
		{drift.Error, "error"},
		{drift.Warning, "warning"},
		{drift.Info, "info"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := string(tt.severity)
			if got != tt.want {
				t.Errorf("drift.Severity = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSchemaVersionIsSet(t *testing.T) {
	if SchemaVersion == "" {
		t.Error("SchemaVersion should not be empty")
	}
	if SchemaVersion != "1.0.0" {
		t.Errorf("SchemaVersion = %q, want %q", SchemaVersion, "1.0.0")
	}
}
