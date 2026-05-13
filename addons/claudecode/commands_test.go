package claudecode_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/addons/claudecode"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
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

// chdir changes the working directory to dir and registers a cleanup to restore
// the original directory when the test finishes.
func chdir(t *testing.T, dir string) {
	t.Helper()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })
}

func TestInitCmd_InvalidPreset(t *testing.T) {
	tmpDir := t.TempDir()
	chdir(t, tmpDir)

	cmd := claudecode.ExportClaudeCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"init", "--permission-preset", "bogus", "--yes"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid permission preset")
	}
	if !strings.Contains(err.Error(), "unknown permission preset") {
		t.Errorf("error should mention 'unknown permission preset', got: %v", err)
	}
	if !strings.Contains(err.Error(), "bogus") {
		t.Errorf("error should include the bad preset name 'bogus', got: %v", err)
	}
}

func TestInitCmd_ExistingSettings_NoForce(t *testing.T) {
	tmpDir := t.TempDir()
	chdir(t, tmpDir)

	// Create existing .claude/settings.json.
	claudeDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	settingsPath := filepath.Join(claudeDir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{"existing": true}`), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := claudecode.ExportClaudeCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"init", "--permission-preset", "standard", "--yes"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when settings.json exists without --force")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error should mention 'already exists', got: %v", err)
	}
}

func TestInitCmd_ExistingSettings_Force(t *testing.T) {
	tmpDir := t.TempDir()
	chdir(t, tmpDir)

	// Create existing .claude/settings.json.
	claudeDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	settingsPath := filepath.Join(claudeDir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{"existing": true}`), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := claudecode.ExportClaudeCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"init", "--permission-preset", "standard", "--yes", "--force"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init --force should succeed: %v", err)
	}

	// Verify settings.json was overwritten.
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("reading settings.json: %v", err)
	}
	if strings.Contains(string(data), `"existing"`) {
		t.Error("settings.json should have been overwritten by --force")
	}
}

func TestUpdateCmd_NoSavedAnswers(t *testing.T) {
	tmpDir := t.TempDir()
	chdir(t, tmpDir)

	cmd := claudecode.ExportClaudeCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"update"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when no saved answers exist")
	}
	if !strings.Contains(err.Error(), "no saved answers") {
		t.Errorf("error should mention 'no saved answers', got: %v", err)
	}
}

func TestUpdateCmd_AfterInit(t *testing.T) {
	tmpDir := t.TempDir()
	chdir(t, tmpDir)

	// First, init.
	cmd := claudecode.ExportClaudeCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"init", "--yes", "--permission-preset", "standard"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Now, update.
	cmd2 := claudecode.ExportClaudeCmd()
	var buf2 bytes.Buffer
	cmd2.SetOut(&buf2)
	cmd2.SetErr(&buf2)
	cmd2.SetArgs([]string{"update", "--force"})

	if err := cmd2.Execute(); err != nil {
		t.Fatalf("update after init failed: %v", err)
	}

	output := buf2.String()
	if !strings.Contains(output, "complete") {
		t.Errorf("update output should confirm completion, got: %s", output)
	}

	// Verify files still exist after update.
	settingsPath := filepath.Join(tmpDir, ".claude", "settings.json")
	if _, err := os.Stat(settingsPath); err != nil {
		t.Errorf("settings.json should still exist after update: %v", err)
	}
}

func TestUpdateCmd_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	chdir(t, tmpDir)

	// First, init.
	cmd := claudecode.ExportClaudeCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"init", "--yes", "--permission-preset", "standard"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Get mod time of settings.json before update.
	settingsPath := filepath.Join(tmpDir, ".claude", "settings.json")
	infoBefore, err := os.Stat(settingsPath)
	if err != nil {
		t.Fatalf("stat settings.json: %v", err)
	}

	// Run update with --dry-run.
	cmd2 := claudecode.ExportClaudeCmd()
	var buf2 bytes.Buffer
	cmd2.SetOut(&buf2)
	cmd2.SetErr(&buf2)
	cmd2.SetArgs([]string{"update", "--dry-run"})

	if err := cmd2.Execute(); err != nil {
		t.Fatalf("update --dry-run failed: %v", err)
	}

	output := buf2.String()
	if !strings.Contains(output, "settings.json") {
		t.Error("dry-run output should mention settings.json")
	}

	// Verify file was NOT re-written.
	infoAfter, err := os.Stat(settingsPath)
	if err != nil {
		t.Fatalf("stat settings.json after dry-run: %v", err)
	}
	if infoAfter.ModTime() != infoBefore.ModTime() {
		t.Error("dry-run should not modify files on disk")
	}
}

func TestAddSkillCmd_Duplicate(t *testing.T) {
	tmpDir := t.TempDir()
	chdir(t, tmpDir)

	// First, init with a skill.
	cmd := claudecode.ExportClaudeCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"init", "--yes", "--permission-preset", "standard", "--skills", "deploy"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Try to add the same skill again.
	cmd2 := claudecode.ExportClaudeCmd()
	var buf2 bytes.Buffer
	cmd2.SetOut(&buf2)
	cmd2.SetErr(&buf2)
	cmd2.SetArgs([]string{"add-skill", "deploy"})

	err := cmd2.Execute()
	if err == nil {
		t.Fatal("expected error for duplicate skill")
	}
	if !strings.Contains(err.Error(), "already configured") {
		t.Errorf("error should mention 'already configured', got: %v", err)
	}
}

