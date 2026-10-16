package devinit

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
)

// mustCatalogProjectProfiles builds the built-in profile registry from the
// default catalog, failing the test on any error.
func mustCatalogProjectProfiles(t *testing.T) *ProjectProfileRegistry {
	t.Helper()
	cat, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	r, err := catalogProjectProfiles(cat)
	if err != nil {
		t.Fatalf("loading built-in profiles: %v", err)
	}
	return r
}

func TestCatalogProjectProfiles_NoErrors(t *testing.T) {
	t.Parallel()
	cat, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	r := mustCatalogProjectProfiles(t)
	if got, want := len(r.Names()), len(cat.ProjectProfiles()); got != want {
		t.Errorf("registered %d profiles, want %d", got, want)
	}
}

// TestCatalogProfile_UnknownNameFails is the regression test for an unknown or
// renamed catalog profile resolving to an empty Profile instead of failing.
func TestCatalogProfile_UnknownNameFails(t *testing.T) {
	t.Parallel()
	cat, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := catalogProfile(cat, "no-such-profile"); err == nil {
		t.Fatal("expected an error for an unknown catalog profile")
	}
}

// TestCatalogProfile_DoesNotAliasCatalog verifies mutating a returned profile
// cannot corrupt the shared catalog definition.
func TestCatalogProfile_DoesNotAliasCatalog(t *testing.T) {
	t.Parallel()
	cat, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	p, err := catalogProfile(cat, "go-web")
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Services) == 0 || len(p.Hooks) == 0 {
		t.Fatal("go-web should declare services and hooks")
	}
	p.Services[0] = "mutated"
	p.Hooks[0] = "mutated"

	again, err := catalogProfile(cat, "go-web")
	if err != nil {
		t.Fatal(err)
	}
	if again.Services[0] == "mutated" || again.Hooks[0] == "mutated" {
		t.Error("catalogProfile returned slices aliasing the catalog definition")
	}
}
