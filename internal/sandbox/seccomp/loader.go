package seccomp

import (
	"fmt"
	"os"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
)

// LoadFilter returns the seccomp BPF filter bytecode. It checks:
// 1. The override path from ldflags (Nix store path)
// 2. A fallback file path if available
// Returns an error if no filter is available.
func LoadFilter() ([]byte, error) {
	if path := sandbox.SeccompFilterFile(); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading seccomp filter from %s: %w", path, err)
		}
		return data, nil
	}

	return nil, fmt.Errorf("seccomp BPF filter not available (not built with Nix or filter not on PATH)")
}

// Available reports whether a seccomp BPF filter can be loaded.
func Available() bool {
	_, err := LoadFilter()
	return err == nil
}

// SeccompSupported checks whether seccomp is supported on the current system
// by reading /proc/sys/kernel/seccomp/actions_avail.
func SeccompSupported() bool {
	data, err := os.ReadFile("/proc/sys/kernel/seccomp/actions_avail")
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "errno")
}
