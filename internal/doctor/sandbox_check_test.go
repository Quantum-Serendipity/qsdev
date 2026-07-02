package doctor

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
)

// stubBinary creates a stat-able file so the resolved sandbox backend reports
// itself Available(). RunSandboxCheck reports the EFFECTIVE tier (the tier of
// the backend that will actually run), so tier assertions must point the probed
// binary paths at real files rather than nonexistent host paths.
func stubBinary(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("writing stub binary %s: %v", path, err)
	}
	return path
}

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
	// Point bwrap at a real (stat-able) file so the bubblewrap backend is
	// actually selectable; otherwise the effective tier degrades to unsandboxed.
	mock.lookPathResults["bwrap"] = stubBinary(t, "bwrap")
	mock.lookPathResults["systemd-run"] = stubBinary(t, "systemd-run")
	mock.files["/proc/sys/kernel/unprivileged_userns_clone"] = []byte("1\n")
	mock.files["/proc/sys/kernel/seccomp/actions_avail"] = []byte("kill errno\n")
	mock.files["/proc/version"] = []byte("Linux version 6.8.0-generic\n")
	mock.fileInfos["/sys/fs/cgroup/cgroup.controllers"] = true

	section := RunSandboxCheck(context.Background(), mock)

	if section == nil {
		t.Fatal("expected non-nil section")
		return
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
		return
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
	// Real systemd-run binary, no bwrap: the effective tier is systemd-run.
	mock.lookPathResults["systemd-run"] = stubBinary(t, "systemd-run")

	section := RunSandboxCheck(context.Background(), mock)

	if section.Tier != "systemd-run" {
		t.Errorf("Tier = %q, want %q", section.Tier, "systemd-run")
	}
	if section.SecurityLevel != "minimal" {
		t.Errorf("SecurityLevel = %q, want %q", section.SecurityLevel, "minimal")
	}
}

// TestRunSandboxCheck_ReportsEffectiveTier is a regression for BL-P0-1: every
// kernel capability for TierFull is probed as present, but the bwrap binary does
// not exist on disk, so no isolating backend is selectable. The doctor must
// report the EFFECTIVE tier (unsandboxed) rather than overclaiming "full" from
// the probed capabilities. The pre-fix code reported DetermineTier(caps) = full.
func TestRunSandboxCheck_ReportsEffectiveTier(t *testing.T) {
	t.Parallel()
	mock := newMockSandboxProber()
	mock.lookPathResults["bwrap"] = "/nonexistent/path/to/bwrap"
	mock.files["/proc/sys/kernel/unprivileged_userns_clone"] = []byte("1\n")
	mock.files["/proc/sys/kernel/seccomp/actions_avail"] = []byte("kill errno\n")
	mock.files["/proc/version"] = []byte("Linux version 6.8.0-generic\n")

	section := RunSandboxCheck(context.Background(), mock)

	if section.Tier != "unsandboxed" {
		t.Errorf("Tier = %q, want %q (probed full but bwrap binary absent)", section.Tier, "unsandboxed")
	}
	if section.SecurityLevel != "none" {
		t.Errorf("SecurityLevel = %q, want %q", section.SecurityLevel, "none")
	}
	if len(section.Recommendations) == 0 {
		t.Error("expected remediation recommendations when effective tier is unsandboxed")
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
