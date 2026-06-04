package bwrap

import (
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
)

// BuildArgs constructs the bwrap command-line arguments for the given sandbox
// configuration and degradation tier. The returned slice does NOT include
// "bwrap" itself; the caller prepends that. It validates all mount source paths
// and the project directory before constructing arguments.
func BuildArgs(cfg *sandbox.SandboxConfig, _ sandbox.DegradationTier) ([]string, error) {
	if cfg.ProjectDir != "" {
		if err := ValidateMountPath(cfg.ProjectDir); err != nil {
			return nil, fmt.Errorf("validating project dir: %w", err)
		}
	}
	for _, m := range cfg.Mounts {
		if err := ValidateMountPath(m.Source); err != nil {
			return nil, fmt.Errorf("validating mount source: %w", err)
		}
	}

	var args []string

	// 1. Namespace flags.
	args = append(args, "--unshare-user", "--unshare-pid", "--unshare-ipc", "--unshare-uts")

	networkAllowed := cfg.HookCategory.NetworkAllowed() ||
		cfg.Network.Mode == "allow" ||
		cfg.Network.Mode == "filtered"
	if !networkAllowed {
		args = append(args, "--unshare-net")
	}

	// 2. Session control.
	args = append(args, "--die-with-parent", "--new-session")

	// 3. Core filesystem.
	args = append(args, "--dev", "/dev", "--proc", "/proc", "--tmpfs", "/tmp")

	// 4. Nix store (read-only).
	args = append(args, "--ro-bind", "/nix/store", "/nix/store")

	// 5. Project directory.
	if cfg.HookCategory.WorktreeReadOnly() {
		args = append(args, "--ro-bind", cfg.ProjectDir, cfg.ProjectDir)
	} else {
		args = append(args, "--bind", cfg.ProjectDir, cfg.ProjectDir)
	}

	// 6. System files (always read-only).
	args = append(args,
		"--ro-bind", "/etc/passwd", "/etc/passwd",
		"--ro-bind", "/etc/group", "/etc/group",
		"--ro-bind", "/etc/hosts", "/etc/hosts",
	)
	if networkAllowed {
		args = append(args, "--ro-bind", "/etc/resolv.conf", "/etc/resolv.conf")
	}

	// 7. Extra mounts.
	for _, m := range cfg.Mounts {
		if m.ReadOnly {
			args = append(args, "--ro-bind", m.Source, m.Target)
		} else {
			args = append(args, "--bind", m.Source, m.Target)
		}
	}

	// 8. Additional Nix store paths.
	for _, p := range cfg.NixStorePaths {
		args = append(args, "--ro-bind", p, p)
	}

	return args, nil
}