func TestAddHookCmd_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	chdir(t, tmpDir)

	// First, init.
	cmd := claudecode.ExportClaudeCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"init", "--yes", "--permission-preset", "standard"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Add hook.
	cmd2 := claudecode.ExportClaudeCmd()
	var buf2 bytes.Buffer
	cmd2.SetOut(&buf2)
	cmd2.SetErr(&buf2)
	cmd2.SetArgs([]string{"add-hook", "audit-log"})

	if err := cmd2.Execute(); err != nil {
		t.Fatalf("add-hook audit-log failed: %v", err)
	}

	output := buf2.String()
	if !strings.Contains(output, "audit-log") {
		t.Errorf("output should mention enabled hook, got: %s", output)
	}

	// Verify answers now have audit-log enabled.
	answers, err := claudecode.ExportLoadAnswers(tmpDir)
	if err != nil {
		t.Fatalf("loading answers: %v", err)
	}
	if !answers.Hooks.AuditLog {
		t.Error("Hooks.AuditLog should be true after add-hook audit-log")
	}
}

func TestAddHookCmd_Invalid(t *testing.T) {
	tmpDir := t.TempDir()
	chdir(t, tmpDir)

	// First, init.
	cmd := claudecode.ExportClaudeCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"init", "--yes", "--permission-preset", "standard"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Try invalid hook.
	cmd2 := claudecode.ExportClaudeCmd()
	var buf2 bytes.Buffer
	cmd2.SetOut(&buf2)
	cmd2.SetErr(&buf2)
	cmd2.SetArgs([]string{"add-hook", "nonexistent"})

	err := cmd2.Execute()
	if err == nil {
		t.Fatal("expected error for invalid hook")
	}
	if !strings.Contains(err.Error(), "unknown hook preset") {
		t.Errorf("error should mention 'unknown hook preset', got: %v", err)
	}
}

func TestListSkillsCmd_ShowsInstalledStatus(t *testing.T) {
	tmpDir := t.TempDir()
	chdir(t, tmpDir)

	// Init with a skill installed.
	cmd := claudecode.ExportClaudeCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"init", "--yes", "--permission-preset", "standard", "--skills", "deploy"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// List skills.
	cmd2 := claudecode.ExportClaudeCmd()
	var buf2 bytes.Buffer
	cmd2.SetOut(&buf2)
	cmd2.SetErr(&buf2)
	cmd2.SetArgs([]string{"list-skills"})

	if err := cmd2.Execute(); err != nil {
		t.Fatalf("list-skills failed: %v", err)
	}

	output := buf2.String()

	// deploy should be marked as installed.
	lines := strings.Split(output, "\n")
	foundInstalled := false
	for _, line := range lines {
		if strings.Contains(line, "deploy") && strings.Contains(line, "(installed)") {
			foundInstalled = true
			break
		}
	}
	if !foundInstalled {
		t.Errorf("deploy skill should be marked as (installed) in output:\n%s", output)
	}
}

func TestInitCmd_NoSafetyBlock(t *testing.T) {
	tmpDir := t.TempDir()
	chdir(t, tmpDir)

	cmd := claudecode.ExportClaudeCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"init", "--yes", "--permission-preset", "standard", "--no-safety-block"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init with --no-safety-block failed: %v", err)
	}

	// Verify answers have safety block disabled.
	answers, err := claudecode.ExportLoadAnswers(tmpDir)
	if err != nil {
		t.Fatalf("loading answers: %v", err)
	}
	if answers.Hooks.SafetyBlock {
		t.Error("Hooks.SafetyBlock should be false when --no-safety-block is used")
	}
}

func TestInitCmd_SavesStateAndAnswers(t *testing.T) {
	tmpDir := t.TempDir()
	chdir(t, tmpDir)

	cmd := claudecode.ExportClaudeCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"init", "--yes", "--permission-preset", "standard"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Verify state file.
	stateFile := filepath.Join(tmpDir, ".claude", ".gdev-claude-state.yaml")
	if _, err := os.Stat(stateFile); err != nil {
		t.Errorf("state file not saved: %v", err)
	}
	stateData, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("reading state file: %v", err)
	}
	if len(stateData) == 0 {
		t.Error("state file should not be empty")
	}

	// Verify answers file.
	answersFile := filepath.Join(tmpDir, ".claude", ".gdev-claude-answers.yaml")
	if _, err := os.Stat(answersFile); err != nil {
		t.Errorf("answers file not saved: %v", err)
	}
	answersData, err := os.ReadFile(answersFile)
	if err != nil {
		t.Fatalf("reading answers file: %v", err)
	}
	if len(answersData) == 0 {
		t.Error("answers file should not be empty")
	}
}

func TestInitCmd_DryRunNoFiles(t *testing.T) {
	tmpDir := t.TempDir()
	chdir(t, tmpDir)

	cmd := claudecode.ExportClaudeCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"init", "--dry-run", "--yes", "--permission-preset", "standard"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init --dry-run failed: %v", err)
	}

	// Verify neither settings.json, state, nor answers were created.
	for _, relPath := range []string{
		".claude/settings.json",
		".claude/.gdev-claude-state.yaml",
		".claude/.gdev-claude-answers.yaml",
		"CLAUDE.md",
	} {
		absPath := filepath.Join(tmpDir, relPath)
		if _, err := os.Stat(absPath); err == nil {
			t.Errorf("dry-run should not write %s", relPath)
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
