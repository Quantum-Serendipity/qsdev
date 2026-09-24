package container

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"time"
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

// probeWaitDelay bounds how long Output waits for I/O after its context ends.
const probeWaitDelay = time.Second

// ExecProber implements Prober using real system calls.
type ExecProber struct{}

func (p *ExecProber) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

func (p *ExecProber) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	// The probes query the host, not the project, so run them outside the
	// caller's working directory: on Windows a probe grandchild that outlives
	// a timeout (a docker CLI plugin, podman's machine helper) holds its
	// working directory open and so blocks renaming or deleting the project.
	cmd.Dir = os.TempDir()
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	// Once ctx is done and the process is killed, stop waiting for any
	// grandchild that still holds the stdout pipe open.
	cmd.WaitDelay = probeWaitDelay
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
