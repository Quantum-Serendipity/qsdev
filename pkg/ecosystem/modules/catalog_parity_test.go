package modules

import (
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// nameDiff returns the names in want that are absent from got (missing) and
// the names in got that are absent from want (extra), each sorted.
func nameDiff(want, got []string) (missing, extra []string) {
	for _, n := range want {
		if !slices.Contains(got, n) {
			missing = append(missing, n)
		}
	}
	for _, n := range got {
		if !slices.Contains(want, n) {
			extra = append(extra, n)
		}
	}
	slices.Sort(missing)
	slices.Sort(extra)
	return missing, extra
}

// moduleNames returns the sorted names of mods.
func moduleNames(mods []ecosystem.EcosystemModule) []string {
	names := make([]string, 0, len(mods))
	for _, m := range mods {
		names = append(names, m.Name())
	}
	slices.Sort(names)
	return names
}

// TestCatalogLanguagesMatchRegistry guards against drift between the
// embedded catalog's languages section (internal/catalog/defaults.yaml) and
// the ecosystem modules registered in DefaultRegistry. The catalog list backs
// config validation, `devenv add-language` and shell completion, so a module
// missing from it is rejected everywhere even though it is detected and
// generated. The guard subtests prove the comparison fails when the catalog
// drops a module name or carries a name no module registers.
func TestCatalogLanguagesMatchRegistry(t *testing.T) {
	t.Parallel()

	cat, err := catalog.LoadEmbeddedOnly()
	if err != nil {
		t.Fatalf("catalog.LoadEmbeddedOnly: %v", err)
	}
	reg := ecosystem.DefaultRegistry()
	registry := reg.Names()

	t.Run("parity", func(t *testing.T) {
		t.Parallel()
		checks := []struct {
			name string
			want []string
			got  []string
		}{
			{"languages.all vs registry", registry, cat.Languages()},
			{"languages.core vs tier-1 modules", moduleNames(reg.ByTier(1)), cat.CoreLanguages()},
		}
		for _, c := range checks {
			missing, extra := nameDiff(c.want, c.got)
			if len(missing) > 0 || len(extra) > 0 {
				t.Errorf("%s: catalog missing %v, catalog extra %v", c.name, missing, extra)
			}
		}
	})

	if len(registry) < 2 {
		t.Fatalf("registry has %d modules; need at least 2", len(registry))
	}
	guards := []struct {
		name        string
		catalog     []string
		wantMissing []string
		wantExtra   []string
	}{
		{"equal", slices.Clone(registry), nil, nil},
		{"missing first", slices.Clone(registry[1:]), registry[:1], nil},
		{"missing last", slices.Clone(registry[:len(registry)-1]), registry[len(registry)-1:], nil},
		{"extra bogus", append(slices.Clone(registry), "bogus"), nil, []string{"bogus"}},
		{"missing and extra", append(slices.Clone(registry[1:]), "bogus"), registry[:1], []string{"bogus"}},
	}
	for _, tt := range guards {
		t.Run("guard/"+tt.name, func(t *testing.T) {
			t.Parallel()
			missing, extra := nameDiff(registry, tt.catalog)
			if !slices.Equal(missing, tt.wantMissing) {
				t.Errorf("missing = %v, want %v", missing, tt.wantMissing)
			}
			if !slices.Equal(extra, tt.wantExtra) {
				t.Errorf("extra = %v, want %v", extra, tt.wantExtra)
			}
		})
	}
}
