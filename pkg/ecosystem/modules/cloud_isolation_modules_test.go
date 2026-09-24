package modules

import (
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/cloudcommon"
)

// TestCloudProviders_MatchModules ties the static cloud isolation check to
// the modules: every cloudcommon provider is a registered module (so a
// project that configures it is assessed) whose generated deny and read-deny
// rules are exactly the ones the fail-safe validator requires.
func TestCloudProviders_MatchModules(t *testing.T) {
	t.Parallel()

	reg := ecosystem.DefaultRegistry()
	for _, p := range cloudcommon.Providers() {
		t.Run(string(p), func(t *testing.T) {
			t.Parallel()
			mod, ok := reg.ByName(string(p))
			if !ok {
				t.Fatalf("no ecosystem module named %q", p)
			}
			drp, ok := mod.(ecosystem.DenyRuleProvider)
			if !ok || !slices.Equal(drp.DenyRules(ecosystem.ModuleConfig{}), cloudcommon.BashDenyRules(p)) {
				t.Errorf("module %q deny rules differ from cloudcommon.BashDenyRules", p)
			}
			rdrp, ok := mod.(ecosystem.ReadDenyRuleProvider)
			if !ok || !slices.Equal(rdrp.ReadDenyRules(ecosystem.ModuleConfig{}), cloudcommon.ReadDenyPaths(p)) {
				t.Errorf("module %q read-deny rules differ from cloudcommon.ReadDenyPaths", p)
			}
		})
	}
}
