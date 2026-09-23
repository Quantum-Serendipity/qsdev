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
