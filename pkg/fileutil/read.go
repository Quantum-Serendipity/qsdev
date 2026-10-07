package fileutil

import (
	"errors"
	"fmt"
	"io"
	"os"
)

var (
	// ErrNotRegular reports that ReadRegularFile's path is not a regular file
	// (a directory, FIFO, device or socket) after following symlinks.
	ErrNotRegular = errors.New("not a regular file")
	// ErrTooLarge reports that ReadRegularFile's file exceeds its limit.
	ErrTooLarge = errors.New("too large")
)

// ReadRegularFile reads the regular file at path, following symlinks, and
// returns at most limit bytes. It stats before opening, so a FIFO, device or
// socket is rejected with ErrNotRegular without being opened (opening a FIFO
// with no writer blocks forever), and rechecks the opened file in case the
// path was swapped in between. A file longer than limit returns ErrTooLarge
// after reading at most limit+1 bytes. A missing file wraps fs.ErrNotExist.
func ReadRegularFile(path string, limit int64) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("reading %s: %w", path, ErrNotRegular)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	if info, err = f.Stat(); err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("reading %s: %w", path, ErrNotRegular)
	}
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("reading %s: over %d bytes: %w", path, limit, ErrTooLarge)
	}
	return data, nil
}
