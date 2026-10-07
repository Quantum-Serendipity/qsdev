package bwrap

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
	"github.com/Quantum-Serendipity/qsdev/internal/sandbox/shim"
)

// BubblewrapBackend implements SandboxBackend using bubblewrap for namespace
// isolation. It supports four tiers depending on the available LSM layers:
// Full (bwrap + Landlock + seccomp), BwrapWithoutLandlock, BwrapWithoutSeccomp
// and BwrapOnly (namespaces alone).
// With WithSystemdRun it also applies the configured cgroup resource limits.
type BubblewrapBackend struct {
	tier           sandbox.DegradationTier
	bwrapBin       string
	hasUserNS      bool
	systemdRunPath string
}

// Option configures optional BubblewrapBackend behaviour.
type Option func(*BubblewrapBackend)

// WithSystemdRun enables cgroup resource limits: each sandbox runs inside a
// transient `systemd-run --user --scope` carrying SandboxConfig.Resources,
// whenever a systemd user session is reachable. Without it (or without a user
// session) the limits cannot be applied and RunHook warns instead.
func WithSystemdRun(path string) Option {
	return func(b *BubblewrapBackend) { b.systemdRunPath = path }
}

// NewBubblewrapBackend creates a BubblewrapBackend with the given tier, bwrap
// binary path, and unprivileged-user-namespace capability. hasUserNS records
// whether the host permits unprivileged user namespaces; every bwrap invocation
// emits --unshare-user, so a backend built without them fails every exec and
// must report itself unavailable (see Available).
func NewBubblewrapBackend(tier sandbox.DegradationTier, bwrapBin string, hasUserNS bool, opts ...Option) *BubblewrapBackend {
	b := &BubblewrapBackend{tier: tier, bwrapBin: bwrapBin, hasUserNS: hasUserNS}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

func (b *BubblewrapBackend) Name() string                  { return "bubblewrap" }
func (b *BubblewrapBackend) Tier() sandbox.DegradationTier { return b.tier }

// Available checks whether bwrap is accessible AND can actually run. Statting
// the binary is not sufficient: bwrap always requests an unprivileged user
// namespace (--unshare-user), so on a host where those are disabled every exec
// fails with "setting up uid map: Permission denied". Reporting such a backend
// Available would let Select pick it over a working fallback (e.g. systemd-run)
// at the same demoted tier, so the missing namespace support is a hard failure.
func (b *BubblewrapBackend) Available() error {
	if b.bwrapBin == "" {
		return fmt.Errorf("bubblewrap binary path not set")
	}
	if _, err := os.Stat(b.bwrapBin); err != nil {
		return fmt.Errorf("bubblewrap binary not found at %s: %w", b.bwrapBin, err)
	}
	if !b.hasUserNS {
		return fmt.Errorf("bubblewrap requires unprivileged user namespaces, which are unavailable on this host")
	}
	return nil
}

// RunHook creates a bubblewrap sandbox, executes the hook, and returns the
// result. The hook runs through the in-sandbox shim (internal/sandbox/shim),
// the host's qsdev binary mounted read-only at shim.SandboxPath(), which
// signals a sandbox.LaunchGuard just before it execs the hook. A run that
// never reached the hook (bwrap or ll-restrict failed, whatever exit code that
// produced) is returned as an error wrapping sandbox.ErrSetupFailed, never as
// the hook's exit code. A run after which a project guardrail path was
// created or replaced returns an error wrapping sandbox.ErrGuardrailModified
// instead of the hook's result, once the change is undone as far as possible
// (sandbox.GuardrailSnapshot.Enforce).
func (b *BubblewrapBackend) RunHook(ctx context.Context, cfg *sandbox.SandboxConfig) (*sandbox.SandboxResult, error) {
	if len(cfg.HookCommand) == 0 {
		return &sandbox.SandboxResult{ExitCode: 0, Tier: b.tier}, nil
	}

	setupStart := time.Now()

	// Taken before BuildArgs decides which guardrails exist to overlay, so a
	// guardrail created in between is reported rather than left writable.
	prepareWritableDirs(cfg)
	guardrails, err := snapshotGuardrails(cfg)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", sandbox.ErrSetupFailed, err)
	}

	args, err := BuildArgs(cfg, b.tier)
	if err != nil {
		return nil, fmt.Errorf("building sandbox args: %w", err)
	}

	hostExe, err := shim.HostExecutable()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", sandbox.ErrSetupFailed, err)
	}
	// The shim mount is a trusted internal directive, like the deny masks, so
	// it bypasses ValidateMountPath; BuildArgs keeps every configured bind
	// off shim.SandboxRoot().
	args = append(args, "--ro-bind", hostExe, shim.SandboxPath())

	// Forbid nested user namespaces at the kernel level when this bwrap can
	// (>= 0.8): seccomp cannot filter clone3's flags, so this is the only
	// complete block on user-namespace-gated kernel attack surface.
	if supportsDisableUserNS(ctx, b.bwrapBin) {
		args = append(args, "--disable-userns")
	}

	// Honesty: "filtered" network has no egress filter yet, so say so rather
	// than let the policy imply an allowlist that is not applied.
	if unenforced := cfg.UnenforcedNetworkControls(); len(unenforced) > 0 {
		slog.Warn("sandbox network controls NOT enforced",
			"category", cfg.HookCategory.String(), "detail", strings.Join(unenforced, "; "))
	}

	// Seccomp layer: pass the compiled BPF filter to bwrap through an inherited
	// file descriptor when one is available (the Nix build injects the path via
	// ldflags). This is a no-op in builds without a filter, so exec still runs.
	seccompFiles, seccompArgs := openSeccompFilter()
	defer func() {
		for _, f := range seccompFiles {
			_ = f.Close()
		}
	}()
	args = append(args, seccompArgs...)

	guard, err := sandbox.NewLaunchGuard()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", sandbox.ErrSetupFailed, err)
	}
	defer func() { _ = guard.Close() }()
	extraFiles, hookCmd, landlockApplied := launchHook(cfg, b.tier, seccompFiles, guard)

	// Honesty: if the selected tier advertises an LSM layer we could not apply
	// (missing ll-restrict binary or BPF filter), say so loudly instead of
	// silently overclaiming protection.
	b.warnUnappliedLayers(landlockApplied, len(seccompArgs) > 0)

	env := FilterEnvironment(sandbox.SourceEnvironment(cfg), cfg.HookCategory)
	name, argv, env := b.limitResources(cfg, args, env)
	argv = append(argv, "--")
	argv = append(argv, hookCmd...)

	sandboxOverhead := time.Since(setupStart)

	cmd := exec.CommandContext(ctx, name, argv...)
	cmd.ExtraFiles = extraFiles
	cmd.Env = sandbox.EnvList(env)
	cfg.Attach(cmd)

	result, err := sandbox.RunCommand(ctx, cmd, b.tier, guard.StderrTap())
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("executing %s: %w", name, err)
		}
		return nil, fmt.Errorf("%w: executing %s: %w", sandbox.ErrSetupFailed, name, err)
	}
	if err := guardrails.Enforce(); err != nil {
		return nil, err
	}
	if err := guard.Err(result.ExitCode, func() string { return shim.LinkageHint(hostExe) }); err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("executing %s: %w", name, ctx.Err())
		}
		return nil, err
	}
	if landlockApplied && isLandlockSetupFailure(result.ExitCode, guard.StderrHead()) {
		return nil, fmt.Errorf("%w: ll-restrict exited %d: %s",
			sandbox.ErrSetupFailed, result.ExitCode, sandbox.FirstLine(guard.StderrHead()))
	}
	result.SandboxOverhead = sandboxOverhead
	return result, nil
}

