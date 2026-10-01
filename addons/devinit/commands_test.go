package devinit

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/internal/check"
	qsdevconfig "github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
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

	output, err := executeInitCmd(t, dir, "--lang", "go", "--yes", "--dry-run")
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
	// attempt to run. In a test environment without a TTY, the terminal
	// detection guard returns an error before the wizard starts.
	_, err := executeInitCmd(t, dir)
	if err == nil {
		t.Error("expected error when neither --yes nor complete flags provided")
	}
	if err != nil && !strings.Contains(err.Error(), "stdin is not a terminal") {
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

	// Claude Code answers live only in the primary answers file: a separate
	// .claude copy went stale and overwrote later changes (F515).
	claudeAnswers := filepath.Join(dir, ".claude", ".qsdev-claude-answers.yaml")
	if _, err := os.Stat(claudeAnswers); !os.IsNotExist(err) {
		t.Error(".claude/.qsdev-claude-answers.yaml should not be created")
	}
	primaryAnswers := filepath.Join(dir, ".devinit", ".qsdev-init-answers.yaml")
	if _, err := os.Stat(primaryAnswers); err != nil {
		t.Errorf("primary answers file not saved: %v", err)
	}
}

// TestInitCmd_GitignoresHookAuditLogs verifies init ignores the audit logs the
// Claude Code hooks write under .claude/ (W047): they hold tool inputs and
// commands, while the rest of .claude/ is meant to be committed.
func TestInitCmd_GitignoresHookAuditLogs(t *testing.T) {
	dir := t.TempDir()
	if output, err := executeInitCmd(t, dir, "--lang", "go", "--yes"); err != nil {
		t.Fatalf("init failed: %v\nOutput: %s", err, output)
	}
	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}
	lines := strings.Split(string(data), "\n")
	for _, want := range []string{".claude/logs/", ".claude/hook-audit.log*"} {
		if !slices.Contains(lines, want) {
			t.Errorf(".gitignore missing %q:\n%s", want, data)
		}
	}
}

func TestInitCmd_Quiet(t *testing.T) {
	dir := t.TempDir()

	output, err := executeInitCmd(t, dir, "--lang", "go", "--yes", "--quiet")
	if err != nil {
		t.Fatalf("init --quiet failed: %v\nOutput: %s", err, output)
	}

	if strings.Contains(output, "Next steps") {
		t.Error("--quiet should suppress post-generation summary, but output contains 'Next steps'")
	}

	devenvNix := filepath.Join(dir, "devenv.nix")
	if _, err := os.Stat(devenvNix); os.IsNotExist(err) {
		t.Error("devenv.nix was not created with --quiet")
	}
}

func TestInitCmd_DryRunWithNonInteractive(t *testing.T) {
	dir := t.TempDir()

	output, err := executeInitCmd(t, dir, "--profile", "go-web", "--yes", "--dry-run")
	if err != nil {
		t.Fatalf("init --yes --dry-run failed: %v\nOutput: %s", err, output)
	}

	if output == "" {
		t.Error("dry-run produced no output")
	}

	devenvNix := filepath.Join(dir, "devenv.nix")
	if _, err := os.Stat(devenvNix); !os.IsNotExist(err) {
		t.Error("devenv.nix should not exist after --dry-run")
	}
}

func TestInitCmd_AnswersFilePlusYes(t *testing.T) {
	dir := t.TempDir()

	answersContent := `languages:
  - name: go
    version: "1.24"
direnv: true
claude_code: true
permission_level: standard
`
	answersPath := filepath.Join(dir, "answers.yaml")
	if err := os.WriteFile(answersPath, []byte(answersContent), 0o644); err != nil {
		t.Fatalf("writing answers file: %v", err)
	}

	output, err := executeInitCmd(t, dir, "--answers-file", answersPath, "--yes")
	if err != nil {
		t.Fatalf("init --answers-file --yes failed: %v\nOutput: %s", err, output)
	}

	devenvNix := filepath.Join(dir, "devenv.nix")
	if _, err := os.Stat(devenvNix); os.IsNotExist(err) {
		t.Error("devenv.nix was not created")
	}

	content, err := os.ReadFile(devenvNix)
	if err != nil {
		t.Fatalf("reading devenv.nix: %v", err)
	}
	if !strings.Contains(string(content), "go") {
		t.Error("devenv.nix does not contain 'go'")
	}
}

