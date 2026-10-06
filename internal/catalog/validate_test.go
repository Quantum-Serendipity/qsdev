package catalog

import (
	"fmt"
	"maps"
	"slices"
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

// A compliance level's required hook that is not a tool must be always on
// (security_hooks or custom_hooks): hook_tiers membership only filters
// hooks, it never renders one, so a hook found only there would never run.
func TestValidate_RequiredNonToolHookMustBeAlwaysOn(t *testing.T) {
	t.Parallel()

	if errs := loadTestCatalog(t).validateComplianceHooks(); len(errs) > 0 {
		t.Fatalf("embedded catalog: validateComplianceHooks() = %v, want none", errs)
	}

	cat := loadTestCatalog(t)
	cat.hookTiers.Tiers = mergeStringSliceMap(cat.hookTiers.Tiers, nil)
	cat.hookTiers.Tiers["baseline"] = append(cat.hookTiers.Tiers["baseline"], "tier-only-hook")
	strict := cat.compliance.Levels["strict"]
	strict.RequiredPreCommitHooks = append(slices.Clone(strict.RequiredPreCommitHooks), "tier-only-hook")
	cat.compliance.Levels = maps.Clone(cat.compliance.Levels)
	cat.compliance.Levels["strict"] = strict

	errs := cat.Validate()
	var found bool
	for _, e := range errs {
		if e.Field == "strict.required_pre_commit_hooks" && strings.Contains(e.Message, `"tier-only-hook"`) {
			found = true
		}
	}
	if !found {
		t.Errorf("Validate() = %v, want an error naming strict.required_pre_commit_hooks and tier-only-hook", errs)
	}
}