// snapshotGuardrails records the project's guardrail paths when the hook
// gets a writable view of them (guardrailsExposed). The read-only overlays
// cover only the paths that exist, so the snapshot is what catches one
// created, or a symlinked one replaced, while the hook runs. Without a
// writable view the snapshot is empty and always verifies.
func snapshotGuardrails(cfg *sandbox.SandboxConfig) (*sandbox.GuardrailSnapshot, error) {
	if !guardrailsExposed(cfg) {
		return &sandbox.GuardrailSnapshot{}, nil
	}
	return sandbox.SnapshotGuardrails(cfg.ProjectDir)
}

// launchHook returns bwrap's ExtraFiles, the argv bwrap runs after its "--"
// and whether that argv applies Landlock. The guard's ready fd follows the files already passed (the optional
// seccomp filter), so its number is computed, never fixed. The hook runs
// through the shim, itself inside ll-restrict when the tier claims Landlock:
// probing sets that claim only when the helper reports a usable ABI, and
// running the helper anyway where Landlock is off would fail every hook.
func launchHook(cfg *sandbox.SandboxConfig, tier sandbox.DegradationTier, files []*os.File, guard *sandbox.LaunchGuard) ([]*os.File, []string, bool) {
	extraFiles, readyFD := guard.AppendChildFile(files)
	shimCmd := append(shim.Argv(shim.SandboxPath(), readyFD), cfg.HookCommand...)
	if !sandbox.TierClaimsLandlock(tier) {
		return extraFiles, shimCmd, false
	}
	hookCmd := InjectLandlock(shimCmd, cfg)
	return extraFiles, hookCmd, len(hookCmd) > len(shimCmd)
}

