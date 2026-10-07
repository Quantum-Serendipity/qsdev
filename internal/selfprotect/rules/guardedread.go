package rules

import "github.com/Quantum-Serendipity/qsdev/pkg/fileutil"

// MaxGuardedFileBytes caps how much of a guarded file the self-protection
// rules read: far above any real config file, small enough that a hook never
// loads an attacker-sized file into memory.
const MaxGuardedFileBytes = 4 << 20

// ReadGuardedFile reads a guarded file the rules inspect, bounded by
// MaxGuardedFileBytes. It never opens a FIFO, device or socket (see
// fileutil.ReadRegularFile), so a hostile target cannot hang the hook.
func ReadGuardedFile(path string) ([]byte, error) {
	return fileutil.ReadRegularFile(path, MaxGuardedFileBytes)
}
