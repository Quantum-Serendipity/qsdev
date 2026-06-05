package cgroup

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
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

// Available checks whether the systemd-run binary exists at the configured path.
func (s *SystemdRunBackend) Available() error {
	if s.systemdRunPath == "" {
		return fmt.Errorf("systemd-run binary path not set")
	}
	if _, err := os.Stat(s.systemdRunPath); err != nil {
		return fmt.Errorf("systemd-run binary not found at %s: %w", s.systemdRunPath, err)
	}
	return nil
}

// Tier returns TierSystemdRun.
func (s *SystemdRunBackend) Tier() sandbox.DegradationTier {
	return sandbox.TierSystemdRun
}

// BuildArgs constructs the systemd-run command arguments from a SandboxConfig.
func BuildArgs(cfg *sandbox.SandboxConfig) []string {
	args := []string{
		"--user",
		"--scope",
	}

	if cfg.Resources.MemoryBytes > 0 {
		args = append(args, "-p", "MemoryMax="+strconv.FormatInt(cfg.Resources.MemoryBytes, 10))
	}

	if cfg.Resources.MaxPIDs > 0 {
		args = append(args, "-p", "TasksMax="+strconv.Itoa(cfg.Resources.MaxPIDs))
	}

	if cfg.Resources.CPUQuotaPercent > 0 && cfg.Resources.CPUQuotaPercent <= 10000 {
		args = append(args, "-p", "CPUQuota="+strconv.Itoa(cfg.Resources.CPUQuotaPercent)+"%")
	}

	args = append(args, "--")
	args = append(args, cfg.HookCommand...)

	return args
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
	execStart := time.Now()

	cmd := exec.CommandContext(ctx, s.systemdRunPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if cfg.Environment != nil {
		filtered := bwrap.FilterEnvironment(cfg.Environment, cfg.HookCategory)
		for k, v := range filtered {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}

	if cfg.ProjectDir != "" {
		cmd.Dir = cfg.ProjectDir
	}

	err := cmd.Run()
	duration := time.Since(execStart)

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("executing systemd-run: %w", err)
		}
	}

	return &sandbox.SandboxResult{
		ExitCode:        exitCode,
		Stdout:          stdout.Bytes(),
		Stderr:          stderr.Bytes(),
		Duration:        duration,
		SandboxOverhead: sandboxOverhead,
		Tier:            sandbox.TierSystemdRun,
	}, nil
}

// Compile-time interface compliance check.
var _ sandbox.SandboxBackend = (*SystemdRunBackend)(nil)
