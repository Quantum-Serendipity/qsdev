package repair

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/posture/drift"
)

func TestExitCode_AllFixed(t *testing.T) {
	r := &RepairResult{
		Fixed: []RepairAction{
			{File: "a.txt", Description: "fixed"},
		},
	}
	if got := r.ExitCode(); got != 0 {
		t.Errorf("ExitCode() = %d, want 0 (all fixed)", got)
	}
}

func TestExitCode_Empty(t *testing.T) {
	r := &RepairResult{}
	if got := r.ExitCode(); got != 0 {
		t.Errorf("ExitCode() = %d, want 0 (nothing to do)", got)
	}
}

func TestExitCode_Skipped(t *testing.T) {
	r := &RepairResult{
		Fixed:   []RepairAction{{File: "a.txt"}},
		Skipped: []RepairAction{{File: "b.txt"}},
	}
	if got := r.ExitCode(); got != 1 {
		t.Errorf("ExitCode() = %d, want 1 (some skipped)", got)
	}
}

// TestExitCode_InfoSkipsDoNotFail verifies that skipped informational findings
// (an intentionally edited user-owned file, version notes) leave the exit code
// at 0, while any skipped actionable finding still yields 1.
func TestExitCode_InfoSkipsDoNotFail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		skipped []RepairAction
		want    int
	}{
		{name: "only info", skipped: []RepairAction{{File: "CLAUDE.md", Severity: drift.Info}, {File: "version", Severity: drift.Info}}, want: 0},
		{name: "warning", skipped: []RepairAction{{File: "CLAUDE.md", Severity: drift.Info}, {File: "pre-commit", Severity: drift.Warning}}, want: 1},
		{name: "error", skipped: []RepairAction{{File: "x", Severity: drift.Error}}, want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := &RepairResult{Skipped: tt.skipped}
			if got := r.ExitCode(); got != tt.want {
				t.Errorf("ExitCode() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestExitCode_Failed(t *testing.T) {
	r := &RepairResult{
		Fixed:  []RepairAction{{File: "a.txt"}},
		Failed: []RepairAction{{File: "c.txt"}},
	}
	if got := r.ExitCode(); got != 2 {
		t.Errorf("ExitCode() = %d, want 2 (some failed)", got)
	}
}

func TestExitCode_FailedTakesPrecedence(t *testing.T) {
	r := &RepairResult{
		Skipped: []RepairAction{{File: "b.txt"}},
		Failed:  []RepairAction{{File: "c.txt"}},
	}
	if got := r.ExitCode(); got != 2 {
		t.Errorf("ExitCode() = %d, want 2 (failed takes precedence over skipped)", got)
	}
}

func TestRepairActionType_Constants(t *testing.T) {
	// Verify constants have distinct values.
	vals := map[RepairActionType]string{
		ActionRegenerate: "ActionRegenerate",
		ActionSkip:       "ActionSkip",
	}
	if len(vals) != 2 {
		t.Error("RepairActionType constants are not all distinct")
	}
}

func TestRepairCategory_Constants(t *testing.T) {
	categories := []RepairCategory{
		CategoryFileDrift,
		CategoryConfigCorrupt,
		CategoryToolMissing,
		CategoryEnvDrift,
		CategoryHookDrift,
		CategoryMarkerDrift,
	}
	seen := make(map[RepairCategory]bool)
	for _, c := range categories {
		if seen[c] {
			t.Errorf("duplicate category: %s", c)
		}
		seen[c] = true
	}
}
