package cmdutil

import (
	"fmt"
	"os"

	"github.com/Quantum-Serendipity/qsdev/internal/logging"
)

// ProjectRoot returns the root of the project enclosing the current working
// directory, located with the shared project marker set (see
// logging.FindProjectRoot), so commands run from a subdirectory act on the
// real project instead of creating a second, nested one, and agree with logs
// and bug reports about where that project is. Outside any project it returns
// the working directory, which keeps commands that may run before
// initialization (e.g. enable) working.
func ProjectRoot() (string, error) {
	wd, err := WorkingDir()
	if err != nil {
		return "", err
	}
	if root, ok := logging.FindProjectRoot(wd); ok {
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
