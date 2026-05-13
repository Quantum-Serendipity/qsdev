package pkgmanager

import (
	"bytes"
	"context"
	"os/exec"
)

// ExecRunner implements CommandRunner using real os/exec calls.
type ExecRunner struct{}

// DefaultRunner returns an ExecRunner for use in production code.
func DefaultRunner() *ExecRunner {
	return &ExecRunner{}
}

func (r *ExecRunner) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

func (r *ExecRunner) Run(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.Run()
}

func (r *ExecRunner) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.Bytes(), err
}

// ensureRunner returns the given runner if non-nil, otherwise DefaultRunner().
func ensureRunner(r CommandRunner) CommandRunner {
	if r == nil {
		return DefaultRunner()
	}
	return r
}
