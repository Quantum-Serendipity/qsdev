package fileutil

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// WriteFileAtomic writes content to path atomically by creating a temporary
// file in the same directory (guaranteeing same-filesystem rename), writing
// the content, setting permissions, and renaming into place. Parent directories
// are created as needed. On failure, the temporary file is cleaned up.
func WriteFileAtomic(path string, content []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create parent directories for %s: %w", path, err)
	}

	tmp, err := os.CreateTemp(dir, branding.Get().TempPrefix)
	if err != nil {
		return fmt.Errorf("create temp file in %s: %w", dir, err)
	}

	success := false
	defer func() {
		if !success {
			_ = tmp.Close()
			_ = os.Remove(tmp.Name())
		}
	}()

	if _, err := tmp.Write(content); err != nil {
		return fmt.Errorf("write to temp file %s: %w", tmp.Name(), err)
	}

	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("sync temp file %s: %w", tmp.Name(), err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file %s: %w", tmp.Name(), err)
	}

	if err := os.Chmod(tmp.Name(), mode); err != nil {
		return fmt.Errorf("chmod temp file %s: %w", tmp.Name(), err)
	}

	if err := renameWithRetry(tmp.Name(), path); err != nil {
		return fmt.Errorf("rename %s to %s: %w", tmp.Name(), path, err)
	}

	success = true
	return nil
}
