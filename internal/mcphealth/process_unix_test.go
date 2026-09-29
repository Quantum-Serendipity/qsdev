//go:build unix

package mcphealth

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestMCPProcessShutdown_KillsGrandchildren is the F256 regression. Launchers
// such as npx, uvx and sh run the real server as a child; killing only the
// launcher orphaned that child, and the stderr copy goroutine kept Close
// blocked until the orphan exited. Shutdown must be bounded and must kill the
// whole process group.
func TestMCPProcessShutdown_KillsGrandchildren(t *testing.T) {
	t.Parallel()

	pidFile := filepath.Join(t.TempDir(), "child.pid")
	script := "sleep 60 & echo $! > " + pidFile + "; sleep 60"

	p, err := startServer("sh", []string{"-c", script}, nil)
	if err != nil {
		t.Fatalf("startServer: %v", err)
	}

	childPID := waitForPIDFile(t, pidFile)

	const grace = 200 * time.Millisecond
	start := time.Now()
	_ = p.shutdown(grace)
	if elapsed := time.Since(start); elapsed > grace+5*time.Second {
		t.Fatalf("shutdown took %s (grace %s): orphaned grandchild kept it blocked", elapsed, grace)
	}

	deadline := time.Now().Add(5 * time.Second)
	for !processGone(childPID) {
		if time.Now().After(deadline) {
			t.Fatalf("grandchild %d survived shutdown", childPID)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// TestMCPProcessClose_Idempotent verifies Close can be called repeatedly (for
// example from both a timeout path and a deferred cleanup) without blocking.
func TestMCPProcessClose_Idempotent(t *testing.T) {
	t.Parallel()

	p, err := startServer("sh", []string{"-c", "cat >/dev/null"}, nil)
	if err != nil {
		t.Fatalf("startServer: %v", err)
	}
	first := p.Close()
	if second := p.Close(); second != first { //nolint:errorlint // identity: Close must return its memoised result
		t.Errorf("second Close = %v, want the first result %v", second, first)
	}
}

func waitForPIDFile(t *testing.T, path string) int {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		data, err := os.ReadFile(path)
		if err == nil {
			if pid, convErr := strconv.Atoi(strings.TrimSpace(string(data))); convErr == nil && pid > 0 {
				return pid
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("grandchild pid file %s never written", path)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// processGone reports whether pid no longer exists. A killed orphan may linger
// briefly as a zombie until its new parent reaps it, which also counts as gone.
func processGone(pid int) bool {
	if err := syscall.Kill(pid, 0); errors.Is(err, syscall.ESRCH) {
		return true
	}
	if runtime.GOOS != "linux" {
		return false
	}
	stat, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return true
	}
	// The state field follows the parenthesised command name.
	if i := strings.LastIndexByte(string(stat), ')'); i >= 0 && i+2 < len(stat) {
		return stat[i+2] == 'Z'
	}
	return false
}
