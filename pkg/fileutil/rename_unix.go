//go:build !windows

package fileutil

import "os"

func renameWithRetry(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}
