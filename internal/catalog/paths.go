package catalog

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// OrgConfigPath returns the expected path for the user-level defaults file.
// Priority: $QSDEV_ORG_CONFIG > ~/.config/qsdev/defaults.yaml
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

// homeOrgConfigPath returns ~/.config/<app>/defaults.yaml, or "" when the
// home directory cannot be determined.
func homeOrgConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(home, ".config", branding.Get().AppName, "defaults.yaml")
}

// OrgConfigFile returns the user-level defaults file path if it exists,
// or empty string if not.
func OrgConfigFile() string {
	p := OrgConfigPath()
	if p == "" {
		return ""
	}
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return ""
}

// ProjectConfigPath returns the expected path for a project-level defaults file.
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
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return ""
}
