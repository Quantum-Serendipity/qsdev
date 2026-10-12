package cmdutil

import (
	"errors"
	"fmt"
	"io"
	"os"
)

// console is the process's console: input from CONIN$, output to CONOUT$.
type console struct {
	in, out *os.File
}

func (c console) Read(p []byte) (int, error)  { return c.in.Read(p) }
func (c console) Write(p []byte) (int, error) { return c.out.Write(p) }
func (c console) Close() error                { return errors.Join(c.in.Close(), c.out.Close()) }

// openControllingTTY opens the process's console. It fails for a process
// without one (a detached or service process).
func openControllingTTY() (io.ReadWriteCloser, error) {
	in, err := os.OpenFile("CONIN$", os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("opening CONIN$: %w", err)
	}
	out, err := os.OpenFile("CONOUT$", os.O_RDWR, 0)
	if err != nil {
		_ = in.Close()
		return nil, fmt.Errorf("opening CONOUT$: %w", err)
	}
	return console{in: in, out: out}, nil
}
