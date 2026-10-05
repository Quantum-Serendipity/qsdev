package devinit

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/internal/check"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
	"github.com/spf13/cobra"
)

// enableToolFilesForTest plans and applies an enable of tool into root with an
// empty generated state, returning the files written.
func enableToolFilesForTest(t *testing.T, tool *toolreg.Tool, name, root string, answers types.WizardAnswers) ([]types.GeneratedFile, error) {
	t.Helper()
	st := types.GeneratedState{Files: map[string]types.FileState{}}
	change, err := planToolEnable(tool, name, root, answers, st, false)
	if err != nil {
		return nil, err
	}
	res, err := applyToolChange(root, change, st)
	return res.written, err
}

// TestEnableTool_AlwaysOnAgentPostmortem_StandardTier is the BL-P1-9
// regression: enabling the always-on agent-postmortem tool at the standard tier
// must write its exclusive SKILL.md (RED before the fix — silently omitted).
func TestEnableTool_AlwaysOnAgentPostmortem_StandardTier(t *testing.T) {
	root := t.TempDir()
	tool, ok := toolreg.DefaultRegistry().ByName("agent-postmortem")
	if !ok {
		t.Fatal("agent-postmortem not registered")
	}
	answers := types.WizardAnswers{
		Tier:         "standard",
		ProjectRoot:  root,
		AgentTools:   types.AgentToolsAnswers{PostmortemEnabled: true},
		EnabledTools: map[string]bool{},
	}

	written, err := enableToolFilesForTest(t, tool, "agent-postmortem", root, answers)
	if err != nil {
		t.Fatalf("enable: %v", err)
	}

	skillPath := filepath.Join(root, ".claude", "skills", "agent-postmortem", "SKILL.md")
	data, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("expected SKILL.md written at standard tier, got: %v", err)
	}
	if !bytes.HasPrefix(data, []byte("---\n")) {
		t.Errorf("generated SKILL.md must carry YAML front-matter, got:\n%.80s", data)
	}

	var sawSkill bool
	for _, f := range written {
		if f.Path == ".claude/skills/agent-postmortem/SKILL.md" {
			sawSkill = true
		}
	}
	if !sawSkill {
		t.Error("SKILL.md not reported in written files")
	}
}

// TestEnableTool_OptInBelowFull_RefusesInsteadOfFalseSuccess is the BL-P1-9
// honesty guard: enabling a Full-gated opt-in tool (lookup-docs) below Full must
// refuse with an actionable tier error and write nothing — never report success
// while silently omitting the SKILL.md and leaving CLAUDE.md advertising it.
func TestEnableTool_OptInBelowFull_RefusesInsteadOfFalseSuccess(t *testing.T) {
	root := t.TempDir()
	tool, ok := toolreg.DefaultRegistry().ByName("lookup-docs")
	if !ok {
		t.Fatal("lookup-docs not registered")
	}
	answers := types.WizardAnswers{
		Tier:         "standard",
		ProjectRoot:  root,
		EnabledTools: map[string]bool{},
	}

	_, err := enableToolFilesForTest(t, tool, "lookup-docs", root, answers)
	if err == nil {
		t.Fatal("expected an actionable error enabling a Full-gated tool below Full, got nil")
	}
	if !strings.Contains(err.Error(), "tier") {
		t.Errorf("error should mention the tier requirement, got: %v", err)
	}

	// Nothing may be written on refusal — no advertisement, no missing SKILL.md.
	if _, statErr := os.Stat(filepath.Join(root, "CLAUDE.md")); statErr == nil {
		t.Error("CLAUDE.md must not be written when enable refuses")
	}
	if _, statErr := os.Stat(filepath.Join(root, ".claude", "skills", "lookup-docs", "SKILL.md")); statErr == nil {
		t.Error("lookup-docs SKILL.md must be absent when enable refuses")
	}
}

func TestPrintWrittenFiles_Empty(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)

	printWrittenFiles(cmd, nil, "wrote")

	if buf.Len() != 0 {
		t.Errorf("expected no output for empty files, got %q", buf.String())
	}
}

