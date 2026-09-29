//go:build linux

package bwrap

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
)

// TestBubblewrapBackend_E3_ForwardsStdin pins that the sandboxed hook receives
// its stdin byte for byte. Claude Code delivers the tool-call payload there; a
// hook that reads /dev/null instead blocks every call or lets every call
// through.
func TestBubblewrapBackend_E3_ForwardsStdin(t *testing.T) {
	t.Parallel()
	backend, shPath, mounts := e3Backend(t)

	payload := "{\"tool_name\":\"Bash\",\"tool_input\":{\"command\":\"rm -rf /\"}}\n"
	var stdout bytes.Buffer
	cfg := &sandbox.SandboxConfig{
		HookCategory: sandbox.CategoryLinter,
		Mounts:       mounts,
		Environment:  map[string]string{"PATH": "/nonexistent", "HOME": t.TempDir()},
		// The shell's read builtin avoids depending on a cat binary being
		// reachable inside the minimal sandbox.
		HookCommand: []string{shPath, "-c", `while IFS= read -r line; do printf '%s\n' "$line"; done`},
		ExecOpts:    sandbox.ExecOpts{Stdin: strings.NewReader(payload), Stdout: &stdout},
	}

	res, err := backend.RunHook(context.Background(), cfg)
	if err != nil {
		t.Fatalf("RunHook: %v (stderr=%q)", err, stderrOf(res))
	}
	if res.ExitCode != 0 {
		t.Fatalf("exit code %d (stderr=%q)", res.ExitCode, stderrOf(res))
	}
	if got := stdout.String(); got != payload {
		t.Errorf("hook echoed %q, want the stdin payload %q", got, payload)
	}
}

// TestBubblewrapBackend_E3_AppliesResourceLimits pins that the bwrap tiers
// apply the policy's resource limits by running the sandbox inside a transient
// systemd user scope, and that the user-bus variables systemd-run needs do not
// leak into the hook. It is skipped without a usable systemd user session.
func TestBubblewrapBackend_E3_AppliesResourceLimits(t *testing.T) {
	t.Parallel()
	_, shPath, mounts := e3Backend(t)

	systemdRun, err := exec.LookPath("systemd-run")
	if err != nil {
		t.Skip("systemd-run not installed")
	}
	if err := sandbox.UserScopeUsable(systemdRun); err != nil {
		t.Skipf("no systemd user session: %v", err)
	}
	bwrapPath, err := exec.LookPath("bwrap")
	if err != nil {
		t.Skip("bwrap not available")
	}
	backend := NewBubblewrapBackend(sandbox.TierFull, bwrapPath, true, WithSystemdRun(systemdRun))

	script := `while IFS= read -r l; do printf 'cgroup=%s\n' "$l"; done < /proc/self/cgroup; ` +
		`printf 'bus=%s\n' "${DBUS_SESSION_BUS_ADDRESS:-unset}"`
	cfg := &sandbox.SandboxConfig{
		HookCategory: sandbox.CategoryLinter,
		Mounts:       mounts,
		Environment:  map[string]string{"PATH": "/nonexistent", "HOME": t.TempDir()},
		Resources:    sandbox.ResourceLimits{MemoryBytes: 512 << 20, MaxPIDs: 256},
		HookCommand:  []string{shPath, "-c", script},
	}

	res, err := backend.RunHook(context.Background(), cfg)
	if err != nil {
		t.Fatalf("RunHook: %v (stderr=%q)", err, stderrOf(res))
	}
	out := string(res.Stdout)
	// The test process itself often already runs in a scope (a terminal or
	// session scope), so the hook must be in a scope of its own.
	own := cgroupLine(t)
	hookCgroup := ""
	for line := range strings.SplitSeq(out, "\n") {
		if v, ok := strings.CutPrefix(line, "cgroup="); ok {
			hookCgroup = v
		}
	}
	if hookCgroup == own || !strings.HasSuffix(hookCgroup, ".scope") {
		t.Errorf("hook is not inside a transient scope of its own (no limits applied): hook cgroup %q, test cgroup %q, stderr=%q",
			hookCgroup, own, res.Stderr)
	}
	if !strings.Contains(out, "bus=unset") {
		t.Errorf("user-bus address leaked into the hook: stdout=%q", out)
	}
}

// TestBubblewrapBackend_E3_EmptyFilteredEnvDoesNotInherit is the fail-open
// regression: a caller-supplied environment with no allowlisted variable used
// to leave exec.Cmd.Env nil, so bwrap and the hook inherited the full parent
// environment, credentials included. It uses t.Setenv, so it is not parallel.
func TestBubblewrapBackend_E3_EmptyFilteredEnvDoesNotInherit(t *testing.T) {
	t.Setenv("AWS_SECRET_ACCESS_KEY", "LEAKED-SECRET")
	backend, shPath, mounts := e3Backend(t)

	cfg := &sandbox.SandboxConfig{
		HookCategory: sandbox.CategoryLinter,
		Mounts:       mounts,
		Environment:  map[string]string{"FOO": "bar"},
		HookCommand:  []string{shPath, "-c", `printf '%s' "${AWS_SECRET_ACCESS_KEY:-ABSENT}"`},
	}

	res, err := backend.RunHook(context.Background(), cfg)
	if err != nil {
		t.Fatalf("RunHook: %v (stderr=%q)", err, stderrOf(res))
	}
	if got := string(res.Stdout); got != "ABSENT" {
		t.Errorf("hook inherited the parent environment: stdout=%q", got)
	}
}

// cgroupLine returns the test process's own cgroup v2 membership line from
// /proc/self/cgroup.
func cgroupLine(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		t.Skipf("reading /proc/self/cgroup: %v", err)
	}
	return strings.TrimSpace(string(data))
}
