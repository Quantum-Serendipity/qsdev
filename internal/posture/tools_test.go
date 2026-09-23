package posture

import (
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/posture/drift"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestAssess_ReportsTools guards `status tools` and `tools.<x>.enabled`
// conformance rules: the report must list every catalog tool with its real
// enablement and availability, not an always-empty slice.
func TestAssess_ReportsTools(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeState(t, root, ".devinit/.qsdev-init-state.yaml", types.GeneratedState{
		QsdevVersion: "1.0.0",
		LastRun:      time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		EnabledTools: map[string]bool{"semgrep": true, "gitleaks": false, "custom-tool": true},
	})

	report, err := Assess(root, AssessOptions{})
	if err != nil {
		t.Fatalf("Assess: %v", err)
	}

	byName := make(map[string]ToolStatus, len(report.Tools))
	for _, ts := range report.Tools {
		byName[ts.Name] = ts
	}
	for _, tool := range toolreg.DefaultRegistry().All() {
		if _, ok := byName[tool.Name]; !ok {
			t.Errorf("catalog tool %q missing from report.Tools", tool.Name)
		}
	}

	tests := []struct {
		name        string
		wantEnabled bool
	}{
		{"semgrep", true},
		{"gitleaks", false},
		{"custom-tool", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, ok := byName[tt.name]
			if !ok {
				t.Fatalf("tool %q missing from report.Tools", tt.name)
			}
			if ts.Enabled != tt.wantEnabled {
				t.Errorf("Enabled = %v, want %v", ts.Enabled, tt.wantEnabled)
			}
			if ts.Available != drift.ToolAvailable(tt.name) {
				t.Errorf("Available = %v, want %v", ts.Available, drift.ToolAvailable(tt.name))
			}
		})
	}

	for i := 1; i < len(report.Tools); i++ {
		if report.Tools[i-1].Name > report.Tools[i].Name {
			t.Fatalf("report.Tools not sorted: %q before %q", report.Tools[i-1].Name, report.Tools[i].Name)
		}
	}
}
