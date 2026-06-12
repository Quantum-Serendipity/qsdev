// Package capture provides a CaptureWriter that tees tool output to a
// capture file alongside the original writer. This preserves ephemeral
// stderr/stdout from external tools for later inclusion in bug reports.
package capture

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

// CaptureWriter tees output to a capture file alongside the original writer.
type CaptureWriter struct {
	original io.Writer
	file     *os.File
	multi    io.Writer
}

// New creates a CaptureWriter that writes to both original and a new file
// in captureDir named "{provider}-{timestamp}.log".
func New(original io.Writer, captureDir, provider string) (*CaptureWriter, error) {
	if err := os.MkdirAll(captureDir, fileutil.ModeDirDefault); err != nil {
		return nil, fmt.Errorf("creating capture dir: %w", err)
	}

	filename := fmt.Sprintf("%s-%s.log", provider, time.Now().Format("2006-01-02T15-04-05"))
	path := filepath.Join(captureDir, filename)

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, fileutil.ModeReadWrite)
	if err != nil {
		return nil, fmt.Errorf("opening capture file: %w", err)
	}

	return &CaptureWriter{
		original: original,
		file:     f,
		multi:    io.MultiWriter(original, f),
	}, nil
}

// Write implements io.Writer, writing to both the original and capture file.
func (w *CaptureWriter) Write(p []byte) (int, error) {
	return w.multi.Write(p)
}

// Close closes the capture file. The original writer is not closed.
func (w *CaptureWriter) Close() error {
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

// Path returns the capture file path.
func (w *CaptureWriter) Path() string {
	if w.file != nil {
		return w.file.Name()
	}
	return ""
}

// CaptureDir returns the appropriate capture directory for the current context.
// Uses .<appname>/logs/capture/ in a project, or ~/.<appname>/logs/capture/ globally.
func CaptureDir(projectRoot string) string {
	dotDir := "." + branding.Get().AppName
	if projectRoot != "" {
		return filepath.Join(projectRoot, dotDir, "logs", "capture")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}
	return filepath.Join(home, dotDir, "logs", "capture")
}
