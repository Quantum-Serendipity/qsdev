//go:build !windows

package policy

import (
	"os"
	"path/filepath"
	"testing"
)

// TestValidateMountDecl_RejectsSymlinkEscape pins the symlink-resolution
// branch of the policy mount check: a declared source or target inside the
// project that resolves to a credential store, or anywhere outside the
// project, is rejected even though its literal path is allowed. It uses
// t.Setenv (a fake HOME), so it is not parallel.
func TestValidateMountDecl_RejectsSymlinkEscape(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	aws := filepath.Join(home, ".aws")
	if err := os.Mkdir(aws, 0o700); err != nil {
		t.Fatalf("creating fake ~/.aws: %v", err)
	}
	project := t.TempDir()
	symlink := func(target, name string) string {
		t.Helper()
		link := filepath.Join(project, name)
		if err := os.Symlink(target, link); err != nil {
			t.Fatalf("creating symlink: %v", err)
		}
		return link
	}
	credLink := symlink(aws, "creds")
	outsideLink := symlink(t.TempDir(), "outside")
	safe := filepath.Join(project, "safe")
	if err := os.Mkdir(safe, 0o700); err != nil {
		t.Fatalf("creating safe dir: %v", err)
	}

	tests := []struct {
		name string
		decl MountDecl
	}{
		{name: "source symlinked to a credential store", decl: MountDecl{Source: credLink, Target: safe, ReadOnly: true}},
		{name: "target symlinked to a credential store", decl: MountDecl{Source: safe, Target: credLink, ReadOnly: true}},
		{name: "source symlinked outside the project", decl: MountDecl{Source: outsideLink, Target: safe, ReadOnly: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateMountDecl(tt.decl, project); err == nil {
				t.Errorf("ValidateMountDecl(%+v) = nil, want an error", tt.decl)
			}
		})
	}

	// Control: a symlink that stays inside the project passes.
	safeLink := symlink(safe, "safe-link")
	if err := ValidateMountDecl(MountDecl{Source: safeLink, Target: safe, ReadOnly: true}, project); err != nil {
		t.Errorf("ValidateMountDecl(symlink within the project) = %v, want nil", err)
	}
}

// TestValidateMountDecl_RejectsCredentialsUnderSymlinkedHome reproduces the
// macOS layout (/var -> /private/var) on any host: HOME is reached through a
// symlink, so a mount naming the resolved ~/.ssh must still be denied.
func TestValidateMountDecl_RejectsCredentialsUnderSymlinkedHome(t *testing.T) {
	realHome := t.TempDir()
	homeLink := filepath.Join(t.TempDir(), "home")
	if err := os.Symlink(realHome, homeLink); err != nil {
		t.Fatalf("creating home symlink: %v", err)
	}
	t.Setenv("HOME", homeLink)
	ssh := filepath.Join(realHome, ".ssh")
	if err := os.Mkdir(ssh, 0o700); err != nil {
		t.Fatalf("creating fake ~/.ssh: %v", err)
	}

	decl := MountDecl{Source: ssh, Target: ssh, ReadOnly: true}
	if err := ValidateMountDecl(decl, realHome); err == nil {
		t.Errorf("ValidateMountDecl(%+v) with HOME=%q = nil, want a deny error", decl, homeLink)
	}
}
