package cgroup

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
)

func TestNewManager(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		uid          string
		wantBasePath string
	}{
		{
			name:         "standard uid",
			uid:          "1000",
			wantBasePath: "/sys/fs/cgroup/user.slice/user-1000.slice",
		},
		{
			name:         "different uid",
			uid:          "1001",
			wantBasePath: "/sys/fs/cgroup/user.slice/user-1001.slice",
		},
		{
			name:         "root uid",
			uid:          "0",
			wantBasePath: "/sys/fs/cgroup/user.slice/user-0.slice",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m, err := NewManager(tt.uid)
			if err != nil {
				t.Fatalf("NewManager(%q) unexpected error: %v", tt.uid, err)
			}
			if m.basePath != tt.wantBasePath {
				t.Errorf("NewManager(%q).basePath = %q, want %q", tt.uid, m.basePath, tt.wantBasePath)
			}
		})
	}
}

func TestNewManager_InvalidUID(t *testing.T) {
	t.Parallel()

	for _, uid := range []string{"abc", "../etc", "", "1000/../../root"} {
		_, err := NewManager(uid)
		if err == nil {
			t.Errorf("NewManager(%q) expected error for non-numeric UID", uid)
		}
	}
}

func TestManagerScopePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		uid       string
		scopeName string
		want      string
	}{
		{
			name:      "simple scope name",
			uid:       "1000",
			scopeName: "my-hook",
			want:      "/sys/fs/cgroup/user.slice/user-1000.slice/qsdev-hooks.scope/my-hook",
		},
		{
			name:      "nested scope name",
			uid:       "1001",
			scopeName: "lint-run-1",
			want:      "/sys/fs/cgroup/user.slice/user-1001.slice/qsdev-hooks.scope/lint-run-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m, err := NewManager(tt.uid)
			if err != nil {
				t.Fatalf("NewManager(%q) unexpected error: %v", tt.uid, err)
			}
			got := m.ScopePath(tt.scopeName)
			if got != tt.want {
				t.Errorf("ScopePath(%q) = %q, want %q", tt.scopeName, got, tt.want)
			}
		})
	}
}

func TestFormatMemoryMax(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{
			name:  "2 GB",
			bytes: 2 * 1024 * 1024 * 1024,
			want:  "2147483648",
		},
		{
			name:  "512 MB",
			bytes: 512 * 1024 * 1024,
			want:  "536870912",
		},
		{
			name:  "zero",
			bytes: 0,
			want:  "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := string(formatMemoryMax(tt.bytes))
			if got != tt.want {
				t.Errorf("formatMemoryMax(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestFormatPIDsMax(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		maxPIDs int
		want    string
	}{
		{
			name:    "default",
			maxPIDs: 4096,
			want:    "4096",
		},
		{
			name:    "small",
			maxPIDs: 64,
			want:    "64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := string(formatPIDsMax(tt.maxPIDs))
			if got != tt.want {
				t.Errorf("formatPIDsMax(%d) = %q, want %q", tt.maxPIDs, got, tt.want)
			}
		})
	}
}

func TestFormatCPUMax(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		percent int
		want    string
	}{
		{
			name:    "200 percent (2 cores)",
			percent: 200,
			want:    "200000 100000",
		},
		{
			name:    "100 percent (1 core)",
			percent: 100,
			want:    "100000 100000",
		},
		{
			name:    "50 percent (half core)",
			percent: 50,
			want:    "50000 100000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := string(formatCPUMax(tt.percent))
			if got != tt.want {
				t.Errorf("formatCPUMax(%d) = %q, want %q", tt.percent, got, tt.want)
			}
		})
	}
}

func TestCreateScopePathConstruction(t *testing.T) {
	t.Parallel()

	// We cannot actually create cgroup directories in tests, but we can
	// verify that the manager constructs the expected scope path from uid
	// and scope name.
	m, err := NewManager("1000")
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	limits := sandbox.DefaultResourceLimits()

	// Verify the scope path that would be created.
	expectedPath := "/sys/fs/cgroup/user.slice/user-1000.slice/qsdev-hooks.scope/test-hook"
	got := m.ScopePath("test-hook")
	if got != expectedPath {
		t.Errorf("scope path = %q, want %q", got, expectedPath)
	}

	// Verify the format functions produce expected values for default limits.
	memMax := string(formatMemoryMax(limits.MemoryBytes))
	if memMax != "2147483648" {
		t.Errorf("memory.max = %q, want %q", memMax, "2147483648")
	}

	pidsMax := string(formatPIDsMax(limits.MaxPIDs))
	if pidsMax != "4096" {
		t.Errorf("pids.max = %q, want %q", pidsMax, "4096")
	}

	cpuMax := string(formatCPUMax(limits.CPUQuotaPercent))
	if cpuMax != "200000 100000" {
		t.Errorf("cpu.max = %q, want %q", cpuMax, "200000 100000")
	}
}
