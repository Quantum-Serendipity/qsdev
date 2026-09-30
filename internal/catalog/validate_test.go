package catalog

import (
	"fmt"
	"strings"
	"testing"
)

// hostileHookIDs are hook ids that are not plain Nix identifiers. Rendered
// as git-hooks.hooks.<id> in devenv.nix, the first two would splice extra
// attributes into the generated file (for example demoting ripsecrets to a
// manual-only hook); the rest would emit broken Nix.
var hostileHookIDs = []string{
	`zz.enable = true; ripsecrets.stages = [ "manual" ]; yy`,
	`evil = { enable = false; }; x`,
	"a.b",
	"a b",
	"",
	"with",
}

// Every hook id section of the catalog rejects an id that is not a plain Nix
// identifier, naming defaults.yaml and the section.
func TestValidate_HookIDGrammar(t *testing.T) {
	t.Parallel()

	sections := []struct {
		name      string
		wantField string
		set       func(c *Catalog, id string)
	}{
		{
			name:      "security_hooks",
			wantField: "security_hooks",
			set: func(c *Catalog, id string) {
				c.security.Hooks.Default = append(c.security.Hooks.Default, id)
			},
		},
		{
			name:      "hook_tiers",
			wantField: "hook_tiers.enhanced",
			set: func(c *Catalog, id string) {
				c.hookTiers.Tiers["enhanced"] = append(c.hookTiers.Tiers["enhanced"], id)
			},
		},
		{
			name:      "custom_hooks",
			wantField: "custom_hooks[0].id",
			set: func(c *Catalog, id string) {
				c.security.CustomHooks[0].ID = id
			},
		},
		{
			name:      "compliance",
			wantField: "compliance.baseline.required_pre_commit_hooks",
			set: func(c *Catalog, id string) {
				lvl := c.compliance.Levels["baseline"]
				lvl.RequiredPreCommitHooks = append(lvl.RequiredPreCommitHooks, id)
				c.compliance.Levels["baseline"] = lvl
			},
		},
	}

	for _, sec := range sections {
		for _, id := range hostileHookIDs {
			t.Run(fmt.Sprintf("%s/%q", sec.name, id), func(t *testing.T) {
				t.Parallel()
				cat := loadTestCatalog(t)
				sec.set(cat, id)

				want := fmt.Sprintf("invalid hook id %q", id)
				for _, e := range cat.Validate() {
					if e.File == "defaults.yaml" && e.Field == sec.wantField && strings.HasPrefix(e.Message, want) {
						return
					}
				}
				t.Errorf("Validate() = %v, want defaults.yaml: %s: %s", cat.Validate(), sec.wantField, want)
			})
		}
	}
}

// The embedded catalog's hook ids, including the hyphenated ones, all pass.
func TestValidate_HookIDGrammarAcceptsEmbedded(t *testing.T) {
	t.Parallel()

	cat := loadTestCatalog(t)
	cat.security.Hooks.Default = append(cat.security.Hooks.Default, "ripsecrets", "check-added-large-files")
	if errs := cat.validateHookIDs("defaults.yaml"); len(errs) > 0 {
		t.Errorf("validateHookIDs() = %v, want none", errs)
	}
}
