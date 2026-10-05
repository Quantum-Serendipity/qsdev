package claudecode_test

import (
	"bytes"
	"errors"
	"maps"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	claudecode "github.com/Quantum-Serendipity/qsdev/addons/claudecode"
)

// hookLibTemplate is the shared Python hook library's template, relative to
// the package directory.
var hookLibTemplate = filepath.Join("templates", "hooks", path.Base(claudecode.HookLibPath))

// TestHookLib_AuditLogDefinedOnce guards U17-14: the audit log writer, the
// interpreter floor and the deadline watchdog live once, in the shared hook
// library, instead of a diverging copy per hook.
func TestHookLib_AuditLogDefinedOnce(t *testing.T) {
	t.Parallel()
	scripts, err := filepath.Glob(filepath.Join("templates", "hooks", "*.py"))
	if err != nil {
		t.Fatal(err)
	}
	var definers []string
	for _, script := range scripts {
		src, err := os.ReadFile(script)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(src, []byte("\ndef audit_log(")) {
			definers = append(definers, filepath.Base(script))
		}
	}
	if want := []string{path.Base(claudecode.HookLibPath)}; !slices.Equal(definers, want) {
		t.Errorf("audit_log defined in %v, want only %v", definers, want)
	}
	for _, script := range slices.Sorted(maps.Keys(failClosedPythonHooks(t))) {
		src, err := os.ReadFile(filepath.Join("templates", "hooks", script))
		if err != nil {
			t.Fatal(err)
		}
		for _, copied := range []string{"_MIN_PYTHON = (", "def _arm_deadline", "def _deadline_seconds", "def _append_private"} {
			if bytes.Contains(src, []byte(copied)) {
				t.Errorf("%s still carries its own %q; use the shared hook library", script, copied)
			}
		}
	}
}

// TestPythonHooks_FailClosedWithoutHookLib pins that a security hook whose
// shared library is missing, empty or stale (lacking what the hook uses)
// blocks (exit 2) instead of running unguarded or crashing with exit 1,
// which Claude Code lets through.
func TestPythonHooks_FailClosedWithoutHookLib(t *testing.T) {
	t.Parallel()
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not available; skipping hook behaviour test")
	}
	libs := []struct {
		name string
		lib  *string // nil: no library beside the hook
	}{
		{"missing", nil},
		{"empty", new("")},
		{"stale", new("MIN_PYTHON = (3, 9)\n\ndef arm_deadline(hook, seconds):\n    pass\n")},
	}
	for _, script := range slices.Sorted(maps.Keys(failClosedPythonHooks(t))) {
		for _, lib := range libs {
			t.Run(script+"/"+lib.name, func(t *testing.T) {
				t.Parallel()
				content, err := os.ReadFile(filepath.Join("templates", "hooks", script))
				if err != nil {
					t.Fatal(err)
				}
				dir := t.TempDir()
				alone := filepath.Join(dir, script)
				if err := os.WriteFile(alone, content, 0o755); err != nil {
					t.Fatal(err)
				}
				if lib.lib != nil {
					libPath := filepath.Join(dir, path.Base(claudecode.HookLibPath))
					if err := os.WriteFile(libPath, []byte(*lib.lib), 0o644); err != nil {
						t.Fatal(err)
					}
				}
				cmd := exec.Command(python, alone)
				cmd.Env = hookEnv(t)
				cmd.Stdin = strings.NewReader(`{"tool_name":"Bash","tool_input":{"command":"ls"}}`)
				var stderr bytes.Buffer
				cmd.Stderr = &stderr
				err = cmd.Run()
				if exitErr, ok := errors.AsType[*exec.ExitError](err); !ok || exitErr.ExitCode() != 2 {
					t.Errorf("exit = %v, want exit status 2 (stderr %q)", err, stderr.String())
				}
				if !strings.Contains(stderr.String(), "hook library unavailable") {
					t.Errorf("stderr %q does not name the unavailable hook library", stderr.String())
				}
			})
		}
	}
}

// runHookUnparseable runs a security hook template on input that is not JSON,
// which every hook records in the audit log, and returns once it exits.
func runHookUnparseable(t *testing.T, script, dir string, env []string) {
	t.Helper()
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not available; skipping hook behaviour test")
	}
	abs, err := filepath.Abs(filepath.Join("templates", "hooks", script))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(python, abs)
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdin = strings.NewReader("not json")
	_ = cmd.Run() // the decision is not under test, only where it is logged
}

// TestAuditLog_NoCwdFallback guards U17-14: without CLAUDE_PROJECT_DIR the
// audit log goes to the user's own ~/.claude/logs, never into whatever
// directory the hook happens to run in.
func TestAuditLog_NoCwdFallback(t *testing.T) {
	t.Parallel()
	for _, script := range slices.Sorted(maps.Keys(failClosedPythonHooks(t))) {
		t.Run(script, func(t *testing.T) {
			t.Parallel()
			for _, tc := range []struct {
				name  string
				unset bool
			}{{"unset", true}, {"empty", false}} {
				t.Run(tc.name, func(t *testing.T) {
					t.Parallel()
					cwd, home := t.TempDir(), t.TempDir()
					env := hookEnv(t, "CLAUDE_PROJECT_DIR=", "HOME="+home, "USERPROFILE="+home)
					if tc.unset {
						env = slices.DeleteFunc(env, func(kv string) bool {
							return strings.HasPrefix(kv, "CLAUDE_PROJECT_DIR=")
						})
					}
					runHookUnparseable(t, script, cwd, env)
					if _, err := os.Stat(filepath.Join(cwd, ".claude", "logs")); err == nil {
						t.Errorf("%s wrote an audit log under its working directory", script)
					}
					_, entries := readJSONLines(t, filepath.Join(home, ".claude", "logs", "hook-audit.jsonl"))
					if len(entries) == 0 {
						t.Errorf("%s logged nothing under $HOME/.claude/logs", script)
					}
				})
			}
		})
	}
}

