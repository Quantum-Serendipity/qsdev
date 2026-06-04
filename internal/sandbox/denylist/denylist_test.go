package denylist

import (
	"os"
	"path/filepath"
	"testing"
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

func TestHomeDenyPaths(t *testing.T) {
	t.Parallel()

	paths := HomeDenyPaths()
	if len(paths) == 0 {
		t.Fatal("HomeDenyPaths returned empty slice")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("getting home dir: %v", err)
	}

	for _, rel := range HomeDenyRelPaths() {
		full := filepath.Join(home, rel)
		found := false
		for _, p := range paths {
			if p == full {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected %q in HomeDenyPaths", full)
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

func TestHomeDenyRelPaths(t *testing.T) {
	t.Parallel()

	rels := HomeDenyRelPaths()
	if len(rels) == 0 {
		t.Fatal("HomeDenyRelPaths returned empty slice")
	}

	// Every relative path should be a suffix of a HomeDenyPaths entry.
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("getting home dir: %v", err)
	}

	expanded := HomeDenyPaths()
	expandedSet := make(map[string]bool, len(expanded))
	for _, p := range expanded {
		expandedSet[p] = true
	}

	for _, rel := range rels {
		full := filepath.Join(home, rel)
		if !expandedSet[full] {
			t.Errorf("HomeDenyRelPaths entry %q does not match any HomeDenyPaths entry (expanded: %q)", rel, full)
		}
	}
}
