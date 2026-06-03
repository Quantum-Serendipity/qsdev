package cgroup

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
)

func TestSystemdRunBackend_InterfaceCompliance(t *testing.T) {
	t.Parallel()

	// Compile-time check is in systemd.go via var _ line, but we also
	// verify at runtime that the concrete type satisfies the interface.
	var _ sandbox.SandboxBackend = (*SystemdRunBackend)(nil)
}

func TestSystemdRunBackend_Name(t *testing.T) {
	t.Parallel()

	b := NewSystemdRunBackend("/usr/bin/systemd-run")
	if got := b.Name(); got != "systemd-run" {
		t.Errorf("Name() = %q, want %q", got, "systemd-run")
	}
}

func TestSystemdRunBackend_Tier(t *testing.T) {
	t.Parallel()

	b := NewSystemdRunBackend("/usr/bin/systemd-run")
	if got := b.Tier(); got != sandbox.TierSystemdRun {
		t.Errorf("Tier() = %v, want %v", got, sandbox.TierSystemdRun)
	}
}

func TestSystemdRunBackend_Available(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
		{
			name:    "nonexistent path",
			path:    "/nonexistent/systemd-run",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			b := NewSystemdRunBackend(tt.path)
			err := b.Available()
			if (err != nil) != tt.wantErr {
				t.Errorf("Available() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBuildArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  *sandbox.SandboxConfig
		want []string
	}{
		{
			name: "all resource limits",
			cfg: &sandbox.SandboxConfig{
				HookCommand: []string{"golangci-lint", "run"},
				Resources: sandbox.ResourceLimits{
					MemoryBytes:     2147483648,
					MaxPIDs:         4096,
					CPUQuotaPercent: 200,
				},
			},
			want: []string{
				"--user", "--scope",
				"-p", "MemoryMax=2147483648",
				"-p", "TasksMax=4096",
				"-p", "CPUQuota=200%",
				"--",
				"golangci-lint", "run",
			},
		},
		{
			name: "zero resource limits omitted",
			cfg: &sandbox.SandboxConfig{
				HookCommand: []string{"echo", "hello"},
				Resources:   sandbox.ResourceLimits{},
			},
			want: []string{
				"--user", "--scope",
				"--",
				"echo", "hello",
			},
		},
		{
			name: "only memory limit",
			cfg: &sandbox.SandboxConfig{
				HookCommand: []string{"lint"},
				Resources: sandbox.ResourceLimits{
					MemoryBytes: 536870912,
				},
			},
			want: []string{
				"--user", "--scope",
				"-p", "MemoryMax=536870912",
				"--",
				"lint",
			},
		},
		{
			name: "only pids limit",
			cfg: &sandbox.SandboxConfig{
				HookCommand: []string{"test"},
				Resources: sandbox.ResourceLimits{
					MaxPIDs: 64,
				},
			},
			want: []string{
				"--user", "--scope",
				"-p", "TasksMax=64",
				"--",
				"test",
			},
		},
		{
			name: "only cpu limit",
			cfg: &sandbox.SandboxConfig{
				HookCommand: []string{"build"},
				Resources: sandbox.ResourceLimits{
					CPUQuotaPercent: 100,
				},
			},
			want: []string{
				"--user", "--scope",
				"-p", "CPUQuota=100%",
				"--",
				"build",
			},
		},
		{
			name: "default resource limits",
			cfg: &sandbox.SandboxConfig{
				HookCommand: []string{"go", "vet", "./..."},
				Resources:   sandbox.DefaultResourceLimits(),
			},
			want: []string{
				"--user", "--scope",
				"-p", "MemoryMax=2147483648",
				"-p", "TasksMax=4096",
				"-p", "CPUQuota=200%",
				"--",
				"go", "vet", "./...",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := BuildArgs(tt.cfg)

			if len(got) != len(tt.want) {
				t.Fatalf("BuildArgs() returned %d args, want %d\ngot:  %v\nwant: %v",
					len(got), len(tt.want), got, tt.want)
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("BuildArgs()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
