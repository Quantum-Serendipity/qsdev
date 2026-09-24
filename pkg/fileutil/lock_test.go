package fileutil

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestLockExclusive_SerializesHolders checks that the lock admits one holder at
// a time: each goroutine stays inside the lock for a moment and records
// whether another holder was inside with it.
func TestLockExclusive_SerializesHolders(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "nested", "state.lock")
	const workers = 16

	var (
		wg      sync.WaitGroup
		inside  int
		mu      sync.Mutex // guards inside only, to detect overlap
		overlap bool
	)
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			lock, err := LockExclusive(context.Background(), path)
			if err != nil {
				t.Errorf("LockExclusive: %v", err)
				return
			}
			mu.Lock()
			inside++
			if inside > 1 {
				overlap = true
			}
			mu.Unlock()

			time.Sleep(time.Millisecond)

			mu.Lock()
			inside--
			mu.Unlock()
			if err := lock.Unlock(); err != nil {
				t.Errorf("Unlock: %v", err)
			}
		}()
	}
	wg.Wait()

	if overlap {
		t.Error("two holders held the lock at the same time")
	}
}

// TestLockExclusive_WaitIsBounded checks that a waiter gives up with the
// context's error while another holder keeps the lock, and succeeds once it is
// released.
func TestLockExclusive_WaitIsBounded(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "state.lock")
	held, err := LockExclusive(context.Background(), path)
	if err != nil {
		t.Fatalf("LockExclusive: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if lock, err := LockExclusive(ctx, path); !errors.Is(err, context.DeadlineExceeded) {
		if lock != nil {
			_ = lock.Unlock()
		}
		t.Fatalf("LockExclusive while held: err = %v, want context.DeadlineExceeded", err)
	}

	if err := held.Unlock(); err != nil {
		t.Fatalf("Unlock: %v", err)
	}
	lock, err := LockExclusive(context.Background(), path)
	if err != nil {
		t.Fatalf("LockExclusive after release: %v", err)
	}
	if err := lock.Unlock(); err != nil {
		t.Fatalf("Unlock: %v", err)
	}
}
