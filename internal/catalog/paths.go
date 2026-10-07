package catalog

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/projectctx"
	"github.com/Quantum-Serendipity/qsdev/internal/userhome"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// OrgConfigPath returns the expected path for the user-level defaults file.
// Priority: $QSDEV_ORG_CONFIG > ~/.config/qsdev/defaults.yaml, where ~ is the
// account's home directory from the user database (see homeOrgConfigPath). It
// returns "" when neither is available.
//
// Inside a test binary the home-directory fallback is not used, so tests
// exercise the embedded catalog rather than whatever overlay the developer's
// machine happens to have (and they are loaded during package init, before
// any TestMain could isolate HOME). Tests that need an org overlay set
// $QSDEV_ORG_CONFIG explicitly.
func OrgConfigPath() string {
	if p := os.Getenv(branding.Get().EnvPrefix + "ORG_CONFIG"); p != "" {
		return p
	}
	return defaultOrgConfigPath()
}

// defaultOrgConfigPath returns the overlay OrgConfigPath falls back to when
// <EnvPrefix>ORG_CONFIG is unset: the account's home overlay, or "" inside a
// test binary (see OrgConfigPath). A variable so tests can give it a home.
var defaultOrgConfigPath = func() string {
	if testing.Testing() {
		return ""
	}
	return homeOrgConfigPath()
}

// accountHome resolves the home directory the user database records for the
// account (a variable for tests).
var accountHome = userhome.Account

// warnUnanchoredOverlay reports, once per process, an overlay below HOME that
// is ignored because the account's home directory cannot be resolved.
var warnUnanchoredOverlay sync.Once

// homeOrgConfigPath returns ~/.config/<app>/defaults.yaml below the home
// directory the user database records for the account (userhome.Account,
// which also asks NSS through getent, so directory-service accounts resolve
// in a static build), or "" when it cannot be resolved. HOME and USERPROFILE
// never move it: a line such as `HOME=/tmp/e qsdev claude update`, however
// the agent spells the assignment or whichever script it runs, would otherwise
// point a regeneration at an overlay of its own making. When the account
// cannot be resolved the home overlay is not read at all, and an overlay
// found below HOME is reported instead of trusted; for an account the user
// database has no entry for, <EnvPrefix>ORG_CONFIG names it (see
// OrgConfigPin.Unanchored).
func homeOrgConfigPath() string {
	home, err := accountHome()
	if err == nil {
		return HomeOrgConfigPath(home)
	}
	if envHome, envErr := envHomeDir(); envErr == nil {
		if p := HomeOrgConfigPath(envHome); fileExists(p) {
			hint := "the account's home directory could not be looked up"
			if errors.Is(err, userhome.ErrNoAccount) {
				hint = "the user database has no entry for this account; set " + branding.Get().EnvPrefix + "ORG_CONFIG to use it"
			}
			warnUnanchoredOverlay.Do(func() {
				slog.Warn("ignoring the org defaults file below HOME: "+hint, "path", p, "error", err)
			})
		}
	}
	return ""
}

// envHomeDir returns the home directory the environment names (HOME, or
// USERPROFILE on Windows), which the agent can change; only a warning and the
// test-binary pins location (pinsHome) use it.
func envHomeDir() (string, error) {
	return projectctx.HomeDir()
}

// HomeOrgConfigPath returns the user-level defaults file below home,
// <home>/.config/<app>/defaults.yaml: the path OrgConfigPath falls back to
// when <EnvPrefix>ORG_CONFIG is unset.
func HomeOrgConfigPath(home string) string {
	return filepath.Join(branding.Get().OrgConfigDir(home), "defaults.yaml")
}

// OrgConfigFile returns the user-level defaults file path if it exists,
// or empty string if not.
func OrgConfigFile() string {
	p := OrgConfigPath()
	if p == "" {
		return ""
	}
	if fileExists(p) {
		return p
	}
	return ""
}

// fileExists reports whether p names an existing file.
func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// ProjectConfigPath returns the expected path for a project-level defaults
// file, <projectRoot>/.qsdev/defaults.yaml. The file is committed with the
// project and may only add or tighten (see Load).
func ProjectConfigPath(projectRoot string) string {
	if projectRoot == "" {
		return ""
	}
	return filepath.Join(projectRoot, "."+branding.Get().AppName, "defaults.yaml")
}

// ProjectConfigFile returns the project-level defaults file path if it exists,
// or "" if not. The file is policy, so an existing one must pass the project
// trust rule (projectctx.CheckTrusted), as must the state directory holding
// it: when the file, the directory, or the project root holding that
// directory could have been written by another local user (it is foreign-
// owned or world-writable), ProjectConfigFile refuses it with an error that
// wraps projectctx.ErrUntrusted and names the file and the fix, rather than
// applying or dropping it.
func ProjectConfigFile(projectRoot string) (string, error) {
	p := ProjectConfigPath(projectRoot)
	if p == "" || !fileExists(p) {
		return "", nil
	}
	for _, entry := range []string{p, filepath.Dir(p)} {
		if err := projectctx.CheckTrusted(entry); err != nil {
			return "", fmt.Errorf("refusing project defaults %s: %w (fix: remove world write access with 'chmod o-w', or 'chown' it to yourself)", p, err)
		}
	}
	return p, nil
}
