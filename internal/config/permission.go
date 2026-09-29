package config

import (
	"cmp"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/tier"
)

// defaultPermissionLevel is the preset generation applies when neither a
// permission level nor a tier selects one (types.WizardAnswers.FillDefaults
// and the Claude Code settings generator use the same fallback).
const defaultPermissionLevel = "standard"

// EffectivePermissionLevel returns the permission preset generation applies
// for a configuration: level when set, otherwise the default preset of the
// tier (named, or inferred from level and mcpServers), otherwise standard.
func EffectivePermissionLevel(level, tierName string, mcpServers []string) string {
	switch {
	case level != "":
		return level
	case tierName != "":
		return tier.Resolve(tierName, level, mcpServers).DefaultPermissionPreset()
	default:
		return defaultPermissionLevel
	}
}

// ComparePermissionLevels compares two permission presets by their catalog
// strictness rank: it returns a negative number when a is less strict than b,
// zero when they are equally strict and a positive number when a is stricter.
// The bool reports whether the presets are comparable at all: a preset the
// catalog gives no strictness rank (custom, supply-chain-only) or does not
// know is comparable only to itself.
func ComparePermissionLevels(a, b string) (int, bool) {
	if a == b {
		return 0, true
	}
	cat, err := catalog.Default()
	if err != nil {
		return 0, false
	}
	ra, okA := cat.PermissionPresetStrictness(a)
	rb, okB := cat.PermissionPresetStrictness(b)
	if !okA || !okB {
		return 0, false
	}
	return cmp.Compare(ra, rb), true
}

// permissionAtLeastAsStrict reports whether candidate is known to be at least
// as strict as floor. Incomparable presets fail closed.
func permissionAtLeastAsStrict(candidate, floor string) bool {
	c, ok := ComparePermissionLevels(candidate, floor)
	return ok && c >= 0
}
