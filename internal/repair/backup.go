package repair

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	pkgfileutil "github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

// backupTimestampLayout is fixed-width, so lexicographic order of backup names
// equals chronological order. Nanosecond precision keeps backups of the same
// file taken within one second distinct.
const backupTimestampLayout = "20060102T150405.000000000"

// maxBackupAttempts bounds the unique-name retries when a backup name is
// already taken.
const maxBackupAttempts = 100

// backupSuffixRe matches the part of a backup filename between "<base>." and
// ".bak": a timestamp (second precision for backups made by older versions,
// nanosecond precision now) with an optional collision counter.
var backupSuffixRe = regexp.MustCompile(`^\d{8}T\d{6}(\.\d{9})?(-\d+)?$`)

// backupDir returns the path to the backup directory within a project.
func backupDir(projectRoot string) string {
	return filepath.Join(projectRoot, "."+branding.Get().AppName, "backups")
}

// backupLocation returns the directory that holds the backups of relPath and
// the filename prefix they share. The project-relative path is mirrored under
// the backup directory so files with the same basename in different
// directories never share (and overwrite) each other's backups.
func backupLocation(projectRoot, relPath string) (dir, prefix string, err error) {
	clean := filepath.Clean(filepath.FromSlash(relPath))
	if !filepath.IsLocal(clean) {
		return "", "", fmt.Errorf("backup path %q is not within the project", relPath)
	}
	mirrored := filepath.Join(backupDir(projectRoot), clean)
	return filepath.Dir(mirrored), filepath.Base(mirrored) + ".", nil
}

// createBackup copies the file at projectRoot/relPath to
// .qsdev/backups/<relPath>.<timestamp>.bak, creating directories as needed.
// An existing backup is never overwritten. Returns the full backup path.
func createBackup(projectRoot, relPath string) (string, error) {
	srcPath := filepath.Join(projectRoot, relPath)

	// Verify source exists and capture permissions.
	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		return "", fmt.Errorf("backup source %s: %w", srcPath, err)
	}

	dir, prefix, err := backupLocation(projectRoot, relPath)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, pkgfileutil.ModeDirDefault); err != nil {
		return "", fmt.Errorf("creating backup dir %s: %w", dir, err)
	}

	stamp := prefix + time.Now().Format(backupTimestampLayout)
	for attempt := range maxBackupAttempts {
		name := stamp + ".bak"
		if attempt > 0 {
			name = fmt.Sprintf("%s-%d.bak", stamp, attempt)
		}
		backupPath := filepath.Join(dir, name)

		err := fileutil.CopyFileExclusive(srcPath, backupPath, srcInfo.Mode())
		if err == nil {
			return backupPath, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return "", fmt.Errorf("copying %s to %s: %w", srcPath, backupPath, err)
		}
	}
	return "", fmt.Errorf("backing up %s: no unused backup name after %d attempts", relPath, maxBackupAttempts)
}

// pruneBackups keeps only the most recent `keep` backups of relPath and
// removes older ones. Only this file's backups are considered: other files,
// even with the same basename, have their own backups and budget.
func pruneBackups(projectRoot, relPath string, keep int) error {
	dir, prefix, err := backupLocation(projectRoot, relPath)
	if err != nil {
		return err
	}
	const suffix = ".bak"

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No backup dir — nothing to prune.
		}
		return fmt.Errorf("listing backup dir %s: %w", dir, err)
	}

	// Collect this file's backups.
	var matches []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, suffix) {
			continue
		}
		if backupSuffixRe.MatchString(strings.TrimSuffix(strings.TrimPrefix(name, prefix), suffix)) {
			matches = append(matches, name)
		}
	}

	if len(matches) <= keep {
		return nil
	}

	// Sort oldest first: the fixed-width timestamps order lexicographically,
	// and a collision counter orders after its uncountered original.
	sort.Slice(matches, func(i, j int) bool {
		ti, ci := backupSortKey(matches[i], prefix, suffix)
		tj, cj := backupSortKey(matches[j], prefix, suffix)
		if ti != tj {
			return ti < tj
		}
		return ci < cj
	})

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

// backupSortKey splits a backup filename into its timestamp and collision
// counter (0 when absent).
func backupSortKey(name, prefix, suffix string) (string, int) {
	stamp := strings.TrimSuffix(strings.TrimPrefix(name, prefix), suffix)
	ts, counter, found := strings.Cut(stamp, "-")
	if !found {
		return ts, 0
	}
	n, err := strconv.Atoi(counter)
	if err != nil {
		return ts, 0
	}
	return ts, n
}
