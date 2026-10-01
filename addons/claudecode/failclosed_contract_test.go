package claudecode_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	claudecode "github.com/Quantum-Serendipity/qsdev/addons/claudecode"
)

// failClosedCase is one fail-closed registry hook as Generate emits it.
type failClosedCase struct {
	name    string
	command string
	event   string
}

// failClosedContractCases writes the real hook templates into a fresh
// project and returns, for every FailClosed registry definition with the
// sandbox off and on, the command buildHooks emits and a PreToolUse event
// its matcher selects. The set comes from the registry, not a list here.
func failClosedContractCases(t *testing.T) (project string, cases []failClosedCase) {
	t.Helper()
	project = t.TempDir()
	if err := os.Mkdir(filepath.Join(project, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	files, err := claudecode.ExportGenerateHookFiles(allHooksAnswers(false))
	if err != nil {
		t.Fatalf("GenerateHookFiles: %v", err)
	}
	for _, f := range files {
		p := filepath.Join(project, filepath.FromSlash(f.Path))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, f.Content, f.Mode); err != nil {
			t.Fatal(err)
		}
	}
	for _, sandbox := range []bool{false, true} {
		built := claudecode.ExportBuildHooks(allHooksAnswers(sandbox))
		for _, d := range claudecode.ExportDefaultHookRegistry().Definitions() {
			if !d.FailClosed {
				continue
			}
			name := d.Owner + "/" + d.Event
			if sandbox {
				name += "/sandbox"
			}
			cases = append(cases, failClosedCase{
				name:    name,
				command: builtFailClosedCommand(t, built[d.Event], d),
				event:   hookEventJSON(t, project, d.Matcher),
			})
		}
	}
	if len(cases) == 0 {
		t.Fatal("registry has no fail-closed hooks")
	}
	return project, cases
}

// builtFailClosedCommand finds d's command among the built matchers by the
// owner named in its fail-closed reason.
func builtFailClosedCommand(t *testing.T, matchers []claudecode.HookMatcher, d claudecode.ExportHookDefinition) string {
	t.Helper()
	marker := "qsdev: " + d.Owner + " hook could not run"
	var found []string
	for _, m := range matchers {
		if m.Matcher != d.Matcher {
			continue
		}
		for _, h := range m.Hooks {
			if strings.Contains(h.Command, marker) {
				found = append(found, h.Command)
			}
		}
	}
	if len(found) != 1 {
		t.Fatalf("%s/%s: %d built fail-closed commands, want 1: %q", d.Owner, d.Event, len(found), found)
	}
	return found[0]
}

// hookEventJSON builds a PreToolUse event for the first tool matcher names
// ("*" selects Bash).
func hookEventJSON(t *testing.T, project, matcher string) string {
	t.Helper()
	tool, _, _ := strings.Cut(matcher, "|")
	var input map[string]string
	switch tool {
	case "*", "Bash", "PowerShell", "Monitor":
		tool, input = "Bash", map[string]string{"command": "npm install lodash"}
	default:
		input = map[string]string{"file_path": filepath.Join(project, "main.go"), "content": "package main\n"}
	}
	b, err := json.Marshal(map[string]any{
		"session_id": "xs-ws1", "cwd": project, "hook_event_name": "PreToolUse",
		"tool_name": tool, "tool_input": input,
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// requireBlocked asserts the fail-closed contract: exit 2 with the wrapper's
// reason, never a pass or a non-blocking error.
func requireBlocked(t *testing.T, rc int, stdout, stderr string) {
	t.Helper()
	if rc != 2 {
		t.Errorf("rc = %d, want 2 (stdout %q, stderr %q)", rc, stdout, stderr)
	}
	if !strings.Contains(stderr, "could not run") {
		t.Errorf("stderr %q lacks the 'could not run' reason", stderr)
	}
}

// TestFailClosedContract_MissingProgramBlocks is XS-WS1 A1/A2: with nothing
// on PATH, neither the hook's interpreter (env python3) nor the qsdev binary
// resolves (exit 127), and every fail-closed hook must still block.
func TestFailClosedContract_MissingProgramBlocks(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("hook commands run under POSIX sh")
	}
	project, cases := failClosedContractCases(t)
	env := []string{"PATH=/nonexistent", "CLAUDE_PROJECT_DIR=" + project}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rc, stdout, stderr := runShInput(t, tc.command, env, tc.event)
			requireBlocked(t, rc, stdout, stderr)
		})
	}
}

// TestFailClosedContract_CrashBlocks is XS-WS1 A5: the interpreter and the
// qsdev binary resolve but crash with an uncaught exception (exit 1), which
// Claude Code would treat as non-blocking; every fail-closed hook must block.
func TestFailClosedContract_CrashBlocks(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("stub programs are POSIX shell scripts")
	}
	project, cases := failClosedContractCases(t)
	// Written once, before the parallel subtests exec them (ETXTBSY).
	stubs := t.TempDir()
	const crash = "#!/bin/sh\necho 'Traceback (most recent call last):' >&2\necho 'RuntimeError: boom' >&2\nexit 1\n"
	for _, name := range []string{"python3", "qsdev"} {
		if err := os.WriteFile(filepath.Join(stubs, name), []byte(crash), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	env := []string{"PATH=" + stubs, "CLAUDE_PROJECT_DIR=" + project}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rc, stdout, stderr := runShInput(t, tc.command, env, tc.event)
			requireBlocked(t, rc, stdout, stderr)
			if !strings.Contains(stderr, "Traceback") {
				t.Errorf("stderr %q: the stub did not run, so this case did not exercise a crash", stderr)
			}
		})
	}
}
