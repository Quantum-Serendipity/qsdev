package sandbox

import (
	"context"
	"fmt"
	"os/exec"
)

// runUnsandboxed executes a hook command directly without any sandbox isolation.
func runUnsandboxed(ctx context.Context, cfg *SandboxConfig) (*SandboxResult, error) {
	if len(cfg.HookCommand) == 0 {
		return &SandboxResult{ExitCode: 0, Tier: TierUnsandboxed}, nil
	}

	cmd := exec.CommandContext(ctx, cfg.HookCommand[0], cfg.HookCommand[1:]...)
	cfg.Attach(cmd)

	// A caller-supplied environment replaces the inherited one entirely. EnvList
	// never returns nil, so an empty map stays empty instead of inheriting.
	if cfg.Environment != nil {
		cmd.Env = EnvList(cfg.Environment)
	}

	if cfg.ProjectDir != "" {
		cmd.Dir = cfg.ProjectDir
	}

	result, err := RunCommand(ctx, cmd, TierUnsandboxed)
	if err != nil {
		return nil, fmt.Errorf("running hook: %w", err)
	}
	return result, nil
}
