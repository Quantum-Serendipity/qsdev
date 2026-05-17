//go:build windows

package fileutil

import (
	"errors"
	"math/rand/v2"
	"os"
	"syscall"
	"time"
)

// renameWithRetry retries os.Rename on Windows when the target file has open
// handles, causing transient ACCESS_DENIED or SHARING_VIOLATION errors.
// Mirrors the strategy in Go's cmd/go/internal/robustio.
func renameWithRetry(oldpath, newpath string) error {
	const timeout = 2 * time.Second

	var (
		deadline  = time.Now().Add(timeout)
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
