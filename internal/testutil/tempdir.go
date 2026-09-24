// Package testutil holds helpers shared by test files across packages.
package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// MarkerFreeTempDir returns a fresh, symlink-resolved temporary directory
// whose ancestors contain nothing isMarker accepts. Tests of upward project
// discovery need this: on Windows the default temp directory lives inside the
// real user profile (C:\Users\<name>\AppData\Local\Temp), so a walk up from
// t.TempDir() reaches the profile and any qsdev data directory in it, which a
// test that points HOME/USERPROFILE elsewhere cannot exclude. When the default
// temp directory is unsuitable it tries RUNNER_TEMP (outside the profile on
// hosted CI runners); if no candidate works the test is skipped, naming the
// ancestor that got in the way.
func MarkerFreeTempDir(t testing.TB, isMarker func(dir string) bool) string {
	t.Helper()
	candidates := []string{""} // "" is the default temp directory
	if rt := os.Getenv("RUNNER_TEMP"); rt != "" {
		candidates = append(candidates, rt)
	}
	var blocked string
	for _, base := range candidates {
		dir := tempDirIn(t, base)
		if a := markedAncestor(dir, isMarker); a != "" {
			blocked = a
			continue
		}
		return dir
	}
	t.Skipf("no temporary directory without project markers above it (%s has one); cannot isolate a walk-up test here", blocked)
	return ""
}

func tempDirIn(t testing.TB, base string) string {
	t.Helper()
	dir := ""
	if base == "" {
		dir = t.TempDir()
	} else {
		d, err := os.MkdirTemp(base, "qsdev-test-*")
		if err != nil {
			t.Fatalf("creating temp dir in %s: %v", base, err)
		}
		t.Cleanup(func() { _ = os.RemoveAll(d) })
		dir = d
	}
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("resolving temp dir %s: %v", dir, err)
	}
	return resolved
}

// markedAncestor returns the nearest strict ancestor of dir that isMarker
// accepts, or "" when there is none.
func markedAncestor(dir string, isMarker func(dir string) bool) string {
	for d := filepath.Dir(dir); ; d = filepath.Dir(d) {
		if isMarker(d) {
			return d
		}
		if parent := filepath.Dir(d); parent == d {
			return ""
		}
	}
}
