package repair

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/posture/drift"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestClassifyFindings_NilReport(t *testing.T) {
	actions := classifyFindings(nil, types.GeneratedState{}, RepairOptions{})
	if len(actions) != 0 {
		t.Errorf("got %d actions for nil report, want 0", len(actions))
	}
}

func TestClassifyFindings_EmptyReport(t *testing.T) {
	report := &drift.Report{}
	actions := classifyFindings(report, types.GeneratedState{}, RepairOptions{})
	if len(actions) != 0 {
		t.Errorf("got %d actions for empty report, want 0", len(actions))
	}
}

func TestClassifyFileModification_Overwrite(t *testing.T) {
	report := &drift.Report{
		Categories: []drift.Category{
			{
				Name: "File Modification",
				Findings: []drift.Finding{
					{
						Subject:     ".claude/settings.json",
						Description: "Machine-owned file \".claude/settings.json\" has been modified (strategy: overwrite)",
						Severity:    drift.Warning,
					},
				},
			},
		},
	}
	genState := types.GeneratedState{
		Files: map[string]types.FileState{
			".claude/settings.json": {Strategy: types.Overwrite},
		},
	}

	actions := classifyFindings(report, genState, RepairOptions{})
	if len(actions) != 1 {
		t.Fatalf("got %d actions, want 1", len(actions))
	}
	a := actions[0]
	if a.ActionType != ActionRegenerate {
		t.Errorf("ActionType = %d, want ActionRegenerate", a.ActionType)
	}
	if !a.AutoFixable {
		t.Error("expected AutoFixable=true for overwrite file")
	}
	if a.Category != CategoryFileDrift {
		t.Errorf("Category = %q, want %q", a.Category, CategoryFileDrift)
	}
}

func TestClassifyFileModification_LibraryManaged(t *testing.T) {
	report := &drift.Report{
		Categories: []drift.Category{
			{
				Name: "File Modification",
				Findings: []drift.Finding{
					{
						Subject:     ".npmrc",
						Description: "Machine-owned file \".npmrc\" has been modified (strategy: library-managed)",
						Severity:    drift.Warning,
					},
				},
			},
		},
	}
	genState := types.GeneratedState{
		Files: map[string]types.FileState{
			".npmrc": {Strategy: types.LibraryManaged},
		},
	}

	actions := classifyFindings(report, genState, RepairOptions{})
	if len(actions) != 1 {
		t.Fatalf("got %d actions, want 1", len(actions))
	}
	if actions[0].ActionType != ActionRegenerate {
		t.Errorf("ActionType = %d, want ActionRegenerate", actions[0].ActionType)
	}
	if !actions[0].AutoFixable {
		t.Error("expected AutoFixable=true for library-managed file")
	}
}

func TestClassifyFileModification_SectionMarker_NoForce(t *testing.T) {
	report := &drift.Report{
		Categories: []drift.Category{
			{
				Name: "File Modification",
				Findings: []drift.Finding{
					{
						Subject:     "CLAUDE.md",
						Description: "Human-edited file \"CLAUDE.md\" has been modified (strategy: section-marker)",
						Severity:    drift.Info,
					},
				},
			},
		},
	}
	genState := types.GeneratedState{
		Files: map[string]types.FileState{
			"CLAUDE.md": {Strategy: types.SectionMarker},
		},
	}

	actions := classifyFindings(report, genState, RepairOptions{})
	if len(actions) != 1 {
		t.Fatalf("got %d actions, want 1", len(actions))
	}
	if actions[0].ActionType != ActionSkip {
		t.Errorf("ActionType = %d, want ActionSkip for section-marker without force", actions[0].ActionType)
	}
	if actions[0].AutoFixable {
		t.Error("expected AutoFixable=false for section-marker without force")
	}
}

