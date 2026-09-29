//go:build windows

package fileutil

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

// lockRange is the byte range locked; any fixed range works for an advisory
// whole-file lock as long as every locker uses the same one.
const lockRange = 1

// tryLockFile takes an exclusive lock without blocking, returning errLockHeld
// when another handle holds it.
func tryLockFile(f *os.File) error {
	ol := new(windows.Overlapped)
	err := windows.LockFileEx(windows.Handle(f.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, lockRange, 0, ol)
	if errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
		return errLockHeld
	}
	return err
}

func unlockFile(f *os.File) error {
	ol := new(windows.Overlapped)
	return windows.UnlockFileEx(windows.Handle(f.Fd()), 0, lockRange, 0, ol)
}
