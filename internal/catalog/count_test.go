package catalog

import "testing"

// The following golden-count tests pin the sizes of the embedded catalog so
// that the numbers quoted in user-facing and internal docs cannot silently
// drift away from the code. Each test names the doc(s) to update if the count
// legitimately changes.

// TestCatalogToolCount pins the number of registered tools in the embedded
// catalog.
//
// If this count changes, update internal-docs/profile-comparison.md
// ("Registered tools" row).
func TestCatalogToolCount(t *testing.T) {
	t.Parallel()

	cat, err := LoadEmbeddedOnly()
	if err != nil {
		t.Fatalf("LoadEmbeddedOnly() error: %v", err)
	}

	const wantTools = 42
	if got := len(cat.Tools()); got != wantTools {
		t.Errorf("catalog tools = %d, want %d; if intentional, update internal-docs/profile-comparison.md",
			got, wantTools)
	}
}

// TestCatalogProjectProfileCount pins the number of project profiles in the
// embedded catalog.
//
// If this count changes, update the count in README.md ("N project profiles")
// and internal-docs/profile-comparison.md ("Project profiles" row).
func TestCatalogProjectProfileCount(t *testing.T) {
	t.Parallel()

	cat, err := LoadEmbeddedOnly()
	if err != nil {
		t.Fatalf("LoadEmbeddedOnly() error: %v", err)
	}

	const wantProjectProfiles = 10
	if got := len(cat.ProjectProfiles()); got != wantProjectProfiles {
		t.Errorf("catalog project profiles = %d, want %d; if intentional, update README.md and internal-docs/profile-comparison.md",
			got, wantProjectProfiles)
	}
}
