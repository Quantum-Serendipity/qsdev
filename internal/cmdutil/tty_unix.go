//go:build !windows

package cmdutil

import (
	"fmt"
	"io"
	"os"
)

// openControllingTTY opens /dev/tty, the process's controlling terminal. It
// fails for a process without one (a detached session, setsid).
func openControllingTTY() (io.ReadWriteCloser, error) {
	f, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("opening /dev/tty: %w", err)
	}
	return f, nil
}
