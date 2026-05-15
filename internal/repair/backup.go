package repair

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// backupDir returns the path to the backup directory within a project.
func backupDir(projectRoot string) string {
	return filepath.Join(projectRoot, ".qsdev", "backups")
}

// createBackup copies the file at projectRoot/relPath to
// .qsdev/backups/<basename>.<20060102T150405>.bak. It creates the backup
// directory if it does not exist. Returns the full backup path.
func createBackup(projectRoot, relPath string) (string, error) {
	srcPath := filepath.Join(projectRoot, relPath)

	// Verify source exists.
	if _, err := os.Stat(srcPath); err != nil {
		return "", fmt.Errorf("backup source %s: %w", srcPath, err)
	}

	dir := backupDir(projectRoot)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating backup dir %s: %w", dir, err)
	}

	base := filepath.Base(relPath)
	timestamp := time.Now().Format("20060102T150405")
	backupName := fmt.Sprintf("%s.%s.bak", base, timestamp)
	backupPath := filepath.Join(dir, backupName)

	if err := copyFile(srcPath, backupPath); err != nil {
		return "", fmt.Errorf("copying %s to %s: %w", srcPath, backupPath, err)
	}

	return backupPath, nil
}

// pruneBackups keeps only the most recent `keep` backups matching the base
// filename pattern and removes older ones. It sorts by the embedded timestamp
// in the filename.
func pruneBackups(projectRoot, relPath string, keep int) error {
	dir := backupDir(projectRoot)
	base := filepath.Base(relPath)
	prefix := base + "."
	suffix := ".bak"

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No backup dir — nothing to prune.
		}
		return fmt.Errorf("listing backup dir %s: %w", dir, err)
	}

	// Collect matching backup filenames.
	var matches []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, suffix) {
			matches = append(matches, name)
		}
	}

	if len(matches) <= keep {
		return nil
	}

	// Sort lexicographically — since timestamps are YYYYMMDDTHHMMSS format,
	// lexicographic order equals chronological order.
	sort.Strings(matches)

	// Remove the oldest entries (those at the beginning of the sorted slice).
	toRemove := matches[:len(matches)-keep]
	for _, name := range toRemove {
		path := filepath.Join(dir, name)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing old backup %s: %w", path, err)
		}
	}

	return nil
}

// copyFile copies src to dst, preserving the original file permissions.
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}
