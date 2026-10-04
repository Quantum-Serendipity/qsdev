package devinit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/answers"
	"github.com/Quantum-Serendipity/qsdev/internal/check"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
)

// runCheckJSON runs `check --format json` in dir and decodes the report.
func runCheckJSON(t *testing.T, dir string) check.CheckReport {
	t.Helper()
	out, _ := runLifecycleCmd(t, dir, checkCmd(), "--format", "json", "--audit-level", "low")
	var report check.CheckReport
	// The JSON report may be followed by cobra's error output.
	if err := json.NewDecoder(strings.NewReader(out[strings.Index(out, "{"):])).Decode(&report); err != nil {
		t.Fatalf("parsing report: %v\n%s", err, out)
	}
	return report
}

// TestLifecycle_ReconcilesUncommittedOptOut verifies enable/disable apply the
// same opt-out rule as init and update: an always-on tool switched off only
// in the local answers file is restored, and a `disable --force` opt-out,
// recorded in the committed tools.disabled, is kept.
func TestLifecycle_ReconcilesUncommittedOptOut(t *testing.T) {
	tests := []struct {
		name        string
		optOut      func(t *testing.T, dir string)
		wantEnabled bool
	}{
		{
			name: "hand-edited answers file",
			optOut: func(t *testing.T, dir string) {
				t.Helper()
				saved, err := answers.LoadPrimary(dir)
				if err != nil {
					t.Fatal(err)
				}
				saved.Hooks.SafetyBlock = false
				saved.EnabledTools[toolreg.ToolAttachGuard] = false
				if err := answers.SavePrimary(dir, saved); err != nil {
					t.Fatal(err)
				}
			},
			wantEnabled: true,
		},
		{
			name: "disable --force",
			optOut: func(t *testing.T, dir string) {
				t.Helper()
				mustDisable(t, dir, toolreg.ToolAttachGuard, "--force")
			},
			wantEnabled: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := initLifecycleProject(t)
			tt.optOut(t, dir)

			mustEnable(t, dir, "semble")

			a := loadProjectAnswers(t, dir)
			if a.EnabledTools[toolreg.ToolAttachGuard] != tt.wantEnabled || a.Hooks.SafetyBlock != tt.wantEnabled {
				t.Errorf("attach-guard enabled = %v, safety_block = %v; want %v",
					a.EnabledTools[toolreg.ToolAttachGuard], a.Hooks.SafetyBlock, tt.wantEnabled)
			}
			if _, disabled := committedTools(t, dir); slices.Contains(disabled, toolreg.ToolAttachGuard) == tt.wantEnabled {
				t.Errorf(".qsdev.yaml tools.disabled = %v, want attach-guard listed = %v", disabled, !tt.wantEnabled)
			}
			settings := readProjectFile(t, dir, ".claude/settings.json")
			if got := strings.Contains(settings, "package-guard.py"); got != tt.wantEnabled {
				t.Errorf("settings.json registers package-guard.py = %v, want %v", got, tt.wantEnabled)
			}
		})
	}
}

// TestInit_SupplyChainOnlyEnforcesNoAgentConfig verifies that at a tier that
// generates no skills, always-on enforcement adds none: init warns about no
// suppressed skill, records no skill-backed tool as enabled, and check does
// not require one, while attach-guard is still enforced.
func TestInit_SupplyChainOnlyEnforcesNoAgentConfig(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/lc\n\ngo 1.24\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := executeInitCmd(t, dir, "--yes", "--lang", "go", "--tier", "supply-chain-only")
	if err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}
	if strings.Contains(out, "will not be generated below the standard tier") {
		t.Errorf("init warned about agent configuration it added itself:\n%s", out)
	}

	a := loadProjectAnswers(t, dir)
	if len(a.Skills) != 0 || a.EnabledTools[toolreg.ToolTrailOfBitsSkills] {
		t.Errorf("skills = %v, trail-of-bits-skills enabled = %v; want none below standard",
			a.Skills, a.EnabledTools[toolreg.ToolTrailOfBitsSkills])
	}
	enabled, _ := committedTools(t, dir)
	if !slices.Contains(enabled, toolreg.ToolAttachGuard) {
		t.Errorf(".qsdev.yaml tools.enabled = %v, want attach-guard", enabled)
	}
	if slices.Contains(enabled, toolreg.ToolTrailOfBitsSkills) {
		t.Errorf(".qsdev.yaml tools.enabled = %v, want no trail-of-bits-skills", enabled)
	}

	for _, c := range runCheckJSON(t, dir).Checks {
		if strings.HasPrefix(c.Name, "tool_missing_") && c.Status == check.StatusFail {
			t.Errorf("check requires %s at supply-chain-only: %s", c.Name, c.Message)
		}
	}
}
