//go:build unix

package fileutil_test

import (
	"errors"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

// TestReadRegularFile_FIFO pins that a FIFO is rejected without being opened:
// opening one with no writer blocks forever, so a hang here is the failure.
func TestReadRegularFile_FIFO(t *testing.T) {
	t.Parallel()
	fifo := filepath.Join(t.TempDir(), "fifo")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Fatalf("mkfifo: %v", err)
	}
	if _, err := fileutil.ReadRegularFile(fifo, 1024); !errors.Is(err, fileutil.ErrNotRegular) {
		t.Fatalf("ReadRegularFile(fifo) error = %v, want ErrNotRegular", err)
	}
}
