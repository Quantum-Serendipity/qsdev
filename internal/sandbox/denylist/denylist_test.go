package denylist

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/pathmatch"
	"github.com/Quantum-Serendipity/qsdev/internal/secrets"
)

func TestSystemDenyPaths(t *testing.T) {
	t.Parallel()

	paths := SystemDenyPaths()
	if len(paths) == 0 {
		t.Fatal("SystemDenyPaths returned empty slice")
	}

	want := map[string]bool{
		"/etc/shadow":    false,
		"/etc/sudoers":   false,
		"/etc/sudoers.d": false,
		"/root":          false,
	}

	for _, p := range paths {
		if _, ok := want[p]; ok {
			want[p] = true
		}
	}

	for path, found := range want {
		if !found {
			t.Errorf("expected %q in SystemDenyPaths", path)
		}
	}
}

// TestHomeDenyPaths_ExpandsCredentialPaths pins that the sandbox home deny
// list is exactly the credential canon (secrets.CredentialPaths) joined with
// the home directory, in order, with nothing added or dropped.
func TestHomeDenyPaths_ExpandsCredentialPaths(t *testing.T) {
	t.Parallel()

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("getting home dir: %v", err)
	}

	rels := secrets.CredentialPaths()
	paths := HomeDenyPaths()
	if len(paths) == 0 || len(paths) != len(rels) {
		t.Fatalf("HomeDenyPaths has %d entries, want one per CredentialPaths entry (%d)", len(paths), len(rels))
	}
	for i, rel := range rels {
		if want := filepath.Join(home, filepath.FromSlash(rel)); paths[i] != want {
			t.Errorf("HomeDenyPaths()[%d] = %q, want %q", i, paths[i], want)
		}
	}
}

func TestAllDenyPaths(t *testing.T) {
	t.Parallel()

	all := AllDenyPaths()
	sys := SystemDenyPaths()
	hom := HomeDenyPaths()

	if len(all) != len(sys)+len(hom) {
		t.Errorf("AllDenyPaths length %d != SystemDenyPaths(%d) + HomeDenyPaths(%d)",
			len(all), len(sys), len(hom))
	}

	// System paths should appear first.
	for i, p := range sys {
		if all[i] != p {
			t.Errorf("AllDenyPaths[%d] = %q, want %q", i, all[i], p)
		}
	}

	// Home paths should follow.
	for i, p := range hom {
		if all[len(sys)+i] != p {
			t.Errorf("AllDenyPaths[%d] = %q, want %q", len(sys)+i, all[len(sys)+i], p)
		}
	}
}

// setHome points the home directory at dir for the rest of the test (HOME on
// Unix, USERPROFILE on Windows). Callers must not be parallel.
func setHome(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
}

// withMatchOptions swaps the package's match options for the rest of the
// test. Callers must not be parallel.
func withMatchOptions(t *testing.T, opts pathmatch.Options) {
	t.Helper()
	saved := matchOptions
	t.Cleanup(func() { matchOptions = saved })
	matchOptions = opts
}

var (
	exactMatch   = pathmatch.Options{}
	foldMatch    = pathmatch.Options{FoldCase: true}
	windowsMatch = pathmatch.Options{FoldCase: true, WindowsAliases: true}
)

// TestCovering pins that Covering compares names the way the host filesystem
// does: on a case-folding filesystem (macOS, Windows) ~/.SSH is ~/.ssh, so
// ~/.SSH/id_rsa is covered by the ~/.ssh entry, and Windows ignores trailing
// dots. The match options are injected so every form runs on every OS. Not
// parallel: it sets the home directory and swaps the package's match options.
func TestCovering(t *testing.T) {
	home := t.TempDir()
	setHome(t, home)
	ssh := filepath.Join(home, ".ssh")

	tests := []struct {
		name     string
		opts     pathmatch.Options
		path     string
		wantDeny string // empty means not covered
	}{
		{"fold upper deny dir", foldMatch, filepath.Join(home, ".SSH"), ssh},
		{"fold descendant", foldMatch, filepath.Join(home, ".SSH", "id_rsa"), ssh},
		{"windows aliases", windowsMatch, filepath.Join(home, ".ssh.", "id_rsa"), ssh},
		{"exact keeps cases distinct", exactMatch, filepath.Join(home, ".SSH", "id_rsa"), ""},
		{"exact entry", exactMatch, ssh, ssh},
		{"system entry", exactMatch, "/etc/shadow", "/etc/shadow"},
		{"component boundary", foldMatch, filepath.Join(home, ".SSHX"), ""},
		{"ancestor is not covered", foldMatch, home, ""},
		{"unrelated path", foldMatch, filepath.Join(home, "sdks", "android"), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withMatchOptions(t, tt.opts)
			deny, ok := Covering(tt.path)
			if ok != (tt.wantDeny != "") || deny != tt.wantDeny {
				t.Errorf("Covering(%q) under %+v = (%q, %v), want %q", tt.path, tt.opts, deny, ok, tt.wantDeny)
			}
		})
	}
}

