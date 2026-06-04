package bwrap

import (
	"os/user"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateMountPath(t *testing.T) {
	t.Parallel()

	home := ""
	if u, err := user.Current(); err == nil {
		home = u.HomeDir
	}

	tests := []struct {
		name    string
		path    string
		wantErr string // empty means no error expected
	}{
		{
			name: "valid absolute path",
			path: "/opt/tools",
		},
		{
			name: "valid nix store path",
			path: "/nix/store/abc123-go",
		},
		{
			name:    "relative path rejected",
			path:    "relative/path",
			wantErr: "must be absolute",
		},
		{
			name:    "dot-relative path rejected",
			path:    "./local",
			wantErr: "must be absolute",
		},
		{
			name:    "deny /etc/shadow",
			path:    "/etc/shadow",
			wantErr: "denied",
		},
		{
			name:    "deny /etc/sudoers",
			path:    "/etc/sudoers",
			wantErr: "denied",
		},
		{
			name:    "deny /etc/sudoers.d",
			path:    "/etc/sudoers.d",
			wantErr: "denied",
		},
		{
			name:    "deny /etc/sudoers.d subpath",
			path:    "/etc/sudoers.d/custom",
			wantErr: "denied",
		},
		{
			name:    "deny /root",
			path:    "/root",
			wantErr: "denied",
		},
		{
			name:    "deny /root subpath",
			path:    "/root/.bashrc",
			wantErr: "denied",
		},
		{
			name: "allow /etc/hosts",
			path: "/etc/hosts",
		},
		{
			name: "allow /etc/passwd",
			path: "/etc/passwd",
		},
	}

	// Add home-relative deny tests only if we can resolve home.
	if home != "" {
		homeTests := []struct {
			name    string
			path    string
			wantErr string
		}{
			{
				name:    "deny ~/.ssh",
				path:    filepath.Join(home, ".ssh"),
				wantErr: "denied",
			},
			{
				name:    "deny ~/.ssh subpath",
				path:    filepath.Join(home, ".ssh", "id_rsa"),
				wantErr: "denied",
			},
			{
				name:    "deny ~/.gnupg",
				path:    filepath.Join(home, ".gnupg"),
				wantErr: "denied",
			},
			{
				name:    "deny ~/.aws",
				path:    filepath.Join(home, ".aws"),
				wantErr: "denied",
			},
			{
				name:    "deny ~/.azure",
				path:    filepath.Join(home, ".azure"),
				wantErr: "denied",
			},
			{
				name:    "deny ~/.config/gcloud",
				path:    filepath.Join(home, ".config", "gcloud"),
				wantErr: "denied",
			},
			{
				name:    "deny ~/.kube",
				path:    filepath.Join(home, ".kube"),
				wantErr: "denied",
			},
			{
				name:    "deny ~/.docker/config.json",
				path:    filepath.Join(home, ".docker", "config.json"),
				wantErr: "denied",
			},
			{
				name:    "deny ~/.netrc",
				path:    filepath.Join(home, ".netrc"),
				wantErr: "denied",
			},
			{
				name: "allow home dir itself",
				path: home,
			},
		}
		for _, ht := range homeTests {
			tests = append(tests, struct {
				name    string
				path    string
				wantErr string
			}{ht.name, ht.path, ht.wantErr})
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateMountPath(tt.path)
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
