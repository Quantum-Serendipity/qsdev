package devinit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/hookio"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/rules"
)

func executeSelfprotect(t *testing.T, stdin string) (string, error) {
	t.Helper()
	cmd := selfprotectCmd()
	var stderr bytes.Buffer
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&stderr)
	cmd.SetArgs(nil)
	err := cmd.Execute()
	return stderr.String(), err
}

// TestRunSelfprotect_VerdictsInProcess drives the real hook entry point in
// process: every deny path must return exit code 2 (not os.Exit) with the
// reason on stderr, and an allowed call must return nil.
func TestRunSelfprotect_VerdictsInProcess(t *testing.T) {
	protected := filepath.Join(t.TempDir(), ".claude", "hooks", "guard.sh")
	payload := func(tool string, input map[string]string) string {
		data, err := json.Marshal(map[string]any{"tool_name": tool, "tool_input": input})
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}

	tests := []struct {
		name       string
		stdin      string
		wantDeny   bool
		wantStderr string
	}{
		{
			name:       "malformed input is denied",
			stdin:      "not json",
			wantDeny:   true,
			wantStderr: "internal error",
		},
		{
			name:       "write to a protected hook script is denied",
			stdin:      payload("Write", map[string]string{"file_path": protected, "content": "exit 0"}),
			wantDeny:   true,
			wantStderr: "qsdev-selfprotect:",
		},
		{
			name:  "benign write is allowed",
			stdin: payload("Write", map[string]string{"file_path": filepath.Join(t.TempDir(), "notes.txt"), "content": "hello"}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stderr, err := executeSelfprotect(t, tt.stdin)
			if !tt.wantDeny {
				if err != nil {
					t.Fatalf("expected allow, got %v (stderr %q)", err, stderr)
				}
				return
			}
			var exitErr *ExitError
			if !errors.As(err, &exitErr) || exitErr.Code != 2 {
				t.Fatalf("expected exit code 2, got %v (stderr %q)", err, stderr)
			}
			if exitErr.Error() != "" {
				t.Errorf("deny error message = %q, want empty so nothing is appended to stderr", exitErr.Error())
			}
			if !strings.Contains(stderr, tt.wantStderr) {
				t.Errorf("stderr %q does not contain %q", stderr, tt.wantStderr)
			}
		})
	}
}

// TestBuildSelfprotectContext_CanonicalizationFailureStaysProtected pins the
// fail-closed fallback: when a protected target cannot be canonicalized (here
// an ancestor directory is unreadable), CanonicalPath must not be left empty,
// which would make every path rule allow the write.
func TestBuildSelfprotectContext_CanonicalizationFailureStaysProtected(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory permission bits do not block traversal on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root bypasses directory permissions")
	}

	locked := filepath.Join(t.TempDir(), "locked")
	if err := os.Mkdir(locked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

	target := filepath.Join(locked, ".claude", "hooks", "guard.sh")
	ctx := buildSelfprotectContext("Write", &hookio.ToolInput{FilePath: target, Content: "exit 0"})

	if ctx.CanonicalPath == "" {
		t.Fatal("CanonicalPath is empty after a canonicalization failure")
	}
	if v, _ := rules.Tier1Rules.EvaluateAll(ctx); v != rules.Deny {
		t.Errorf("write to protected path behind an unreadable directory = %v, want Deny", v)
	}
}

