package posture

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/posture/drift"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg/toolregtest"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// failingRegistry is a loadRegistry stand-in for a tool registry that cannot
// be built.
func failingRegistry(err error) func() (*toolreg.Registry, error) {
	return func() (*toolreg.Registry, error) { return nil, err }
}

// TestBuildToolStatusesRegistryError: when the tool registry cannot be built,
// the tool listing reports that error instead of panicking or returning an
// empty listing that reads as "no tools".
func TestBuildToolStatusesRegistryError(t *testing.T) {
	t.Parallel()
	errBuild := errors.New("registry build failed")
	got, err := buildToolStatuses(failingRegistry(errBuild), map[string]bool{"semgrep": true})
	if !errors.Is(err, errBuild) {
		t.Fatalf("buildToolStatuses() error = %v, want %v", err, errBuild)
	}
	if got != nil {
		t.Errorf("buildToolStatuses() = %v, want nil on error", got)
	}
}

// TestLoadAnswersFallbackRegistryError: when the registry needed to infer the
// enabled tools cannot be built, the fallback records the error and marks the
// tool state unknown rather than crediting nothing (or panicking).
func TestLoadAnswersFallbackRegistryError(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, ".devinit")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".qsdev-init-answers.yaml"), []byte("languages: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	errBuild := errors.New("registry build failed")
	merged := &MergedState{EnabledTools: map[string]bool{}}

	loadAnswersFallback(root, merged, failingRegistry(errBuild))

	if !merged.ToolStateUnknown {
		t.Error("ToolStateUnknown = false, want true")
	}
	if len(merged.Errors) != 1 || !errors.Is(merged.Errors[0].Err, errBuild) {
		t.Errorf("Errors = %v, want one wrapping %v", merged.Errors, errBuild)
	}
	if len(merged.EnabledTools) != 0 {
		t.Errorf("EnabledTools = %v, want none", merged.EnabledTools)
	}
}

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
	for _, tool := range toolregtest.Default(t).All() {
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
