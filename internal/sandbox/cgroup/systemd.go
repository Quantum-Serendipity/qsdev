package cgroup

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
	"github.com/Quantum-Serendipity/qsdev/internal/sandbox/bwrap"
	"github.com/Quantum-Serendipity/qsdev/internal/sandbox/shim"
)

// SystemdRunBackend implements sandbox.SandboxBackend using systemd-run --user
// to execute hooks inside a transient user scope with resource limits.
type SystemdRunBackend struct {
	systemdRunPath string
}

// NewSystemdRunBackend creates a SystemdRunBackend with the given systemd-run
// binary path.
func NewSystemdRunBackend(path string) *SystemdRunBackend {
	return &SystemdRunBackend{systemdRunPath: path}
}

// Name returns the backend identifier.
func (s *SystemdRunBackend) Name() string { return "systemd-run" }

// Available checks that the systemd-run binary exists AND that a systemd user
// session bus is reachable. Without one, every `systemd-run --user` invocation
// fails, and selecting this backend over the unsandboxed fallback would turn
// every hook into a spurious exit 1.
func (s *SystemdRunBackend) Available() error {
	return sandbox.UserScopeUsable(s.systemdRunPath)
}

// Tier returns TierSystemdRun.
func (s *SystemdRunBackend) Tier() sandbox.DegradationTier {
	return sandbox.TierSystemdRun
}

// BuildArgs constructs the systemd-run command arguments from a SandboxConfig:
// the scope, then the launcher argv (RunHook's shim) when given, then the hook.
func BuildArgs(cfg *sandbox.SandboxConfig, launcher ...string) []string {
	args := append(sandbox.SystemdScopeArgs(cfg.Resources), launcher...)
	return append(args, cfg.HookCommand...)
}

// hookEnvironment returns the environment for systemd-run: the hook's filtered
// environment (always filtered, including when the caller supplied none and the
// process environment is the source) plus the user-bus variables systemd-run
// itself needs. In scope mode systemd-run execs the hook with its own
// environment, so the bus variables reach the hook as well; this tier has no
// namespace isolation, and the bus socket's location is derivable from the uid
// anyway, so that exposes nothing the hook could not already reach.
func hookEnvironment(cfg *sandbox.SandboxConfig) []string {
	env := bwrap.FilterEnvironment(sandbox.SourceEnvironment(cfg), cfg.HookCategory)
	for k, v := range sandbox.UserBusEnv() {
		env[k] = v
	}
	return sandbox.EnvList(env)
}

// RunHook executes the hook command inside a systemd-run --user scope with
// resource limits derived from the sandbox configuration. The hook runs
// through the shim (internal/sandbox/shim), the host's qsdev binary itself
// since this tier has no namespace, which signals a sandbox.LaunchGuard just
// before it execs the hook. A run that never reached the hook (systemd-run
// could not reach the user bus, or start the scope) is returned as an error
// wrapping sandbox.ErrSetupFailed, never as the hook's exit code.
func (s *SystemdRunBackend) RunHook(ctx context.Context, cfg *sandbox.SandboxConfig) (*sandbox.SandboxResult, error) {
	if len(cfg.HookCommand) == 0 {
		return &sandbox.SandboxResult{ExitCode: 0, Tier: sandbox.TierSystemdRun}, nil
	}

	setupStart := time.Now()

	hostExe, err := shim.HostExecutable()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", sandbox.ErrSetupFailed, err)
	}
	guard, err := sandbox.NewLaunchGuard()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", sandbox.ErrSetupFailed, err)
	}
	defer func() { _ = guard.Close() }()
	extraFiles, readyFD := guard.AppendChildFile(nil)
	args := BuildArgs(cfg, shim.Argv(hostExe, readyFD)...)

	sandboxOverhead := time.Since(setupStart)

	cmd := exec.CommandContext(ctx, s.systemdRunPath, args...)
	cmd.ExtraFiles = extraFiles
	cfg.Attach(cmd)
	cmd.Env = hookEnvironment(cfg)

	if cfg.ProjectDir != "" {
		cmd.Dir = cfg.ProjectDir
	}

	result, err := sandbox.RunCommand(ctx, cmd, sandbox.TierSystemdRun, guard.StderrTap())
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("executing systemd-run: %w", err)
		}
		return nil, fmt.Errorf("%w: executing systemd-run: %w", sandbox.ErrSetupFailed, err)
	}
	if err := guard.Err(result.ExitCode, func() string { return "" }); err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("executing systemd-run: %w", ctx.Err())
		}
		return nil, err
	}
	result.SandboxOverhead = sandboxOverhead
	return result, nil
}

// Compile-time interface compliance check.
var _ sandbox.SandboxBackend = (*SystemdRunBackend)(nil)
