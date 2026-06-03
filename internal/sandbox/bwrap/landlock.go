package bwrap

import (
	"os/user"
	"path/filepath"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
)

// credentialDenyPaths are paths that hooks must never read, regardless of
// policy. Paths starting with ~ are expanded to the user's home directory.
var credentialDenyPaths = []string{
	"~/.ssh",
	"~/.gnupg",
	"~/.aws",
	"~/.azure",
	"~/.config/gcloud",
	"~/.kube",
	"~/.docker/config.json",
	"~/.netrc",
}

// PrepareLandlockFlags builds the ll-restrict CLI flags for the given config.
// Returns nil if ll-restrict is unavailable.
func PrepareLandlockFlags(cfg *sandbox.SandboxConfig) []string {
	llBin := sandbox.LLRestrictBin()
	if llBin == "" {
		return nil
	}

	var flags []string

	// Nix store is always read-only.
	flags = append(flags, "--ro", "/nix/store")

	// System files.
	flags = append(flags, "--ro", "/etc")

	// Tmp is always writable.
	flags = append(flags, "--rw", "/tmp")

	// Project directory: ro for linters, rw for formatters/generators.
	if cfg.ProjectDir != "" {
		if cfg.HookCategory.WorktreeReadOnly() {
			flags = append(flags, "--ro", cfg.ProjectDir)
		} else {
			flags = append(flags, "--rw", cfg.ProjectDir)
		}
	}

	// Extra mounts from config.
	for _, m := range cfg.Mounts {
		if m.ReadOnly {
			flags = append(flags, "--ro", m.Source)
		} else {
			flags = append(flags, "--rw", m.Source)
		}
	}

	// Network denial.
	if !cfg.HookCategory.NetworkAllowed() {
		flags = append(flags, "--deny-net")
	}

	return flags
}

// InjectLandlock modifies the hook command to run through ll-restrict.
// The original command is prefixed with: ll-restrict <flags> -- <original-cmd>
// Returns the original command unchanged if ll-restrict is unavailable.
func InjectLandlock(hookCmd []string, cfg *sandbox.SandboxConfig) []string {
	flags := PrepareLandlockFlags(cfg)
	if flags == nil {
		return hookCmd
	}

	llBin := sandbox.LLRestrictBin()
	result := []string{llBin}
	result = append(result, flags...)
	result = append(result, "--")
	result = append(result, hookCmd...)
	return result
}

// expandDenyPaths resolves ~ in credential deny paths to the user's home.
func expandDenyPaths() []string {
	home := ""
	if u, err := user.Current(); err == nil {
		home = u.HomeDir
	}

	paths := make([]string, 0, len(credentialDenyPaths))
	for _, p := range credentialDenyPaths {
		if len(p) > 0 && p[0] == '~' && home != "" {
			paths = append(paths, filepath.Join(home, p[1:]))
		} else {
			paths = append(paths, p)
		}
	}
	return paths
}
