package bwrap

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
)

// BubblewrapBackend implements SandboxBackend using bubblewrap for namespace
// isolation. It supports three tiers depending on available kernel features:
// Full (bwrap + Landlock + seccomp), BwrapWithoutLandlock, BwrapWithoutSeccomp.
type BubblewrapBackend struct {
	tier      sandbox.DegradationTier
	bwrapBin  string
	hasUserNS bool
}

// NewBubblewrapBackend creates a BubblewrapBackend with the given tier, bwrap
// binary path, and unprivileged-user-namespace capability. hasUserNS records
// whether the host permits unprivileged user namespaces; every bwrap invocation
// emits --unshare-user, so a backend built without them fails every exec and
// must report itself unavailable (see Available).
func NewBubblewrapBackend(tier sandbox.DegradationTier, bwrapBin string, hasUserNS bool) *BubblewrapBackend {
	return &BubblewrapBackend{tier: tier, bwrapBin: bwrapBin, hasUserNS: hasUserNS}
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

	// Seccomp layer: pass the compiled BPF filter to bwrap through an inherited
	// file descriptor when one is available (the Nix build injects the path via
	// ldflags). This is a no-op in builds without a filter, so exec still runs.
	var extraFiles []*os.File
	defer func() {
		for _, f := range extraFiles {
			_ = f.Close()
		}
	}()
	seccompApplied := false
	if fp := sandbox.SeccompFilterFile(); fp != "" {
		if f, openErr := os.Open(fp); openErr == nil { //nolint:gosec // path is a trusted build-time constant
			// cmd.ExtraFiles entries are handed to the child starting at fd 3.
			childFD := 3 + len(extraFiles)
			args = append(args, "--seccomp", strconv.Itoa(childFD))
			extraFiles = append(extraFiles, f)
			seccompApplied = true
		} else {
			slog.Warn("seccomp filter present but unreadable; syscall filtering NOT applied",
				"path", fp, "error", openErr)
		}
	}

	// Landlock layer: wrap the hook command with ll-restrict when it is
	// available. InjectLandlock returns the command unchanged when ll-restrict
	// is absent, so this is also a safe no-op in unprovisioned environments.
	hookCmd := InjectLandlock(cfg.HookCommand, cfg)
	landlockApplied := len(hookCmd) > len(cfg.HookCommand)

	// Honesty: if the selected tier advertises an LSM layer we could not apply
	// (missing ll-restrict binary or BPF filter), say so loudly instead of
	// silently overclaiming protection.
	b.warnUnappliedLayers(landlockApplied, seccompApplied)

	// Append the hook command after the bwrap args.
	args = append(args, "--")
	args = append(args, hookCmd...)

	sandboxOverhead := time.Since(setupStart)
	execStart := time.Now()

	cmd := exec.CommandContext(ctx, b.bwrapBin, args...)
	cmd.ExtraFiles = extraFiles

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Set filtered environment.
	filteredEnv := FilterEnvironment(currentEnv(cfg), cfg.HookCategory)
	for k, v := range filteredEnv {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	err = cmd.Run()
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
