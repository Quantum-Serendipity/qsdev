//go:build !windows

package policy

import (
	"os"
	"path/filepath"
	"testing"
)

// TestValidateMountDecl_RejectsSymlinkToDenyPath pins the symlink-resolution
// branch of the policy mount check: a declared source or target that resolves
// to a credential store is rejected even though its literal path is harmless.
// It uses t.Setenv (a fake HOME), so it is not parallel.
func TestValidateMountDecl_RejectsSymlinkToDenyPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	aws := filepath.Join(home, ".aws")
	if err := os.Mkdir(aws, 0o700); err != nil {
		t.Fatalf("creating fake ~/.aws: %v", err)
	}
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(aws, link); err != nil {
		t.Fatalf("creating symlink: %v", err)
	}
	safe := t.TempDir()

	tests := []struct {
		name string
		decl MountDecl
	}{
		{name: "symlinked source", decl: MountDecl{Source: link, Target: "/creds", ReadOnly: true}},
		{name: "symlinked target", decl: MountDecl{Source: safe, Target: link, ReadOnly: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateMountDecl(tt.decl); err == nil {
				t.Errorf("ValidateMountDecl(%+v) = nil, want a deny error", tt.decl)
			}
		})
	}

	// Control: the same shape with a symlink to a harmless directory passes.
	safeLink := filepath.Join(t.TempDir(), "safe-link")
	if err := os.Symlink(safe, safeLink); err != nil {
		t.Fatalf("creating symlink: %v", err)
	}
	if err := ValidateMountDecl(MountDecl{Source: safeLink, Target: "/data", ReadOnly: true}); err != nil {
		t.Errorf("ValidateMountDecl(symlink to safe dir) = %v, want nil", err)
	}
}
