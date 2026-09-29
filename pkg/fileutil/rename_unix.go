//go:build !windows

package fileutil

import "os"

func renameWithRetry(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}

// syncDir fsyncs dir so a completed rename survives power loss. It is best
// effort: some filesystems do not support syncing directories.
func syncDir(dir string) {
	d, err := os.Open(dir)
	if err != nil {
		return
	}
	_ = d.Sync()
	_ = d.Close()
}
