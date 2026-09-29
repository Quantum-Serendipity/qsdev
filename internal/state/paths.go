package state

import (
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// InitStateFile returns the devinit (init/join/update) state file path
// relative to the project root.
func InitStateFile() string {
	b := branding.Get()
	return b.StateDir + "/." + b.AppName + "-init-state.yaml"
}

// DevenvStateFile returns the devenv addon state file path relative to the
// project root.
func DevenvStateFile() string {
	return ".devenv/." + branding.Get().AppName + "-state.yaml"
}

// ClaudeStateFile returns the Claude Code addon state file path relative to
// the project root.
func ClaudeStateFile() string {
	return ".claude/." + branding.Get().AppName + "-claude-state.yaml"
}

// StateFilePaths returns the standard state file locations relative to the
// project root. Used by posture scoring and teardown to locate all state files.
func StateFilePaths() [3]string {
	return [3]string{
		InitStateFile(),
		DevenvStateFile(),
		ClaudeStateFile(),
	}
}
