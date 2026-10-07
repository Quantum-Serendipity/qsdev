package denylist

import (
	"os"
	"path/filepath"
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

// TestOverlaps_CaseFold pins that the mount validators' containment checks
// compare names the way the host filesystem does: on a case-folding
// filesystem (macOS, Windows) ~/.SSH is ~/.ssh, so binding it must be
// refused. The folding options are injected so the check runs on every OS.
// Not parallel: it swaps the package's match options.
func TestOverlaps_CaseFold(t *testing.T) {
	saved := matchOptions
	t.Cleanup(func() { matchOptions = saved })

	tests := []struct {
		name               string
		opts               pathmatch.Options
		path, deny         string
		overlaps, ancestor bool
	}{
		{"fold upper deny dir", pathmatch.Options{FoldCase: true}, "/home/me/.SSH", "/home/me/.ssh", true, false},
		{"fold descendant", pathmatch.Options{FoldCase: true}, "/home/me/.SSH/id_rsa", "/home/me/.ssh", true, false},
		{"fold home ancestor", pathmatch.Options{FoldCase: true}, "/HOME/ME", "/home/me/.ssh", false, true},
		{"windows drive and aliases", pathmatch.Options{FoldCase: true, WindowsAliases: true}, "c:/users/me/.ssh./id_rsa", "C:/Users/me/.ssh", true, false},
		{"exact keeps cases distinct", pathmatch.Options{}, "/home/me/.SSH", "/home/me/.ssh", false, false},
		{"exact ancestor", pathmatch.Options{}, "/home/me", "/home/me/.ssh", false, true},
		{"root is ancestor", pathmatch.Options{}, "/", "/etc/shadow", false, true},
		{"component boundary", pathmatch.Options{FoldCase: true}, "/home/me/.SSHX", "/home/me/.ssh", false, false},
	}
	for _, tt := range tests {
		matchOptions = tt.opts
		if got := Overlaps(tt.path, tt.deny); got != tt.overlaps {
			t.Errorf("%s: Overlaps(%q, %q) under %+v = %v, want %v", tt.name, tt.path, tt.deny, tt.opts, got, tt.overlaps)
		}
		if got := IsStrictAncestor(tt.path, tt.deny); got != tt.ancestor {
			t.Errorf("%s: IsStrictAncestor(%q, %q) under %+v = %v, want %v", tt.name, tt.path, tt.deny, tt.opts, got, tt.ancestor)
		}
	}
}