func TestClassifyFileModification_SectionMarker_WithForce(t *testing.T) {
	report := &drift.Report{
		Categories: []drift.Category{
			{
				Name: "File Modification",
				Findings: []drift.Finding{
					{
						Subject:     "CLAUDE.md",
						Description: "Human-edited file \"CLAUDE.md\" has been modified (strategy: section-marker)",
						Severity:    drift.Info,
					},
				},
			},
		},
	}
	genState := types.GeneratedState{
		Files: map[string]types.FileState{
			"CLAUDE.md": {Strategy: types.SectionMarker},
		},
	}

	actions := classifyFindings(report, genState, RepairOptions{Force: true})
	if len(actions) != 1 {
		t.Fatalf("got %d actions, want 1", len(actions))
	}
	if actions[0].ActionType != ActionRegenerate {
		t.Errorf("ActionType = %d, want ActionRegenerate for section-marker with force", actions[0].ActionType)
	}
	if !actions[0].AutoFixable {
		t.Error("expected AutoFixable=true for section-marker with force")
	}
}

func TestClassifyFileModification_SectionMarker_WithReset(t *testing.T) {
	report := &drift.Report{
		Categories: []drift.Category{
			{
				Name: "File Modification",
				Findings: []drift.Finding{
					{
						Subject:     "CLAUDE.md",
						Description: "Human-edited file \"CLAUDE.md\" has been modified (strategy: section-marker)",
						Severity:    drift.Info,
					},
				},
			},
		},
	}
	genState := types.GeneratedState{
		Files: map[string]types.FileState{
			"CLAUDE.md": {Strategy: types.SectionMarker},
		},
	}

	actions := classifyFindings(report, genState, RepairOptions{Reset: true})
	if len(actions) != 1 {
		t.Fatalf("got %d actions, want 1", len(actions))
	}
	if actions[0].ActionType != ActionRegenerate {
		t.Errorf("ActionType = %d, want ActionRegenerate for section-marker with reset", actions[0].ActionType)
	}
}

func TestClassifyFileModification_DevenvNix_AlwaysSkipped(t *testing.T) {
	report := &drift.Report{
		Categories: []drift.Category{
			{
				Name: "File Modification",
				Findings: []drift.Finding{
					{
						Subject:     "devenv.nix",
						Description: "Machine-owned file \"devenv.nix\" has been modified (strategy: overwrite)",
						Severity:    drift.Warning,
					},
				},
			},
		},
	}
	genState := types.GeneratedState{
		Files: map[string]types.FileState{
			"devenv.nix": {Strategy: types.Overwrite},
		},
	}

	// Even with --force and --reset, devenv.nix is never auto-modified.
	actions := classifyFindings(report, genState, RepairOptions{Force: true, Reset: true})
	if len(actions) != 1 {
		t.Fatalf("got %d actions, want 1", len(actions))
	}
	if actions[0].ActionType != ActionSkip {
		t.Errorf("ActionType = %d, want ActionSkip for devenv.nix", actions[0].ActionType)
	}
	if actions[0].AutoFixable {
		t.Error("devenv.nix should never be AutoFixable")
	}
}

func TestClassifyFileModification_Deleted(t *testing.T) {
	report := &drift.Report{
		Categories: []drift.Category{
			{
				Name: "File Modification",
				Findings: []drift.Finding{
					{
						Subject:     ".envrc",
						Description: "Generated file \".envrc\" has been deleted",
						Severity:    drift.Error,
					},
				},
			},
		},
	}
	genState := types.GeneratedState{
		Files: map[string]types.FileState{
			".envrc": {Strategy: types.Overwrite},
		},
	}

	actions := classifyFindings(report, genState, RepairOptions{})
	if len(actions) != 1 {
		t.Fatalf("got %d actions, want 1", len(actions))
	}
	if actions[0].ActionType != ActionRegenerate {
		t.Errorf("ActionType = %d, want ActionRegenerate for deleted file", actions[0].ActionType)
	}
	if !actions[0].AutoFixable {
		t.Error("expected AutoFixable=true for deleted file")
	}
}

