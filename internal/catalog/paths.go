package catalog

import (
	"os"
	"path/filepath"
)

// OrgConfigDir returns the directory for organization-level catalog overrides.
// Priority: $QSDEV_ORG_CONFIG > ~/.config/qsdev/catalog/
// Returns empty string if no override directory is configured or exists.
func OrgConfigDir() string {
	if dir := os.Getenv("QSDEV_ORG_CONFIG"); dir != "" {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	dir := filepath.Join(home, ".config", "qsdev", "catalog")
	if info, err := os.Stat(dir); err == nil && info.IsDir() {
		return dir
	}

	return ""
}

// ProjectConfigDir returns the directory for project-level catalog overrides.
// Looks for .qsdev/catalog/ relative to the given project root.
// Returns empty string if the directory does not exist.
func ProjectConfigDir(projectRoot string) string {
	if projectRoot == "" {
		return ""
	}

	dir := filepath.Join(projectRoot, ".qsdev", "catalog")
	if info, err := os.Stat(dir); err == nil && info.IsDir() {
		return dir
	}

	return ""
}
