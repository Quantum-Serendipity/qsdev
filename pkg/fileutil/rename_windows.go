//go:build windows

package fileutil

import (
	"errors"
	"math/rand/v2"
	"os"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// fileRenameInfo is the variable-length FILE_RENAME_INFO struct used with
// SetFileInformationByHandle(FileRenameInfoEx). The Flags field (instead of
// ReplaceIfExists) is only valid with the Ex class.
type fileRenameInfo struct {
	Flags          uint32
	RootDirectory  syscall.Handle
	FileNameLength uint32
	FileName       [syscall.MAX_PATH]uint16
}

func renameWithRetry(oldpath, newpath string) error {
	// Try POSIX-semantics rename (Windows 10 1607+, NTFS). Succeeds even
	// when the target has open reader handles.
	err := posixRename(oldpath, newpath)
	if err == nil {
		return nil
	}
	// Fall back to retry-based os.Rename for older systems or non-NTFS
	// filesystems that don't support POSIX semantics.
	return retryRename(oldpath, newpath)
}

func posixRename(oldpath, newpath string) error {
	oldp, err := syscall.UTF16PtrFromString(oldpath)
	if err != nil {
		return err
	}

	// Open source with DELETE access and full sharing so it can be renamed.
	h, err := windows.CreateFile(
		oldp,
		windows.DELETE|windows.SYNCHRONIZE,
		windows.FILE_SHARE_DELETE|windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT,
		0,
	)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(h)

	// Resolve to absolute path for the rename target.
	absNew, err := filepath.Abs(newpath)
	if err != nil {
		return err
	}
	newp, err := syscall.UTF16FromString(absNew)
	if err != nil {
		return err
	}
	if len(newp) > syscall.MAX_PATH {
		return syscall.EINVAL
	}

	info := fileRenameInfo{
		Flags:          windows.FILE_RENAME_REPLACE_IF_EXISTS | windows.FILE_RENAME_POSIX_SEMANTICS,
		RootDirectory:  0,
		FileNameLength: uint32((len(newp) - 1) * 2), // byte count, excluding NUL
	}
	copy(info.FileName[:], newp)

	return windows.SetFileInformationByHandle(
		h,
		windows.FileRenameInfoEx,
		(*byte)(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
	)
}

const retryTimeout = 2 * time.Second

func retryRename(oldpath, newpath string) error {
	var (
		deadline  = time.Now().Add(retryTimeout)
		nextSleep = 1 * time.Millisecond
	)

	for {
		err := os.Rename(oldpath, newpath)
		if err == nil {
			return nil
		}
		if !isRetryableError(err) || time.Now().After(deadline) {
			return err
		}

		sleep := nextSleep
		if sleep > 500*time.Millisecond {
			sleep = 500 * time.Millisecond
		}
		time.Sleep(sleep + time.Duration(rand.Int64N(int64(sleep))))
		nextSleep *= 2
	}
}

const errorSharingViolation = syscall.Errno(32)

func isRetryableError(err error) bool {
	var errno syscall.Errno
	if errors.As(err, &errno) {
		return errno == syscall.ERROR_ACCESS_DENIED || errno == errorSharingViolation
	}
	return false
}