// TestConflict_CaseFold pins how Conflict classifies a path against the deny
// list: one equal to or under an entry is covered (exposes=false), one that is
// a strict ancestor of an entry would re-expose it (exposes=true), and an
// unrelated one is clear. The fold and Windows-alias forms are injected so
// they run on every OS. Not parallel: it sets the home directory and swaps the
// package's match options.
func TestConflict_CaseFold(t *testing.T) {
	home := t.TempDir()
	setHome(t, home)
	ssh := filepath.Join(home, ".ssh")

	tests := []struct {
		name        string
		opts        pathmatch.Options
		path        string
		ok, exposes bool
	}{
		{"fold descendant is covered", foldMatch, filepath.Join(home, ".SSH", "id_rsa"), true, false},
		{"windows alias descendant is covered", windowsMatch, filepath.Join(home, ".SSH.", "id_rsa"), true, false},
		{"entry itself is covered", exactMatch, ssh, true, false},
		{"home exposes", exactMatch, home, true, true},
		{"fold upper home exposes", foldMatch, strings.ToUpper(home), true, true},
		{"windows alias home exposes", windowsMatch, home + ".", true, true},
		{"exact upper home is clear", exactMatch, strings.ToUpper(home), false, false},
		{"root exposes", exactMatch, "/", true, true},
		{"root exposes under fold", foldMatch, "/", true, true},
		{"etc exposes shadow", exactMatch, "/etc", true, true},
		{"sdks is clear", foldMatch, filepath.Join(home, "sdks"), false, false},
		{"sdks is clear under windows", windowsMatch, filepath.Join(home, "sdks"), false, false},
		{"component boundary is clear", foldMatch, filepath.Join(home, ".SSHX"), false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withMatchOptions(t, tt.opts)
			deny, exposes, ok := Conflict(tt.path)
			if ok != tt.ok || exposes != tt.exposes {
				t.Fatalf("Conflict(%q) under %+v = (%q, %v, %v), want exposes=%v ok=%v",
					tt.path, tt.opts, deny, exposes, ok, tt.exposes, tt.ok)
			}
			switch {
			case !ok:
				if deny != "" {
					t.Errorf("Conflict(%q) = %q with ok=false, want empty", tt.path, deny)
				}
			case exposes:
				if !matchOptions.StrictlyWithin(deny, tt.path) {
					t.Errorf("Conflict(%q) exposes %q, which is not below the path", tt.path, deny)
				}
			default:
				if covered, _ := Covering(tt.path); covered != deny {
					t.Errorf("Conflict(%q) = %q, but Covering = %q", tt.path, deny, covered)
				}
			}
		})
	}
}

// TestConflict_RootExposesEveryEntry pins that the filesystem root re-exposes
// the whole deny list: binding "/" would show every entry inside the sandbox.
// Not parallel: it sets the home directory.
func TestConflict_RootExposesEveryEntry(t *testing.T) {
	setHome(t, t.TempDir())
	for _, deny := range ExpandedDenyPaths() {
		if !matchOptions.StrictlyWithin(deny, "/") {
			t.Errorf("deny entry %q is not below /", deny)
		}
	}
	if _, exposes, ok := Conflict("/"); !ok || !exposes {
		t.Errorf("Conflict(/) = exposes=%v ok=%v, want both true", exposes, ok)
	}
}

// TestConflict_SingleResolutionPass pins that one Conflict call resolves the
// deny list and the path's candidates exactly once, shared by its covering and
// exposing checks: each lookup is a filesystem call per catalog entry, paid on
// every mount validation. Not parallel: it sets the home directory and swaps
// the package's symlink resolver.
func TestConflict_SingleResolutionPass(t *testing.T) {
	setHome(t, t.TempDir())
	calls := 0
	saved := evalSymlinks
	t.Cleanup(func() { evalSymlinks = saved })
	evalSymlinks = func(p string) (string, error) {
		calls++
		return saved(p)
	}
	path := filepath.Join(t.TempDir(), "unrelated", "child")

	ExpandedDenyPaths()
	CandidatePaths(path)
	want := calls
	calls = 0

	if deny, exposes, ok := Conflict(path); ok {
		t.Fatalf("Conflict(%q) = (%q, %v, true), want no conflict", path, deny, exposes)
	}
	if want == 0 || calls != want {
		t.Errorf("Conflict made %d symlink lookups, want %d (one ExpandedDenyPaths plus one CandidatePaths)", calls, want)
	}
}
