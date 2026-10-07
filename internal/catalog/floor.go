package catalog

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/sliceutil"
)

// ErrOverlayLoosens reports an org defaults file that would lower the
// built-in security floor: keep a credential the catalog strips, weaken a
// built-in compliance level, map a tier to a weaker compliance level, or
// tier an always-on hook out of a security level. The always-on hooks and
// stripped variables cannot be lowered at all, because MergeCatalogs only
// adds to them.
var ErrOverlayLoosens = errors.New("defaults file loosens the built-in security floor")

// securityFloorViolations returns how merged falls below the security floor
// set by floor (the embedded catalog). file names the overlay in each error.
// Every layer may only add to the floor:
//
//   - no variable in unset_vars may also be in keep_vars;
//   - a built-in compliance level keeps its order and is not weakened (see
//     levelWeakenings);
//   - a tier the floor maps to a compliance level never maps to one of lower
//     order, nor to another level weaker than its built-in one;
//   - the built-in hook tiers stay first in hook_tier_order, in order, and
//     no always-on hook (security_hooks, custom_hooks) moves to a higher
//     hook tier, where lower security levels would drop it.
func securityFloorViolations(floor, merged *Catalog, file string) []CatalogError {
	var errs []CatalogError
	add := func(field, format string, args ...any) {
		errs = append(errs, CatalogError{file, field, fmt.Sprintf(format, args...)})
	}

	unset := merged.security.CleanEnvironment.UnsetVars
	for _, v := range merged.security.CleanEnvironment.KeepVars {
		if slices.Contains(unset, v) {
			add("keep_vars", "cannot keep %q: it is a credential the shell strips (unset_vars)", v)
		}
	}

	for _, name := range slices.Sorted(maps.Keys(floor.compliance.Levels)) {
		want, have := floor.compliance.Levels[name], merged.compliance.Levels[name]
		field := "compliance." + name
		if have.Order != want.Order {
			add(field+".order", "order of built-in compliance level cannot change (got %d, built-in %d)", have.Order, want.Order)
		}
		for _, w := range levelWeakenings(merged, want, have) {
			add(field+"."+w.field, "%s", w.msg)
		}
	}

	for _, tier := range slices.Sorted(maps.Keys(floor.derivations.TierToCompliance)) {
		want, have := floor.derivations.TierToCompliance[tier], merged.derivations.TierToCompliance[tier]
		wantLevel := floor.compliance.Levels[want]
		haveLevel, ok := merged.compliance.Levels[have]
		field := sectionTierToCompliance + "." + tier
		switch {
		case !ok || have == want:
			// An unknown level is reported by Validate; a built-in level
			// is held to its own floor above.
		case haveLevel.Order < wantLevel.Order:
			add(field, "cannot lower compliance from %q to %q", want, have)
		default:
			for _, w := range levelWeakenings(merged, wantLevel, haveLevel) {
				add(field, "compliance level %q is weaker than the built-in %q: %s", have, want, w.msg)
			}
		}
	}

	hookTierFloorViolations(floor, merged, add)
	return errs
}

// levelWeakening is one way a compliance level is weaker than another.
type levelWeakening struct{ field, msg string }

// levelWeakenings returns how have demands less than want: a required hook
// dropped, a shorter age gate, script blocking, the Claude audit log or
// license scanning turned off, or a less strict Claude permission preset (as
// c ranks presets; an unranked preset is less strict than any ranked one). mcp_server_policy and sbom_policy have no strictness
// order and are not compared.
func levelWeakenings(c *Catalog, want, have ComplianceLevelDef) []levelWeakening {
	var out []levelWeakening
	var dropped []string
	for _, h := range want.RequiredPreCommitHooks {
		if !slices.Contains(have.RequiredPreCommitHooks, h) {
			dropped = append(dropped, h)
		}
	}
	if len(dropped) > 0 {
		out = append(out, levelWeakening{"required_pre_commit_hooks", "cannot drop required hooks " + strings.Join(dropped, ", ")})
	}
	if have.AgeGatingThresholdHours < want.AgeGatingThresholdHours {
		out = append(out, levelWeakening{"age_gating_threshold_hours",
			fmt.Sprintf("age gate of %d hours is below the built-in %d", have.AgeGatingThresholdHours, want.AgeGatingThresholdHours)})
	}
	for _, b := range []struct {
		field      string
		want, have bool
	}{
		{"script_blocking", want.ScriptBlocking, have.ScriptBlocking},
		{"claude_audit_log", want.ClaudeAuditLog, have.ClaudeAuditLog},
		{"license_scanning", want.LicenseScanning, have.LicenseScanning},
	} {
		if b.want && !b.have {
			out = append(out, levelWeakening{b.field, b.field + " cannot be turned off"})
		}
	}
	if want.ClaudePermissionLevel != have.ClaudePermissionLevel {
		wantRank, ranked := c.PermissionPresetStrictness(want.ClaudePermissionLevel)
		haveRank, _ := c.PermissionPresetStrictness(have.ClaudePermissionLevel)
		_, known := c.PermissionPreset(have.ClaudePermissionLevel) // Validate reports an unknown one
		if ranked && known && haveRank < wantRank {
			out = append(out, levelWeakening{"claude_permission_level",
				fmt.Sprintf("permission preset %q is less strict than the built-in %q", have.ClaudePermissionLevel, want.ClaudePermissionLevel)})
		}
	}
	return out
}

// hookTierFloorViolations reports, through add, a merged hook tier layout
// that would run an always-on hook of floor at fewer security levels than
// floor does. devenv.nix drops every hook tiered above the project's level
// (see devenv.FilterHooksByTier), so the built-in tiers must keep their
// relative order ahead of any new tier, and each always-on hook must stay in
// a tier no higher than its built-in one (an untiered hook runs at every
// level, like one in the lowest tier).
func hookTierFloorViolations(floor, merged *Catalog, add func(field, format string, args ...any)) {
	builtinOrder, order := floor.hookTiers.TierOrder, merged.hookTiers.TierOrder
	if len(order) < len(builtinOrder) || !slices.Equal(order[:len(builtinOrder)], builtinOrder) {
		add("hook_tier_order", "must start with the built-in hook tiers %v, in order (got %v)", builtinOrder, order)
		return
	}
	alwaysOn := slices.Clone(floor.security.Hooks.Default)
	for _, h := range floor.security.CustomHooks {
		alwaysOn = append(alwaysOn, h.ID)
	}
	for _, id := range sliceutil.Dedup(alwaysOn) {
		builtinTier, builtinAt := hookTierOf(floor, id)
		tier, at := hookTierOf(merged, id)
		if at > builtinAt {
			add(sectionHookTiers+"."+tier, "cannot move always-on hook %q above built-in hook tier %q", id, cmp.Or(builtinTier, builtinOrder[0]))
		}
	}
}

// hookTierOf returns the hook tier c places hook id in and that tier's
// index in hook_tier_order; an untiered hook (index 0) runs at every level.
func hookTierOf(c *Catalog, id string) (string, int) {
	for i, tier := range c.hookTiers.TierOrder {
		if slices.Contains(c.hookTiers.Tiers[tier], id) {
			return tier, i
		}
	}
	return "", 0
}
