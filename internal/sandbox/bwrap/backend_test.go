//go:build !windows

package bwrap

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
)

// TestBubblewrapBackend_WarnsWhenLayersUnapplied verifies the tier-honesty
// signal: when a TierFull backend cannot actually apply Landlock or seccomp,
// RunHook must emit a warning per un-applied layer instead of silently
// overclaiming. This is a pure unit test of the warning logic.
func TestBubblewrapBackend_WarnsWhenLayersUnapplied(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	b := NewBubblewrapBackend(sandbox.TierFull, "/nonexistent/bwrap", true)

	// Route slog through our buffer for the duration of the call.
	prev := slog.Default()
	slog.SetDefault(logger)
	b.warnUnappliedLayers(false, false)
	slog.SetDefault(prev)

	out := buf.String()
	if !strings.Contains(out, "Landlock NOT applied") {
		t.Errorf("expected a Landlock-not-applied warning for TierFull, got: %q", out)
	}
	if !strings.Contains(out, "seccomp NOT applied") {
		t.Errorf("expected a seccomp-not-applied warning for TierFull, got: %q", out)
	}
}

// TestBubblewrapBackend_NoFalseWarnWhenApplied ensures no warning is emitted
// when the claimed layers are actually applied.
func TestBubblewrapBackend_NoFalseWarnWhenApplied(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	b := NewBubblewrapBackend(sandbox.TierFull, "/nonexistent/bwrap", true)

	prev := slog.Default()
	slog.SetDefault(logger)
	b.warnUnappliedLayers(true, true)
	slog.SetDefault(prev)

	if out := buf.String(); out != "" {
		t.Errorf("expected no warnings when both layers applied, got: %q", out)
	}
}

// --- E3 integration tests (real bwrap on the host) -------------------------
//
// These model the WU-23 adversarial findings: a hardened `sandbox exec` must
// strip credential env vars, isolate the PID namespace, and deny writes to
// $HOME. They are skipped when bwrap or unprivileged user namespaces are
// unavailable so CI stays green on hosts without them.

// e3Backend returns a real bubblewrap backend, a resolved POSIX shell path that
// is reachable inside the sandbox, and the extra mounts required for it. It
// skips the test if the environment cannot run a real bwrap sandbox.
func e3Backend(t *testing.T) (*BubblewrapBackend, string, []sandbox.MountSpec) {
	t.Helper()

	bwrapPath, err := exec.LookPath("bwrap")
	if err != nil {
		t.Skip("bwrap not available; skipping E3 integration test")
	}

	shPath, mounts := resolveSandboxShell(t)

	backend := NewBubblewrapBackend(sandbox.TierFull, bwrapPath, true)

	// Smoke test: if a trivial sandboxed command cannot run (e.g. user
	// namespaces disabled), skip rather than fail.
	cfg := &sandbox.SandboxConfig{
		HookCategory: sandbox.CategoryLinter,
		Mounts:       mounts,
		Environment:  map[string]string{"PATH": "/nonexistent", "HOME": t.TempDir()},
		HookCommand:  []string{shPath, "-c", "exit 0"},
	}
	res, err := backend.RunHook(context.Background(), cfg)
	if err != nil || res.ExitCode != 0 {
		t.Skipf("bwrap sandbox not usable here (err=%v, stderr=%q); skipping E3", err, stderrOf(res))
	}
	return backend, shPath, mounts
}

// resolveSandboxShell finds a POSIX shell and the read-only mounts needed to run
// it inside the minimal sandbox. On NixOS the shell and its libraries live under
// /nix/store, which BuildArgs already mounts, so no extra mounts are needed.
func resolveSandboxShell(t *testing.T) (string, []sandbox.MountSpec) {
	t.Helper()

	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no POSIX shell found; skipping E3 integration test")
	}
	if resolved, rerr := filepath.EvalSymlinks(sh); rerr == nil {
		sh = resolved
	}

	if strings.HasPrefix(sh, "/nix/store/") {
		return sh, nil // libraries are under the already-mounted /nix/store
	}

	// Non-Nix host: expose the standard library/binary directories read-only so
	// the shell and its dynamic dependencies resolve.
	var mounts []sandbox.MountSpec
	for _, dir := range []string{"/usr", "/lib", "/lib64", "/bin"} {
		if info, statErr := os.Stat(dir); statErr == nil && info.IsDir() {
			mounts = append(mounts, sandbox.MountSpec{Source: dir, Target: dir, ReadOnly: true})
		}
	}
	return sh, mounts
}

func stderrOf(res *sandbox.SandboxResult) string {
	if res == nil {
		return ""
	}
	return string(res.Stderr)
}

