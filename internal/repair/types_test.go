package repair

import "testing"

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
		ActionReinstall:  "ActionReinstall",
		ActionSkip:       "ActionSkip",
	}
	if len(vals) != 3 {
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
