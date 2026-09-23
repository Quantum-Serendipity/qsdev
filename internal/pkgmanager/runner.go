package pkgmanager

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// errTailBytes bounds how much of a failed command's stderr is quoted in the
// returned error.
const errTailBytes = 2048

// ExecRunner implements CommandRunner using real os/exec calls.
//
// Run streams the command's stdout and stderr to Stdout and Stderr (a nil
// writer discards that stream) and, on failure, quotes the tail of stderr in
// the returned error so the cause is visible even when the output scrolled
// past. Stdin is not connected: installers run non-interactively.
type ExecRunner struct {
	Stdout io.Writer
	Stderr io.Writer
}

// DefaultRunner returns an ExecRunner for use in production code. It streams
// installer output to the process's stdout and stderr.
func DefaultRunner() *ExecRunner {
	return &ExecRunner{Stdout: os.Stdout, Stderr: os.Stderr}
}

func (r *ExecRunner) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

func (r *ExecRunner) Run(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	tail := &tailBuffer{max: errTailBytes}
	cmd.Stdout = r.Stdout
	cmd.Stderr = tail
	if r.Stderr != nil {
		cmd.Stderr = io.MultiWriter(r.Stderr, tail)
	}
	if err := cmd.Run(); err != nil {
		cmdline := strings.Join(append([]string{name}, args...), " ")
		if msg := strings.TrimSpace(tail.String()); msg != "" {
			return fmt.Errorf("running %s: %w\n%s", cmdline, err, msg)
		}
		return fmt.Errorf("running %s: %w", cmdline, err)
	}
	return nil
}

func (r *ExecRunner) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.Bytes(), err
}

// tailBuffer is an io.Writer that keeps only the last max bytes written.
type tailBuffer struct {
	max int
	buf []byte
}

func (t *tailBuffer) Write(p []byte) (int, error) {
	t.buf = append(t.buf, p...)
	if over := len(t.buf) - t.max; over > 0 {
		t.buf = append(t.buf[:0], t.buf[over:]...)
	}
	return len(p), nil
}

func (t *tailBuffer) String() string { return string(t.buf) }

// ensureRunner returns the given runner if non-nil, otherwise DefaultRunner().
func ensureRunner(r CommandRunner) CommandRunner {
	if r == nil {
		return DefaultRunner()
	}
	return r
}
