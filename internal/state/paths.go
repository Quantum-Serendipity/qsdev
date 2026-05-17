package state

import (
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// StateFilePaths returns the standard state file locations relative to the
// project root. Used by posture scoring and teardown to locate all state files.
func StateFilePaths() [3]string {
	b := branding.Get()
	return [3]string{
		b.StateDir + "/." + b.AppName + "-init-state.yaml",
		".devenv/." + b.AppName + "-state.yaml",
		".claude/." + b.AppName + "-claude-state.yaml",
	}
}