// TestInitCmd_WarnsPoetryProjectFiles guards W072: init warns when the chosen
// Poetry package manager lacks the pyproject.toml or poetry.lock the devenv
// shell needs, and stays quiet for a complete Poetry project or a
// Claude-only run that generates no devenv configuration.
func TestInitCmd_WarnsPoetryProjectFiles(t *testing.T) {
	tests := []struct {
		name    string
		files   []string
		extra   []string
		want    string
		notWant []string
	}{
		{name: "requirements project", files: []string{"requirements.txt"}, want: "Warning: Python: poetry is the package manager but pyproject.toml is missing"},
		{name: "no lockfile", files: []string{"pyproject.toml"}, want: "Warning: Python: poetry.lock is missing"},
		{name: "complete project", files: []string{"pyproject.toml", "poetry.lock"}, notWant: []string{"pyproject.toml is missing", "poetry.lock is missing"}},
		{name: "claude only", files: []string{"requirements.txt"}, extra: []string{"--claude-only"}, notWant: []string{"pyproject.toml is missing"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, f := range tt.files {
				if err := os.WriteFile(filepath.Join(dir, f), nil, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			args := append([]string{"--lang", "python", "--python-pkg-mgr", "poetry", "--yes", "--dry-run"}, tt.extra...)
			output, err := executeInitCmd(t, dir, args...)
			if err != nil {
				t.Fatalf("init failed: %v\nOutput: %s", err, output)
			}
			if tt.want != "" && !strings.Contains(output, tt.want) {
				t.Errorf("output does not contain %q:\n%s", tt.want, output)
			}
			for _, nw := range tt.notWant {
				if strings.Contains(output, nw) {
					t.Errorf("output unexpectedly contains %q:\n%s", nw, output)
				}
			}
		})
	}
}

// TestUpdateAndJoin_WarnPoetryProjectFiles guards W072 for existing
// projects: `init --update` and join regenerate the Poetry lock check too,
// so they warn about a missing poetry.lock or pyproject.toml like init does.
func TestUpdateAndJoin_WarnPoetryProjectFiles(t *testing.T) {
	t.Run("update", func(t *testing.T) {
		dir := t.TempDir()
		for _, f := range []string{"pyproject.toml", "poetry.lock"} {
			if err := os.WriteFile(filepath.Join(dir, f), nil, 0o644); err != nil {
				t.Fatal(err)
			}
		}
		if out, err := executeInitCmd(t, dir, "--lang", "python", "--python-pkg-mgr", "poetry", "--yes"); err != nil {
			t.Fatalf("init failed: %v\n%s", err, out)
		}
		if err := os.Remove(filepath.Join(dir, "poetry.lock")); err != nil {
			t.Fatal(err)
		}
		out, err := executeInitCmd(t, dir, "--update", "--dry-run")
		if err != nil {
			t.Fatalf("update failed: %v\n%s", err, out)
		}
		if want := "Warning: Python: poetry.lock is missing"; !strings.Contains(out, want) {
			t.Errorf("update output does not contain %q:\n%s", want, out)
		}
	})
	t.Run("join", func(t *testing.T) {
		t.Setenv("QSDEV_SKIP_SETUP", "1")
		dir := t.TempDir()
		config := "version: 1\nlanguages:\n  - name: python\n    package_manager: poetry\n"
		if err := os.WriteFile(filepath.Join(dir, ".qsdev.yaml"), []byte(config), 0o644); err != nil {
			t.Fatal(err)
		}
		cmd, buf := newJoinTestCmd()
		if err := runJoin(cmd, InitOptions{DryRun: true, Yes: true, Quiet: true}, dir); err != nil {
			t.Fatalf("runJoin: %v", err)
		}
		if want := "Warning: Python: poetry is the package manager but pyproject.toml is missing"; !strings.Contains(buf.String(), want) {
			t.Errorf("join output does not contain %q:\n%s", want, buf.String())
		}
	})
}

// initGoProject writes a go.mod into a fresh directory and runs init there
// with args, returning the directory and the command output.
func initGoProject(t *testing.T, args ...string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/aon\n\ngo 1.24\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := executeInitCmd(t, dir, append([]string{"--yes", "--lang", "go"}, args...)...)
	if err != nil {
		t.Fatalf("init %v: %v\n%s", args, err, out)
	}
	return dir, out
}

// committedTools returns tools.enabled and tools.disabled from .qsdev.yaml.
func committedTools(t *testing.T, dir string) (enabled, disabled []string) {
	t.Helper()
	cfg, err := qsdevconfig.ParseQsdevConfig(filepath.Join(dir, ".qsdev.yaml"))
	if err != nil {
		t.Fatalf("parsing .qsdev.yaml: %v", err)
	}
	return cfg.Tools.Enabled, cfg.Tools.Disabled
}

// assertAttachGuardOn checks every place that records the package guard: the
// committed tool list, the saved hook answer, the hook script and its
// settings.json registration.
func assertAttachGuardOn(t *testing.T, dir string) {
	t.Helper()
	if enabled, _ := committedTools(t, dir); !slices.Contains(enabled, toolreg.ToolAttachGuard) {
		t.Errorf(".qsdev.yaml tools.enabled = %v, want it to contain %s", enabled, toolreg.ToolAttachGuard)
	}
	a := loadProjectAnswers(t, dir)
	if !a.Hooks.SafetyBlock || !a.EnabledTools[toolreg.ToolAttachGuard] {
		t.Errorf("answers hooks.safety_block = %v, enabled_tools[%s] = %v, want both true",
			a.Hooks.SafetyBlock, toolreg.ToolAttachGuard, a.EnabledTools[toolreg.ToolAttachGuard])
	}
	if _, err := os.Stat(filepath.Join(dir, ".claude", "hooks", "package-guard.py")); err != nil {
		t.Errorf("package-guard.py not generated: %v", err)
	}
	if settings := readProjectFile(t, dir, ".claude/settings.json"); !strings.Contains(settings, "package-guard.py") {
		t.Error("settings.json does not register package-guard.py")
	}
}

// TestInit_ClaudeHooksKeepsPackageGuard is the U28-01 regression: naming
// hook presets with --claude-hooks adds them to the always-on set instead of
// replacing it, so the package guard is still generated and recorded.
func TestInit_ClaudeHooksKeepsPackageGuard(t *testing.T) {
	dir, out := initGoProject(t, "--claude-hooks", "audit-log")
	if strings.Contains(out, "always-on tool") {
		t.Errorf("--claude-hooks init warned although nothing opted out:\n%s", out)
	}

	assertAttachGuardOn(t, dir)
	if !loadProjectAnswers(t, dir).Hooks.AuditLog {
		t.Error("answers hooks.audit_log = false, want the --claude-hooks preset enabled")
	}
	if _, err := os.Stat(filepath.Join(dir, ".claude", "hooks", "audit-log.sh")); err != nil {
		t.Errorf("audit-log hook not generated: %v", err)
	}
}

// TestReinitForce_ClaudeHooksKeepsAttachGuard is the U28-V01 regression:
// re-initialising with --force and --claude-hooks must not drop attach-guard
// from .qsdev.yaml nor switch off the safety block recorded in the answers.
func TestReinitForce_ClaudeHooksKeepsAttachGuard(t *testing.T) {
	dir, _ := initGoProject(t)
	assertAttachGuardOn(t, dir)

	if out, err := executeInitCmd(t, dir, "--yes", "--lang", "go", "--force", "--claude-hooks", "audit-log"); err != nil {
		t.Fatalf("re-init: %v\n%s", err, out)
	}
	assertAttachGuardOn(t, dir)
}

// TestReinit_RespectsExplicitAlwaysOnDisable verifies `disable --force` is a
// real opt-out: a later re-init keeps attach-guard in tools.disabled and does
// not restore the safety block.
func TestReinit_RespectsExplicitAlwaysOnDisable(t *testing.T) {
	dir, _ := initGoProject(t)
	mustDisable(t, dir, toolreg.ToolAttachGuard, "--force")

	out, err := executeInitCmd(t, dir, "--yes", "--lang", "go", "--force")
	if err != nil {
		t.Fatalf("re-init: %v\n%s", err, out)
	}
	enabled, disabled := committedTools(t, dir)
	if !slices.Contains(disabled, toolreg.ToolAttachGuard) || slices.Contains(enabled, toolreg.ToolAttachGuard) {
		t.Errorf(".qsdev.yaml tools.enabled = %v, tools.disabled = %v, want %s only disabled",
			enabled, disabled, toolreg.ToolAttachGuard)
	}
	a := loadProjectAnswers(t, dir)
	if enabled, set := a.EnabledTools[toolreg.ToolAttachGuard]; !set || enabled {
		t.Errorf("answers enabled_tools[%s] = %v (set %v), want an explicit false", toolreg.ToolAttachGuard, enabled, set)
	}
	if a.Hooks.SafetyBlock {
		t.Error("answers hooks.safety_block = true, want the explicit opt-out kept")
	}
	if settings := readProjectFile(t, dir, ".claude/settings.json"); strings.Contains(settings, "package-guard.py") {
		t.Error("settings.json registers package-guard.py although attach-guard was disabled")
	}
	if strings.Contains(out, "always-on tool") {
		t.Errorf("re-init warned about restoring an opted-out tool:\n%s", out)
	}
}

// TestUpdate_WarnsWhenAlwaysOnRestored verifies update restores an always-on
// tool lost from a project and says so, whether the saved answers dropped it
// (a U28-01 victim) or it was deleted from .qsdev.yaml tools.enabled by hand.
func TestUpdate_WarnsWhenAlwaysOnRestored(t *testing.T) {
	tests := []struct {
		name string
		drop func(t *testing.T, dir string)
	}{
		{"saved answers without the safety block", func(t *testing.T, dir string) {
			a := loadProjectAnswers(t, dir)
			a.Hooks.SafetyBlock = false
			delete(a.EnabledTools, toolreg.ToolAttachGuard)
			if err := saveAnswers(dir, a); err != nil {
				t.Fatal(err)
			}
		}},
		{"entry deleted from tools.enabled", func(t *testing.T, dir string) {
			path := filepath.Join(dir, ".qsdev.yaml")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			lines := strings.SplitAfter(string(data), "\n")
			kept := slices.DeleteFunc(lines, func(l string) bool {
				return strings.TrimSpace(l) == "- "+toolreg.ToolAttachGuard
			})
			if len(kept) == len(lines) {
				t.Fatalf(".qsdev.yaml has no %s entry:\n%s", toolreg.ToolAttachGuard, data)
			}
			if err := os.WriteFile(path, []byte(strings.Join(kept, "")), 0o644); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, _ := initGoProject(t)
			tt.drop(t, dir)

			out, err := executeInitCmd(t, dir, "--update")
			if err != nil {
				t.Fatalf("update: %v\n%s", err, out)
			}
			want := "always-on tool \"attach-guard\" kept enabled; opt out with `qsdev disable attach-guard --force`"
			if !strings.Contains(out, want) {
				t.Errorf("update output does not contain %q:\n%s", want, out)
			}
			assertAttachGuardOn(t, dir)
		})
	}
}

// safetyBlockOptOutWarning is the warning init, update and claude
// init/update print while the package guard is opted out.
const safetyBlockOptOutWarning = "Warning: the package-install guard (attach-guard) is disabled by tools.disabled in .qsdev.yaml; re-enable with `qsdev enable attach-guard`"

// TestInitUpdate_WarnsOnCommittedSafetyBlockOptOut verifies update keeps a
// committed `disable attach-guard --force` opt-out, records it in the answers
// and says so, while a project without the opt-out gets no warning.
func TestInitUpdate_WarnsOnCommittedSafetyBlockOptOut(t *testing.T) {
	tests := []struct {
		name   string
		optOut bool
	}{
		{name: "committed opt-out", optOut: true},
		{name: "no opt-out"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, _ := initGoProject(t)
			if tt.optOut {
				mustDisable(t, dir, toolreg.ToolAttachGuard, "--force")
			}

			out, err := executeInitCmd(t, dir, "--update")
			if err != nil {
				t.Fatalf("update: %v\n%s", err, out)
			}
			if got := strings.Contains(out, safetyBlockOptOutWarning); got != tt.optOut {
				t.Errorf("update printed the opt-out warning = %v, want %v:\n%s", got, tt.optOut, out)
			}
			a := loadProjectAnswers(t, dir)
			if a.Hooks.SafetyBlockOptOut != tt.optOut || a.Hooks.SafetyBlock == tt.optOut {
				t.Errorf("answers safety_block = %v, safety_block_opt_out = %v; want opt-out = %v",
					a.Hooks.SafetyBlock, a.Hooks.SafetyBlockOptOut, tt.optOut)
			}
		})
	}
}

// checkReport runs `check --format json` in dir and returns the report.
func checkReport(t *testing.T, dir string) check.CheckReport {
	t.Helper()
	out, _ := runLifecycleCmd(t, dir, checkCmd(), "--format", "json", "--audit-level", "low")
	i := strings.Index(out, "{")
	if i < 0 {
		t.Fatalf("check printed no JSON report:\n%s", out)
	}
	var report check.CheckReport
	// The JSON report may be followed by cobra's error output.
	if err := json.NewDecoder(strings.NewReader(out[i:])).Decode(&report); err != nil {
		t.Fatalf("parsing report: %v\n%s", err, out)
	}
	return report
}

// missingTools returns the tool_missing_* checks that failed.
func missingTools(report check.CheckReport) []string {
	var names []string
	for _, c := range report.Checks {
		if strings.HasPrefix(c.Name, "tool_missing_") && c.Status == check.StatusFail {
			names = append(names, c.Name)
		}
	}
	return names
}

// TestInit_DefaultProjectHasEveryAlwaysOnTool verifies a default init
// records every always-on tool and generates what backs it, so check finds
// none missing and the trail-of-bits skill it records is on disk.
func TestInit_DefaultProjectHasEveryAlwaysOnTool(t *testing.T) {
	dir, out := initGoProject(t)

	if strings.Contains(out, "always-on tool") {
		t.Errorf("default init warned about an always-on tool:\n%s", out)
	}
	if missing := missingTools(checkReport(t, dir)); len(missing) != 0 {
		t.Errorf("check reports missing always-on tools: %v", missing)
	}
	if _, err := os.Stat(filepath.Join(dir, ".claude", "skills", "security-review-owasp", "SKILL.md")); err != nil {
		t.Errorf("trail-of-bits-skills recorded but its skill is not generated: %v", err)
	}
}

// TestInit_DevenvOnlyRecordsNoClaudeCodeTools is the --devenv-only
// regression: without Claude Code, the always-on tools that configure it are
// neither recorded, warned about, nor required by check.
func TestInit_DevenvOnlyRecordsNoClaudeCodeTools(t *testing.T) {
	dir, out := initGoProject(t, "--devenv-only")

	if strings.Contains(out, "always-on tool") {
		t.Errorf("--devenv-only init warned about an always-on tool:\n%s", out)
	}
	enabled, _ := committedTools(t, dir)
	for _, name := range []string{toolreg.ToolAttachGuard, toolreg.ToolTrailOfBitsSkills} {
		if slices.Contains(enabled, name) {
			t.Errorf(".qsdev.yaml tools.enabled = %v, want no %s without Claude Code", enabled, name)
		}
	}
	if !slices.Contains(enabled, "branch-naming") {
		t.Errorf(".qsdev.yaml tools.enabled = %v, want the project tool branch-naming", enabled)
	}
	if missing := missingTools(checkReport(t, dir)); len(missing) != 0 {
		t.Errorf("check requires Claude Code tools in a --devenv-only project: %v", missing)
	}
}

// TestInit_AnswersFileCannotOptOutAlwaysOn verifies an answers file is not
// an opt-out: its explicit off for attach-guard is dropped with a warning,
// and init does not write the tool to tools.disabled.
func TestInit_AnswersFileCannotOptOutAlwaysOn(t *testing.T) {
	src, _ := initGoProject(t)
	a := loadProjectAnswers(t, src)
	a.Hooks.SafetyBlock = false
	a.EnabledTools[toolreg.ToolAttachGuard] = false
	data, err := yaml.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	answersFile := filepath.Join(t.TempDir(), "answers.yaml")
	if err := os.WriteFile(answersFile, data, 0o644); err != nil {
		t.Fatal(err)
	}

	dir, out := initGoProject(t, "--answers-file", answersFile)

	want := "always-on tool \"attach-guard\" kept enabled; opt out with `qsdev disable attach-guard --force`"
	if !strings.Contains(out, want) {
		t.Errorf("init output does not contain %q:\n%s", want, out)
	}
	if _, disabled := committedTools(t, dir); slices.Contains(disabled, toolreg.ToolAttachGuard) {
		t.Errorf(".qsdev.yaml tools.disabled = %v, want init never to write the opt-out", disabled)
	}
	assertAttachGuardOn(t, dir)
}

// TestInit_AgentPostmortemFalseRejected verifies --agent-postmortem=false is
// rejected with a pointer to the only opt-out rather than silently ignored.
func TestInit_AgentPostmortemFalseRejected(t *testing.T) {
	dir := t.TempDir()
	out, err := executeInitCmd(t, dir, "--yes", "--lang", "go", "--agent-postmortem=false")
	if err == nil {
		t.Fatalf("init --agent-postmortem=false succeeded:\n%s", out)
	}
	if want := "qsdev disable agent-postmortem --force"; !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want it to point at %q", err, want)
	}
	if _, statErr := os.Stat(filepath.Join(dir, ".qsdev.yaml")); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf(".qsdev.yaml written despite the rejected flag: %v", statErr)
	}
}

