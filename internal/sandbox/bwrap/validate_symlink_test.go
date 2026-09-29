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

// TestValidateMountPath_RejectsAbsentEntryUnderSymlinkedParent reproduces the
// macOS layout that TestValidateMountPath_RejectsResolvedHomeUnderSymlinkedHome
// only hits on a real Mac: the home directory itself sits below a symlinked
// directory (/var -> /private/var), HOME is a further symlink to it, and the
// mount path names an absent deny entry through the unresolved parent. Both
// sides must resolve through their deepest existing ancestor to meet.
func TestValidateMountPath_RejectsAbsentEntryUnderSymlinkedParent(t *testing.T) {
	root := t.TempDir()
	privateVar := filepath.Join(root, "private", "var")
	if err := os.MkdirAll(filepath.Join(privateVar, "folders", "home"), 0o700); err != nil {
		t.Fatalf("creating fake /private/var: %v", err)
	}
	varLink := filepath.Join(root, "var")
	if err := os.Symlink(privateVar, varLink); err != nil {
		t.Fatalf("creating /var symlink: %v", err)
	}
	homeViaVar := filepath.Join(varLink, "folders", "home") // literal, like /var/folders/...
	homeLink := filepath.Join(root, "homelink")
	if err := os.Symlink(homeViaVar, homeLink); err != nil {
		t.Fatalf("creating home symlink: %v", err)
	}
	t.Setenv("HOME", homeLink)

	absent := filepath.Join(homeViaVar, ".aws") // does not exist
	if err := ValidateMountPath(absent); err == nil {
		t.Errorf("ValidateMountPath(%q) with HOME=%q = nil, want a deny error", absent, homeLink)
	}
}

// TestValidateMountPath_RejectsResolvedHomeUnderSymlinkedHome reproduces the
// macOS layout on any host: HOME is reached through a symlink (as /var is a
// symlink to /private/var on macOS), so the deny entries built from HOME are
// literal while a mount path may name the resolved location directly.
func TestValidateMountPath_RejectsResolvedHomeUnderSymlinkedHome(t *testing.T) {
	realHome := t.TempDir()
	homeLink := filepath.Join(t.TempDir(), "home")
	if err := os.Symlink(realHome, homeLink); err != nil {
		t.Fatalf("creating home symlink: %v", err)
	}
	t.Setenv("HOME", homeLink)
	if err := os.Mkdir(filepath.Join(realHome, ".ssh"), 0o700); err != nil {
		t.Fatalf("creating fake ~/.ssh: %v", err)
	}

	for _, path := range []string{
		filepath.Join(realHome, ".ssh"), // the credential store itself
		filepath.Join(realHome, ".aws"), // an absent entry under the resolved home
		realHome,                        // an ancestor that would re-expose ~/.ssh
	} {
		if err := ValidateMountPath(path); err == nil {
			t.Errorf("ValidateMountPath(%q) with HOME=%q = nil, want a deny error", path, homeLink)
		}
	}
}
