//go:build !windows

package policy

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateMountDecl(t *testing.T) {
	t.Parallel()

	home, _ := os.UserHomeDir()
	in := func(rel string) string { return filepath.Join(testProjectDir, rel) }

	type testCase struct {
		name       string
		mount      MountDecl
		projectDir string
		wantErr    string
	}
	tests := []testCase{
		{
			name:  "project-relative mount",
			mount: MountDecl{Source: in("tools"), Target: in("tools"), ReadOnly: true},
		},
		{
			name:  "writable project-relative mount",
			mount: MountDecl{Source: in(".cache"), Target: in("cache"), ReadOnly: false},
		},
		{
			name:  "nix store path",
			mount: MountDecl{Source: "/nix/store/abc-tool", Target: "/nix/store/abc-tool", ReadOnly: true},
		},
		{
			name:    "source outside the project",
			mount:   MountDecl{Source: "/opt/tools", Target: in("tools"), ReadOnly: true},
			wantErr: "outside the project directory",
		},
		{
			name:    "target outside the project",
			mount:   MountDecl{Source: in("tools"), Target: "/opt/tools", ReadOnly: true},
			wantErr: "outside the project directory",
		},
		{
			name:    "dot-dot escapes the project",
			mount:   MountDecl{Source: in("../other"), Target: in("other"), ReadOnly: true},
			wantErr: "outside the project directory",
		},
		{
			name:    "sibling with the project as a name prefix",
			mount:   MountDecl{Source: testProjectDir + "-evil", Target: in("x"), ReadOnly: true},
			wantErr: "outside the project directory",
		},
		{
			name:    "systemd user bus",
			mount:   MountDecl{Source: "/run/user/1000", Target: "/run/user/1000", ReadOnly: false},
			wantErr: "runtime directory /run",
		},
		{
			name:    "docker socket",
			mount:   MountDecl{Source: "/var/run/docker.sock", Target: in("docker.sock"), ReadOnly: false},
			wantErr: "runtime directory /var/run",
		},
		{
			name:       "project under /run is still refused",
			mount:      MountDecl{Source: "/run/proj/sub", Target: "/run/proj/sub", ReadOnly: true},
			projectDir: "/run/proj",
			wantErr:    "runtime directory /run",
		},
		{
			name:       "no project directory allows only the nix store",
			mount:      MountDecl{Source: in("tools"), Target: in("tools"), ReadOnly: true},
			projectDir: "-",
			wantErr:    "outside the project directory",
		},
		{
			name:    "relative source",
			mount:   MountDecl{Source: "relative/path", Target: in("out"), ReadOnly: true},
			wantErr: "must be absolute",
		},
		{
			name:    "relative target",
			mount:   MountDecl{Source: in("tools"), Target: "relative/path", ReadOnly: true},
			wantErr: "must be absolute",
		},
		{
			name:    "root filesystem source",
			mount:   MountDecl{Source: "/", Target: in("mnt"), ReadOnly: true},
			wantErr: "root filesystem",
		},
		{
			name:    "root filesystem target",
			mount:   MountDecl{Source: in("data"), Target: "/", ReadOnly: true},
			wantErr: "root filesystem",
		},
		{
			name:       "deny /etc/shadow even when the project is /etc",
			mount:      MountDecl{Source: "/etc/shadow", Target: "/etc/shadow", ReadOnly: true},
			projectDir: "/etc",
			wantErr:    "overlaps sensitive path",
		},
	}

	if home != "" {
		// With the project at $HOME the allowlist admits every home path, so
		// these pin that the deny list still applies inside the project.
		for _, tc := range []testCase{
			{name: "deny ~/.ssh source", mount: MountDecl{Source: filepath.Join(home, ".ssh"), Target: filepath.Join(home, "ssh"), ReadOnly: true}},
			{name: "deny ~/.aws target", mount: MountDecl{Source: filepath.Join(home, "aws"), Target: filepath.Join(home, ".aws"), ReadOnly: true}},
			{name: "deny ~/.docker/config.json", mount: MountDecl{Source: filepath.Join(home, ".docker", "config.json"), Target: filepath.Join(home, "d"), ReadOnly: true}},
			{name: "deny home dir source (ancestor of credential dirs)", mount: MountDecl{Source: home, Target: filepath.Join(home, "h"), ReadOnly: true}},
		} {
			tc.projectDir = home
			tc.wantErr = "overlaps sensitive path"
			tests = append(tests, tc)
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			projectDir := tt.projectDir
			switch projectDir {
			case "":
				projectDir = testProjectDir
			case "-":
				projectDir = ""
			}
			err := ValidateMountDecl(tt.mount, projectDir)
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

// TestValidateMountDecl_RejectsSocketSource pins that a socket inside the
// project (a devenv service socket, say) cannot be bound into the sandbox.
func TestValidateMountDecl_RejectsSocketSource(t *testing.T) {
	t.Parallel()

	// Unix socket paths are length-limited, so use a short directory.
	dir, err := os.MkdirTemp("", "qsp")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	sock := filepath.Join(dir, "s.sock")
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Skipf("cannot create a unix socket: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	err = ValidateMountDecl(MountDecl{Source: sock, Target: filepath.Join(dir, "t.sock")}, dir)
	if err == nil || !strings.Contains(err.Error(), "socket") {
		t.Errorf("ValidateMountDecl(socket) = %v, want a socket error", err)
	}
}

func TestToSandboxConfig_SkipsInvalidCategoryMounts(t *testing.T) {
	t.Parallel()

	spec := DefaultPolicy()
	spec.HookCategories["linter"] = CategoryPolicy{
		WorktreeAccess: "ro",
		Network:        "deny",
		ExtraMounts: []MountDecl{
			{Source: testProjectDir + "/valid", Target: testProjectDir + "/valid", ReadOnly: true},
			{Source: "relative/bad", Target: "/tmp/bad", ReadOnly: true},
			{Source: "/", Target: "/mnt", ReadOnly: true},
		},
	}

	cfg := ToSandboxConfig(spec, 0, "test-hook", testProjectDir) // CategoryLinter = 0

	// Deny entries travel in cfg.Deny, so Mounts holds only the extra mounts.
	if len(cfg.Mounts) != 1 {
		t.Errorf("expected 1 valid extra mount, got %d: %v", len(cfg.Mounts), cfg.Mounts)
	}
}

func TestToSandboxConfig_SkipsInvalidOverrideMounts(t *testing.T) {
	t.Parallel()

	spec := DefaultPolicy()
	spec.HookOverrides = map[string]HookOverride{
		"my-hook": {
			ExtraMounts: []MountDecl{
				{Source: testProjectDir + "/valid", Target: testProjectDir + "/valid", ReadOnly: true},
				{Source: "/etc/shadow", Target: "/tmp/shadow", ReadOnly: true},
				{Source: "/run/user/1000", Target: "/run/user/1000", ReadOnly: false},
			},
		},
	}

	cfg := ToSandboxConfig(spec, 0, "my-hook", testProjectDir) // CategoryLinter = 0

	if len(cfg.Mounts) != 1 {
		t.Errorf("expected 1 valid extra mount, got %d: %v", len(cfg.Mounts), cfg.Mounts)
	}
}
