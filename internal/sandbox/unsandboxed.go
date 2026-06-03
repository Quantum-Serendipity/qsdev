package sandbox

import (
	"bytes"
	"context"
	"os/exec"
	"time"
)

// runUnsandboxed executes a hook command directly without any sandbox isolation.
func runUnsandboxed(ctx context.Context, cfg *SandboxConfig) (*SandboxResult, error) {
	if len(cfg.HookCommand) == 0 {
		return &SandboxResult{ExitCode: 0, Tier: TierUnsandboxed}, nil
	}

	start := time.Now()

	cmd := exec.CommandContext(ctx, cfg.HookCommand[0], cfg.HookCommand[1:]...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if cfg.Environment != nil {
		for k, v := range cfg.Environment {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}

	if cfg.ProjectDir != "" {
		cmd.Dir = cfg.ProjectDir
	}

	err := cmd.Run()
	duration := time.Since(start)

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, err
		}
	}

	return &SandboxResult{
		ExitCode: exitCode,
		Stdout:   stdout.Bytes(),
		Stderr:   stderr.Bytes(),
		Duration: duration,
		Tier:     TierUnsandboxed,
	}, nil
}