// TestAuditLog_PlantedSymlinkNotFollowed pins that no security hook appends
// its audit entry through a symlink planted in place of the log.
func TestAuditLog_PlantedSymlinkNotFollowed(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need privileges on Windows")
	}
	for _, script := range slices.Sorted(maps.Keys(failClosedPythonHooks(t))) {
		t.Run(script, func(t *testing.T) {
			t.Parallel()
			project := t.TempDir()
			victim := filepath.Join(t.TempDir(), "victim")
			if err := os.WriteFile(victim, []byte("original\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			logs := filepath.Join(project, ".claude", "logs")
			if err := os.MkdirAll(logs, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(victim, filepath.Join(logs, "hook-audit.jsonl")); err != nil {
				t.Fatal(err)
			}
			runHookUnparseable(t, script, project, hookEnv(t, "CLAUDE_PROJECT_DIR="+project))
			if data, _ := os.ReadFile(victim); string(data) != "original\n" {
				t.Errorf("%s wrote through the planted symlink: %q", script, data)
			}
		})
	}
}

// TestAuditLog_PlantedLockSymlink pins that a symlink planted in place of the
// audit log's lock file is not followed and does not silence the log: the
// entry is still appended, unlocked.
func TestAuditLog_PlantedLockSymlink(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need privileges on Windows")
	}
	for _, script := range slices.Sorted(maps.Keys(failClosedPythonHooks(t))) {
		t.Run(script, func(t *testing.T) {
			t.Parallel()
			project := t.TempDir()
			victim := filepath.Join(t.TempDir(), "victim")
			if err := os.WriteFile(victim, []byte("original\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			logs := filepath.Join(project, ".claude", "logs")
			if err := os.MkdirAll(logs, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(victim, filepath.Join(logs, "hook-audit.jsonl.lock")); err != nil {
				t.Fatal(err)
			}
			runHookUnparseable(t, script, project, hookEnv(t, "CLAUDE_PROJECT_DIR="+project))
			if data, _ := os.ReadFile(victim); string(data) != "original\n" {
				t.Errorf("%s wrote through the planted lock symlink: %q", script, data)
			}
			if _, entries := readJSONLines(t, filepath.Join(logs, "hook-audit.jsonl")); len(entries) == 0 {
				t.Errorf("%s logged nothing when the lock file could not be opened", script)
			}
		})
	}
}

// auditLogWriter is one concurrent audit-log writer: it loads the shared hook
// library by path, sets the rotation threshold to four and a half lines,
// widens the window between the size check and the append (each lstat
// returns 50ms late), waits at a barrier (a directory every writer adds its
// name to) until all writers are ready so they race, then appends one entry.
const auditLogWriter = `import importlib.util, json, os, sys, time
from datetime import datetime, timezone
spec = importlib.util.spec_from_file_location('_qsdev_hooklib', os.environ['LIB_PATH'])
lib = importlib.util.module_from_spec(spec)
spec.loader.exec_module(lib)
entry = {'event': 'concurrent', 'n': int(os.environ['WRITER']), 'pad': 'x' * 200}
line = json.dumps(dict(entry, timestamp=datetime.now(timezone.utc).isoformat())) + '\n'
lib.AUDIT_LOG_MAX_BYTES = len(line) * 9 // 2
real_lstat = os.lstat
def slow_lstat(path):
    try:
        return real_lstat(path)
    finally:
        time.sleep(0.05)
os.lstat = slow_lstat
barrier = os.environ['BARRIER']
open(os.path.join(barrier, os.environ['WRITER']), 'w').close()
for _ in range(20000):
    if len(os.listdir(barrier)) >= int(os.environ['WRITERS']):
        break
    time.sleep(0.001)
lib.audit_log(entry)
`

// TestAuditLog_ConcurrentRotation guards U17-14: rotation and append share
// one lock, so concurrent hooks crossing the size threshold rotate exactly
// once and no entry is lost or torn. Eight writers cross the threshold once.
// Without the lock, writers act on a size another writer has already
// changed: none rotates, or a second rotation overwrites .1.
func TestAuditLog_ConcurrentRotation(t *testing.T) {
	t.Parallel()
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not available; skipping hook behaviour test")
	}
	lib, err := filepath.Abs(hookLibTemplate)
	if err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	barrier := t.TempDir()
	const writers = 8
	cmds := make([]*exec.Cmd, writers)
	for i := range cmds {
		cmd := exec.Command(python, "-c", auditLogWriter)
		cmd.Env = hookEnv(t, "CLAUDE_PROJECT_DIR="+project, "LIB_PATH="+lib, "BARRIER="+barrier,
			"WRITER="+strconv.Itoa(i), "WRITERS="+strconv.Itoa(writers))
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		cmds[i] = cmd
	}
	for _, cmd := range cmds {
		if err := cmd.Wait(); err != nil {
			t.Fatalf("writer failed: %v", err)
		}
	}
	log := filepath.Join(project, ".claude", "logs", "hook-audit.jsonl")
	_, current := readJSONLines(t, log)
	_, rotated := readJSONLines(t, log+".1")
	if got := len(current) + len(rotated); got != writers {
		t.Errorf("entries in log (%d) + .1 (%d) = %d, want %d", len(current), len(rotated), got, writers)
	}
}
