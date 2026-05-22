// Package shellintegration provides PATH management and shell completion
// installation for the qsdev binary.
package shellintegration

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fastcat.org/go/gdev/addons/bootstrap/textedit"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

func pathMarkerStart() string { return "# " + branding.Get().AppName + ": PATH setup" }
func pathMarkerEnd() string   { return "# " + branding.Get().AppName + " end PATH setup" }

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
		pathMarkerStart(),
		exportLine,
		pathMarkerEnd(),
	)

	// Ensure parent directory exists (the RC file may not exist yet).
	if err := os.MkdirAll(filepath.Dir(rcFile), 0o755); err != nil {
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

func normalizeShellName(shell string) string {
	name := shell
	if i := strings.LastIndexAny(shell, `/\`); i >= 0 {
		name = shell[i+1:]
	}
	name = strings.TrimSuffix(name, ".exe")
	return strings.ToLower(name)
}
