package projectctx

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// logsDirName is the name of the session-log directory below Legacy and
// State (Dirs.Logs).
const logsDirName = "logs"

// modeStateParent is the mode of a state directory MigrateLegacyLogs creates
// to hold the moved logs; the logs directory keeps its own (private) mode.
// It matches pkg/fileutil.ModeDirDefault, which this leaf package cannot
// import.
const modeStateParent os.FileMode = 0o755

// MigrateLegacyLogs moves the global session logs from their historical
// location, Legacy/logs, to State/logs, and reports whether it moved any.
//
// It is a best-effort, one-time move that never copies and never overwrites.
// It does nothing when Legacy/logs is not a directory (missing, a file or a
// symlink). When State/logs does not exist, Legacy/logs is renamed to it.
// When it does, as it will once a hook or MCP server process (which never
// migrates) has written a session there first, the legacy entries are moved
// into it one by one (mergeDir), and Legacy/logs is removed once empty. A
// rename that fails, for example across filesystems, leaves the entry where it
// is and is reported in the returned error.
func MigrateLegacyLogs(d Dirs) (bool, error) {
	legacy := filepath.Join(d.Legacy, logsDirName)
	target := d.Logs()

	info, err := os.Lstat(legacy)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("inspecting legacy logs %s: %w", legacy, err)
	}
	if !info.IsDir() {
		return false, nil
	}
	targetInfo, err := os.Lstat(target)
	switch {
	case err == nil && targetInfo.IsDir():
		return mergeDir(legacy, target)
	case err == nil:
		return false, nil // not a directory: leave both alone
	case !errors.Is(err, fs.ErrNotExist):
		return false, fmt.Errorf("inspecting log directory %s: %w", target, err)
	}

	if err := os.MkdirAll(d.State, modeStateParent); err != nil {
		return false, fmt.Errorf("creating state directory %s: %w", d.State, err)
	}
	if err := os.Rename(legacy, target); err != nil {
		return false, fmt.Errorf("moving legacy logs %s to %s: %w", legacy, target, err)
	}
	return true, nil
}

// mergeDir moves each entry of src that has no namesake in dst into dst,
// descending into directories present in both (such as the automated
// sub-tier), then removes src if that left it empty. An entry whose name is
// taken in dst stays in src: nothing is overwritten. Session log names carry
// a timestamp and a random suffix, so a collision means the same file, and
// the window between the existence check and the rename cannot be raced into
// overwriting a session written meanwhile. It reports whether it moved
// anything; per-entry failures are joined into the error and do not stop the
// other entries.
func mergeDir(src, dst string) (bool, error) {
	entries, err := os.ReadDir(src)
	if err != nil {
		return false, fmt.Errorf("reading legacy logs %s: %w", src, err)
	}
	moved := false
	var errs []error
	for _, e := range entries {
		from, to := filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())
		toInfo, err := os.Lstat(to)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			if err := os.Rename(from, to); err != nil {
				errs = append(errs, fmt.Errorf("moving legacy log %s to %s: %w", from, to, err))
				continue
			}
			moved = true
		case err != nil:
			errs = append(errs, fmt.Errorf("inspecting log %s: %w", to, err))
		case e.IsDir() && toInfo.IsDir():
			sub, err := mergeDir(from, to)
			moved = moved || sub
			if err != nil {
				errs = append(errs, err)
			}
		}
	}
	// Fails, harmlessly, when src still holds a colliding entry.
	_ = os.Remove(src)
	return moved, errors.Join(errs...)
}
