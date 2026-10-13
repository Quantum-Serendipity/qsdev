//go:build unix

package devinit

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/hookio"
)

// mkfifo creates a FIFO at path. Opening it with no writer blocks forever.
func mkfifo(t *testing.T, path string) {
	t.Helper()
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Fatalf("mkfifo: %v", err)
	}
}

// TestDetectResultGateDodge_FIFOTargetDenies pins U05-14: a guarded .npmrc
// that is a symlink to a FIFO is denied at once as not a regular file, rather
// than blocking on open until the SP-TIMEOUT backstop fires. The bound is
// wall time against a real deadline (the open would block forever).
func TestDetectResultGateDodge_FIFOTargetDenies(t *testing.T) {
	dir := t.TempDir()
	fifo := filepath.Join(dir, "pipe")
	mkfifo(t, fifo)
	npmrc := filepath.Join(dir, ".npmrc")
	if err := os.Symlink(fifo, npmrc); err != nil {
		t.Fatal(err)
	}
	input := hookio.ToolInput{FilePath: npmrc, Content: "registry=https://x\n"}

	type verdict struct {
		blocked        bool
		ruleID, reason string
	}
	done := make(chan verdict, 1)
	go func() {
		b, id, r := detectResultGateDodge("Write", input, npmrc)
		done <- verdict{b, id, r}
	}()
	select {
	case v := <-done:
		if !v.blocked || v.ruleID != "GD-004" || !strings.Contains(v.reason, "not a regular file") {
			t.Errorf("detectResultGateDodge = (%v, %q, %q), want GD-004 deny naming 'not a regular file'", v.blocked, v.ruleID, v.reason)
		}
	case <-time.After(100 * time.Millisecond * raceScale):
		t.Fatal("detectResultGateDodge blocked on the FIFO")
	}

	t.Run("end to end", func(t *testing.T) {
		assertSelfprotectDenies(t, writePayload(t, npmrc, "registry=https://x\n"), "GD-004")
	})
}

// TestSelfprotect_QsdevYAMLFIFODeniesGD001 pins that the GD-001 check
// (rules.FileChange) denies a FIFO .qsdev.yaml instead of hanging on it.
func TestSelfprotect_QsdevYAMLFIFODeniesGD001(t *testing.T) {
	target := filepath.Join(t.TempDir(), ".qsdev.yaml")
	mkfifo(t, target)
	assertSelfprotectDenies(t, writePayload(t, target, "security:\n  level: strict\n"), "GD-001")
}
