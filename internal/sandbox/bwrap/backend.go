package bwrap

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
)

// BubblewrapBackend implements SandboxBackend using bubblewrap for namespace
// isolation. It supports three tiers depending on available kernel features:
// Full (bwrap + Landlock + seccomp), BwrapWithoutLandlock, BwrapWithoutSeccomp.
type BubblewrapBackend struct {
	tier     sandbox.DegradationTier
	bwrapBin string
}

// NewBubblewrapBackend creates a BubblewrapBackend with the given tier and
// bwrap binary path.
func NewBubblewrapBackend(tier sandbox.DegradationTier, bwrapBin string) *BubblewrapBackend {
	return &BubblewrapBackend{tier: tier, bwrapBin: bwrapBin}
}

func (b *BubblewrapBackend) Name() string                  { return "bubblewrap" }
func (b *BubblewrapBackend) Tier() sandbox.DegradationTier { return b.tier }

// Available checks whether bwrap is accessible.
func (b *BubblewrapBackend) Available() error {
	if b.bwrapBin == "" {
		return fmt.Errorf("bubblewrap binary path not set")
	}
	if _, err := os.Stat(b.bwrapBin); err != nil {
		return fmt.Errorf("bubblewrap binary not found at %s: %w", b.bwrapBin, err)
	}
	return nil
}

// RunHook creates a bubblewrap sandbox, executes the hook, and returns the result.
func (b *BubblewrapBackend) RunHook(ctx context.Context, cfg *sandbox.SandboxConfig) (*sandbox.SandboxResult, error) {
	if len(cfg.HookCommand) == 0 {
		return &sandbox.SandboxResult{ExitCode: 0, Tier: b.tier}, nil
	}

	setupStart := time.Now()

	args, err := BuildArgs(cfg, b.tier)
	if err != nil {
		return nil, fmt.Errorf("building sandbox args: %w", err)
	}

	// Append the hook command after the bwrap args.
	args = append(args, "--")
	args = append(args, cfg.HookCommand...)

	sandboxOverhead := time.Since(setupStart)
	execStart := time.Now()

	cmd := exec.CommandContext(ctx, b.bwrapBin, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Set filtered environment.
	filteredEnv := FilterEnvironment(currentEnv(cfg), cfg.HookCategory)
	for k, v := range filteredEnv {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	err := cmd.Run()
	duration := time.Since(execStart)

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("executing bwrap: %w", err)
		}
	}

	return &sandbox.SandboxResult{
		ExitCode:        exitCode,
		Stdout:          stdout.Bytes(),
		Stderr:          stderr.Bytes(),
		Duration:        duration,
		SandboxOverhead: sandboxOverhead,
		Tier:            b.tier,
	}, nil
}

// currentEnv builds the environment map from the config or from the current
// process environment.
func currentEnv(cfg *sandbox.SandboxConfig) map[string]string {
	if cfg.Environment != nil {
		return cfg.Environment
	}
	env := make(map[string]string)
	for _, e := range os.Environ() {
		if k, v, ok := splitEnvVar(e); ok {
			env[k] = v
		}
	}
	return env
}

// splitEnvVar splits "KEY=VALUE" into key and value.
func splitEnvVar(s string) (string, string, bool) {
	for i := range s {
		if s[i] == '=' {
			return s[:i], s[i+1:], true
		}
	}
	return "", "", false
}

var _ sandbox.SandboxBackend = (*BubblewrapBackend)(nil)