func TestClassifyFileModification_DeletedDevenvNix(t *testing.T) {
	report := &drift.Report{
		Categories: []drift.Category{
			{
				Name: "File Modification",
				Findings: []drift.Finding{
					{
						Subject:     "devenv.nix",
						Description: "Generated file \"devenv.nix\" has been deleted",
						Severity:    drift.Error,
					},
				},
			},
		},
	}
	genState := types.GeneratedState{
		Files: map[string]types.FileState{
			"devenv.nix": {Strategy: types.Overwrite},
		},
	}

	actions := classifyFindings(report, genState, RepairOptions{})
	if len(actions) != 1 {
		t.Fatalf("got %d actions, want 1", len(actions))
	}
	// devenv.nix is always skipped, even when deleted.
	if actions[0].ActionType != ActionSkip {
		t.Errorf("ActionType = %d, want ActionSkip for devenv.nix even if deleted", actions[0].ActionType)
	}
}

func TestClassifyHookDrift(t *testing.T) {
	report := &drift.Report{
		Categories: []drift.Category{
			{
				Name: "Pre-Commit Hook Drift",
				Findings: []drift.Finding{
					{
						Subject:     "pre-commit",
						Description: "Git pre-commit hook is not installed",
						Severity:    drift.Warning,
					},
				},
			},
		},
	}

	actions := classifyFindings(report, types.GeneratedState{}, RepairOptions{})
	if len(actions) != 1 {
		t.Fatalf("got %d actions, want 1", len(actions))
	}
	a := actions[0]
	if a.ActionType != ActionReinstall {
		t.Errorf("ActionType = %d, want ActionReinstall", a.ActionType)
	}
	if !a.AutoFixable {
		t.Error("expected AutoFixable=true for hook drift")
	}
	if a.Category != CategoryHookDrift {
		t.Errorf("Category = %q, want %q", a.Category, CategoryHookDrift)
	}
}

func TestClassifyMarkerDrift(t *testing.T) {
	report := &drift.Report{
		Categories: []drift.Category{
			{
				Name: "Section Marker Integrity",
				Findings: []drift.Finding{
					{
						Subject:     "marker:security",
						Description: "Opening marker has no matching closing marker",
						Severity:    drift.Warning,
					},
				},
			},
		},
	}

	actions := classifyFindings(report, types.GeneratedState{}, RepairOptions{})
	if len(actions) != 1 {
		t.Fatalf("got %d actions, want 1", len(actions))
	}
	a := actions[0]
	if a.ActionType != ActionRegenerate {
		t.Errorf("ActionType = %d, want ActionRegenerate", a.ActionType)
	}
	if !a.AutoFixable {
		t.Error("expected AutoFixable=true for marker drift")
	}
	if a.Category != CategoryMarkerDrift {
		t.Errorf("Category = %q, want %q", a.Category, CategoryMarkerDrift)
	}
}

func TestClassifyVersionDrift(t *testing.T) {
	report := &drift.Report{
		Categories: []drift.Category{
			{
				Name: "Version Drift",
				Findings: []drift.Finding{
					{
						Subject:     "qsdev version",
						Description: "Configuration was generated with a different qsdev version",
						Severity:    drift.Info,
					},
				},
			},
		},
	}

	actions := classifyFindings(report, types.GeneratedState{}, RepairOptions{})
	if len(actions) != 1 {
		t.Fatalf("got %d actions, want 1", len(actions))
	}
	if actions[0].ActionType != ActionSkip {
		t.Errorf("ActionType = %d, want ActionSkip", actions[0].ActionType)
	}
	if actions[0].AutoFixable {
		t.Error("version drift should not be auto-fixable")
	}
}

func TestClassifyToolAvailability(t *testing.T) {
	report := &drift.Report{
		Categories: []drift.Category{
			{
				Name: "Tool Availability",
				Findings: []drift.Finding{
					{
						Subject:     "semgrep",
						Description: "Required binary \"semgrep\" for tool \"semgrep\" is not available on PATH",
						Severity:    drift.Warning,
					},
				},
			},
		},
	}

	actions := classifyFindings(report, types.GeneratedState{}, RepairOptions{})
	if len(actions) != 1 {
		t.Fatalf("got %d actions, want 1", len(actions))
	}
	if actions[0].ActionType != ActionSkip {
		t.Errorf("ActionType = %d, want ActionSkip", actions[0].ActionType)
	}
	if actions[0].Category != CategoryToolMissing {
		t.Errorf("Category = %q, want %q", actions[0].Category, CategoryToolMissing)
	}
}