func TestPrintWrittenFiles_MultipleFiles(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)

	files := []types.GeneratedFile{
		{Path: "secretspec.toml"},
		{Path: ".claude/settings.json"},
	}
	printWrittenFiles(cmd, files, "wrote")

	out := buf.String()
	if !strings.Contains(out, "Files wrote:") {
		t.Error("output should contain 'Files wrote:' header")
	}
	if !strings.Contains(out, "secretspec.toml") {
		t.Error("output should list secretspec.toml")
	}
	if !strings.Contains(out, ".claude/settings.json") {
		t.Error("output should list .claude/settings.json")
	}
}

func TestPrintWrittenFiles_VerbParam(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)

	files := []types.GeneratedFile{{Path: "test.txt"}}
	printWrittenFiles(cmd, files, "would write")

	if !strings.Contains(buf.String(), "Files would write:") {
		t.Errorf("expected 'Files would write:' header, got %q", buf.String())
	}
}

func TestIsMachineReadableFormat(t *testing.T) {
	t.Parallel()
	tests := []struct {
		format check.OutputFormat
		want   bool
	}{
		{check.FormatJSON, true},
		{check.FormatSARIF, true},
		{check.FormatJUnit, true},
		{check.FormatHuman, false},
		{check.OutputFormat("unknown"), false},
	}
	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			t.Parallel()
			if got := isMachineReadableFormat(tt.format); got != tt.want {
				t.Errorf("isMachineReadableFormat(%q) = %v, want %v", tt.format, got, tt.want)
			}
		})
	}
}

// pythonHooks returns the Python hook scripts the project in dir holds,
// besides the shared hook library.
func pythonHooks(t *testing.T, dir string) []string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, ".claude", "hooks", "*.py"))
	if err != nil {
		t.Fatal(err)
	}
	var hooks []string
	for _, m := range matches {
		if rel := ".claude/hooks/" + filepath.Base(m); rel != claudecode.HookLibPath {
			hooks = append(hooks, rel)
		}
	}
	return hooks
}

// TestDisable_RemovesHookLibWhenNoPythonHookRemains pins that disabling the
// last Python hook's tool also removes the shared hook library the generator
// no longer writes, and forgets it, instead of leaving an inert library
// behind. The lifecycle project's only Python hook is package-guard.py.
func TestDisable_RemovesHookLibWhenNoPythonHookRemains(t *testing.T) {
	dir := initLifecycleProject(t)
	if got := pythonHooks(t, dir); !slices.Equal(got, []string{claudecode.PackageGuardPath}) {
		t.Fatalf("fixture: Python hooks = %v, want only %s", got, claudecode.PackageGuardPath)
	}
	requireFileExists(t, dir, claudecode.HookLibPath)

	out, err := disableTool(t, dir, "attach-guard", "--force")
	if err != nil {
		t.Fatalf("disable: %v\n%s", err, out)
	}

	requireFileNotExists(t, dir, claudecode.HookLibPath)
	if _, tracked := loadProjectState(t, dir).Files[claudecode.HookLibPath]; tracked {
		t.Errorf("%s is still tracked after disable", claudecode.HookLibPath)
	}
	if !strings.Contains(out, claudecode.HookLibPath) {
		t.Errorf("disable output does not report removing %s:\n%s", claudecode.HookLibPath, out)
	}
}

// TestDisable_KeepsModifiedHookLib pins that a hook library the user edited is
// never deleted by disable: it is left in place and no longer tracked.
func TestDisable_KeepsModifiedHookLib(t *testing.T) {
	dir := initLifecycleProject(t)
	lib := filepath.Join(dir, filepath.FromSlash(claudecode.HookLibPath))
	edited := []byte("# my edits\n")
	if err := os.WriteFile(lib, edited, 0o644); err != nil {
		t.Fatal(err)
	}

	mustDisable(t, dir, "attach-guard", "--force")

	if got, err := os.ReadFile(lib); err != nil || !bytes.Equal(got, edited) {
		t.Errorf("edited %s = %q, %v; want it kept as edited", claudecode.HookLibPath, got, err)
	}
	if _, tracked := loadProjectState(t, dir).Files[claudecode.HookLibPath]; tracked {
		t.Errorf("edited %s is still tracked after disable", claudecode.HookLibPath)
	}
}

