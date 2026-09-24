//go:build !windows

package denylist

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// TestCandidatePaths_ResolvesSymlinks pins that a symlink's resolved target is
// part of the deny comparison, so a link such as /tmp/x -> ~/.ssh cannot slip
// a credential store past the mount validators.
func TestCandidatePaths_ResolvesSymlinks(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	target := filepath.Join(dir, "secret-store")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatalf("creating target: %v", err)
	}
	link := filepath.Join(dir, "innocent-link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("creating symlink: %v", err)
	}
	wantTarget, err := filepath.EvalSymlinks(target)
	if err != nil {
		t.Fatalf("resolving target: %v", err)
	}

	got := CandidatePaths(link)

	if !slices.Contains(got, filepath.Clean(link)) {
		t.Errorf("CandidatePaths(%q) = %v, missing the literal path", link, got)
	}
	if !slices.Contains(got, wantTarget) {
		t.Errorf("CandidatePaths(%q) = %v, missing the resolved target %q", link, got, wantTarget)
	}
}

// TestExpandedDenyPaths_IncludesResolvedForms pins that the deny list used for
// comparison carries each entry's symlink-resolved location, including entries
// that do not exist but live under a symlinked directory.
func TestExpandedDenyPaths_IncludesResolvedForms(t *testing.T) {
	realHome := t.TempDir()
	homeLink := filepath.Join(t.TempDir(), "home")
	if err := os.Symlink(realHome, homeLink); err != nil {
		t.Fatalf("creating home symlink: %v", err)
	}
	t.Setenv("HOME", homeLink)
	if err := os.Mkdir(filepath.Join(realHome, ".ssh"), 0o700); err != nil {
		t.Fatalf("creating fake ~/.ssh: %v", err)
	}
	resolvedHome, err := filepath.EvalSymlinks(realHome)
	if err != nil {
		t.Fatalf("resolving home: %v", err)
	}

	got := ExpandedDenyPaths()

	for _, want := range []string{
		filepath.Join(homeLink, ".ssh"),     // literal entry is kept
		filepath.Join(resolvedHome, ".ssh"), // existing entry, resolved
		filepath.Join(resolvedHome, ".aws"), // absent entry, resolved parent
		filepath.Join(resolvedHome, ".config", "gcloud"),
	} {
		if !slices.Contains(got, want) {
			t.Errorf("ExpandedDenyPaths() missing %q; got %v", want, got)
		}
	}
	seen := make(map[string]bool, len(got))
	for _, p := range got {
		if seen[p] {
			t.Errorf("ExpandedDenyPaths() has duplicate %q", p)
		}
		seen[p] = true
	}
}
