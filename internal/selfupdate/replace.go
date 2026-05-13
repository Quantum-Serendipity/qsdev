package selfupdate

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// DoUpdate downloads, verifies, and replaces the current binary with the
// new version from the given release. It implements a safe replacement
// strategy with rollback on failure:
//
//  1. Download and verify the new binary
//  2. Rename current binary to .bak
//  3. Copy new binary to original path
//  4. Verify the new binary executes successfully
//  5. Remove .bak on success, or restore it on failure
func DoUpdate(ctx context.Context, cfg Config, release *Release) error {
	// Find current binary path.
	currentPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding current binary: %w", err)
	}

	// Resolve symlinks to get the actual file path.
	currentPath, err = resolveExecutable(currentPath)
	if err != nil {
		return fmt.Errorf("resolving binary path: %w", err)
	}

	// Get the current file's permissions.
	currentInfo, err := os.Stat(currentPath)
	if err != nil {
		return fmt.Errorf("stating current binary: %w", err)
	}
	currentMode := currentInfo.Mode()

	// Download and verify the new binary to a temp directory.
	tmpDir, err := os.MkdirTemp("", "qsdev-update-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	newBinaryPath, err := DownloadAndVerify(ctx, release, cfg, runtime.GOOS, runtime.GOARCH, tmpDir)
	if err != nil {
		return fmt.Errorf("downloading update: %w", err)
	}

	// Clean up stale backup from previous update (Windows can't delete running binaries).
	backupPath := currentPath + ".bak"
	if _, statErr := os.Stat(backupPath); statErr == nil {
		os.Remove(backupPath)
	}

	// Create backup.
	if err := os.Rename(currentPath, backupPath); err != nil {
		return fmt.Errorf("creating backup: %w", err)
	}

	// Copy new binary to original path.
	if err := copyFile(newBinaryPath, currentPath, currentMode); err != nil {
		// Restore backup on copy failure.
		_ = os.Rename(backupPath, currentPath)
		return fmt.Errorf("installing new binary: %w", err)
	}

	// Verify the new binary runs.
	if err := verifyBinary(ctx, currentPath); err != nil {
		// Restore backup on verification failure.
		_ = os.Remove(currentPath)
		if restoreErr := os.Rename(backupPath, currentPath); restoreErr != nil {
			return fmt.Errorf("verification failed (%w) and restore also failed: %v", err, restoreErr)
		}
		return fmt.Errorf("new binary verification failed (restored previous version): %w", err)
	}

	// Success — remove backup.
	_ = os.Remove(backupPath)

	// Print changelog summary.
	fmt.Fprintf(os.Stderr, "Successfully updated to %s\n", release.Version)
	if release.Body != "" {
		summary := truncateChangelog(release.Body, 10)
		fmt.Fprintf(os.Stderr, "\nChangelog:\n%s\n", summary)
	}
	if release.URL != "" {
		fmt.Fprintf(os.Stderr, "\nRelease: %s\n", release.URL)
	}

	return nil
}

// resolveExecutable resolves symlinks for the executable path.
func resolveExecutable(path string) (string, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		// If resolution fails, fall back to original path.
		return path, nil
	}
	return resolved, nil
}

// copyFile copies src to dst with the given permissions.
func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

// verifyBinary runs the binary with "version" to confirm it executes.
func verifyBinary(ctx context.Context, binaryPath string) error {
	cmd := exec.CommandContext(ctx, binaryPath, "version")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run()
}

// truncateChangelog returns at most maxLines lines from the changelog.
func truncateChangelog(body string, maxLines int) string {
	lines := splitLines(body)
	if len(lines) <= maxLines {
		return body
	}
	result := make([]byte, 0, 512)
	for i := 0; i < maxLines; i++ {
		result = append(result, lines[i]...)
		result = append(result, '\n')
	}
	result = append(result, "  ...\n"...)
	return string(result)
}

// splitLines splits text into individual lines.
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
