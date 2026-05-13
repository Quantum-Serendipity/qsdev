// Package shellintegration provides PATH management and shell completion
// installation for the qsdev binary.
package shellintegration

import (
	"fmt"
	"os"
	"strings"

	"fastcat.org/go/gdev/addons/bootstrap/textedit"
)

const (
	pathMarkerStart = "# qsdev: PATH setup"
	pathMarkerEnd   = "# qsdev end PATH setup"
)

// EnsurePath idempotently ensures that dir is on the user's PATH by editing
// the given shell RC file. It uses marker comments to allow future updates.
// If dir is already present in the current PATH environment variable, the
// function still ensures the RC file has the appropriate export line so that
// future shell sessions also include it.
func EnsurePath(dir string, shell string, rcFile string) error {
	if dir == "" {
		return fmt.Errorf("dir must not be empty")
	}
	if rcFile == "" {
		return fmt.Errorf("rcFile must not be empty")
	}

	exportLine := pathExportLine(dir, shell)

	editor := textedit.SpliceRange(
		pathMarkerStart,
		exportLine,
		pathMarkerEnd,
	)

	// Ensure parent directory exists (the RC file may not exist yet).
	if err := os.MkdirAll(parentDir(rcFile), 0o755); err != nil {
		return fmt.Errorf("creating parent directory for %s: %w", rcFile, err)
	}

	_, err := textedit.EditFile(rcFile, editor)
	if err != nil {
		return fmt.Errorf("editing %s: %w", rcFile, err)
	}
	return nil
}

// pathExportLine returns the shell-appropriate line that prepends dir to PATH.
func pathExportLine(dir string, shell string) string {
	switch normalizeShellName(shell) {
	case "fish":
		return fmt.Sprintf("fish_add_path %s", dir)
	case "pwsh", "powershell":
		return fmt.Sprintf(`$env:PATH = "%s" + [IO.Path]::PathSeparator + $env:PATH`, dir)
	default: // bash, zsh, and anything POSIX-like
		return fmt.Sprintf(`export PATH="%s:$PATH"`, dir)
	}
}

// normalizeShellName extracts the base shell name from a path or name, and
// lowercases it for comparison.
func normalizeShellName(shell string) string {
	// Handle paths like /usr/bin/zsh -> zsh
	name := shell
	if idx := strings.LastIndex(shell, "/"); idx >= 0 {
		name = shell[idx+1:]
	}
	return strings.ToLower(name)
}

// parentDir returns the parent directory of a file path. This is a simple
// string operation that avoids importing path/filepath just for Dir().
func parentDir(path string) string {
	idx := strings.LastIndex(path, "/")
	if idx < 0 {
		return "."
	}
	if idx == 0 {
		return "/"
	}
	return path[:idx]
}
