package ecosystem

import (
	"slices"
	"testing"
)

// TestOrderedLockFiles_ManifestsLast checks every catalog ecosystem: no
// dedicated lockfile is ever ordered after a dual-role manifest, and no entry
// is lost.
func TestOrderedLockFiles_ManifestsLast(t *testing.T) {
	t.Parallel()
	for eco, lockfiles := range LockFilesByEcosystem {
		t.Run(eco, func(t *testing.T) {
			t.Parallel()
			got := OrderedLockFiles(eco)
			if len(got) != len(lockfiles) {
				t.Fatalf("OrderedLockFiles(%q) = %v, lost entries from %v", eco, got, lockfiles)
			}
			seenManifest := false
			for _, name := range got {
				isManifest := slices.Contains(ManifestsByEcosystem[eco], name)
				if seenManifest && !isManifest {
					t.Errorf("dedicated lockfile %q ordered after a manifest in %v", name, got)
				}
				seenManifest = seenManifest || isManifest
			}
		})
	}
}

func TestOrderedLockFiles_Python(t *testing.T) {
	t.Parallel()
	catalog := slices.Clone(LockFilesByEcosystem[NamePython])
	got := OrderedLockFiles(NamePython)
	if got[len(got)-1] != "requirements.txt" {
		t.Errorf("OrderedLockFiles(python) = %v, want requirements.txt last", got)
	}
	if !slices.Equal(LockFilesByEcosystem[NamePython], catalog) {
		t.Error("OrderedLockFiles mutated the catalog slice")
	}
}
