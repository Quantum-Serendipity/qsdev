package fileutil

import (
	"fmt"
	"io"
	"os"
)

// CopyFile copies src to dst with the given permissions. If the copy or
// final close fails the destination file is removed on a best-effort basis.
func CopyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening source %s: %w", src, err)
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("creating destination %s: %w", dst, err)
	}

	if _, err := io.Copy(out, in); err != nil {
		if closeErr := out.Close(); closeErr != nil {
			os.Remove(dst)
			return fmt.Errorf("copying to %s: %w (also failed to close: %v)", dst, err, closeErr)
		}
		os.Remove(dst)
		return fmt.Errorf("copying to %s: %w", dst, err)
	}
	if err := out.Close(); err != nil {
		os.Remove(dst)
		return fmt.Errorf("closing %s: %w", dst, err)
	}
	return nil
}
