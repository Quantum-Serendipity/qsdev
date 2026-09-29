package devinit

import (
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	qsdevconfig "github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestCreate_DefaultTierMatchesExplicitTier is the regression test for
// `init --yes` without --tier building full-tier output because the default
// MCP servers made the legacy inference pick full: the default path must
// record the catalog's default tier and generate exactly what an explicit
// `--tier <default>` does, including the committed .qsdev.yaml.
func TestCreate_DefaultTierMatchesExplicitTier(t *testing.T) {
	t.Parallel()
	defaultTier := catalog.MustDefault().DefaultTier()
	if defaultTier == "" {
		t.Fatal("catalog declares no default tier")
	}

	// One project dir, so the project name derived from it cannot differ.
	dir := newGoProject(t)
	implicit := createAnswers(t, dir, "--lang", "go")
	explicit := createAnswers(t, dir, "--lang", "go", "--tier", defaultTier)

	if implicit.Tier != defaultTier {
		t.Errorf("default create Tier = %q, want catalog default %q", implicit.Tier, defaultTier)
	}
	if implicit.PermissionLevel != explicit.PermissionLevel {
		t.Errorf("PermissionLevel: default %q, --tier %s %q", implicit.PermissionLevel, defaultTier, explicit.PermissionLevel)
	}

	implicitCfg := qsdevconfig.AnswersToConfig(implicit, "test")
	explicitCfg := qsdevconfig.AnswersToConfig(explicit, "test")
	if implicitCfg.Tier != defaultTier {
		t.Errorf(".qsdev.yaml tier = %q, want %q", implicitCfg.Tier, defaultTier)
	}
	if !reflect.DeepEqual(implicitCfg, explicitCfg) {
		t.Errorf(".qsdev.yaml differs between default and --tier %s:\ndefault: %+v\nexplicit: %+v", defaultTier, implicitCfg, explicitCfg)
	}

	implicitFiles := generatedContent(t, implicit)
	explicitFiles := generatedContent(t, explicit)
	implicitPaths := slices.Sorted(maps.Keys(implicitFiles))
	explicitPaths := slices.Sorted(maps.Keys(explicitFiles))
	if !slices.Equal(implicitPaths, explicitPaths) {
		t.Fatalf("generated file set differs between default and --tier %s:\ndefault:  %v\nexplicit: %v", defaultTier, implicitPaths, explicitPaths)
	}
	for _, path := range implicitPaths {
		if implicitFiles[path] != explicitFiles[path] {
			t.Errorf("%s differs between default and --tier %s", path, defaultTier)
		}
	}
}

// TestPermissionSummary verifies the plan preview names the preset settings
// generation applies: the explicit level, else the tier's preset.
func TestPermissionSummary(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		answers types.WizardAnswers
		want    string
	}{
		{"explicit level wins", types.WizardAnswers{PermissionLevel: "minimal", Tier: "full"}, "minimal permissions"},
		{"standard tier preset", types.WizardAnswers{Tier: "standard"}, "standard permissions"},
		{"supply-chain-only tier preset", types.WizardAnswers{Tier: "supply-chain-only"}, "supply-chain-only permissions"},
		{"nothing set", types.WizardAnswers{}, "standard permissions"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := permissionSummary(tt.answers); got != tt.want {
				t.Errorf("permissionSummary = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestTierFlagUsage verifies --tier help names every catalog tier and the
// catalog's default tier rather than a hardcoded list.
func TestTierFlagUsage(t *testing.T) {
	t.Parallel()
	cat := catalog.MustDefault()
	usage := tierFlagUsage()
	for _, name := range cat.TierOrder() {
		if !strings.Contains(usage, name) {
			t.Errorf("--tier usage %q omits tier %q", usage, name)
		}
	}
	if want := "(default: " + cat.DefaultTier() + ")"; !strings.Contains(usage, want) {
		t.Errorf("--tier usage %q omits %q", usage, want)
	}
}
