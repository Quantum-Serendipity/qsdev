package container

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
)

// Prober abstracts the system calls needed by container runtime detection.
// Tests supply a mock; production code uses ExecProber.
type Prober interface {
	LookPath(name string) (string, error)
	Output(ctx context.Context, name string, args ...string) ([]byte, error)
	ReadFile(path string) ([]byte, error)
	Stat(path string) (os.FileInfo, error)
	Glob(pattern string) ([]string, error)
	CurrentUser() string
	Getenv(key string) string
}

// ExecProber implements Prober using real system calls.
type ExecProber struct{}

func (p *ExecProber) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

func (p *ExecProber) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err := cmd.Run()
	return stdout.Bytes(), err
}

func (p *ExecProber) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (p *ExecProber) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

func (p *ExecProber) Glob(pattern string) ([]string, error) {
	return filepath.Glob(pattern)
}

func (p *ExecProber) CurrentUser() string {
	u, err := user.Current()
	if err != nil {
		return ""
	}
	return u.Username
}

func (p *ExecProber) Getenv(key string) string {
	return os.Getenv(key)
}
