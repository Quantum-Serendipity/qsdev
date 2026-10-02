package catalog

import (
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"

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
// found below HOME is reported instead of trusted; set <EnvPrefix>ORG_CONFIG
// to use it.
func homeOrgConfigPath() string {
	home, err := accountHome()
	if err == nil {
		return HomeOrgConfigPath(home)
	}
	if envHome, envErr := os.UserHomeDir(); envErr == nil {
		if p := HomeOrgConfigPath(envHome); fileExists(p) {
			warnUnanchoredOverlay.Do(func() {
				slog.Warn("ignoring the org defaults file below HOME: the user database has no home directory for this account; set "+
					branding.Get().EnvPrefix+"ORG_CONFIG to use it",
					"path", p, "error", err)
			})
		}
	}
	return ""
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
// or empty string if not.
func ProjectConfigFile(projectRoot string) string {
	p := ProjectConfigPath(projectRoot)
	if p == "" {
		return ""
	}
	if fileExists(p) {
		return p
	}
	return ""
}
