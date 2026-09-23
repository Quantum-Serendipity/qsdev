package bwrap

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
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
// result. A failure to establish the sandbox (bwrap cannot start, or the
// ll-restrict helper fails before exec'ing the hook) is returned as an error
// wrapping sandbox.ErrSetupFailed, never as the hook's exit code.
func (b *BubblewrapBackend) RunHook(ctx context.Context, cfg *sandbox.SandboxConfig) (*sandbox.SandboxResult, error) {
	if len(cfg.HookCommand) == 0 {
		return &sandbox.SandboxResult{ExitCode: 0, Tier: b.tier}, nil
	}

	setupStart := time.Now()

	args, err := BuildArgs(cfg, b.tier)
	if err != nil {
		return nil, fmt.Errorf("building sandbox args: %w", err)
	}

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
	extraFiles, seccompArgs := openSeccompFilter()
	defer func() {
		for _, f := range extraFiles {
			_ = f.Close()
		}
	}()
	args = append(args, seccompArgs...)

	// Landlock layer: wrap the hook command with ll-restrict, but only when the
	// tier claims Landlock. Probing sets that claim only when the helper reports
	// a usable ABI; running the helper anyway on a host where Landlock is off
	// (e.g. missing from the boot lsm= list) fails every hook.
	hookCmd := cfg.HookCommand
	if sandbox.TierClaimsLandlock(b.tier) {
		hookCmd = InjectLandlock(cfg.HookCommand, cfg)
	}
	landlockApplied := len(hookCmd) > len(cfg.HookCommand)

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
	var taps []io.Writer
	stderrHead := &headWriter{limit: stderrHeadLimit}
	if landlockApplied {
		taps = append(taps, stderrHead)
	}

	result, err := sandbox.RunCommand(ctx, cmd, b.tier, taps...)
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("executing %s: %w", name, err)
		}
		return nil, fmt.Errorf("%w: executing %s: %w", sandbox.ErrSetupFailed, name, err)
	}
	if landlockApplied && isLandlockSetupFailure(result.ExitCode, stderrHead.buf) {
		return nil, fmt.Errorf("%w: ll-restrict exited %d: %s",
			sandbox.ErrSetupFailed, result.ExitCode, firstLine(stderrHead.buf))
	}
	result.SandboxOverhead = sandboxOverhead
	return result, nil
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

// firstLine returns the first line of b, for error messages.
func firstLine(b []byte) string {
	line, _, _ := bytes.Cut(b, []byte("\n"))
	return string(line)
}

var _ sandbox.SandboxBackend = (*BubblewrapBackend)(nil)
