package devenv

import (
	"reflect"
	"slices"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestBuildDevenvNixData_CatalogWiring checks that the generated devenv.nix
// data and devenv.yaml take the catalog's base packages, security hooks,
// unset and kept environment variables as they are, for a project with no
// languages, services or hook tier that would add to or filter them.
func TestBuildDevenvNixData_CatalogWiring(t *testing.T) {
	t.Parallel()
	cat := catalog.MustDefault()
	answers := types.WizardAnswers{ProjectName: "wiring"}

	data, err := BuildDevenvNixData(answers, nil)
	if err != nil {
		t.Fatalf("BuildDevenvNixData: %v", err)
	}
	base := cat.BasePackages()
	if len(base) == 0 {
		t.Fatal("catalog has no base packages")
	}
	if len(data.Packages) < len(base) || !slices.Equal(data.Packages[:len(base)], base) {
		t.Errorf("Packages = %v, want catalog BasePackages() %v as its prefix", data.Packages, base)
	}
	if want := cat.UnsetVars(); !reflect.DeepEqual(data.UnsetEnvVars, want) {
		t.Errorf("UnsetEnvVars = %v, want catalog UnsetVars() %v", data.UnsetEnvVars, want)
	}
	if want := cat.SecurityHooks(); !reflect.DeepEqual(data.SecurityHooks, want) {
		t.Errorf("SecurityHooks = %v, want catalog SecurityHooks() %v", data.SecurityHooks, want)
	}

	file, err := GenerateDevenvYaml(answers, nil)
	if err != nil {
		t.Fatalf("GenerateDevenvYaml: %v", err)
	}
	var dy DevenvYaml
	if err := yaml.Unmarshal(file.Content, &dy); err != nil {
		t.Fatalf("parsing devenv.yaml: %v", err)
	}
	if want := cat.KeepVars(); !reflect.DeepEqual(dy.Clean.Keep, want) {
		t.Errorf("devenv.yaml clean.keep = %v, want catalog KeepVars() %v", dy.Clean.Keep, want)
	}
}

func TestDefaultSpecializedHooks_MatchesCatalog(t *testing.T) {
	t.Parallel()
	cat := catalog.MustDefault()
	got := specializedHooks(cat, nil)
	catHooks := cat.CustomHooks()

	if len(got) != len(catHooks) {
		t.Fatalf("specializedHooks() count = %d, want catalog.CustomHooks() count = %d", len(got), len(catHooks))
	}

	catIDs := make(map[string]bool, len(catHooks))
	for _, h := range catHooks {
		catIDs[h.ID] = true
	}
	for _, h := range got {
		if !catIDs[h.ID] {
			t.Errorf("specializedHooks() has ID %q not found in catalog.CustomHooks()", h.ID)
		}
	}
}
