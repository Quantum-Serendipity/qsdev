//go:build windows

package fileutil

import (
	"os"
	"path/filepath"
)

// renameWithRetry uses os.Root.Rename which internally calls NtSetInformationFile
// with FILE_RENAME_POSIX_SEMANTICS on Windows 10+. This allows renaming over a
// file even when readers hold it open (like POSIX). Falls back to non-POSIX
// rename on filesystems that don't support it (e.g., ReFS, FAT32).
func renameWithRetry(oldpath, newpath string) error {
	dir := filepath.Dir(newpath)
	root, err := os.OpenRoot(dir)
	if err != nil {
		return os.Rename(oldpath, newpath)
	}
	defer root.Close()

	return root.Rename(filepath.Base(oldpath), filepath.Base(newpath))
}
