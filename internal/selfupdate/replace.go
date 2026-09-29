package selfupdate

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// executablePath locates the running binary. It is a variable so tests can
// point DoUpdate at a scratch binary instead of the test executable.
var executablePath = os.Executable

// DoUpdate downloads, verifies, and replaces the current binary with the
// new version from the given release. The live binary is never missing or
// partially written (see replaceBinary):
//
//  1. Download and verify the new binary (checksum and signature)
//  2. Stage it in a synced temp file next to the current binary
//  3. Run the staged binary to confirm it executes
//  4. Atomically rename it over the current binary
func DoUpdate(ctx context.Context, cfg Config, release *Release) error {
	// Find current binary path.
	currentPath, err := executablePath()
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
	tmpDir, err := os.MkdirTemp("", branding.Get().AppName+"-update-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	newBinaryPath, err := DownloadAndVerify(ctx, release, cfg, runtime.GOOS, runtime.GOARCH, tmpDir)
	if err != nil {
		return fmt.Errorf("downloading update: %w", err)
	}

	if err := replaceBinary(ctx, currentPath, newBinaryPath, currentMode, verifyBinary); err != nil {
		return err
	}

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

// replaceBinary installs newBinaryPath at currentPath (with mode) so that the
// install path always holds a complete, working binary:
//
//   - the new binary is staged in a temp file in currentPath's directory (same
//     filesystem), fsynced and chmodded;
//   - verify runs against the STAGED file, before the live binary is touched,
//     so a broken download never replaces a working install;
//   - the staged file is renamed over currentPath, which is atomic on Unix
//     (running processes keep the old inode, new ones get the new file).
//
// Windows cannot rename over a running executable, so there the live binary is
// first moved aside to <path>.bak and restored if the final rename fails.
func replaceBinary(ctx context.Context, currentPath, newBinaryPath string, mode os.FileMode, verify func(context.Context, string) error) error {
	staged, err := stageBinary(currentPath, newBinaryPath, mode)
	if err != nil {
		return fmt.Errorf("installing new binary: %w", err)
	}
	installed := false
	defer func() {
		if !installed {
			_ = os.Remove(staged)
		}
	}()

	if err := verify(ctx, staged); err != nil {
		return fmt.Errorf("new binary verification failed (current version left in place): %w", err)
	}

	if err := swapBinary(staged, currentPath, runtime.GOOS == "windows"); err != nil {
		return fmt.Errorf("installing new binary: %w", err)
	}
	installed = true
	return nil
}

// stageBinary copies src into a new temp file beside currentPath, syncs it to
// disk, and applies mode. The temp name keeps currentPath's extension so the
// staged binary is directly executable on Windows. It returns the temp path.
func stageBinary(currentPath, src string, mode os.FileMode) (_ string, retErr error) {
	base := filepath.Base(currentPath)
	ext := filepath.Ext(base)
	tmp, err := os.CreateTemp(filepath.Dir(currentPath), "."+strings.TrimSuffix(base, ext)+".new-*"+ext)
	if err != nil {
		return "", fmt.Errorf("creating staging file: %w", err)
	}
	defer func() {
		if retErr != nil {
			_ = tmp.Close()
			_ = os.Remove(tmp.Name())
		}
	}()

	in, err := os.Open(src) //nolint:gosec // src is the verified binary extracted into our own temp dir.
	if err != nil {
		return "", fmt.Errorf("opening new binary: %w", err)
	}
	defer func() { _ = in.Close() }()

	if _, err := io.Copy(tmp, in); err != nil {
		return "", fmt.Errorf("writing staging file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return "", fmt.Errorf("syncing staging file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return "", fmt.Errorf("closing staging file: %w", err)
	}
	if err := os.Chmod(tmp.Name(), mode); err != nil {
		return "", fmt.Errorf("setting staging file mode: %w", err)
	}
	return tmp.Name(), nil
}

// swapBinary renames staged over currentPath. With viaBackup (Windows, where a
// running executable cannot be replaced), the live binary is first renamed to
// <currentPath>.bak and restored if the final rename fails; the .bak is left
// behind because a running executable cannot be deleted, and is removed by the
// next update.
func swapBinary(staged, currentPath string, viaBackup bool) error {
	backupPath := currentPath + ".bak"
	// Clean up a stale backup from a previous update.
	if err := os.Remove(backupPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		slog.Warn("removing stale backup", "path", backupPath, "error", err)
	}

	if !viaBackup {
		return os.Rename(staged, currentPath)
	}

	if err := os.Rename(currentPath, backupPath); err != nil {
		return fmt.Errorf("moving current binary aside: %w", err)
	}
	if err := os.Rename(staged, currentPath); err != nil {
		if restoreErr := os.Rename(backupPath, currentPath); restoreErr != nil {
			return fmt.Errorf("install failed and restoring the previous binary also failed: %w", errors.Join(err, restoreErr))
		}
		return err
	}
	if err := os.Remove(backupPath); err != nil {
		slog.Debug("backup of running binary left for the next update", "path", backupPath, "error", err)
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
