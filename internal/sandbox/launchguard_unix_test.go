//go:build !windows

package sandbox

import (
	"errors"
	"os"
	"syscall"
	"testing"
	"time"
)

// TestLaunchGuard_LingeringWriterBounded: a process that still holds a copy
// of the write end (a grandchild that inherited it) must not hang Started.
// The bound is a real deadline, so wall time is what it is checked against.
func TestLaunchGuard_LingeringWriterBounded(t *testing.T) {
	t.Parallel()
	g := newTestGuard(t)
	fd, err := syscall.Dup(int(g.ChildFile().Fd()))
	if err != nil {
		t.Fatal(err)
	}
	lingering := os.NewFile(uintptr(fd), "lingering")
	t.Cleanup(func() { _ = lingering.Close() })

	start := time.Now()
	if g.Started() {
		t.Error("Started = true with nothing written")
	}
	if elapsed := time.Since(start); elapsed > launchGuardWait+5*time.Second {
		t.Errorf("Started took %v with a lingering writer, want about %v", elapsed, launchGuardWait)
	}
}

// newBlockingPipeGuard builds a guard over a blocking pipe, which, like an
// os.Pipe on Windows, has no read deadline (os.ErrNoDeadline), so Started
// takes its goroutine path on Linux too.
func newBlockingPipeGuard(t *testing.T) *LaunchGuard {
	t.Helper()
	var p [2]int
	if err := syscall.Pipe(p[:]); err != nil {
		t.Fatal(err)
	}
	r, w := os.NewFile(uintptr(p[0]), "r"), os.NewFile(uintptr(p[1]), "w")
	if err := r.SetReadDeadline(time.Now()); !errors.Is(err, os.ErrNoDeadline) {
		t.Fatalf("SetReadDeadline on a blocking pipe = %v, want os.ErrNoDeadline", err)
	}
	g := newLaunchGuard(r, w)
	t.Cleanup(func() { _ = g.Close() })
	return g
}

// TestLaunchGuard_NoDeadline: on a pipe without read deadlines Started still
// sees the ready byte, keeps its answer on a second call, and stays bounded
// when a lingering process holds the write end.
func TestLaunchGuard_NoDeadline(t *testing.T) {
	t.Parallel()

	t.Run("started", func(t *testing.T) {
		t.Parallel()
		g := newBlockingPipeGuard(t)
		if _, err := g.ChildFile().Write([]byte{'R'}); err != nil {
			t.Fatal(err)
		}
		if !g.Started() {
			t.Error("Started = false after the ready byte was written")
		}
		if !g.Started() {
			t.Error("second Started = false after the ready byte was written")
		}
	})

	t.Run("not started", func(t *testing.T) {
		t.Parallel()
		g := newBlockingPipeGuard(t)
		if g.Started() {
			t.Error("Started = true with nothing written")
		}
	})

	t.Run("lingering writer", func(t *testing.T) {
		t.Parallel()
		g := newBlockingPipeGuard(t)
		fd, err := syscall.Dup(int(g.ChildFile().Fd()))
		if err != nil {
			t.Fatal(err)
		}
		lingering := os.NewFile(uintptr(fd), "lingering")
		// Registered after the guard's Close, so it runs first and ends the
		// pending read with EOF.
		t.Cleanup(func() { _ = lingering.Close() })

		start := time.Now()
		for i := range 2 {
			if g.Started() {
				t.Errorf("Started call %d = true with nothing written", i+1)
			}
		}
		if elapsed := time.Since(start); elapsed > 2*launchGuardWait+5*time.Second {
			t.Errorf("two Started calls took %v with a lingering writer, want about %v", elapsed, 2*launchGuardWait)
		}
	})
}
