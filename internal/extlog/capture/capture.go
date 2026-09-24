// Package capture provides a CaptureWriter that tees tool output to a
// capture file alongside the original writer. This preserves ephemeral
// stderr/stdout from external tools for later inclusion in bug reports.
package capture

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

const (
	// maxCapturesPerProvider bounds how many capture files are kept for one
	// provider; the oldest beyond it are removed when a new capture starts.
	maxCapturesPerProvider = 20
	// maxCaptureAge is how long a capture file is kept at all.
	maxCaptureAge = 30 * 24 * time.Hour
	// modeDirPrivate keeps the capture directory owner-only: captured tool
	// output can contain environment details and, before scrubbing, secrets.
	modeDirPrivate os.FileMode = 0o700
)

// CaptureWriter tees output to a capture file alongside the original writer.
type CaptureWriter struct {
	original io.Writer
	file     *os.File
	sink     *fileSink
	multi    io.Writer
}

// fileSink writes to the capture file best-effort. Capture is a diagnostic
// side channel, so a failed write (a full disk, say) must never surface as an
// error of the command whose output is being captured; after the first
// failure the sink stops writing. It is safe for the concurrent stdout and
// stderr copiers of an exec.Cmd.
type fileSink struct {
	f      *os.File
	failed atomic.Bool
}

func (s *fileSink) Write(p []byte) (int, error) {
	if !s.failed.Load() {
		if _, err := s.f.Write(p); err != nil {
			s.failed.Store(true)
			slog.Debug("capture write failed; capture stopped", "path", s.f.Name(), "error", err)
		}
	}
	return len(p), nil
}

// New creates a CaptureWriter that writes to both original and a new,
// owner-only file in captureDir named "{provider}-{timestamp}-{pid}.log".
// Older captures of the same provider beyond the retention limits are pruned.
func New(original io.Writer, captureDir, provider string) (*CaptureWriter, error) {
	if captureDir == "" {
		return nil, errors.New("creating capture file: no capture directory available")
	}
	if err := os.MkdirAll(filepath.Dir(captureDir), fileutil.ModeDirDefault); err != nil {
		return nil, fmt.Errorf("creating capture dir parent: %w", err)
	}
	if err := os.Mkdir(captureDir, modeDirPrivate); err != nil && !errors.Is(err, fs.ErrExist) {
		return nil, fmt.Errorf("creating capture dir: %w", err)
	}

	filename := fmt.Sprintf("%s-%s-%d.log", provider, time.Now().Format("2006-01-02T15-04-05"), os.Getpid())
	path := filepath.Join(captureDir, filename)

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, fileutil.ModePrivate)
	if err != nil {
		return nil, fmt.Errorf("opening capture file: %w", err)
	}

	prune(captureDir, provider, filename)

	sink := &fileSink{f: f}
	return &CaptureWriter{
		original: original,
		file:     f,
		sink:     sink,
		multi:    io.MultiWriter(original, sink),
	}, nil
}

// Write implements io.Writer, writing to both the original and capture file.
// Only an error from the original writer is returned.
func (w *CaptureWriter) Write(p []byte) (int, error) {
	return w.multi.Write(p)
}

// Tee returns a writer that sends a second stream (typically stderr, when the
// CaptureWriter wraps stdout) to other and to the same capture file, so a
// command's interleaved output is captured in one place.
func (w *CaptureWriter) Tee(other io.Writer) io.Writer {
	return io.MultiWriter(other, w.sink)
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
// Uses .<appname>/logs/capture/ in a project, or ~/.<appname>/logs/capture/
// globally. It returns "" when there is no project and no home directory,
// rather than a predictable path under the shared temp directory.
func CaptureDir(projectRoot string) string {
	dotDir := "." + branding.Get().AppName
	if projectRoot != "" {
		return filepath.Join(projectRoot, dotDir, "logs", "capture")
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, dotDir, "logs", "capture")
}

// prune removes provider's capture files in dir that are older than
// maxCaptureAge or beyond the newest maxCapturesPerProvider, never touching
// keep (the capture just opened). Failures are logged and otherwise ignored:
// retention is housekeeping and must not fail the captured command.
func prune(dir, provider, keep string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	type capture struct {
		name    string
		modTime time.Time
	}
	var captures []capture
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || name == keep || !strings.HasPrefix(name, provider+"-") || !strings.HasSuffix(name, ".log") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		captures = append(captures, capture{name: name, modTime: info.ModTime()})
	}
	sort.Slice(captures, func(i, j int) bool { return captures[i].modTime.After(captures[j].modTime) })

	cutoff := time.Now().Add(-maxCaptureAge)
	for i, c := range captures {
		// The new capture occupies one of the maxCapturesPerProvider slots.
		if i < maxCapturesPerProvider-1 && c.modTime.After(cutoff) {
			continue
		}
		path := filepath.Join(dir, c.name)
		if err := os.Remove(path); err != nil {
			slog.Debug("failed to prune capture file", "path", path, "error", err)
		}
	}
}
