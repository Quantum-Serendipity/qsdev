package bwrap

import (
	"path/filepath"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
	"github.com/Quantum-Serendipity/qsdev/internal/sandbox/denylist"
)

// PrepareLandlockFlags builds the ll-restrict CLI flags for the given config.
// Returns nil if ll-restrict is unavailable.
func PrepareLandlockFlags(cfg *sandbox.SandboxConfig) []string {
	if sandbox.LLRestrictBin() == "" {
		return nil
	}
	return landlockFlags(cfg)
}

// landlockFlags builds the ll-restrict CLI flags for cfg. It performs no binary
// lookup, so the policy it encodes can be tested on hosts without ll-restrict.
func landlockFlags(cfg *sandbox.SandboxConfig) []string {
	var flags []string

	// Nix store is always read-only.
	flags = append(flags, "--ro", "/nix/store")

	// System files.
	flags = append(flags, "--ro", "/etc")

	// Tmp is always writable.
	flags = append(flags, "--rw", "/tmp")

	// Device nodes and procfs. bwrap's --dev already limits /dev to a minimal
	// set (null, zero, full, random, urandom, tty, pts, shm), and ordinary
	// hooks need it (`cmd >/dev/null`, /dev/urandom); /proc/self is read by
	// most runtimes.
	flags = append(flags, "--rw", "/dev", "--ro", "/proc")

	// Project directory: ro for linters, rw for formatters/generators.
	if cfg.ProjectDir != "" {
		if cfg.WorktreeReadOnly() {
			flags = append(flags, "--ro", cfg.ProjectDir)
		} else {
			flags = append(flags, "--rw", cfg.ProjectDir)
		}
	}

	// Extra mounts from config, granted at the Target where bwrap mounted
	// them. Deny entries are never Mounts (cfg.Deny carries them and
	// BuildArgs rejects any mount that would expose one); as defense in depth
	// a mount touching a built-in or policy deny path is still never granted.
	for _, m := range cfg.Mounts {
		if touchesDenyPath(m, cfg.Deny) {
			continue
		}
		if m.ReadOnly {
			flags = append(flags, "--ro", m.Target)
		} else {
			flags = append(flags, "--rw", m.Target)
		}
	}

	// Network denial, derived from the same resolved mode as bwrap's
	// --unshare-net so the two layers never disagree.
	if cfg.NetworkIsolated() {
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

// touchesDenyPath reports whether mount m's source or target overlaps the
// built-in credential deny list or lies under one of the policy's deny entries.
func touchesDenyPath(m sandbox.MountSpec, deny []string) bool {
	if IsDenyPath(m.Source) || IsDenyPath(m.Target) {
		return true
	}
	for _, d := range deny {
		for _, p := range []string{m.Source, m.Target} {
			if denylist.Overlaps(filepath.Clean(p), filepath.Clean(d)) {
				return true
			}
		}
	}
	return false
}