func TestBubblewrapBackend_E3_StripsCredentialEnv(t *testing.T) {
	t.Parallel()
	backend, shPath, mounts := e3Backend(t)

	cfg := &sandbox.SandboxConfig{
		HookCategory: sandbox.CategoryLinter,
		Mounts:       mounts,
		Environment: map[string]string{
			"PATH":                  "/nonexistent",
			"HOME":                  t.TempDir(),
			"AWS_SECRET_ACCESS_KEY": "canary-secret-value",
			"KEEP_ME":               "ok", // not on allowlist -> also dropped, harmless
		},
		HookCommand: []string{shPath, "-c", `printf '%s' "${AWS_SECRET_ACCESS_KEY:-ABSENT}"`},
	}

	res, err := backend.RunHook(context.Background(), cfg)
	if err != nil {
		t.Fatalf("RunHook error: %v (stderr=%q)", err, stderrOf(res))
	}
	got := string(res.Stdout)
	if strings.Contains(got, "canary-secret-value") {
		t.Errorf("credential leaked into sandbox: stdout=%q", got)
	}
	if got != "ABSENT" {
		t.Errorf("expected AWS_SECRET_ACCESS_KEY to be stripped (stdout=ABSENT), got %q", got)
	}
}

func TestBubblewrapBackend_E3_PIDNamespaceIsolation(t *testing.T) {
	t.Parallel()
	backend, shPath, mounts := e3Backend(t)

	// Count host PIDs for comparison.
	hostEntries, _ := filepath.Glob("/proc/[0-9]*")
	hostCount := len(hostEntries)
	if hostCount < 50 {
		t.Skipf("host shows only %d PIDs; PID-namespace comparison not meaningful", hostCount)
	}

	script := `n=0; for p in /proc/[0-9]*; do [ -d "$p" ] && n=$((n+1)); done; printf '%s' "$n"`
	cfg := &sandbox.SandboxConfig{
		HookCategory: sandbox.CategoryLinter,
		Mounts:       mounts,
		Environment:  map[string]string{"PATH": "/nonexistent", "HOME": t.TempDir()},
		HookCommand:  []string{shPath, "-c", script},
	}

	res, err := backend.RunHook(context.Background(), cfg)
	if err != nil {
		t.Fatalf("RunHook error: %v (stderr=%q)", err, stderrOf(res))
	}
	sandboxCount, convErr := strconv.Atoi(strings.TrimSpace(string(res.Stdout)))
	if convErr != nil {
		t.Fatalf("could not parse sandbox PID count %q: %v", res.Stdout, convErr)
	}
	if sandboxCount < 1 {
		t.Errorf("expected at least the sandbox init PID, got %d", sandboxCount)
	}
	// A PID namespace hides the host's processes: the sandbox should see only a
	// handful, far fewer than the host.
	if sandboxCount >= hostCount {
		t.Errorf("no PID isolation: sandbox saw %d PIDs, host has %d", sandboxCount, hostCount)
	}
	if sandboxCount > 30 {
		t.Errorf("expected only a few PIDs inside the PID namespace, saw %d", sandboxCount)
	}
}

func TestBubblewrapBackend_E3_HomeNotWritable(t *testing.T) {
	t.Parallel()
	backend, shPath, mounts := e3Backend(t)

	// Use the real host home so a successful write would be observable on the
	// host. The sandbox does not mount it, so the write must fail.
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		t.Skip("cannot determine home directory; skipping")
	}
	probe := filepath.Join(home, ".qsdev-sandbox-e3-probe")
	_ = os.Remove(probe)
	t.Cleanup(func() { _ = os.Remove(probe) })

	script := `if echo escaped > "$HOME/.qsdev-sandbox-e3-probe" 2>/dev/null; then printf WROTE; else printf BLOCKED; fi`
	cfg := &sandbox.SandboxConfig{
		HookCategory: sandbox.CategoryLinter, // read-only worktree category
		Mounts:       mounts,
		Environment:  map[string]string{"PATH": "/nonexistent", "HOME": home},
		HookCommand:  []string{shPath, "-c", script},
	}

	res, err := backend.RunHook(context.Background(), cfg)
	if err != nil {
		t.Fatalf("RunHook error: %v (stderr=%q)", err, stderrOf(res))
	}
	if got := string(res.Stdout); got != "BLOCKED" {
		t.Errorf("expected $HOME write to be BLOCKED, got %q", got)
	}
	if _, statErr := os.Stat(probe); statErr == nil {
		t.Errorf("filesystem escape: probe file persisted on host at %s", probe)
	}
}
