//go:build unix

package rules

import (
	"errors"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

// TestGuardedReads_FIFO pins that the guarded-file readers never open a FIFO,
// which blocks forever with no writer: FileChange errors (callers deny) and
// hookCommands registers no hooks. A hang here is the failure.
func TestGuardedReads_FIFO(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	fifo := filepath.Join(dir, "fifo")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Fatalf("mkfifo: %v", err)
	}

	ctx := EvalContext{ToolName: "Write", CanonicalPath: fifo, Content: "x"}
	if _, _, err := ctx.FileChange(); !errors.Is(err, fileutil.ErrNotRegular) {
		t.Errorf("FileChange(fifo) error = %v, want ErrNotRegular", err)
	}
	if got := hookCommands(fifo); got != nil {
		t.Errorf("hookCommands(fifo) = %q, want none", got)
	}
}
