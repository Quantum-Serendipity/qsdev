package bwrap

import (
	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
)

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