// openSeccompFilter opens the compiled BPF filter, when the build provides one
// (the Nix build injects the path via ldflags), for bwrap to load through an
// inherited file descriptor. It returns the files to hand to the child and the
// bwrap arguments referencing them; both are empty when no filter is usable, so
// exec still runs in builds without one.
func openSeccompFilter() ([]*os.File, []string) {
	fp := sandbox.SeccompFilterFile()
	if fp == "" {
		return nil, nil
	}
	f, err := os.Open(fp) //nolint:gosec // path is a trusted build-time constant
	if err != nil {
		slog.Warn("seccomp filter present but unreadable; syscall filtering NOT applied",
			"path", fp, "error", err)
		return nil, nil
	}
	// cmd.ExtraFiles entries are handed to the child starting at fd 3.
	return []*os.File{f}, []string{"--seccomp", strconv.Itoa(3)}
}

// limitResources returns the program, its leading arguments and the process
// environment that run bwrap (with bwrapArgs, not yet terminated by "--")
// under cfg.Resources. When limits are requested and a systemd user session is
// usable, bwrap runs inside a transient `systemd-run --user --scope`: the scope
// execs bwrap in place, so stdio, the environment and the seccomp descriptor
// pass straight through. systemd-run needs the user-bus variables, which the
// hook allowlist strips, so they are added for systemd-run and removed again
// by bwrap (--unsetenv) before the hook starts. Otherwise bwrap runs directly
// and the unapplied limits are reported.
func (b *BubblewrapBackend) limitResources(cfg *sandbox.SandboxConfig, bwrapArgs []string, env map[string]string) (string, []string, map[string]string) {
	if !cfg.Resources.Any() {
		return b.bwrapBin, bwrapArgs, env
	}
	if b.systemdRunPath == "" {
		slog.Warn("systemd-run unavailable; sandbox resource limits NOT applied", "tier", b.tier.String())
		return b.bwrapBin, bwrapArgs, env
	}
	if err := sandbox.UserScopeUsable(b.systemdRunPath); err != nil {
		slog.Warn("systemd user scope unusable; sandbox resource limits NOT applied",
			"tier", b.tier.String(), "error", err)
		return b.bwrapBin, bwrapArgs, env
	}

	argv := append(sandbox.SystemdScopeArgs(cfg.Resources), b.bwrapBin)
	argv = append(argv, bwrapArgs...)
	for k, v := range sandbox.UserBusEnv() {
		if _, hookHasIt := env[k]; hookHasIt {
			continue
		}
		env[k] = v
		argv = append(argv, "--unsetenv", k)
	}
	return b.systemdRunPath, argv, env
}

// warnUnappliedLayers emits a warning for each LSM layer the backend's tier
// advertises but could not actually apply, so a reported tier never silently
// overstates the isolation delivered.
func (b *BubblewrapBackend) warnUnappliedLayers(landlockApplied, seccompApplied bool) {
	if sandbox.TierClaimsLandlock(b.tier) && !landlockApplied {
		slog.Warn("sandbox tier advertises Landlock but ll-restrict is unavailable; Landlock NOT applied",
			"tier", b.tier.String())
	}
	if sandbox.TierClaimsSeccomp(b.tier) && !seccompApplied {
		slog.Warn("sandbox tier advertises seccomp but no BPF filter is available; seccomp NOT applied",
			"tier", b.tier.String())
	}
}

var _ sandbox.SandboxBackend = (*BubblewrapBackend)(nil)
