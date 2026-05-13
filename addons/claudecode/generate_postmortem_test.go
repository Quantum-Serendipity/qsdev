package claudecode_test

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/addons/claudecode"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

func TestGeneratePostmortemSkill_GoProject(t *testing.T) {
	reg := newTestRegistry(t, goMock())
	answers := types.WizardAnswers{
		Languages:  []types.LanguageChoice{{Name: "go"}},
		AgentTools: types.AgentToolsAnswers{PostmortemEnabled: true},
	}

	gen := claudecode.NewClaudeCodeGenerator(reg, claudecode.Config{})
	files, err := gen.Generate(answers)
	if err != nil {
		t.Fatal(err)
	}

	var skillFile *types.GeneratedFile
	for i, f := range files {
		if f.Path == ".claude/skills/agent-postmortem/SKILL.md" {
			skillFile = &files[i]
			break
		}
	}

	if skillFile == nil {
		t.Fatal("expected postmortem skill file")
	}

	content := string(skillFile.Content)
	if !strings.Contains(content, "go test ./...") {
		t.Error("skill should contain 'go test ./...'")
	}
	if !strings.Contains(content, "go build ./...") {
		t.Error("skill should contain 'go build ./...'")
	}
	if !strings.Contains(content, "golangci-lint run") {
		t.Error("skill should contain 'golangci-lint run'")
	}
}

func TestGeneratePostmortemSkill_MultiLanguage(t *testing.T) {
	reg := newTestRegistry(t, goMock(), jsMock())
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "go"},
			{Name: "javascript"},
		},
		AgentTools: types.AgentToolsAnswers{PostmortemEnabled: true},
	}

	gen := claudecode.NewClaudeCodeGenerator(reg, claudecode.Config{})
	files, err := gen.Generate(answers)
	if err != nil {
		t.Fatal(err)
	}

	var skillFile *types.GeneratedFile
	for i, f := range files {
		if f.Path == ".claude/skills/agent-postmortem/SKILL.md" {
			skillFile = &files[i]
			break
		}
	}

	if skillFile == nil {
		t.Fatal("expected postmortem skill file")
	}

	content := string(skillFile.Content)
	if !strings.Contains(content, "go test ./...") {
		t.Error("should contain Go test command")
	}
	if !strings.Contains(content, "npm test") {
		t.Error("should contain npm test command")
	}
}

func TestGeneratePostmortemSkill_Disabled(t *testing.T) {
	reg := newTestRegistry(t, goMock())
	answers := types.WizardAnswers{
		Languages:  []types.LanguageChoice{{Name: "go"}},
		AgentTools: types.AgentToolsAnswers{PostmortemEnabled: false},
	}

	gen := claudecode.NewClaudeCodeGenerator(reg, claudecode.Config{})
	files, err := gen.Generate(answers)
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range files {
		if f.Path == ".claude/skills/agent-postmortem/SKILL.md" {
			t.Error("postmortem skill should not be generated when disabled")
		}
	}
}

func TestGeneratePostmortemSkill_PreservesBaseSkillStructure(t *testing.T) {
	reg := newTestRegistry(t, goMock())
	answers := types.WizardAnswers{
		Languages:  []types.LanguageChoice{{Name: "go"}},
		AgentTools: types.AgentToolsAnswers{PostmortemEnabled: true},
	}

	gen := claudecode.NewClaudeCodeGenerator(reg, claudecode.Config{})
	files, err := gen.Generate(answers)
	if err != nil {
		t.Fatal(err)
	}

	var skillFile *types.GeneratedFile
	for i, f := range files {
		if f.Path == ".claude/skills/agent-postmortem/SKILL.md" {
			skillFile = &files[i]
			break
		}
	}

	if skillFile == nil {
		t.Fatal("expected postmortem skill file")
	}

	content := string(skillFile.Content)
	requiredSections := []string{
		"## Step 1 - Intent Snapshot",
		"## Step 2 - Evidence Collection",
		"## Step 3 - Verification Check",
		"## Step 4 - Postmortem Output",
		"## Anti-Fake-Done Guardrails",
		"VERIFIED DONE | NOT VERIFIED",
		"No proof, no done.",
	}
	for _, section := range requiredSections {
		if !strings.Contains(content, section) {
			t.Errorf("skill should contain %q", section)
		}
	}
}

func TestGeneratePostmortemSkill_FileMetadata(t *testing.T) {
	reg := newTestRegistry(t, goMock())
	answers := types.WizardAnswers{
		Languages:  []types.LanguageChoice{{Name: "go"}},
		AgentTools: types.AgentToolsAnswers{PostmortemEnabled: true},
	}

	gen := claudecode.NewClaudeCodeGenerator(reg, claudecode.Config{})
	files, err := gen.Generate(answers)
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range files {
		if f.Path == ".claude/skills/agent-postmortem/SKILL.md" {
			if f.Mode != 0o644 {
				t.Errorf("Mode = %#o, want %#o", f.Mode, 0o644)
			}
			if f.Strategy != types.LibraryManaged {
				t.Errorf("Strategy = %v, want LibraryManaged", f.Strategy)
			}
			return
		}
	}
	t.Error("postmortem skill file not found")
}

func TestGeneratePostmortemSkill_NoLanguages(t *testing.T) {
	reg := newTestRegistry(t)
	answers := types.WizardAnswers{
		AgentTools: types.AgentToolsAnswers{PostmortemEnabled: true},
	}

	gen := claudecode.NewClaudeCodeGenerator(reg, claudecode.Config{})
	files, err := gen.Generate(answers)
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range files {
		if f.Path == ".claude/skills/agent-postmortem/SKILL.md" {
			content := string(f.Content)
			if !strings.Contains(content, "## Step 1") {
				t.Error("skill should still have base structure even with no languages")
			}
			return
		}
	}
	t.Error("postmortem skill should be generated even with no languages")
}

func TestGeneratePostmortemSkill_PackageManagerAware(t *testing.T) {
	reg := newTestRegistry(t, &ecosystem.MockModule{
		NameVal:        "javascript",
		DisplayNameVal: "JavaScript/TypeScript",
		TierVal:        1,
		PackageManagersVal: []ecosystem.PackageManagerInfo{
			{Name: "pnpm"},
		},
		VerificationCommandsVal: ecosystem.VerificationCommands{
			Build: []string{"pnpm run build"},
			Test:  []string{"pnpm test"},
			Lint:  []string{"pnpm run lint"},
		},
	})
	answers := types.WizardAnswers{
		Languages:  []types.LanguageChoice{{Name: "javascript", PackageManager: "pnpm"}},
		AgentTools: types.AgentToolsAnswers{PostmortemEnabled: true},
	}

	gen := claudecode.NewClaudeCodeGenerator(reg, claudecode.Config{})
	files, err := gen.Generate(answers)
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range files {
		if f.Path == ".claude/skills/agent-postmortem/SKILL.md" {
			content := string(f.Content)
			if !strings.Contains(content, "pnpm test") {
				t.Error("skill should contain pnpm-specific test command")
			}
			return
		}
	}
	t.Error("postmortem skill file not found")
}
