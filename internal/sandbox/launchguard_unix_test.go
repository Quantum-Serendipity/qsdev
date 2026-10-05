//go:build !windows

package sandbox

import (
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
