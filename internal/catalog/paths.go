package catalog

import (
	"os"
	"path/filepath"
)

// OrgConfigPath returns the expected path for the user-level defaults file.
// Priority: $QSDEV_ORG_CONFIG > ~/.config/qsdev/defaults.yaml
func OrgConfigPath() string {
	if p := os.Getenv("QSDEV_ORG_CONFIG"); p != "" {
		return p
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(home, ".config", "qsdev", "defaults.yaml")
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
	return filepath.Join(projectRoot, ".qsdev", "defaults.yaml")
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
