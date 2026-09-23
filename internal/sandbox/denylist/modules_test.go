package denylist_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox/denylist"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules"
)

// TestHomeDenyRelPathsCoverModuleReadDenyRules derives the expected sandbox
// deny list from the ecosystem modules: every home credential file a module
// read-denies to the agent must also be kept out of sandbox mounts. Before
// this, Cargo and NuGet credentials were protected by neither.
func TestHomeDenyRelPathsCoverModuleReadDenyRules(t *testing.T) {
	t.Parallel()

	rels := denylist.HomeDenyRelPaths()
	var checked int
	for _, mod := range ecosystem.DefaultRegistry().All() {
		rdp, ok := mod.(ecosystem.ReadDenyRuleProvider)
		if !ok {
			continue
		}
		for _, rule := range rdp.ReadDenyRules(ecosystem.ModuleConfig{}) {
			rel, ok := strings.CutPrefix(rule, "~/")
			if !ok {
				continue
			}
			rel = strings.TrimSuffix(strings.TrimSuffix(rel, "/**"), "/*")
			checked++
			if !slices.ContainsFunc(rels, func(deny string) bool { return denylist.Overlaps(rel, deny) }) {
				t.Errorf("module %s read-denies %q, but no HomeDenyRelPaths entry covers it", mod.Name(), rule)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no module contributed a home read-deny rule")
	}
}
