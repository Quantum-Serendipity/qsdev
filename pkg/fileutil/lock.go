package fileutil

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// lockPollInterval is how often LockExclusive retries a lock another holder
// has.
const lockPollInterval = 10 * time.Millisecond

// errLockHeld is returned by tryLockFile when another holder has the lock.
var errLockHeld = errors.New("lock held by another holder")

// FileLock is an exclusive, advisory, cross-process lock held on a lock file.
// Every process that reads, modifies and rewrites a shared file must take the
// same lock around the whole sequence; a reader that only reads a file
// replaced with WriteFileAtomic does not need it.
type FileLock struct {
	f *os.File
}

// LockExclusive opens (creating it and its parent directories when needed)
// the lock file at path and waits until this process holds an exclusive lock
// on it or ctx is done. The wait is bounded by ctx so a holder that never
// lets go (a hung process, or one deliberately sitting on the lock) turns into
// an error the caller can fail closed on, instead of a hang that outlives the
// caller's own deadline. The lock is released by Unlock or when the process
// exits.
func LockExclusive(ctx context.Context, path string) (*FileLock, error) {
	if err := os.MkdirAll(filepath.Dir(path), ModeDirDefault); err != nil {
		return nil, fmt.Errorf("creating lock directory: %w", err)
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, ModePrivate) //nolint:gosec // lock path chosen by the caller.
	if err != nil {
		return nil, fmt.Errorf("opening lock file: %w", err)
	}
	if err := waitLock(ctx, f); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("locking %s: %w", path, err)
	}
	return &FileLock{f: f}, nil
}

// waitLock retries a non-blocking lock attempt until it succeeds, fails for a
// reason other than contention, or ctx is done.
func waitLock(ctx context.Context, f *os.File) error {
	ticker := time.NewTicker(lockPollInterval)
	defer ticker.Stop()
	for {
		err := tryLockFile(f)
		if !errors.Is(err, errLockHeld) {
			return err
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("waiting for lock: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

// Unlock releases the lock and closes the lock file. The lock file itself is
// left in place: removing it would let a waiter lock the unlinked file while a
// newcomer locks a fresh one.
func (l *FileLock) Unlock() error {
	unlockErr := unlockFile(l.f)
	closeErr := l.f.Close()
	if unlockErr != nil {
		return fmt.Errorf("unlocking %s: %w", l.f.Name(), unlockErr)
	}
	if closeErr != nil {
		return fmt.Errorf("closing lock file: %w", closeErr)
	}
	return nil
}
