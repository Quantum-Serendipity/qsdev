package doctor

import (
	"context"
	"os"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
)

type mockSandboxProber struct {
	lookPathResults map[string]string
	outputResults   map[string][]byte
	files           map[string][]byte
	fileInfos       map[string]bool
	envVars         map[string]string
}

func newMockSandboxProber() *mockSandboxProber {
	return &mockSandboxProber{
		lookPathResults: make(map[string]string),
		outputResults:   make(map[string][]byte),
		files:           make(map[string][]byte),
		fileInfos:       make(map[string]bool),
		envVars:         make(map[string]string),
	}
}

func (m *mockSandboxProber) LookPath(name string) (string, error) {
	if path, ok := m.lookPathResults[name]; ok {
		return path, nil
	}
	return "", &os.PathError{Op: "LookPath", Path: name, Err: os.ErrNotExist}
}

func (m *mockSandboxProber) Output(_ context.Context, name string, _ ...string) ([]byte, error) {
	if out, ok := m.outputResults[name]; ok {
		return out, nil
	}
	return nil, &os.PathError{Op: "exec", Path: name, Err: os.ErrNotExist}
}

func (m *mockSandboxProber) ReadFile(path string) ([]byte, error) {
	if data, ok := m.files[path]; ok {
		return data, nil
	}
	return nil, &os.PathError{Op: "ReadFile", Path: path, Err: os.ErrNotExist}
}

func (m *mockSandboxProber) Stat(path string) (os.FileInfo, error) {
	if m.fileInfos[path] {
		return nil, nil
	}
	return nil, &os.PathError{Op: "Stat", Path: path, Err: os.ErrNotExist}
}

func (m *mockSandboxProber) Getenv(key string) string {
	return m.envVars[key]
}

var _ sandbox.SandboxProber = (*mockSandboxProber)(nil)

func TestRunSandboxCheck_FullSupport(t *testing.T) {
	t.Parallel()
	mock := newMockSandboxProber()
	mock.lookPathResults["bwrap"] = "/usr/bin/bwrap"
	mock.lookPathResults["systemd-run"] = "/usr/bin/systemd-run"
	mock.files["/proc/sys/kernel/unprivileged_userns_clone"] = []byte("1\n")
	mock.files["/proc/sys/kernel/seccomp/actions_avail"] = []byte("kill errno\n")
	mock.files["/proc/version"] = []byte("Linux version 6.8.0-generic\n")
	mock.fileInfos["/sys/fs/cgroup/cgroup.controllers"] = true

	section := RunSandboxCheck(context.Background(), mock)

	if section == nil {
		t.Fatal("expected non-nil section")
	}
	if !section.Detected {
		t.Error("expected Detected = true")
	}
	if section.Tier != "full" {
		t.Errorf("Tier = %q, want %q", section.Tier, "full")
	}
	if section.SecurityLevel != "strong" {
		t.Errorf("SecurityLevel = %q, want %q", section.SecurityLevel, "strong")
	}
	if len(section.Recommendations) != 0 {
		t.Errorf("expected no recommendations for full tier, got %v", section.Recommendations)
	}
}

func TestRunSandboxCheck_NoBwrap(t *testing.T) {
	t.Parallel()
	mock := newMockSandboxProber()

	section := RunSandboxCheck(context.Background(), mock)

	if section == nil {
		t.Fatal("expected non-nil section")
	}
	if section.Tier != "unsandboxed" {
		t.Errorf("Tier = %q, want %q", section.Tier, "unsandboxed")
	}
	if len(section.Recommendations) == 0 {
		t.Error("expected recommendations for unsandboxed tier")
	}
}

func TestRunSandboxCheck_SystemdRunOnly(t *testing.T) {
	t.Parallel()
	mock := newMockSandboxProber()
	mock.lookPathResults["systemd-run"] = "/usr/bin/systemd-run"

	section := RunSandboxCheck(context.Background(), mock)

	if section.Tier != "systemd-run" {
		t.Errorf("Tier = %q, want %q", section.Tier, "systemd-run")
	}
	if section.SecurityLevel != "minimal" {
		t.Errorf("SecurityLevel = %q, want %q", section.SecurityLevel, "minimal")
	}
}

func TestRunSandboxCheck_ItemCount(t *testing.T) {
	t.Parallel()
	mock := newMockSandboxProber()

	section := RunSandboxCheck(context.Background(), mock)

	if len(section.Items) != 7 {
		t.Errorf("expected 7 items, got %d", len(section.Items))
	}
}
