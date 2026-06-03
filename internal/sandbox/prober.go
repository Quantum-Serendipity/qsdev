package sandbox

import (
	"bytes"
	"context"
	"os"
	"os/exec"
)

// SandboxProber abstracts system calls needed by sandbox capability detection.
// Tests supply a mock; production code uses ExecSandboxProber.
type SandboxProber interface {
	LookPath(name string) (string, error)
	Output(ctx context.Context, name string, args ...string) ([]byte, error)
	ReadFile(path string) ([]byte, error)
	Stat(path string) (os.FileInfo, error)
	Getenv(key string) string
}

// ExecSandboxProber implements SandboxProber using real system calls.
type ExecSandboxProber struct{}

func (p *ExecSandboxProber) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

func (p *ExecSandboxProber) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err := cmd.Run()
	return stdout.Bytes(), err
}

func (p *ExecSandboxProber) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (p *ExecSandboxProber) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

func (p *ExecSandboxProber) Getenv(key string) string {
	return os.Getenv(key)
}
