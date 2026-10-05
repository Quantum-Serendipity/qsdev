package sandbox

import (
	"os"
	"path"
	"strings"
)

// NixStoreDir is the Nix store, which every namespace sandbox mounts read-only.
const NixStoreDir = "/nix/store"

// llRestrictPath and seccompFilterPath are set via -ldflags at Nix build time.
// Example: -X github.com/Quantum-Serendipity/qsdev/internal/sandbox.llRestrictPath=/nix/store/.../bin/ll-restrict
var (
	llRestrictPath    string //nolint:gochecknoglobals
	seccompFilterPath string //nolint:gochecknoglobals
)

// LLRestrictBin returns the path to the ll-restrict helper, or "" when it is
// unavailable. Only the Nix store path injected at build time is used, and
// only when it exists: the helper must be visible inside the sandbox (which
// mounts the store), and a helper found on PATH is neither trusted nor
// reachable there, so it would make the tier claim Landlock and then fail
// every hook (U19-02).
func LLRestrictBin() string {
	p := path.Clean(llRestrictPath)
	if !strings.HasPrefix(p, NixStoreDir+"/") {
		return ""
	}
	if _, err := os.Stat(p); err != nil {
		return ""
	}
	return p
}

// SeccompFilterFile returns the path to the pre-compiled seccomp BPF filter.
// It checks the Nix store path injected at build time first.
// Returns empty string if unavailable (caller should use embedded BPF).
func SeccompFilterFile() string {
	if seccompFilterPath != "" {
		return seccompFilterPath
	}
	return ""
}
