//go:build !windows

package bwrap

import (
	"os"
	"path/filepath"
	"testing"
)

// fakeHomeWithSSHLink points HOME at a temporary home containing ~/.ssh and
// returns a symlink, outside that home, that resolves to ~/.ssh. It uses
// t.Setenv, so callers must not be parallel.
func fakeHomeWithSSHLink(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	ssh := filepath.Join(home, ".ssh")
	if err := os.Mkdir(ssh, 0o700); err != nil {
		t.Fatalf("creating fake ~/.ssh: %v", err)
	}
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(ssh, link); err != nil {
		t.Fatalf("creating symlink: %v", err)
	}
	return link
}

// TestValidateMountPath_RejectsSymlinkToDenyPath pins the symlink-resolution
// branch of the deny check: a mount source whose literal path is harmless but
// which resolves to a credential store must be rejected.
func TestValidateMountPath_RejectsSymlinkToDenyPath(t *testing.T) {
	link := fakeHomeWithSSHLink(t)

	if err := ValidateMountPath(link); err == nil {
		t.Errorf("ValidateMountPath(%q -> ~/.ssh) = nil, want a deny error", link)
	}
	if !IsDenyPath(link) {
		t.Errorf("IsDenyPath(%q -> ~/.ssh) = false, want true", link)
	}
}

// TestValidateMountPath_AllowsSymlinkToSafePath is the control case: a symlink
// is not rejected merely for being a symlink.
func TestValidateMountPath_AllowsSymlinkToSafePath(t *testing.T) {
	fakeHomeWithSSHLink(t)
	target := t.TempDir()
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("creating symlink: %v", err)
	}

	if err := ValidateMountPath(link); err != nil {
		t.Errorf("ValidateMountPath(%q -> %q) = %v, want nil", link, target, err)
	}
	if IsDenyPath(link) {
		t.Errorf("IsDenyPath(%q -> %q) = true, want false", link, target)
	}
}
