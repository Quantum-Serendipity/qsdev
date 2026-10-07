//go:build !windows

package bwrap

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
	"github.com/Quantum-Serendipity/qsdev/internal/testutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// TestBubblewrapBackend_WarnsWhenLayersUnapplied verifies the tier-honesty
// signal: when a TierFull backend cannot actually apply Landlock or seccomp,
// RunHook must emit a warning per un-applied layer instead of silently
// overclaiming. This is a pure unit test of the warning logic.
func TestBubblewrapBackend_WarnsWhenLayersUnapplied(t *testing.T) {
	// Not parallel: it swaps the global slog default.

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
	// Not parallel: it swaps the global slog default.

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
// $HOME. They skip when bwrap or unprivileged user namespaces are
// unavailable, and fail instead under testutil.RequireE3, which the CI job
// that provisions bubblewrap sets.

// e3Backend returns a real bubblewrap backend, a resolved POSIX shell path that
// is reachable inside the sandbox, and the extra mounts required for it.
//
// Host capability is probed with a fixed, hand-written bwrap invocation that is
// independent of BuildArgs; only a failure of THAT probe skips. Once the host
// is known to run bwrap, a failing smoke run through the production BuildArgs
// path is a regression (e.g. an invalid flag) and fails the test.
func e3Backend(t *testing.T) (*BubblewrapBackend, string, []sandbox.MountSpec) {
	t.Helper()

	bwrapPath := testutil.RequireTool(t, "bwrap", testutil.RequireE3)

	shPath, mounts := resolveSandboxShell(t)

	probe := exec.CommandContext(context.Background(), bwrapPath, //nolint:gosec // fixed test probe
		"--unshare-user", "--ro-bind", "/", "/", "--", shPath, "-c", "exit 0")
	if out, probeErr := probe.CombinedOutput(); probeErr != nil {
		testutil.Unavailable(t, testutil.RequireE3, "host cannot run bwrap with a user namespace (err=%v, output=%q)", probeErr, out)
	}

	backend := NewBubblewrapBackend(sandbox.TierFull, bwrapPath, true)

	cfg := &sandbox.SandboxConfig{
		HookCategory: sandbox.CategoryLinter,
		Mounts:       mounts,
		Environment:  map[string]string{"PATH": "/nonexistent", "HOME": t.TempDir()},
		HookCommand:  []string{shPath, "-c", "exit 0"},
	}
	res, err := backend.RunHook(context.Background(), cfg)
	if err != nil || res.ExitCode != 0 {
		t.Fatalf("bwrap works on this host but the production sandbox does not (err=%v, exit=%v, stderr=%q)",
			err, exitCodeOf(res), stderrOf(res))
	}
	return backend, shPath, mounts
}

func exitCodeOf(res *sandbox.SandboxResult) any {
	if res == nil {
		return nil
	}
	return res.ExitCode
}

// resolveSandboxShell finds a POSIX shell and the read-only mounts needed to run
// it inside the minimal sandbox. On NixOS the shell and its libraries live under
// /nix/store, which BuildArgs already mounts, so no extra mounts are needed.
func resolveSandboxShell(t *testing.T) (string, []sandbox.MountSpec) {
	t.Helper()

	sh, err := exec.LookPath("sh")
	if err != nil {
		testutil.Unavailable(t, testutil.RequireE3, "no POSIX shell found")
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
	for _, dir := range hostSystemDirs {
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

// TestBubblewrapBackend_E3_PolicyDenyPathHidden is the live regression for the
// inverted deny: a filesystem.deny entry that is not on the built-in list was
// bind-mounted read-only, so the hook could read exactly what policy denied.
func TestBubblewrapBackend_E3_PolicyDenyPathHidden(t *testing.T) {
	t.Parallel()
	backend, shPath, mounts := e3Backend(t)

	root, secretDir, secretFile := policyDenyFixture(t)
	// Print whatever each file yields (the fixtures have no trailing newline,
	// so read's EOF status is ignored); an unopenable file prints nothing.
	script := `for f in "$1/secret.txt" "$2" "$3"; do l=; { read -r l || :; } <"$f" && printf '%s;' "$l"; done 2>/dev/null`
	cfg := &sandbox.SandboxConfig{
		HookCategory: sandbox.CategoryLinter,
		Mounts:       append(mounts, sandbox.MountSpec{Source: root, Target: root, ReadOnly: true}),
		Deny:         []string{secretDir, secretFile},
		Environment:  map[string]string{"PATH": "/nonexistent", "HOME": t.TempDir()},
		HookCommand:  []string{shPath, "-c", script, "sh", secretDir, secretFile, filepath.Join(root, "public.txt")},
	}

	res, err := backend.RunHook(context.Background(), cfg)
	if err != nil {
		t.Fatalf("RunHook error: %v (stderr=%q)", err, stderrOf(res))
	}
	got := string(res.Stdout)
	if strings.Contains(got, "TOPSECRET") || strings.Contains(got, "TOKEN") {
		t.Errorf("policy-denied paths readable inside the sandbox: stdout=%q", got)
	}
	if !strings.Contains(got, "PUBLIC") {
		t.Errorf("non-denied file under the same mount must stay readable: stdout=%q stderr=%q", got, stderrOf(res))
	}
}

// TestBubblewrapBackend_E3_LandlockAllowsDevAndProc is the regression for the
// Landlock allowlist omitting /dev and /proc: ll-restrict denies every path it
// is not given, so `cmd >/dev/null` and reads of /proc/self failed with EACCES
// at exactly the strongest tier. It runs only where ll-restrict is available.
func TestBubblewrapBackend_E3_LandlockAllowsDevAndProc(t *testing.T) {
	t.Parallel()
	if sandbox.LLRestrictBin() == "" {
		testutil.Unavailable(t, testutil.RequireE3, "ll-restrict not available (build with the ll-restrict ldflags)")
	}
	backend, shPath, mounts := e3Backend(t)

	script := `echo x >/dev/null && : </dev/urandom && read -r line </proc/self/status && printf OK`
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
	if got := string(res.Stdout); got != "OK" || res.ExitCode != 0 {
		t.Errorf("hook touching /dev and /proc under Landlock: exit=%d stdout=%q stderr=%q",
			res.ExitCode, got, stderrOf(res))
	}
}

// TestBubblewrapBackend_E3_NestedUserNamespaceBlocked is the regression for
// the namespace-escape gap: seccomp blocks unshare/setns only when a filter is
// loaded, and a hook could otherwise create nested user namespaces. With
// --disable-userns (bwrap >= 0.8) the kernel refuses them by any route.
func TestBubblewrapBackend_E3_NestedUserNamespaceBlocked(t *testing.T) {
	t.Parallel()
	backend, _, mounts := e3Backend(t)

	if !supportsDisableUserNS(context.Background(), backend.bwrapBin) {
		testutil.Unavailable(t, testutil.RequireE3, "bwrap at %s is older than 0.8 and cannot disable nested user namespaces", backend.bwrapBin)
	}
	unshareBin, err := exec.LookPath("unshare")
	if err != nil {
		testutil.Unavailable(t, testutil.RequireE3, "unshare(1) not available")
	}
	if resolved, rerr := filepath.EvalSymlinks(unshareBin); rerr == nil {
		unshareBin = resolved
	}

	cfg := &sandbox.SandboxConfig{
		HookCategory: sandbox.CategoryLinter,
		Mounts:       hostToolMounts(mounts, unshareBin),
		Environment:  map[string]string{"PATH": "/nonexistent", "HOME": t.TempDir()},
		HookCommand:  []string{unshareBin, "--user", "--", unshareBin, "--version"},
	}

	res, err := backend.RunHook(context.Background(), cfg)
	if err != nil {
		t.Fatalf("RunHook error: %v (stderr=%q)", err, stderrOf(res))
	}
	if res.ExitCode == 0 {
		t.Errorf("nested user namespace was created inside the sandbox (stdout=%q)", res.Stdout)
	}
}

// TestRunHook_ReadyFDNumbering: the ready fd follows the optional seccomp
// file in ExtraFiles, so the shim is told fd 4 when a filter is passed and fd
// 3 when none is; the hook runs through the shim mounted at /.qsdev/bin/qsdev.
func TestRunHook_ReadyFDNumbering(t *testing.T) {
	t.Parallel()
	seccomp, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = seccomp.Close() })

	tests := []struct {
		name   string
		before []*os.File
		wantFD string
	}{
		{name: "with seccomp file", before: []*os.File{seccomp}, wantFD: "4"},
		{name: "without seccomp file", before: nil, wantFD: "3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			guard, err := sandbox.NewLaunchGuard()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = guard.Close() })
			cfg := &sandbox.SandboxConfig{HookCategory: sandbox.CategoryLinter, HookCommand: []string{"hook", "arg"}}

			files, argv, landlock := launchHook(cfg, sandbox.TierBwrapOnly, tt.before, guard)
			if landlock {
				t.Error("Landlock applied at a tier that does not claim it")
			}

			if len(files) != len(tt.before)+1 || files[len(files)-1] != guard.ChildFile() {
				t.Errorf("ExtraFiles = %v, want %v then the guard's child file", files, tt.before)
			}
			want := []string{"/.qsdev/bin/qsdev", "sandbox", "shim", "--ready-fd", tt.wantFD, "--", "hook", "arg"}
			if !slices.Equal(argv, want) {
				t.Errorf("hook argv = %v, want %v", argv, want)
			}
		})
	}
}

// TestRunHook_SetupFailureIsErrSetupFailed is the live regression for
// U19-01/XS-WS1 A4: when bwrap itself cannot set up the sandbox (a strict
// bind whose source does not exist), its exit status must never pass as the
// hook's: RunHook returns ErrSetupFailed, which blocks.
func TestRunHook_SetupFailureIsErrSetupFailed(t *testing.T) {
	t.Parallel()
	backend, shPath, mounts := e3Backend(t)

	tests := []struct {
		name   string
		mutate func(cfg *sandbox.SandboxConfig)
	}{
		{name: "missing project dir", mutate: func(cfg *sandbox.SandboxConfig) { cfg.ProjectDir = "/nonexistent-u19" }},
		{name: "policy mount with a missing source", mutate: func(cfg *sandbox.SandboxConfig) {
			cfg.Mounts = append(cfg.Mounts, sandbox.MountSpec{Source: "/nonexistent-u19-src", Target: "/opt/u19", ReadOnly: true})
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := &sandbox.SandboxConfig{
				HookCategory: sandbox.CategoryLinter,
				Mounts:       slices.Clone(mounts),
				Environment:  map[string]string{"PATH": "/nonexistent", "HOME": t.TempDir()},
				HookCommand:  []string{shPath, "-c", "printf HOOKRAN"},
			}
			tt.mutate(cfg)

			res, err := backend.RunHook(context.Background(), cfg)
			if !errors.Is(err, sandbox.ErrSetupFailed) {
				t.Fatalf("RunHook = (exit %v, err %v), want ErrSetupFailed", exitCodeOf(res), err)
			}
			if !strings.Contains(err.Error(), "nonexistent-u19") {
				t.Errorf("error %q does not carry bwrap's diagnostic", err)
			}
		})
	}
}

// TestRunHook_HookExitCodePreserved: once the sandbox reached the hook, the
// hook's own status is the result, including 127, the shim's own not-started
// code, and the hook does not inherit the ready fd.
func TestRunHook_HookExitCodePreserved(t *testing.T) {
	t.Parallel()
	backend, shPath, mounts := e3Backend(t)

	for _, code := range []int{0, 1, 7, 127} {
		t.Run(strconv.Itoa(code), func(t *testing.T) {
			t.Parallel()
			cfg := &sandbox.SandboxConfig{
				HookCategory: sandbox.CategoryLinter,
				Mounts:       mounts,
				Environment:  map[string]string{"PATH": "/nonexistent", "HOME": t.TempDir()},
				HookCommand:  []string{shPath, "-c", "[ -e /proc/self/fd/3 ] && printf LEAKED; exit " + strconv.Itoa(code)},
			}
			res, err := backend.RunHook(context.Background(), cfg)
			if err != nil {
				t.Fatalf("RunHook error: %v", err)
			}
			if res.ExitCode != code {
				t.Errorf("exit code = %d, want %d (stderr=%q)", res.ExitCode, code, res.Stderr)
			}
			if strings.Contains(string(res.Stdout), "LEAKED") {
				t.Error("the hook inherited the ready fd")
			}
		})
	}
}

// guardrailRun runs script under sh in a real sandbox of category cat with
// project as the project directory; "$P" in script is replaced with it (the
// hook environment allowlist would strip a variable). tools are the host
// tools (see hostTool) script runs.
func guardrailRun(t *testing.T, cat sandbox.HookCategory, project, script string, tools ...string) (*sandbox.SandboxResult, error) {
	t.Helper()
	backend, shPath, mounts := e3Backend(t)
	cfg := &sandbox.SandboxConfig{
		ProjectDir:   project,
		HookCategory: cat,
		Mounts:       hostToolMounts(mounts, tools...),
		Environment:  map[string]string{"PATH": "/nonexistent", "HOME": t.TempDir()},
		HookCommand:  []string{shPath, "-c", strings.ReplaceAll(script, "$P", project)},
	}
	return backend.RunHook(context.Background(), cfg)
}

// TestRunHook_GuardrailWriteBlocked is the live U05-05 regression: a
// read-write category can no longer plant a git hook.
func TestRunHook_GuardrailWriteBlocked(t *testing.T) {
	t.Parallel()
	project := t.TempDir()
	hooks := filepath.Join(project, ".git", "hooks")
	if err := os.MkdirAll(hooks, 0o755); err != nil {
		t.Fatal(err)
	}

	res, err := guardrailRun(t, sandbox.CategoryGenerator, project, `echo x > "$P/.git/hooks/post-checkout"`)
	if err != nil {
		t.Fatalf("RunHook: %v", err)
	}
	if res.ExitCode == 0 {
		t.Errorf("write to .git/hooks succeeded inside the sandbox")
	}
	if !strings.Contains(stderrOf(res), "Read-only file system") {
		t.Errorf("stderr = %q, want Read-only file system", stderrOf(res))
	}
	if _, err := os.Lstat(filepath.Join(hooks, "post-checkout")); err == nil {
		t.Error("post-checkout was created on the host")
	}
}

// TestRunHook_GuardrailCreationFailsClosed: a guardrail absent at launch
// cannot be overlaid, so creating it is detected after the hook and fails
// closed with ErrGuardrailModified naming the path.
func TestRunHook_GuardrailCreationFailsClosed(t *testing.T) {
	t.Parallel()
	project := t.TempDir()

	res, err := guardrailRun(t, sandbox.CategoryTestRunner, project, `echo x > "$P/.envrc"`)
	if !errors.Is(err, sandbox.ErrGuardrailModified) {
		t.Fatalf("RunHook = (exit %v, err %v), want ErrGuardrailModified", exitCodeOf(res), err)
	}
	if !strings.Contains(err.Error(), ".envrc") {
		t.Errorf("error %q does not name .envrc", err)
	}
	// The created file is moved aside, so direnv does not run it next time.
	if _, err := os.Lstat(filepath.Join(project, ".envrc")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf(".envrc still in place on the host after the run (err %v)", err)
	}
	if q, _ := filepath.Glob(sandbox.QuarantineName(filepath.Join(project, ".envrc"), "*")); len(q) != 1 {
		t.Errorf("quarantined .envrc = %v, want one copy", q)
	}
}

// TestRunHook_GuardrailDirectoryCannotBeSwapped: the directory holding a
// read-only guardrail is pinned, so a hook cannot rename .git away and plant
// a fresh .git/hooks beside it; the rest of .git stays writable.
func TestRunHook_GuardrailDirectoryCannotBeSwapped(t *testing.T) {
	t.Parallel()
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, ".git", "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}

	res, err := guardrailRun(t, sandbox.CategoryGenerator, project,
		hostCommand(t, "mv")+` "$P/.git" "$P/.git2" && exit 3; echo x > "$P/.git/objects-ok" || exit 4; exit 0`,
		hostTool(t, "mv"))
	if err != nil {
		t.Fatalf("RunHook: %v", err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("exit %d, stderr %q: want the rename refused and .git itself still writable", res.ExitCode, stderrOf(res))
	}
	if !strings.Contains(stderrOf(res), "busy") {
		t.Errorf("stderr = %q, want the rename refused with EBUSY", stderrOf(res))
	}
	if _, err := os.Lstat(filepath.Join(project, ".git2")); err == nil {
		t.Error(".git was renamed on the host")
	}
	if _, err := os.Stat(filepath.Join(project, ".git", "objects-ok")); err != nil {
		t.Errorf("write inside .git did not reach the host: %v", err)
	}
}

// TestRunHook_GuardrailPinNeverWidens: with a read-only worktree exposed by a
// writable mount elsewhere, the pins still bind .git, and must keep it as
// read-only as the worktree.
func TestRunHook_GuardrailPinNeverWidens(t *testing.T) {
	t.Parallel()
	backend, shPath, mounts := e3Backend(t)
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, ".git", "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	cache := t.TempDir()
	cfg := &sandbox.SandboxConfig{
		ProjectDir:   project,
		HookCategory: sandbox.CategoryLinter,
		Mounts:       append(slices.Clone(mounts), sandbox.MountSpec{Source: cache, Target: cache}),
		Environment:  map[string]string{"PATH": "/nonexistent", "HOME": t.TempDir()},
		HookCommand:  []string{shPath, "-c", `echo x > "` + project + `/.git/obj"`},
	}
	if args, err := BuildArgs(cfg, sandbox.TierFull); err != nil || !slices.Contains(args, filepath.Join(project, ".git")) {
		t.Fatalf("expected a pin of .git in %v (err %v)", args, err)
	}
	res, err := backend.RunHook(context.Background(), cfg)
	if err != nil {
		t.Fatalf("RunHook: %v", err)
	}
	if res.ExitCode == 0 || !strings.Contains(stderrOf(res), "Read-only file system") {
		t.Errorf("exit %d, stderr %q: the pin made a read-only worktree writable", res.ExitCode, stderrOf(res))
	}
}

// TestRunHook_SymlinkedGuardrailRestored is the regression for a symlinked
// guardrail (the default .pre-commit-config.yaml links into the Nix store),
// which no overlay can pin: replacing it fails closed, and the host is left
// with the original symlink and the hook's file moved aside, not in use.
func TestRunHook_SymlinkedGuardrailRestored(t *testing.T) {
	t.Parallel()
	rm := hostCommand(t, "rm")
	project, store := t.TempDir(), t.TempDir()
	dest := filepath.Join(store, "pc.json")
	if err := os.WriteFile(dest, []byte("repos: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(project, ".pre-commit-config.yaml")
	if err := os.Symlink(dest, cfgPath); err != nil {
		t.Fatal(err)
	}
	const evil = "repos: [{repo: local, hooks: [{id: x, entry: evil, language: system}]}]"

	res, err := guardrailRun(t, sandbox.CategoryTestRunner, project,
		rm+` -f "$P/.pre-commit-config.yaml" && echo "`+evil+`" > "$P/.pre-commit-config.yaml"`, hostTool(t, "rm"))
	if !errors.Is(err, sandbox.ErrGuardrailModified) {
		t.Fatalf("RunHook = (exit %v, err %v, stderr %q), want ErrGuardrailModified", exitCodeOf(res), err, stderrOf(res))
	}
	if got, err := os.Readlink(cfgPath); err != nil || got != dest {
		t.Errorf("%s -> %q (err %v), want the symlink to %s restored", cfgPath, got, err, dest)
	}
	q, _ := filepath.Glob(sandbox.QuarantineName(cfgPath, "*"))
	if len(q) != 1 {
		t.Fatalf("quarantined config = %v, want the hook's file kept aside", q)
	}
	if data, err := os.ReadFile(q[0]); err != nil || !strings.Contains(string(data), "evil") {
		t.Errorf("quarantined config = %q (err %v)", data, err)
	}
	if !strings.Contains(err.Error(), q[0]) {
		t.Errorf("error %q does not name the quarantined file %s", err, q[0])
	}
}

// TestRunHook_DevenvStateWritable is the live regression for devenv's state
// directory: with GOPATH under .devenv/state, a test runner or generator can
// still fill the module cache and sync a venv, even when .devenv did not exist
// yet in a devenv project, while the rest of .devenv (the shell scripts devenv
// runs on entry) stays read-only.
func TestRunHook_DevenvStateWritable(t *testing.T) {
	t.Parallel()
	for _, cat := range []sandbox.HookCategory{sandbox.CategoryGenerator, sandbox.CategoryTestRunner} {
		t.Run(cat.String(), func(t *testing.T) {
			t.Parallel()
			mkdir := hostCommand(t, "mkdir")
			project := t.TempDir()
			if err := os.WriteFile(filepath.Join(project, "devenv.nix"), []byte("{ }\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			state := sandbox.DevenvStateDir(project)
			rel, err := filepath.Rel(project, state)
			if err != nil {
				t.Fatal(err)
			}
			script := mkdir + ` -p "$P/` + rel + `/go/pkg/mod/cache/download" && echo x > "$P/` + rel + `/venv-marker"`
			res, err := guardrailRun(t, cat, project, script, hostTool(t, "mkdir"))
			if err != nil {
				t.Fatalf("RunHook: %v", err)
			}
			if res.ExitCode != 0 {
				t.Fatalf("exit %d, stderr %q", res.ExitCode, stderrOf(res))
			}
			for _, p := range []string{filepath.Join(state, "go", "pkg", "mod", "cache", "download"), filepath.Join(state, "venv-marker")} {
				if _, err := os.Stat(p); err != nil {
					t.Errorf("devenv state write did not reach the host: %v", err)
				}
			}

			entry := filepath.Join(filepath.Dir(state), "entry.sh")
			res, err = guardrailRun(t, cat, project, `echo x > "`+entry+`"`)
			if err != nil {
				t.Fatalf("RunHook: %v", err)
			}
			if res.ExitCode == 0 || !strings.Contains(stderrOf(res), "Read-only file system") {
				t.Errorf("exit %d, stderr %q: .devenv outside its state dir must stay read-only", res.ExitCode, stderrOf(res))
			}
		})
	}
}

// TestRunHook_ProjectStillWritable: the overlays protect only the control
// plane; the rest of the project stays writable for read-write categories.
func TestRunHook_ProjectStillWritable(t *testing.T) {
	t.Parallel()
	for _, cat := range []sandbox.HookCategory{sandbox.CategoryGenerator, sandbox.CategoryTestRunner} {
		t.Run(cat.String(), func(t *testing.T) {
			t.Parallel()
			project := t.TempDir()
			if err := os.MkdirAll(filepath.Join(project, ".git", "hooks"), 0o755); err != nil {
				t.Fatal(err)
			}
			res, err := guardrailRun(t, cat, project, `echo x > "$P/generated.txt"`)
			if err != nil {
				t.Fatalf("RunHook: %v", err)
			}
			if res.ExitCode != 0 {
				t.Fatalf("exit %d, stderr %q", res.ExitCode, stderrOf(res))
			}
			if _, err := os.Stat(filepath.Join(project, "generated.txt")); err != nil {
				t.Errorf("project write did not reach the host: %v", err)
			}
		})
	}
}

// hostTool returns the symlink-resolved path of a host binary for a live
// sandbox test, skipping the test when it is not installed.
func hostTool(t *testing.T, name string) string {
	t.Helper()
	p, err := exec.LookPath(name)
	if err != nil {
		t.Skipf("%s not available: %v", name, err)
	}
	if r, err := filepath.EvalSymlinks(p); err == nil {
		p = r
	}
	return p
}

// hostCommand returns a shell command word running host tool name inside the
// sandbox (see hostTool). A multi-call binary (Nix's coreutils), which the
// resolved path names instead of the tool, is told which tool to be.
func hostCommand(t *testing.T, name string) string {
	t.Helper()
	p := hostTool(t, name)
	if filepath.Base(p) == "coreutils" {
		return p + " --coreutils-prog=" + name
	}
	return p
}

// hostSystemDirs are the standard directories a host binary outside the Nix
// store and its dynamic dependencies resolve from.
var hostSystemDirs = []string{"/usr", "/lib", "/lib64", "/bin"}

// hostToolMounts returns mounts plus the read-only mounts the sandbox needs
// to run the host tools at paths (see hostTool). A tool under /nix/store
// needs none, as BuildArgs mounts the store. Any other tool needs its own
// directory and hostSystemDirs: it is resolved on the host PATH, so it can
// live outside the store even when the sandbox shell does not (a Nix shell
// on a distribution host resolves git to /usr/bin/git), and without them
// it does not exist in the sandbox.
func hostToolMounts(mounts []sandbox.MountSpec, paths ...string) []sandbox.MountSpec {
	out := slices.Clone(mounts)
	add := func(dir string) {
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			return
		}
		for _, m := range out {
			if dir == m.Target || strings.HasPrefix(dir, m.Target+"/") {
				return
			}
		}
		out = append(out, sandbox.MountSpec{Source: dir, Target: dir, ReadOnly: true})
	}
	for _, p := range paths {
		if strings.HasPrefix(p, "/nix/store/") {
			continue
		}
		for _, dir := range hostSystemDirs {
			add(dir)
		}
		add(filepath.Dir(p))
	}
	return out
}

// TestRunHook_GitControlReadOnlyGitAddWorks: only git's code-executing
// control files are read-only, so a formatter can still stage its output
// while planting a hook or rewriting the config fails.
func TestRunHook_GitControlReadOnlyGitAddWorks(t *testing.T) {
	t.Parallel()
	git := hostTool(t, "git")
	project := t.TempDir()
	initCmd := exec.CommandContext(context.Background(), git, "init", "-q", project) //nolint:gosec // fixed test command
	if out, err := initCmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}

	tests := []struct {
		name, script string
		wantOK       bool
	}{
		{"git add", `echo x > "$P/gen.txt" && ` + git + ` -C "$P" add gen.txt`, true},
		{"plant a git hook", `echo x > "$P/.git/hooks/post-checkout"`, false},
		{"rewrite git config", `echo '[core]' >> "$P/.git/config"`, false},
		{"rewrite git attributes", `echo '* filter=x' >> "$P/.git/info/attributes"`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := guardrailRun(t, sandbox.CategoryFormatter, project, tt.script, git)
			if err != nil {
				t.Fatalf("RunHook: %v", err)
			}
			if ok := res.ExitCode == 0; ok != tt.wantOK {
				t.Fatalf("exit %d (stderr %q), want success=%v", res.ExitCode, stderrOf(res), tt.wantOK)
			}
			if !tt.wantOK && !strings.Contains(stderrOf(res), "Read-only file system") {
				t.Errorf("stderr = %q, want Read-only file system", stderrOf(res))
			}
		})
	}
	if _, err := os.Lstat(filepath.Join(project, ".git", "hooks", "post-checkout")); err == nil {
		t.Error("post-checkout was created on the host")
	}
}

// TestRunHook_ControlFilesProtected covers the control files beyond git and
// .claude: an existing one is read-only, and creating an absent one fails
// closed with ErrGuardrailModified naming it.
func TestRunHook_ControlFilesProtected(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"devenv.local.nix", "devenv.local.yaml", ".npmrc", branding.Get().ConfigFile} {
		t.Run("create "+name, func(t *testing.T) {
			t.Parallel()
			project := t.TempDir()
			res, err := guardrailRun(t, sandbox.CategoryTestRunner, project, `echo x > "$P/`+name+`"`)
			if !errors.Is(err, sandbox.ErrGuardrailModified) {
				t.Fatalf("RunHook = (exit %v, err %v), want ErrGuardrailModified", exitCodeOf(res), err)
			}
			if !strings.Contains(err.Error(), name) {
				t.Errorf("error %q does not name %s", err, name)
			}
		})
		t.Run("rewrite "+name, func(t *testing.T) {
			t.Parallel()
			project := t.TempDir()
			if err := os.WriteFile(filepath.Join(project, name), nil, 0o644); err != nil {
				t.Fatal(err)
			}
			res, err := guardrailRun(t, sandbox.CategoryTestRunner, project, `echo x >> "$P/`+name+`"`)
			if err != nil {
				t.Fatalf("RunHook: %v", err)
			}
			if res.ExitCode == 0 || !strings.Contains(stderrOf(res), "Read-only file system") {
				t.Errorf("exit %d, stderr %q: want a Read-only file system failure", res.ExitCode, stderrOf(res))
			}
		})
	}
}

// TestRunHook_AuditLogTemplateWrites runs the shipped audit-log hook in the
// generator category it is registered with: its log under the read-only
// .claude must still reach the host.
func TestRunHook_AuditLogTemplateWrites(t *testing.T) {
	t.Parallel()
	backend, shPath, mounts := e3Backend(t)
	var path []string
	for _, tool := range []string{"bash", "python3", "date", "mkdir"} {
		p := hostTool(t, tool)
		path = append(path, filepath.Dir(p))
		mounts = hostToolMounts(mounts, p)
	}

	project := t.TempDir()
	template, err := os.ReadFile(filepath.Join("..", "..", "..", "addons", "claudecode", "templates", "hooks", "audit-log.sh"))
	if err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(project, ".claude", "hooks", "audit-log.sh")
	if err := os.MkdirAll(filepath.Dir(script), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(script, template, 0o755); err != nil { //nolint:gosec // a hook script must be executable
		t.Fatal(err)
	}

	cfg := &sandbox.SandboxConfig{
		ProjectDir:   project,
		HookCategory: sandbox.CategoryGenerator,
		Mounts:       mounts,
		Environment: map[string]string{
			"PATH": strings.Join(path, ":"), "HOME": t.TempDir(), "CLAUDE_PROJECT_DIR": project,
		},
		HookCommand: []string{shPath, "-c", `printf '%s' '{"tool_name":"Bash","tool_input":{"command":"ls"}}' | bash "` + script + `"`},
	}
	res, err := backend.RunHook(context.Background(), cfg)
	if err != nil || res.ExitCode != 0 {
		t.Fatalf("RunHook = (exit %v, err %v), stderr %q", exitCodeOf(res), err, stderrOf(res))
	}
	logs, err := filepath.Glob(filepath.Join(sandbox.HookLogDir(project), "audit-*.jsonl"))
	if err != nil || len(logs) != 1 {
		t.Fatalf("audit log files = %v (err %v), want one", logs, err)
	}
	data, err := os.ReadFile(logs[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"Bash"`) {
		t.Errorf("audit log %q does not record the tool call", data)
	}
}
