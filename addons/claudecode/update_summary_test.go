package claudecode_test

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestBuildUpdateSummary_AllUpToDate(t *testing.T) {
	content := []byte("unchanged content")
	hash := state.ComputeHash(content)

	prevState := types.GeneratedState{
		Files: map[string]types.FileState{
			".claude/skills/deploy.md": {Hash: hash},
		},
	}
	newFiles := []types.GeneratedFile{
		{Path: ".claude/skills/deploy.md", Content: content},
	}
	vDiff := claudecode.ExportVersionDiff{}

	summary := claudecode.ExportBuildUpdateSummary(prevState, newFiles, vDiff)

	if summary.String() != "All files up to date." {
		t.Errorf("expected %q, got %q", "All files up to date.", summary.String())
	}
}

func TestBuildUpdateSummary_SkillsUpdated(t *testing.T) {
	oldContent := []byte("old skill content")
	newContent := []byte("new skill content")

	prevState := types.GeneratedState{
		Files: map[string]types.FileState{
			".claude/skills/deploy.md": {Hash: state.ComputeHash(oldContent)},
		},
	}
	newFiles := []types.GeneratedFile{
		{Path: ".claude/skills/deploy.md", Content: newContent},
	}
	vDiff := claudecode.ExportVersionDiff{SkillLibraryChanged: true}

	summary := claudecode.ExportBuildUpdateSummary(prevState, newFiles, vDiff)

	if summary.SkillsUpdated != 1 {
		t.Errorf("expected SkillsUpdated == 1, got %d", summary.SkillsUpdated)
	}
	if summary.SkillsAdded != 0 {
		t.Errorf("expected SkillsAdded == 0, got %d", summary.SkillsAdded)
	}

	msg := summary.String()
	if msg != "Updated 1 skill(s)." {
		t.Errorf("expected %q, got %q", "Updated 1 skill(s).", msg)
	}
}

func TestBuildUpdateSummary_NewSkill(t *testing.T) {
	newContent := []byte("brand new skill")

	prevState := types.GeneratedState{
		Files: map[string]types.FileState{},
	}
	newFiles := []types.GeneratedFile{
		{Path: ".claude/skills/new-skill.md", Content: newContent},
	}
	vDiff := claudecode.ExportVersionDiff{}

	summary := claudecode.ExportBuildUpdateSummary(prevState, newFiles, vDiff)

	if summary.SkillsAdded != 1 {
		t.Errorf("expected SkillsAdded == 1, got %d", summary.SkillsAdded)
	}
	if summary.SkillsUpdated != 0 {
		t.Errorf("expected SkillsUpdated == 0, got %d", summary.SkillsUpdated)
	}

	msg := summary.String()
	if msg != "Added 1 skill(s)." {
		t.Errorf("expected %q, got %q", "Added 1 skill(s).", msg)
	}
}

func TestBuildUpdateSummary_MixedChanges(t *testing.T) {
	oldSkillContent := []byte("old skill")
	newSkillContent := []byte("new skill")
	ruleContent := []byte("new rule")
	templateContent := []byte("new template")

	prevState := types.GeneratedState{
		Files: map[string]types.FileState{
			".claude/skills/deploy.md": {Hash: state.ComputeHash(oldSkillContent)},
		},
	}
	newFiles := []types.GeneratedFile{
		{Path: ".claude/skills/deploy.md", Content: newSkillContent},
		{Path: ".claude/rules/go-conventions.md", Content: ruleContent},
		{Path: "CLAUDE.md", Content: templateContent},
	}
	vDiff := claudecode.ExportVersionDiff{TemplateChanged: true, SkillLibraryChanged: true}

	summary := claudecode.ExportBuildUpdateSummary(prevState, newFiles, vDiff)

	if summary.SkillsUpdated != 1 {
		t.Errorf("expected SkillsUpdated == 1, got %d", summary.SkillsUpdated)
	}
	if summary.RulesAdded != 1 {
		t.Errorf("expected RulesAdded == 1, got %d", summary.RulesAdded)
	}
	if summary.TemplatesUpdated != 1 {
		t.Errorf("expected TemplatesUpdated == 1, got %d", summary.TemplatesUpdated)
	}
	if !summary.VersionBump {
		t.Error("expected VersionBump == true")
	}

	msg := summary.String()
	// Should contain all parts.
	if msg == "All files up to date." {
		t.Error("should not be 'All files up to date.' with mixed changes")
	}
}
