package devinit

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// executeInitCmd creates and runs the init command in the given directory with
// the provided args. It returns the combined stdout/stderr and any error.
func executeInitCmd(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	t.Setenv("QSDEV_SKIP_SETUP", "1")
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir to %s: %v", dir, err)
	}

	cmd := initCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return buf.String(), err
}

func TestInitCmd_HasCorrectUseAndFlags(t *testing.T) {
	cmd := initCmd()

	if cmd.Use != "init" {
		t.Errorf("Use = %q, want %q", cmd.Use, "init")
	}

	// Verify key flags are registered.
	expectedFlags := []string{
		"lang", "service", "yes", "force", "dry-run",
		"devenv-only", "claude-only", "profile", "list-profiles",
		"go-version", "node-version", "node-pkg-mgr",
		"python-version", "python-pkg-mgr", "rust-channel",
		"java-version", "java-build-tool",
		"direnv", "git-hooks", "packages", "env",
		"nix-hardening-guide", "infra-profile",
		"claude-code", "claude-permissions", "claude-skills",
		"claude-hooks", "mcp",
	}

	for _, name := range expectedFlags {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag %q not found", name)
		}
	}
}

func TestInitCmd_DryRun(t *testing.T) {
	dir := t.TempDir()

	output, err := executeInitCmd(t, dir, "--yes", "--dry-run")
	if err != nil {
		t.Fatalf("dry-run failed: %v\nOutput: %s", err, output)
	}

	// Dry-run should show preview output.
	if output == "" {
		t.Error("dry-run produced no output")
	}

	// Verify no files were written (only the temp dir should exist, empty).
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		t.Errorf("dry-run wrote unexpected file/dir: %s", e.Name())
	}
}

func TestInitCmd_WritesFiles(t *testing.T) {
	dir := t.TempDir()

	output, err := executeInitCmd(t, dir, "--lang", "go", "--yes")
	if err != nil {
		t.Fatalf("init failed: %v\nOutput: %s", err, output)
	}

	// Verify devenv files were created.
	devenvNix := filepath.Join(dir, "devenv.nix")
	if _, err := os.Stat(devenvNix); os.IsNotExist(err) {
		t.Error("devenv.nix was not created")
	}

	// Verify Claude Code files were created (claude-code defaults to true).
	settingsJSON := filepath.Join(dir, ".claude", "settings.json")
	if _, err := os.Stat(settingsJSON); os.IsNotExist(err) {
		t.Error(".claude/settings.json was not created")
	}
}

func TestInitCmd_DevenvOnly(t *testing.T) {
	dir := t.TempDir()

	output, err := executeInitCmd(t, dir, "--lang", "go", "--yes", "--devenv-only")
	if err != nil {
		t.Fatalf("init --devenv-only failed: %v\nOutput: %s", err, output)
	}

	// Verify devenv files exist.
	devenvNix := filepath.Join(dir, "devenv.nix")
	if _, err := os.Stat(devenvNix); os.IsNotExist(err) {
		t.Error("devenv.nix was not created with --devenv-only")
	}

	// Verify Claude Code files were NOT created.
	settingsJSON := filepath.Join(dir, ".claude", "settings.json")
	if _, err := os.Stat(settingsJSON); !os.IsNotExist(err) {
		t.Error(".claude/settings.json should not exist with --devenv-only")
	}
}

func TestInitCmd_ClaudeOnly(t *testing.T) {
	dir := t.TempDir()

	output, err := executeInitCmd(t, dir, "--lang", "go", "--yes", "--claude-only")
	if err != nil {
		t.Fatalf("init --claude-only failed: %v\nOutput: %s", err, output)
	}

	// Verify devenv files were NOT created.
	devenvNix := filepath.Join(dir, "devenv.nix")
	if _, err := os.Stat(devenvNix); !os.IsNotExist(err) {
		t.Error("devenv.nix should not exist with --claude-only")
	}

	// Verify Claude Code files were created.
	settingsJSON := filepath.Join(dir, ".claude", "settings.json")
	if _, err := os.Stat(settingsJSON); os.IsNotExist(err) {
		t.Error(".claude/settings.json was not created with --claude-only")
	}
}

func TestInitCmd_ForceOverwrite(t *testing.T) {
	dir := t.TempDir()

	// Create an existing devenv.nix.
	existingContent := []byte("# existing devenv.nix\n")
	if err := os.WriteFile(filepath.Join(dir, "devenv.nix"), existingContent, 0o644); err != nil {
		t.Fatalf("creating existing devenv.nix: %v", err)
	}

	// Without --force, should fail.
	_, err := executeInitCmd(t, dir, "--lang", "go", "--yes")
	if err == nil {
		t.Error("expected error without --force when existing config found")
	}

	// With --force, should succeed.
	output, err := executeInitCmd(t, dir, "--lang", "go", "--yes", "--force")
	if err != nil {
		t.Fatalf("init --force failed: %v\nOutput: %s", err, output)
	}

	// Verify devenv.nix was overwritten.
	content, err := os.ReadFile(filepath.Join(dir, "devenv.nix"))
	if err != nil {
		t.Fatalf("reading devenv.nix: %v", err)
	}
	if string(content) == string(existingContent) {
		t.Error("devenv.nix was not overwritten with --force")
	}
}

