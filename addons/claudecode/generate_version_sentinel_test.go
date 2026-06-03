package claudecode_test

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestVersionSentinel_Disabled(t *testing.T) {
	reg := newTestRegistry(t, goMock())
	answers := types.WizardAnswers{
		Languages:  []types.LanguageChoice{{Name: "go"}},
		AgentTools: types.AgentToolsAnswers{VersionSentinel: false},
	}

	gen := claudecode.NewClaudeCodeGenerator(reg, claudecode.Config{})
	files, err := gen.Generate(answers)
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range files {
		if strings.Contains(f.Path, "version-sentinel") {
			t.Errorf("version-sentinel file %q should not be generated when disabled", f.Path)
		}
	}
}

func TestVersionSentinelIgnore_GoProject(t *testing.T) {
	reg := newTestRegistry(t, goMock())
	answers := types.WizardAnswers{
		Tier:       "full",
		Languages:  []types.LanguageChoice{{Name: "go"}},
		AgentTools: types.AgentToolsAnswers{VersionSentinel: true},
	}

	gen := claudecode.NewClaudeCodeGenerator(reg, claudecode.Config{})
	files, err := gen.Generate(answers)
	if err != nil {
		t.Fatal(err)
	}

	var ignoreFile *types.GeneratedFile
	for i, f := range files {
		if f.Path == ".version-sentinel/ignore" {
			ignoreFile = &files[i]
			break
		}
	}

	if ignoreFile == nil {
		t.Fatal("expected .version-sentinel/ignore file for Go project (VS doesn't cover go.mod)")
		return
	}

	content := string(ignoreFile.Content)
	if !strings.Contains(content, "go.mod") {
		t.Errorf("ignore file should contain go.mod, got:\n%s", content)
	}
}

func TestVersionSentinelIgnore_NpmProject(t *testing.T) {
	reg := newTestRegistry(t, jsMock())
	answers := types.WizardAnswers{
		Tier:       "full",
		Languages:  []types.LanguageChoice{{Name: "javascript"}},
		AgentTools: types.AgentToolsAnswers{VersionSentinel: true},
	}

	gen := claudecode.NewClaudeCodeGenerator(reg, claudecode.Config{})
	files, err := gen.Generate(answers)
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range files {
		if f.Path == ".version-sentinel/ignore" {
			t.Error("npm project should NOT have .version-sentinel/ignore (npm is covered)")
		}
	}
}

func TestVersionSentinelRecoverySkill(t *testing.T) {
	reg := newTestRegistry(t, goMock())
	answers := types.WizardAnswers{
		Tier:       "full",
		Languages:  []types.LanguageChoice{{Name: "go"}},
		AgentTools: types.AgentToolsAnswers{VersionSentinel: true},
	}

	gen := claudecode.NewClaudeCodeGenerator(reg, claudecode.Config{})
	files, err := gen.Generate(answers)
	if err != nil {
		t.Fatal(err)
	}

	var skillFile *types.GeneratedFile
	for i, f := range files {
		if f.Path == ".claude/skills/version-sentinel/SKILL.md" {
			skillFile = &files[i]
			break
		}
	}

	if skillFile == nil {
		t.Fatal("expected version-sentinel recovery skill")
		return
	}

	content := string(skillFile.Content)
	if !strings.Contains(content, "BLOCKED") {
		t.Error("skill should mention BLOCKED")
	}
	if !strings.Contains(content, "/vs-record") {
		t.Error("skill should mention /vs-record command")
	}
}

func TestVersionSentinelIgnore_GoAndJsProject(t *testing.T) {
	reg := newTestRegistry(t, goMock(), jsMock())
	answers := types.WizardAnswers{
		Tier: "full",
		Languages: []types.LanguageChoice{
			{Name: "go"},
			{Name: "javascript"},
		},
		AgentTools: types.AgentToolsAnswers{VersionSentinel: true},
	}

	gen := claudecode.NewClaudeCodeGenerator(reg, claudecode.Config{})
	files, err := gen.Generate(answers)
	if err != nil {
		t.Fatal(err)
	}

	var ignoreFile *types.GeneratedFile
	for i, f := range files {
		if f.Path == ".version-sentinel/ignore" {
			ignoreFile = &files[i]
			break
		}
	}

	if ignoreFile == nil {
		t.Fatal("expected ignore file for Go+JS project (go.mod is uncovered)")
		return
	}

	content := string(ignoreFile.Content)
	if !strings.Contains(content, "go.mod") {
		t.Error("ignore file should contain go.mod")
	}
	if strings.Contains(content, "package.json") {
		t.Error("ignore file should NOT contain package.json (npm is covered)")
	}
}
