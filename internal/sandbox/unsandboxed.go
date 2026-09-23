package sandbox

import (
	"context"
	"os/exec"
)

// runUnsandboxed executes a hook command directly without any sandbox isolation.
func runUnsandboxed(ctx context.Context, cfg *SandboxConfig) (*SandboxResult, error) {
	if len(cfg.HookCommand) == 0 {
		return &SandboxResult{ExitCode: 0, Tier: TierUnsandboxed}, nil
	}

	cmd := exec.CommandContext(ctx, cfg.HookCommand[0], cfg.HookCommand[1:]...)

	// Claude Code delivers the tool call on stdin; RunCommand captures
	// stdout and stderr.
	cmd.Stdin = cfg.Stdin

	if cfg.Environment != nil {
		for k, v := range cfg.Environment {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}

	if cfg.ProjectDir != "" {
		cmd.Dir = cfg.ProjectDir
	}

	return RunCommand(ctx, cmd, TierUnsandboxed)
}