// TestUpdate_RestoresTrailOfBitsSkill covers a project from a release that
// recorded trail-of-bits-skills without generating its skill: check fails
// with tool_missing, and update restores both the entry and the skill.
func TestUpdate_RestoresTrailOfBitsSkill(t *testing.T) {
	dir, _ := initGoProject(t)
	a := loadProjectAnswers(t, dir)
	a.Skills = slices.DeleteFunc(a.Skills, func(s string) bool { return s == "security-review-owasp" })
	delete(a.EnabledTools, toolreg.ToolTrailOfBitsSkills)
	if err := saveAnswers(dir, a); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(dir, ".qsdev.yaml")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	stripped := strings.ReplaceAll(string(data), "        - "+toolreg.ToolTrailOfBitsSkills+"\n", "")
	stripped = strings.ReplaceAll(stripped, "    - "+toolreg.ToolTrailOfBitsSkills+"\n", "")
	if stripped == string(data) {
		t.Fatalf(".qsdev.yaml has no %s entry:\n%s", toolreg.ToolTrailOfBitsSkills, data)
	}
	if err := os.WriteFile(cfgPath, []byte(stripped), 0o644); err != nil {
		t.Fatal(err)
	}
	// That release never generated the skill, so neither disk nor state has it.
	if err := os.RemoveAll(filepath.Join(dir, ".claude", "skills", "security-review-owasp")); err != nil {
		t.Fatal(err)
	}
	st, err := state.LoadStateFromFile(filepath.Join(dir, stateFilePath()))
	if err != nil {
		t.Fatal(err)
	}
	delete(st.Files, ".claude/skills/security-review-owasp/SKILL.md")
	if err := state.SaveInitState(dir, st); err != nil {
		t.Fatal(err)
	}

	if missing := missingTools(checkReport(t, dir)); !slices.Equal(missing, []string{"tool_missing_" + toolreg.ToolTrailOfBitsSkills}) {
		t.Errorf("check missing = %v, want only %s", missing, toolreg.ToolTrailOfBitsSkills)
	}
	out, err := executeInitCmd(t, dir, "--update")
	if err != nil {
		t.Fatalf("update: %v\n%s", err, out)
	}
	if enabled, _ := committedTools(t, dir); !slices.Contains(enabled, toolreg.ToolTrailOfBitsSkills) {
		t.Errorf(".qsdev.yaml tools.enabled = %v, want %s restored", enabled, toolreg.ToolTrailOfBitsSkills)
	}
	if _, err := os.Stat(filepath.Join(dir, ".claude", "skills", "security-review-owasp", "SKILL.md")); err != nil {
		t.Errorf("update did not restore the skill: %v", err)
	}
}
