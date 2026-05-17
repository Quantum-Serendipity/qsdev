package cmdutil

import (
	"fmt"
	"os"
	"path/filepath"
)

// CheckNotExists returns an error if the file at projectRoot/relPath exists,
// unless force is true.
func CheckNotExists(projectRoot, relPath string, force bool) error {
	if force {
		return nil
	}
	absPath := filepath.Join(projectRoot, relPath)
	if _, err := os.Stat(absPath); err == nil {
		return fmt.Errorf("%s already exists; use --force to overwrite", relPath)
	}
	return nil
}
