package devenv

import (
	"slices"
	"testing"
)

// withCatalogHookExcludes adds the catalog's excludes to a module's override
// of the same hook (keeping its scoping, without duplicates), creates an
// override for a hook no module scoped, and leaves other hooks alone.
func TestWithCatalogHookExcludes(t *testing.T) {
	t.Parallel()

	moduleExcludes := []string{`(^|/)vendor/`}
	overrides := []HookOverrideData{
		{ID: "shellcheck", ExcludeTypes: []string{"zsh"}, Excludes: moduleExcludes},
	}
	catalogExcludes := map[string][]string{
		"shellcheck": {`(^|/)vendor/`, `^third_party/`},
		"ripsecrets": {`(^|/)vendor/`},
	}
	got := withCatalogHookExcludes(overrides, []string{"ripsecrets", "shellcheck", "statix"},
		func(id string) []string { return slices.Clone(catalogExcludes[id]) })

	want := []HookOverrideData{
		{ID: "shellcheck", ExcludeTypes: []string{"zsh"}, Excludes: []string{`(^|/)vendor/`, `^third_party/`}},
		{ID: "ripsecrets", Excludes: []string{`(^|/)vendor/`}},
	}
	if !slices.EqualFunc(got, want, func(a, b HookOverrideData) bool {
		return a.ID == b.ID && slices.Equal(a.TypesOr, b.TypesOr) &&
			slices.Equal(a.ExcludeTypes, b.ExcludeTypes) && slices.Equal(a.Excludes, b.Excludes)
	}) {
		t.Errorf("withCatalogHookExcludes() = %+v, want %+v", got, want)
	}
	if !slices.Equal(moduleExcludes, []string{`(^|/)vendor/`}) {
		t.Errorf("module's exclude slice was modified: %q", moduleExcludes)
	}
}