// TestRunSelfprotect_HookSurfaces drives the hook entry point against the
// surfaces that change which hooks run (F150): a nested Claude Code session
// started without the hook settings (SP-008), and a file placed where a
// registered hook program resolves through PATH (SP-011), read from the
// environment Claude Code gives its hooks.
func TestRunSelfprotect_HookSurfaces(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project")
	early := filepath.Join(root, "early")
	bin := filepath.Join(root, "bin")
	settings := `{"hooks":{"PreToolUse":[{"matcher":"*","hooks":[{"type":"command","command":"hookprog run"}]}]}}`
	for p, content := range map[string]string{
		filepath.Join(project, ".claude", "settings.json"): settings,
		filepath.Join(bin, "hookprog"):                     "binary",
	} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(early, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLAUDE_PROJECT_DIR", project)
	t.Setenv("PATH", early+string(os.PathListSeparator)+bin)

	payload := func(tool string, input map[string]string) string {
		data, err := json.Marshal(map[string]any{"tool_name": tool, "tool_input": input, "cwd": project})
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	tests := []struct {
		name, stdin, wantRule string
	}{
		{"nested session without hooks", payload("Bash", map[string]string{"command": "claude --bare -p x"}), "SP-008"},
		{"shadowed hook program", payload("Write", map[string]string{"file_path": filepath.Join(early, "hookprog"), "content": "exit 0"}), "SP-011"},
		{"plain nested session", payload("Bash", map[string]string{"command": `claude -p "review"`}), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stderr, err := executeSelfprotect(t, tt.stdin)
			if tt.wantRule == "" {
				if err != nil {
					t.Fatalf("expected allow, got %v (stderr %q)", err, stderr)
				}
				return
			}
			var exitErr *ExitError
			if !errors.As(err, &exitErr) || exitErr.Code != 2 || !strings.Contains(stderr, tt.wantRule) {
				t.Fatalf("expected %s deny with exit code 2, got %v (stderr %q)", tt.wantRule, err, stderr)
			}
		})
	}
}

// newSelfprotectTestCmd returns a selfprotect command wired to in-memory
// stdin and stderr.
func newSelfprotectTestCmd(stdin string) (*cobra.Command, *bytes.Buffer) {
	cmd := selfprotectCmd()
	var stderr bytes.Buffer
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetErr(&stderr)
	cmd.SetContext(context.Background())
	return cmd, &stderr
}

func requireSelfprotectDeny(t *testing.T, err error, stderr, want string) {
	t.Helper()
	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 2 {
		t.Fatalf("expected exit code 2, got %v (stderr %q)", err, stderr)
	}
	if !strings.Contains(stderr, want) {
		t.Errorf("stderr %q does not contain %q", stderr, want)
	}
}

// TestRunSelfprotect_DeadlineDenies verifies an evaluation that overruns the
// deadline is denied (fail closed) instead of running until Claude Code's own
// hook timeout, which would let the call through.
func TestRunSelfprotect_DeadlineDenies(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	blocking := func(_ context.Context, _ io.Reader, w io.Writer) error {
		<-release
		_, _ = io.WriteString(w, "late")
		return nil
	}

	cmd, stderr := newSelfprotectTestCmd(`{"tool_name":"Bash","tool_input":{"command":"true"}}`)
	start := time.Now()
	err := runSelfprotectWith(cmd, 50*time.Millisecond, blocking)
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("runSelfprotectWith returned after %v, want the 50ms deadline", elapsed)
	}
	requireSelfprotectDeny(t, err, stderr.String(), "SP-TIMEOUT")
}

// TestRunSelfprotect_PanicInEvaluatorDenies verifies a panic anywhere in the
// evaluation is a deny, not a crash whose exit code Claude Code would ignore.
func TestRunSelfprotect_PanicInEvaluatorDenies(t *testing.T) {
	t.Parallel()

	panicking := func(context.Context, io.Reader, io.Writer) error { panic("rule bug") }
	cmd, stderr := newSelfprotectTestCmd(`{"tool_name":"Bash","tool_input":{"command":"true"}}`)
	err := runSelfprotectWith(cmd, time.Second, panicking)
	requireSelfprotectDeny(t, err, stderr.String(), "rule bug")
}

// TestSelfprotect_TooManySimpleCommandsDenies verifies a command with more
// than MaxSimpleCommands simple commands is denied outright, while a long but
// realistic chain is still evaluated and allowed.
func TestSelfprotect_TooManySimpleCommandsDenies(t *testing.T) {
	t.Parallel()

	bash := func(command string) string {
		return toolCallJSON(t, "Bash", map[string]any{"command": command})
	}

	stderr, err := executeSelfprotect(t, bash(strings.Repeat("true; ", hookio.MaxSimpleCommands+1)))
	requireSelfprotectDeny(t, err, stderr, "SP-LIMIT")

	stderr, err = executeSelfprotect(t, bash(strings.Repeat("echo hi && ", 800)+"true"))
	if err != nil {
		t.Errorf("800-segment echo chain: expected allow, got %v (stderr %q)", err, stderr)
	}
}

// TestSelfprotect_AdversarialInputsUnderBudget runs the production hook as a
// process against inputs that used to take tens of seconds (or never finish),
// long past Claude Code's hook timeout, which then allows the call. Each must
// now be denied with exit 2 well inside the budget. This is XS-WS1 A3.
func TestSelfprotect_AdversarialInputsUnderBudget(t *testing.T) {
	t.Parallel()

	const budget = 2 * time.Second
	tests := []struct {
		name    string
		command string
	}{
		{"unclosed braces hiding a delete", "echo " + strings.Repeat("{", 200000) + "; rm -rf .claude/settings.json"},
		{"repeated unclosed brace words", strings.Repeat("echo {x", 200000)},
		{"unclosed braces", "echo " + strings.Repeat("{", 160000)},
		{"long cd chain", strings.Repeat("cd a && ", 5000) + "true"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			payload := toolCallJSON(t, "Bash", map[string]any{"command": tt.command})
			start := time.Now()
			code, stderr := runSelfprotectHook(t, t.TempDir(), payload)
			elapsed := time.Since(start)
			if code != 2 {
				t.Fatalf("exit code = %d, want 2 (stderr %.200q)", code, stderr)
			}
			if elapsed > budget {
				t.Errorf("hook took %v, want under %v", elapsed, budget)
			}
		})
	}
}
