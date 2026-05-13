package claudecode_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fastcat.org/go/gdev-secure-devenv-bootstrap/addons/claudecode"
	"fastcat.org/go/gdev-secure-devenv-bootstrap/pkg/types"
)

func TestClaudeCmd_HasSubcommands(t *testing.T) {
	cmd := claudecode.ExportClaudeCmd()
	subs := cmd.Commands()

	if len(subs) != 5 {
		names := make([]string, len(subs))
		for i, s := range subs {
			names[i] = s.Name()
		}
		t.Fatalf("expected 5 subcommands, got %d: %v", len(subs), names)
	}

	expected := map[string]bool{
		"init":        false,
		"update":      false,
		"add-skill":   false,
		"add-hook":    false,
		"list-skills": false,
	}

	for _, sub := range subs {
		name := sub.Name()
		if _, ok := expected[name]; !ok {
			t.Errorf("unexpected subcommand: %q", name)
		} else {
			expected[name] = true
		}
	}

	for name, found := range expected {
		if !found {
			t.Errorf("missing expected subcommand: %q", name)
		}
	}
}

func TestInitCmd_DryRun(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a minimal go.mod so detection has something.
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := claudecode.ExportClaudeCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"init", "--dry-run", "--yes", "--permission-preset", "standard"})

	// Change to temp dir so os.Getwd() works.
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init --dry-run failed: %v", err)
	}

	output := buf.String()

	// Dry-run should show file preview, not write files.
	if !strings.Contains(output, "settings.json") {
		t.Error("dry-run output should mention settings.json")
	}

	// Verify no files were actually written.
	settingsPath := filepath.Join(tmpDir, ".claude", "settings.json")
	if _, err := os.Stat(settingsPath); err == nil {
		t.Error("dry-run should not write .claude/settings.json")
	}
}

func TestInitCmd_WritesFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a minimal go.mod so detection has something.
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := claudecode.ExportClaudeCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"init", "--yes", "--permission-preset", "standard"})

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Verify key files were created.
	expectedFiles := []string{
		filepath.Join(tmpDir, ".claude", "settings.json"),
		filepath.Join(tmpDir, "CLAUDE.md"),
	}

	for _, f := range expectedFiles {
		if _, err := os.Stat(f); err != nil {
			t.Errorf("expected file %q was not created: %v", f, err)
		}
	}

	// Verify answers were saved.
	answersFile := filepath.Join(tmpDir, ".claude", ".gdev-claude-answers.yaml")
	if _, err := os.Stat(answersFile); err != nil {
		t.Errorf("answers file not saved: %v", err)
	}

	// Verify state was saved.
	stateFile := filepath.Join(tmpDir, ".claude", ".gdev-claude-state.yaml")
	if _, err := os.Stat(stateFile); err != nil {
		t.Errorf("state file not saved: %v", err)
	}
}

func TestAddSkillCmd_ValidSkill(t *testing.T) {
	tmpDir := t.TempDir()

	// First run init to create saved answers.
	cmd := claudecode.ExportClaudeCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"init", "--yes", "--permission-preset", "standard"})

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Now add a skill.
	cmd2 := claudecode.ExportClaudeCmd()
	var buf2 bytes.Buffer
	cmd2.SetOut(&buf2)
	cmd2.SetErr(&buf2)
	cmd2.SetArgs([]string{"add-skill", "deploy"})

	if err := cmd2.Execute(); err != nil {
		t.Fatalf("add-skill failed: %v", err)
	}

	output := buf2.String()
	if !strings.Contains(output, "deploy") {
		t.Errorf("output should mention added skill, got: %s", output)
	}

	// Verify skill file was created.
	skillFile := filepath.Join(tmpDir, ".claude", "skills", "deploy.md")
	if _, err := os.Stat(skillFile); err != nil {
		t.Errorf("skill file not created: %v", err)
	}
}

func TestAddSkillCmd_UnknownSkill(t *testing.T) {
	tmpDir := t.TempDir()

	// First run init to create saved answers.
	cmd := claudecode.ExportClaudeCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"init", "--yes"})

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Try to add a nonexistent skill.
	cmd2 := claudecode.ExportClaudeCmd()
	var buf2 bytes.Buffer
	cmd2.SetOut(&buf2)
	cmd2.SetErr(&buf2)
	cmd2.SetArgs([]string{"add-skill", "nonexistent-skill"})

	err := cmd2.Execute()
	if err == nil {
		t.Fatal("expected error for unknown skill, got nil")
	}
	if !strings.Contains(err.Error(), "unknown skill") {
		t.Errorf("error should mention 'unknown skill', got: %v", err)
	}
}

func TestListSkillsCmd_ShowsAvailable(t *testing.T) {
	tmpDir := t.TempDir()

	cmd := claudecode.ExportClaudeCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"list-skills"})

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	if err := cmd.Execute(); err != nil {
		t.Fatalf("list-skills failed: %v", err)
	}

	output := buf.String()

	// Verify known skills appear in output.
	expectedSkills := []string{"deploy", "review-pr", "security-review", "generate-tests", "refactor", "db-migration"}
	for _, name := range expectedSkills {
		if !strings.Contains(output, name) {
			t.Errorf("output should contain skill %q, got:\n%s", name, output)
		}
	}
}