// TestDisable_KeepsHookLibWhilePythonHookRemains pins that disable leaves the
// library alone while another Python hook still loads it, and that it never
// removes a tool's file as a no-longer-generated orphan: those belong to that
// tool's own disable.
func TestDisable_KeepsHookLibWhilePythonHookRemains(t *testing.T) {
	dir := initLifecycleProject(t)
	before := loadProjectState(t, dir)

	mustDisable(t, dir, "semgrep", "--force")

	requireFileExists(t, dir, claudecode.HookLibPath)
	requireFileExists(t, dir, claudecode.PackageGuardPath)
	after := loadProjectState(t, dir)
	for rel, entry := range before.Files {
		if _, tracked := after.Files[rel]; !tracked && entry.Owner != "semgrep" {
			if _, err := os.Lstat(filepath.Join(dir, filepath.FromSlash(rel))); err != nil {
				t.Errorf("disabling semgrep removed %s (owner %q): %v", rel, entry.Owner, err)
			}
		}
	}
}

// TestEnable_RestoresHookLib pins that enabling a tool whose Python hook
// loads the shared hook library also writes and tracks the library when the
// project lacks it: a hook without its library fails closed and blocks every
// shell call, even `ls`. A disable that stopped the library's last user
// removed it; a project from an older qsdev never had it.
func TestEnable_RestoresHookLib(t *testing.T) {
	generated := claudecode.HookScriptContents()[claudecode.HookLibPath]
	cases := []struct {
		name    string
		prepare func(t *testing.T, lib string) // after the disable removed the library
		force   bool
		wantErr bool
	}{
		{name: "after disable removed it", prepare: func(*testing.T, string) {}},
		{name: "untracked copy already present", prepare: func(t *testing.T, lib string) {
			t.Helper()
			if err := os.WriteFile(lib, generated, 0o644); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "foreign file is not overwritten", prepare: func(t *testing.T, lib string) {
			t.Helper()
			if err := os.WriteFile(lib, []byte("# mine\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}, wantErr: true},
		{name: "foreign file is replaced with --force", prepare: func(t *testing.T, lib string) {
			t.Helper()
			if err := os.WriteFile(lib, []byte("# mine\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}, force: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := initLifecycleProject(t)
			lib := filepath.Join(dir, filepath.FromSlash(claudecode.HookLibPath))
			mustDisable(t, dir, "attach-guard", "--force")
			requireFileNotExists(t, dir, claudecode.HookLibPath)
			tc.prepare(t, lib)

			args := []string{"attach-guard"}
			if tc.force {
				args = append(args, "--force")
			}
			out, err := enableTool(t, dir, args...)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("enable over a foreign %s succeeded:\n%s", claudecode.HookLibPath, out)
				}
				return
			}
			if err != nil {
				t.Fatalf("enable: %v\n%s", err, out)
			}
			if got, err := os.ReadFile(lib); err != nil || !bytes.Equal(got, generated) {
				t.Fatalf("%s after enable: %v (generated content: %t)", claudecode.HookLibPath, err, bytes.Equal(got, generated))
			}
			if entry, tracked := loadProjectState(t, dir).Files[claudecode.HookLibPath]; !tracked || entry.Owner != "hooks-lib" {
				t.Errorf("%s tracked = %t owner %q, want tracked with owner hooks-lib", claudecode.HookLibPath, tracked, entry.Owner)
			}
			if code := runProjectGuard(t, dir, "ls"); code != 0 {
				t.Errorf("project package-guard on `ls` exited %d, want 0", code)
			}
		})
	}
}

// runProjectGuard runs the project's own package-guard.py on a Bash command
// and returns its exit code.
func runProjectGuard(t *testing.T, dir, command string) int {
	t.Helper()
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not available; skipping the guard run")
	}
	payload, err := json.Marshal(map[string]any{"tool_name": "Bash", "tool_input": map[string]any{"command": command}})
	if err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	cmd := exec.Command(python, filepath.Join(dir, filepath.FromSlash(claudecode.PackageGuardPath)))
	cmd.Stdin = bytes.NewReader(payload)
	cmd.Env = append(os.Environ(), "PYTHONDONTWRITEBYTECODE=1", "HOME="+home, "USERPROFILE="+home,
		"CLAUDE_PROJECT_DIR="+dir, "CLAUDE_AUDIT_DIR="+t.TempDir())
	out, err := cmd.CombinedOutput()
	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
		t.Logf("package-guard output: %s", out)
		return exitErr.ExitCode()
	}
	if err != nil {
		t.Fatalf("running package-guard: %v", err)
	}
	return 0
}
