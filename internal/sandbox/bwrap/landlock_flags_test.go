package bwrap

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
)

// landlockGrant returns the access flag ("--ro"/"--rw") landlockFlags
// emitted for path, or "" when the path is not granted at all.
func landlockGrant(flags []string, path string) string {
	for i := 0; i+1 < len(flags); i++ {
		if (flags[i] == "--ro" || flags[i] == "--rw") && flags[i+1] == path {
			return flags[i]
		}
	}
	return ""
}

func TestPrepareLandlockFlags_Grants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		cfg   sandbox.SandboxConfig
		want  map[string]string // path -> expected grant ("" = not granted)
		isNet bool
	}{
		{
			name: "device nodes and procfs are reachable",
			cfg:  sandbox.SandboxConfig{HookCategory: sandbox.CategoryLinter},
			want: map[string]string{"/dev": "--rw", "/proc": "--ro", "/tmp": "--rw", "/nix/store": "--ro"},
		},
		{
			name: "extra mounts are granted at the in-sandbox target",
			cfg: sandbox.SandboxConfig{
				HookCategory: sandbox.CategoryFormatter,
				Mounts: []sandbox.MountSpec{
					{Source: "/home/u/.cache/tool", Target: "/opt/tool", ReadOnly: true},
					{Source: "/data/out", Target: "/out", ReadOnly: false},
				},
			},
			want: map[string]string{
				"/opt/tool":           "--ro",
				"/out":                "--rw",
				"/home/u/.cache/tool": "",
				"/data/out":           "",
			},
		},
		{
			name: "project dir follows the category worktree access",
			cfg:  sandbox.SandboxConfig{ProjectDir: "/work/p", HookCategory: sandbox.CategoryLinter},
			want: map[string]string{"/work/p": "--ro"},
		},
		{
			name: "policy deny entries are never granted",
			cfg: sandbox.SandboxConfig{
				HookCategory: sandbox.CategoryFormatter,
				Deny:         []string{"/opt/company-secrets"},
			},
			want: map[string]string{"/opt/company-secrets": ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			flags := landlockFlags(&tt.cfg)
			for path, want := range tt.want {
				if got := landlockGrant(flags, path); got != want {
					t.Errorf("grant for %q = %q, want %q; flags: %v", path, got, want, flags)
				}
			}
		})
	}
}
