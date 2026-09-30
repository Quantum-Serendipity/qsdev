package state

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// ErrNotJoined reports a checkout that has a committed project config but no
// local init state (a fresh clone): it must be joined with init before any
// command regenerates or removes the team's committed files.
var ErrNotJoined = errors.New("project checkout not joined")

// ConfigExists reports whether the project config file exists in projectRoot.
func ConfigExists(projectRoot string) (bool, error) {
	return fileExists(filepath.Join(projectRoot, branding.Get().ConfigFile))
}

// NeedsJoin reports whether projectRoot is a fresh clone: the project config
// exists but the init state file does not. It is the single definition shared
// by init's mode detection and the mutating commands' join guard.
func NeedsJoin(projectRoot string) (bool, error) {
	hasConfig, err := ConfigExists(projectRoot)
	if err != nil || !hasConfig {
		return false, err
	}
	hasState, err := fileExists(filepath.Join(projectRoot, filepath.FromSlash(InitStateFile())))
	if err != nil {
		return false, err
	}
	return !hasState, nil
}

// fileExists stats path; only a not-exist result counts as absent, any other
// stat failure is returned so callers do not guess.
func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, fs.ErrNotExist):
		return false, nil
	default:
		return false, fmt.Errorf("checking %s: %w", path, err)
	}
}
