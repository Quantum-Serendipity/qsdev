package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/check"
)

// TestInvalidProjectDefaultsNoPanic is the end-to-end regression test for
// G-05: a project .qsdev/defaults.yaml that tries to loosen the built-in
// defaults made check, status and list panic in the tool registry and every
// command log the catalog failure at startup. Each command now fails with a
// clean error (check with an in-band report), version stays quiet, the
// defaults commands still diagnose the file, and the self-protection hook
// keeps its allow/deny contract.
func TestInvalidProjectDefaultsNoPanic(t *testing.T) {
	t.Parallel()
	env := guardrailEnv(t)
	dir := newGuardrailProject(t, env)
	bad := catalog.ProjectConfigPath(dir)
	if err := os.MkdirAll(filepath.Dir(bad), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bad, []byte("keep_vars: [AWS_SECRET_ACCESS_KEY]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	const violation = "may only add or tighten"
	noCrash := func(t *testing.T, out string) {
		t.Helper()
		for _, s := range []string{"panic:", "goroutine "} {
			if strings.Contains(out, s) {
				t.Errorf("output contains %q:\n%s", s, out)
			}
		}
	}

	for _, args := range [][]string{{"check"}, {"status"}, {"list"}, {"disable", "x"}, {"disable", "gitleaks"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			t.Parallel()
			out, code := runQsdev(t, env, dir, nil, args...)
			noCrash(t, out)
			if code != 1 || !strings.Contains(out, violation) || !strings.Contains(out, "defaults validate") {
				t.Errorf("exit %d, want 1 naming the violation and 'defaults validate':\n%s", code, out)
			}
			// The gate and check share one wording; check must not stutter
			// the tool registry's own "loading catalog" wrap.
			if !strings.Contains(out, "loading qsdev defaults: project defaults "+violation) {
				t.Errorf("output does not read 'loading qsdev defaults: project defaults %s':\n%s", violation, out)
			}
		})
	}

	// Commands that read no defaults catalog keep working: bare groups only
	// print their usage, gdev's addons lists addons, config show prints the
	// project config as written.
	for _, args := range [][]string{{"addons"}, {"config"}, {"config", "show"}, {"claude"}, {"devenv"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			t.Parallel()
			out, code := runQsdev(t, env, dir, nil, args...)
			noCrash(t, out)
			if code != 0 || strings.Contains(out, violation) {
				t.Errorf("exit %d, want 0 without the catalog error:\n%s", code, out)
			}
		})
	}

	t.Run("check --format json", func(t *testing.T) {
		t.Parallel()
		stdout, stderr, code := runQsdevSplit(t, env, dir, "check", "--format", "json")
		noCrash(t, stdout+stderr)
		var report check.CheckReport
		if err := json.Unmarshal([]byte(stdout), &report); err != nil {
			t.Fatalf("stdout is not a JSON report: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
		}
		if code != 1 {
			t.Errorf("exit %d, want 1", code)
		}
	})

	t.Run("version", func(t *testing.T) {
		t.Parallel()
		_, stderr, code := runQsdevSplit(t, env, dir, "version")
		if code != 0 || strings.Contains(stderr, "ERROR loading built-in project profiles") {
			t.Errorf("exit %d, want 0 without the startup ERROR line:\n%s", code, stderr)
		}
	})

	t.Run("defaults validate", func(t *testing.T) {
		t.Parallel()
		out, code := runQsdev(t, env, dir, nil, "defaults", "validate")
		noCrash(t, out)
		if code != 1 || !strings.Contains(out, violation) {
			t.Errorf("exit %d, want 1 naming the violation:\n%s", code, out)
		}
	})

	selfprotect := func(t *testing.T, tool string, input map[string]string) (string, int) {
		t.Helper()
		payload, err := json.Marshal(map[string]any{
			"hook_event_name": "PreToolUse",
			"cwd":             dir,
			"tool_name":       tool,
			"tool_input":      input,
		})
		if err != nil {
			t.Fatal(err)
		}
		return runQsdev(t, agentEnv(env), dir, bytes.NewReader(payload), "selfprotect")
	}
	t.Run("selfprotect allow", func(t *testing.T) {
		t.Parallel()
		if out, code := selfprotect(t, "Bash", map[string]string{"command": "ls"}); code != 0 {
			t.Errorf("exit %d, want 0:\n%s", code, out)
		}
	})
	t.Run("selfprotect deny", func(t *testing.T) {
		t.Parallel()
		settings := filepath.Join(dir, ".claude", "settings.json")
		out, code := selfprotect(t, "Edit", map[string]string{"file_path": settings, "old_string": "a", "new_string": "b"})
		if code != 2 {
			t.Errorf("exit %d, want 2 (deny):\n%s", code, out)
		}
	})
}

// runQsdevSplit runs qsdev with args in dir and returns its stdout, stderr
// and exit code.
func runQsdevSplit(t *testing.T, env []string, dir string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	cmd := exec.Command(os.Args[0], args...)
	cmd.Dir = dir
	cmd.Env = env
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	err := cmd.Run()
	var exitErr *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &exitErr):
		code = exitErr.ExitCode()
	default:
		t.Fatalf("running qsdev %s: %v", strings.Join(args, " "), err)
	}
	return out.String(), errOut.String(), code
}
