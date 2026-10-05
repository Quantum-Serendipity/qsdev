package projectctx

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// ErrNoUserDir reports that a per-user directory cannot be determined from
// the environment.
var ErrNoUserDir = errors.New("per-user directory unavailable")

// Dirs are the per-user directories of the application.
type Dirs struct {
	// Home is the user's home directory.
	Home string
	// State holds data that persists between runs but is not configuration,
	// such as session logs and bug-report drafts.
	State string
	// Cache holds data that can be deleted and recomputed, such as the
	// update-check cache.
	Cache string
	// Legacy is the historical ~/.<app> directory. Security state, installed
	// binaries and the docs corpus still live there.
	Legacy string
}

// Logs is the global session-log directory, State/logs.
func (d Dirs) Logs() string {
	return filepath.Join(d.State, logsDirName)
}

// UserDirs returns the per-user directories for the current process.
func UserDirs() (Dirs, error) {
	home, err := HomeDir()
	if err != nil {
		return Dirs{}, err
	}
	return userDirs(runtime.GOOS, os.Getenv, home)
}

// LegacyDir returns the legacy per-user directory ~/.<app>. It depends on the
// home directory alone, so callers that keep security state there (the user
// policy, session grants, trusted keys, sandbox approvals) never fail because
// the state or cache directory cannot be determined (an unset %LocalAppData%
// on Windows, say).
func LegacyDir() (string, error) {
	home, err := HomeDir()
	if err != nil {
		return "", err
	}
	return legacyDir(home), nil
}

func legacyDir(home string) string {
	return filepath.Join(home, DataDirName())
}

// userDirs is UserDirs as a pure function of the target OS, the environment
// and the home directory:
//
//   - State is $XDG_STATE_HOME/<app> when that is set and absolute, else
//     ~/.local/state/<app> on linux and the BSDs, ~/Library/Application
//     Support/<app> on darwin and %LocalAppData%\<app> on windows.
//   - Cache is $XDG_CACHE_HOME/<app> when that is set and absolute, else the
//     OS cache directory (as os.UserCacheDir) joined with <app>.
//   - Legacy is ~/.<app>.
//
// Relative XDG values are ignored, as the XDG base directory specification
// requires.
func userDirs(goos string, getenv func(string) string, home string) (Dirs, error) {
	app := branding.Get().AppName
	stateBase, cacheBase, err := osDefaults(goos, getenv, home)
	if xdg := getenv("XDG_STATE_HOME"); filepath.IsAbs(xdg) {
		stateBase = xdg
	} else if err != nil {
		return Dirs{}, err
	}
	if xdg := getenv("XDG_CACHE_HOME"); filepath.IsAbs(xdg) {
		cacheBase = xdg
	} else if err != nil {
		return Dirs{}, err
	}
	return Dirs{
		Home:   home,
		State:  filepath.Join(stateBase, app),
		Cache:  filepath.Join(cacheBase, app),
		Legacy: legacyDir(home),
	}, nil
}

// osDefaults returns the OS default state and cache base directories, used
// when the XDG variables are unset.
func osDefaults(goos string, getenv func(string) string, home string) (state, cache string, err error) {
	switch goos {
	case "windows":
		lad := getenv("LocalAppData")
		if !filepath.IsAbs(lad) {
			return "", "", fmt.Errorf("%%LocalAppData%% is %q: %w", lad, ErrNoUserDir)
		}
		return lad, lad, nil
	case "darwin", "ios":
		return filepath.Join(home, "Library", "Application Support"), filepath.Join(home, "Library", "Caches"), nil
	default:
		return filepath.Join(home, ".local", "state"), filepath.Join(home, ".cache"), nil
	}
}
