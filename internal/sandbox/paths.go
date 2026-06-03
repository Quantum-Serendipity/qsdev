package sandbox

import "os/exec"

// llRestrictPath and seccompFilterPath are set via -ldflags at Nix build time.
// Example: -X github.com/Quantum-Serendipity/qsdev/internal/sandbox.llRestrictPath=/nix/store/.../bin/ll-restrict
var (
	llRestrictPath    string //nolint:gochecknoglobals
	seccompFilterPath string //nolint:gochecknoglobals
)

// LLRestrictBin returns the path to the ll-restrict binary. It checks (in
// order): the Nix store path injected at build time, then $PATH.
// Returns empty string if unavailable.
func LLRestrictBin() string {
	if llRestrictPath != "" {
		return llRestrictPath
	}
	if path, err := exec.LookPath("ll-restrict"); err == nil {
		return path
	}
	return ""
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