func TestClassifyLockfileDrift(t *testing.T) {
	report := &drift.Report{
		Categories: []drift.Category{
			{
				Name: "Lock File Drift",
				Findings: []drift.Finding{
					{
						Subject:     "package-lock.json",
						Description: "Lockfile is older than manifest",
						Severity:    drift.Warning,
					},
				},
			},
		},
	}

	actions := classifyFindings(report, types.GeneratedState{}, RepairOptions{})
	if len(actions) != 1 {
		t.Fatalf("got %d actions, want 1", len(actions))
	}
	if actions[0].ActionType != ActionSkip {
		t.Errorf("ActionType = %d, want ActionSkip", actions[0].ActionType)
	}
}

func TestClassifyFindings_MultipleCategories(t *testing.T) {
	report := &drift.Report{
		Categories: []drift.Category{
			{
				Name: "File Modification",
				Findings: []drift.Finding{
					{Subject: ".npmrc", Description: "Machine-owned file has been modified", Severity: drift.Warning},
				},
			},
			{
				Name: "Pre-Commit Hook Drift",
				Findings: []drift.Finding{
					{Subject: "pre-commit", Description: "Hook not installed", Severity: drift.Warning},
				},
			},
			{
				Name: "Version Drift",
				Findings: []drift.Finding{
					{Subject: "qsdev version", Description: "Version mismatch", Severity: drift.Info},
				},
			},
		},
	}
	genState := types.GeneratedState{
		Files: map[string]types.FileState{
			".npmrc": {Strategy: types.Overwrite},
		},
	}

	actions := classifyFindings(report, genState, RepairOptions{})
	if len(actions) != 3 {
		t.Fatalf("got %d actions, want 3", len(actions))
	}

	// Verify classification of each.
	if actions[0].ActionType != ActionRegenerate {
		t.Errorf("action[0] type = %d, want ActionRegenerate", actions[0].ActionType)
	}
	if actions[1].ActionType != ActionReinstall {
		t.Errorf("action[1] type = %d, want ActionReinstall", actions[1].ActionType)
	}
	if actions[2].ActionType != ActionSkip {
		t.Errorf("action[2] type = %d, want ActionSkip", actions[2].ActionType)
	}
}

func TestClassifyFileModification_UnknownFile(t *testing.T) {
	report := &drift.Report{
		Categories: []drift.Category{
			{
				Name: "File Modification",
				Findings: []drift.Finding{
					{
						Subject:     "unknown-file.txt",
						Description: "File \"unknown-file.txt\" has been modified",
						Severity:    drift.Warning,
					},
				},
			},
		},
	}
	// File not in state.
	genState := types.GeneratedState{
		Files: map[string]types.FileState{},
	}

	actions := classifyFindings(report, genState, RepairOptions{})
	if len(actions) != 1 {
		t.Fatalf("got %d actions, want 1", len(actions))
	}
	if actions[0].ActionType != ActionSkip {
		t.Errorf("ActionType = %d, want ActionSkip for unknown file", actions[0].ActionType)
	}
}

func TestClassifyUnknownCategory(t *testing.T) {
	report := &drift.Report{
		Categories: []drift.Category{
			{
				Name: "Future Category",
				Findings: []drift.Finding{
					{
						Subject:     "something",
						Description: "Something happened",
						Severity:    drift.Info,
					},
				},
			},
		},
	}

	actions := classifyFindings(report, types.GeneratedState{}, RepairOptions{})
	if len(actions) != 1 {
		t.Fatalf("got %d actions, want 1", len(actions))
	}
	if actions[0].ActionType != ActionSkip {
		t.Errorf("ActionType = %d, want ActionSkip for unknown category", actions[0].ActionType)
	}
}
