package denylist_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/pathmatch"
	"github.com/Quantum-Serendipity/qsdev/internal/secrets"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules"
)

// TestCredentialPathsCoverModuleReadDenyRules derives the expected sandbox
// deny list from the ecosystem modules: every home credential file a module
// read-denies to the agent must also be kept out of sandbox mounts. Before
// this, Cargo and NuGet credentials were protected by neither.
func TestCredentialPathsCoverModuleReadDenyRules(t *testing.T) {
	t.Parallel()

	rels := secrets.CredentialPaths()
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
			if !slices.ContainsFunc(rels, func(deny string) bool { return pathmatch.Within(rel, deny) }) {
				t.Errorf("module %s read-denies %q, but no secrets.CredentialPaths entry covers it", mod.Name(), rule)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no module contributed a home read-deny rule")
	}
}

// TestModuleReadDenyRulesDoNotDuplicateCredentialCanon keeps the credential
// canon the single owner of the home credential stores: the generated
// settings read-deny every secrets.CredentialPaths entry at every preset, so
// a module that read-denies a whole canon store again is a private copy that
// can drift. A module may still name a path inside a canon store (the cloud
// modules' ~/.aws/credentials), since that is narrower than the store.
func TestModuleReadDenyRulesDoNotDuplicateCredentialCanon(t *testing.T) {
	t.Parallel()

	rels := secrets.CredentialPaths()
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
			if slices.Contains(rels, rel) {
				t.Errorf("module %s read-denies %q, a copy of the secrets.CredentialPaths entry %q", mod.Name(), rule, rel)
			}
		}
	}
}
