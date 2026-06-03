package cgroup

import (
	"context"
	"os"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
)

// mockProber implements sandbox.SandboxProber for testing.
type mockProber struct {
	files     map[string][]byte
	fileInfos map[string]bool
}

func newMockProber() *mockProber {
	return &mockProber{
		files:     make(map[string][]byte),
		fileInfos: make(map[string]bool),
	}
}

func (m *mockProber) LookPath(name string) (string, error) {
	return "", &os.PathError{Op: "LookPath", Path: name, Err: os.ErrNotExist}
}

func (m *mockProber) Output(_ context.Context, name string, _ ...string) ([]byte, error) {
	return nil, &os.PathError{Op: "exec", Path: name, Err: os.ErrNotExist}
}

func (m *mockProber) ReadFile(path string) ([]byte, error) {
	if data, ok := m.files[path]; ok {
		return data, nil
	}
	return nil, &os.PathError{Op: "ReadFile", Path: path, Err: os.ErrNotExist}
}

func (m *mockProber) Stat(path string) (os.FileInfo, error) {
	if m.fileInfos[path] {
		return nil, nil
	}
	return nil, &os.PathError{Op: "Stat", Path: path, Err: os.ErrNotExist}
}

func (m *mockProber) Getenv(_ string) string { return "" }

var _ sandbox.SandboxProber = (*mockProber)(nil)

func TestDetectCgroupV2(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		cgroupFile bool
		want       bool
	}{
		{
			name:       "controllers file exists",
			cgroupFile: true,
			want:       true,
		},
		{
			name:       "controllers file missing",
			cgroupFile: false,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mock := newMockProber()
			if tt.cgroupFile {
				mock.fileInfos["/sys/fs/cgroup/cgroup.controllers"] = true
			}

			got := DetectCgroupV2(mock)
			if got != tt.want {
				t.Errorf("DetectCgroupV2() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetectDelegation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		uid     string
		content string
		exists  bool
		want    bool
	}{
		{
			name:    "delegation available",
			uid:     "1000",
			content: "cpu memory pids\n",
			exists:  true,
			want:    true,
		},
		{
			name:    "delegation file exists but empty",
			uid:     "1000",
			content: "",
			exists:  true,
			want:    false,
		},
		{
			name:    "delegation file exists whitespace only",
			uid:     "1000",
			content: "  \n",
			exists:  true,
			want:    false,
		},
		{
			name:   "delegation file missing",
			uid:    "1000",
			exists: false,
			want:   false,
		},
		{
			name:   "empty uid",
			uid:    "",
			exists: false,
			want:   false,
		},
		{
			name:    "different uid",
			uid:     "1001",
			content: "cpu memory\n",
			exists:  true,
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mock := newMockProber()
			if tt.exists {
				path := "/sys/fs/cgroup/user.slice/user-" + tt.uid + ".slice/cgroup.controllers"
				mock.files[path] = []byte(tt.content)
			}

			got := DetectDelegation(mock, tt.uid)
			if got != tt.want {
				t.Errorf("DetectDelegation(uid=%q) = %v, want %v", tt.uid, got, tt.want)
			}
		})
	}
}