func TestInitCmd_ListProfiles(t *testing.T) {
	dir := t.TempDir()

	output, err := executeInitCmd(t, dir, "--list-profiles")
	if err != nil {
		t.Fatalf("--list-profiles failed: %v\nOutput: %s", err, output)
	}

	// Should contain the built-in profile names.
	expectedProfiles := []string{"go-web", "ts-fullstack", "python-data", "rust-cli"}
	for _, name := range expectedProfiles {
		if !strings.Contains(output, name) {
			t.Errorf("output missing profile %q:\n%s", name, output)
		}
	}
}

func TestInitCmd_Profile(t *testing.T) {
	dir := t.TempDir()

	output, err := executeInitCmd(t, dir, "--profile", "go-web", "--yes")
	if err != nil {
		t.Fatalf("init --profile go-web failed: %v\nOutput: %s", err, output)
	}

	// The go-web profile includes Go, so devenv.nix should be generated.
	devenvNix := filepath.Join(dir, "devenv.nix")
	if _, err := os.Stat(devenvNix); os.IsNotExist(err) {
		t.Error("devenv.nix was not created with --profile go-web")
	}

	// The go-web profile includes ClaudeCode=true, so settings.json should exist.
	settingsJSON := filepath.Join(dir, ".claude", "settings.json")
	if _, err := os.Stat(settingsJSON); os.IsNotExist(err) {
		t.Error(".claude/settings.json was not created with --profile go-web")
	}
}

func TestInitCmd_SavesAnswers(t *testing.T) {
	dir := t.TempDir()

	output, err := executeInitCmd(t, dir, "--lang", "go", "--yes")
	if err != nil {
		t.Fatalf("init failed: %v\nOutput: %s", err, output)
	}

	answersFile := filepath.Join(dir, ".devinit", ".qsdev-init-answers.yaml")
	if _, err := os.Stat(answersFile); os.IsNotExist(err) {
		t.Error(".devinit/.qsdev-init-answers.yaml was not created")
	}
}

func TestInitCmd_SavesState(t *testing.T) {
	dir := t.TempDir()

	output, err := executeInitCmd(t, dir, "--lang", "go", "--yes")
	if err != nil {
		t.Fatalf("init failed: %v\nOutput: %s", err, output)
	}

	stateFile := filepath.Join(dir, ".devinit", ".qsdev-init-state.yaml")
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		t.Error(".devinit/.qsdev-init-state.yaml was not created")
	}
}

func TestInitCmd_RequiresWizardOrYes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("huh TUI forms hang on Windows without TTY")
	}
	dir := t.TempDir()

	// Without --yes and without a complete set of flags, the wizard will
	// attempt to run. In a test environment without a TTY, this results in
	// a "running wizard" error (wrapping the TTY open failure).
	_, err := executeInitCmd(t, dir)
	if err == nil {
		t.Error("expected error when neither --yes nor complete flags provided")
	}
	if err != nil && !strings.Contains(err.Error(), "running wizard") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestInitCmd_UnknownProfile(t *testing.T) {
	dir := t.TempDir()

	_, err := executeInitCmd(t, dir, "--profile", "nonexistent", "--yes")
	if err == nil {
		t.Error("expected error for unknown profile")
	}
	if err != nil && !strings.Contains(err.Error(), "unknown profile") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestInitCmd_SavesPerAddonAnswers(t *testing.T) {
	dir := t.TempDir()

	output, err := executeInitCmd(t, dir, "--lang", "go", "--yes")
	if err != nil {
		t.Fatalf("init failed: %v\nOutput: %s", err, output)
	}

	// Verify devenv answers were saved.
	devenvAnswers := filepath.Join(dir, ".devenv", ".qsdev-answers.yaml")
	if _, err := os.Stat(devenvAnswers); os.IsNotExist(err) {
		t.Error(".devenv/.qsdev-answers.yaml was not created")
	}

	// Verify Claude Code answers were saved (claude-code defaults to true).
	claudeAnswers := filepath.Join(dir, ".claude", ".qsdev-claude-answers.yaml")
	if _, err := os.Stat(claudeAnswers); os.IsNotExist(err) {
		t.Error(".claude/.qsdev-claude-answers.yaml was not created")
	}
}
