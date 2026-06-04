package policy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateMountDecl(t *testing.T) {
	t.Parallel()

	home, _ := os.UserHomeDir()

	tests := []struct {
		name    string
		mount   MountDecl
		wantErr string
	}{
		{
			name:  "valid mount",
			mount: MountDecl{Source: "/opt/tools", Target: "/opt/tools", ReadOnly: true},
		},
		{
			name:  "valid writable mount",
			mount: MountDecl{Source: "/tmp/cache", Target: "/tmp/cache", ReadOnly: false},
		},
		{
			name:    "relative source",
			mount:   MountDecl{Source: "relative/path", Target: "/opt/out", ReadOnly: true},
			wantErr: "must be absolute",
		},
		{
			name:    "relative target",
			mount:   MountDecl{Source: "/opt/tools", Target: "relative/path", ReadOnly: true},
			wantErr: "must be absolute",
		},
		{
			name:    "root filesystem source",
			mount:   MountDecl{Source: "/", Target: "/mnt", ReadOnly: true},
			wantErr: "root filesystem",
		},
		{
			name:    "deny /etc/shadow source",
			mount:   MountDecl{Source: "/etc/shadow", Target: "/tmp/shadow", ReadOnly: true},
			wantErr: "overlaps sensitive path",
		},
		{
			name:    "deny /etc/sudoers target",
			mount:   MountDecl{Source: "/tmp/sudoers", Target: "/etc/sudoers", ReadOnly: false},
			wantErr: "overlaps sensitive path",
		},
		{
			name:    "deny /root source",
			mount:   MountDecl{Source: "/root", Target: "/tmp/root", ReadOnly: true},
			wantErr: "overlaps sensitive path",
		},
		{
			name:    "deny /root subpath",
			mount:   MountDecl{Source: "/root/.bashrc", Target: "/tmp/bashrc", ReadOnly: true},
			wantErr: "overlaps sensitive path",
		},
		{
			name:  "allow /etc/hosts",
			mount: MountDecl{Source: "/etc/hosts", Target: "/etc/hosts", ReadOnly: true},
		},
	}

	if home != "" {
		tests = append(tests,
			struct {
				name    string
				mount   MountDecl
				wantErr string
			}{
				name:    "deny ~/.ssh source",
				mount:   MountDecl{Source: filepath.Join(home, ".ssh"), Target: "/tmp/ssh", ReadOnly: true},
				wantErr: "overlaps sensitive path",
			},
			struct {
				name    string
				mount   MountDecl
				wantErr string
			}{
				name:    "deny ~/.aws target",
				mount:   MountDecl{Source: "/tmp/aws", Target: filepath.Join(home, ".aws"), ReadOnly: true},
				wantErr: "overlaps sensitive path",
			},
			struct {
				name    string
				mount   MountDecl
				wantErr string
			}{
				name:    "deny ~/.gnupg",
				mount:   MountDecl{Source: filepath.Join(home, ".gnupg"), Target: "/tmp/gnupg", ReadOnly: true},
				wantErr: "overlaps sensitive path",
			},
			struct {
				name    string
				mount   MountDecl
				wantErr string
			}{
				name:    "deny ~/.docker/config.json",
				mount:   MountDecl{Source: filepath.Join(home, ".docker", "config.json"), Target: "/tmp/docker", ReadOnly: true},
				wantErr: "overlaps sensitive path",
			},
		)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateMountDecl(tt.mount)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Errorf("expected error containing %q, got nil", tt.wantErr)
				return
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q should contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestToSandboxConfig_SkipsInvalidCategoryMounts(t *testing.T) {
	t.Parallel()

	spec := DefaultPolicy()
	spec.HookCategories["linter"] = CategoryPolicy{
		WorktreeAccess: "ro",
		Network:        "deny",
		ExtraMounts: []MountDecl{
			{Source: "/opt/valid", Target: "/opt/valid", ReadOnly: true},
			{Source: "relative/bad", Target: "/tmp/bad", ReadOnly: true},
			{Source: "/", Target: "/mnt", ReadOnly: true},
		},
	}

	cfg := ToSandboxConfig(spec, 0, "test-hook") // CategoryLinter = 0

	// Count extra mounts beyond the deny-list mounts.
	denyCount := len(spec.Filesystem.Deny)
	extraCount := len(cfg.Mounts) - denyCount
	if extraCount != 1 {
		t.Errorf("expected 1 valid extra mount, got %d (total mounts: %d, deny: %d)", extraCount, len(cfg.Mounts), denyCount)
	}
}

func TestToSandboxConfig_SkipsInvalidOverrideMounts(t *testing.T) {
	t.Parallel()

	spec := DefaultPolicy()
	spec.HookOverrides = map[string]HookOverride{
		"my-hook": {
			ExtraMounts: []MountDecl{
				{Source: "/opt/valid", Target: "/opt/valid", ReadOnly: true},
				{Source: "/etc/shadow", Target: "/tmp/shadow", ReadOnly: true},
			},
		},
	}

	cfg := ToSandboxConfig(spec, 0, "my-hook") // CategoryLinter = 0

	denyCount := len(spec.Filesystem.Deny)
	extraCount := len(cfg.Mounts) - denyCount
	if extraCount != 1 {
		t.Errorf("expected 1 valid extra mount, got %d", extraCount)
	}
}
