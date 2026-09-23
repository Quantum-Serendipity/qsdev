package cmdutil

import (
	"fmt"
	"os"

	"github.com/Quantum-Serendipity/qsdev/internal/logging"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

// ProjectRoot returns the root of the project enclosing the current working
// directory (see FindProjectRoot), so commands run from a subdirectory act on
// the real project instead of creating a second, nested one. Outside any
// project it returns the working directory, which keeps commands that may run
// before initialization (e.g. enable) working.
func ProjectRoot() (string, error) {
	wd, err := WorkingDir()
	if err != nil {
		return "", err
	}
	if root, ok := FindProjectRoot(wd); ok {
		return root, nil
	}
	return wd, nil
}

// WorkingDir returns the current working directory, wrapping any error with a
// consistent message. Commands that deliberately target the current directory
// rather than an enclosing project (init, which creates a project where it is
// run) use it instead of ProjectRoot.
func WorkingDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("determining working directory: %w", err)
	}
	return wd, nil
}

// FindProjectRoot walks up from start to the nearest directory (start itself
// included) that holds the project config file or the generated-state
// directory, the two markers only an initialized (or tool-enabled) project
// has. It reports false when no such directory exists.
func FindProjectRoot(start string) (string, bool) {
	b := branding.Get()
	return logging.WalkUp(start, func(dir string) bool {
		return fileutil.FileExists(dir, b.ConfigFile) || fileutil.DirExists(dir, b.StateDir)
	})
}