func TestBuildClaudeAnswersFromFlags(t *testing.T) {
	projectRoot := "/tmp/test-project"
	preset := "standard"
	skills := []string{"deploy", "review-pr"}
	mcpServers := []string{"github"}
	yes := true

	answers := claudecode.ExportBuildClaudeAnswersFromFlags(projectRoot, preset, skills, mcpServers, yes, false)

	if answers.ProjectRoot != projectRoot {
		t.Errorf("ProjectRoot = %q, want %q", answers.ProjectRoot, projectRoot)
	}
	if answers.ProjectName != "test-project" {
		t.Errorf("ProjectName = %q, want %q", answers.ProjectName, "test-project")
	}
	if answers.PermissionLevel != "standard" {
		t.Errorf("PermissionLevel = %q, want %q", answers.PermissionLevel, "standard")
	}
	if !answers.ClaudeCode {
		t.Error("ClaudeCode should be true")
	}
	if !answers.Confirmed {
		t.Error("Confirmed should be true")
	}
	if !answers.Hooks.SafetyBlock {
		t.Error("Hooks.SafetyBlock should be true by default")
	}
	if len(answers.Skills) != 2 {
		t.Fatalf("Skills count = %d, want 2", len(answers.Skills))
	}
	if answers.Skills[0] != "deploy" {
		t.Errorf("Skills[0] = %q, want %q", answers.Skills[0], "deploy")
	}
	if answers.Skills[1] != "review-pr" {
		t.Errorf("Skills[1] = %q, want %q", answers.Skills[1], "review-pr")
	}
	if len(answers.MCPServers) != 1 {
		t.Fatalf("MCPServers count = %d, want 1", len(answers.MCPServers))
	}
	if answers.MCPServers[0] != "github" {
		t.Errorf("MCPServers[0] = %q, want %q", answers.MCPServers[0], "github")
	}
}

func TestInvalidPermissionPresetRejected(t *testing.T) {
	// Verify validPermissionPresets contains the expected values.
	expectedPresets := []string{"minimal", "standard", "permissive", "custom"}
	presets := claudecode.ExportValidPermissionPresets
	if len(presets) != len(expectedPresets) {
		t.Fatalf("validPermissionPresets has %d entries, want %d", len(presets), len(expectedPresets))
	}
	for _, expected := range expectedPresets {
		if !claudecode.ExportContains(presets, expected) {
			t.Errorf("validPermissionPresets missing expected value %q", expected)
		}
	}

	// Verify the contains function correctly rejects invalid values.
	invalidPresets := []string{"bogus", "", "STANDARD", "Minimal", "super-permissive"}
	for _, invalid := range invalidPresets {
		if claudecode.ExportContains(presets, invalid) {
			t.Errorf("contains(validPermissionPresets, %q) = true, want false", invalid)
		}
	}
}

func TestSaveLoadAnswers_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()

	original := types.WizardAnswers{
		ProjectName:     "test-project",
		ProjectRoot:     tmpDir,
		ClaudeCode:      true,
		PermissionLevel: "standard",
		Skills:          []string{"deploy", "review-pr"},
		MCPServers:      []string{"github"},
		Hooks: types.HookChoices{
			SafetyBlock: true,
			AutoFormat:  true,
		},
		Confirmed: true,
	}

	if err := claudecode.ExportSaveAnswers(tmpDir, original); err != nil {
		t.Fatalf("saveAnswers failed: %v", err)
	}

	loaded, err := claudecode.ExportLoadAnswers(tmpDir)
	if err != nil {
		t.Fatalf("loadAnswers failed: %v", err)
	}

	// Verify key fields match.
	if loaded.ProjectName != original.ProjectName {
		t.Errorf("ProjectName = %q, want %q", loaded.ProjectName, original.ProjectName)
	}
	if loaded.PermissionLevel != original.PermissionLevel {
		t.Errorf("PermissionLevel = %q, want %q", loaded.PermissionLevel, original.PermissionLevel)
	}
	if len(loaded.Skills) != len(original.Skills) {
		t.Fatalf("Skills count = %d, want %d", len(loaded.Skills), len(original.Skills))
	}
	for i, skill := range loaded.Skills {
		if skill != original.Skills[i] {
			t.Errorf("Skills[%d] = %q, want %q", i, skill, original.Skills[i])
		}
	}
	if len(loaded.MCPServers) != len(original.MCPServers) {
		t.Fatalf("MCPServers count = %d, want %d", len(loaded.MCPServers), len(original.MCPServers))
	}
	for i, srv := range loaded.MCPServers {
		if srv != original.MCPServers[i] {
			t.Errorf("MCPServers[%d] = %q, want %q", i, srv, original.MCPServers[i])
		}
	}
	if loaded.Hooks.SafetyBlock != original.Hooks.SafetyBlock {
		t.Errorf("Hooks.SafetyBlock = %v, want %v", loaded.Hooks.SafetyBlock, original.Hooks.SafetyBlock)
	}
	if loaded.Hooks.AutoFormat != original.Hooks.AutoFormat {
		t.Errorf("Hooks.AutoFormat = %v, want %v", loaded.Hooks.AutoFormat, original.Hooks.AutoFormat)
	}
	if loaded.ClaudeCode != original.ClaudeCode {
		t.Errorf("ClaudeCode = %v, want %v", loaded.ClaudeCode, original.ClaudeCode)
	}
}
