package cgroup

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
	"github.com/Quantum-Serendipity/qsdev/internal/sandbox/bwrap"
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

// BuildArgs constructs the systemd-run command arguments from a SandboxConfig.
func BuildArgs(cfg *sandbox.SandboxConfig) []string {
	return append(sandbox.SystemdScopeArgs(cfg.Resources), cfg.HookCommand...)
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
// resource limits derived from the sandbox configuration.
func (s *SystemdRunBackend) RunHook(ctx context.Context, cfg *sandbox.SandboxConfig) (*sandbox.SandboxResult, error) {
	if len(cfg.HookCommand) == 0 {
		return &sandbox.SandboxResult{ExitCode: 0, Tier: sandbox.TierSystemdRun}, nil
	}

	setupStart := time.Now()

	args := BuildArgs(cfg)

	sandboxOverhead := time.Since(setupStart)

	cmd := exec.CommandContext(ctx, s.systemdRunPath, args...)
	cfg.Attach(cmd)
	cmd.Env = hookEnvironment(cfg)

	if cfg.ProjectDir != "" {
		cmd.Dir = cfg.ProjectDir
	}

	result, err := sandbox.RunCommand(ctx, cmd, sandbox.TierSystemdRun)
	if err != nil {
		return nil, fmt.Errorf("executing systemd-run: %w", err)
	}
	result.SandboxOverhead = sandboxOverhead
	return result, nil
}

// Compile-time interface compliance check.
var _ sandbox.SandboxBackend = (*SystemdRunBackend)(nil)
