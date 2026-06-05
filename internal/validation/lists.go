// Package validation provides canonical lists of supported languages, services,
// permission presets, and other enumerated values used across addons. All lists
// are exposed via functions (not package-level vars) so callers cannot mutate
// the source of truth.
package validation

import "github.com/Quantum-Serendipity/qsdev/internal/catalog"

// ---------------------------------------------------------------------------
// Public accessors — each returns a fresh copy to prevent mutation.
// Backed by the YAML catalog at internal/catalog/defaults/validation.yaml.
// ---------------------------------------------------------------------------

// Languages returns all supported language/ecosystem identifiers in canonical order.
func Languages() []string { return catalog.MustDefault().Languages() }

// CoreLanguages returns the subset of languages for which the devenv addon
// can generate full configuration.
func CoreLanguages() []string { return catalog.MustDefault().CoreLanguages() }

// Services returns all supported development service identifiers.
func Services() []string { return catalog.MustDefault().Services() }

// PermissionPresets returns all valid Claude Code permission preset names.
func PermissionPresets() []string { return catalog.MustDefault().PermissionPresets() }

// HookPresets returns all valid Claude Code hook preset names.
func HookPresets() []string { return catalog.MustDefault().HookPresets() }

// NodePackageManagers returns valid Node.js package manager names.
func NodePackageManagers() []string { return catalog.MustDefault().PackageManagers("node") }

// PythonPackageManagers returns valid Python package manager names.
func PythonPackageManagers() []string { return catalog.MustDefault().PackageManagers("python") }

// Tiers returns all valid security onboarding tier names.
func Tiers() []string { return catalog.MustDefault().TierOrder() }

// SecurityLevels returns all valid security posture levels.
func SecurityLevels() []string { return catalog.MustDefault().SecurityLevels() }

// DataClassifications returns all valid data classification labels.
func DataClassifications() []string { return catalog.MustDefault().DataClassifications() }

// ---------------------------------------------------------------------------
// Membership checks
// ---------------------------------------------------------------------------

// IsValidLanguage checks if lang is a supported language identifier.
func IsValidLanguage(lang string) bool { return containsStr(catalog.MustDefault().Languages(), lang) }

// IsValidCoreLanguage checks if lang belongs to the core devenv language set.
func IsValidCoreLanguage(lang string) bool {
	return containsStr(catalog.MustDefault().CoreLanguages(), lang)
}

// IsValidService checks if svc is a supported service identifier.
func IsValidService(svc string) bool { return containsStr(catalog.MustDefault().Services(), svc) }

// IsValidPermissionPreset checks if preset is a valid permission preset name.
func IsValidPermissionPreset(preset string) bool {
	return containsStr(catalog.MustDefault().PermissionPresets(), preset)
}

// IsValidHookPreset checks if preset is a valid hook preset name.
func IsValidHookPreset(preset string) bool {
	return containsStr(catalog.MustDefault().HookPresets(), preset)
}

// IsValidNodePackageManager checks if pm is a valid Node.js package manager.
func IsValidNodePackageManager(pm string) bool {
	return containsStr(catalog.MustDefault().PackageManagers("node"), pm)
}

// IsValidPythonPackageManager checks if pm is a valid Python package manager.
func IsValidPythonPackageManager(pm string) bool {
	return containsStr(catalog.MustDefault().PackageManagers("python"), pm)
}

// IsValidTier checks if t is a valid security onboarding tier.
func IsValidTier(t string) bool { return containsStr(catalog.MustDefault().TierOrder(), t) }

// IsValidSecurityLevel checks if level is a valid security posture level.
func IsValidSecurityLevel(level string) bool {
	return containsStr(catalog.MustDefault().SecurityLevels(), level)
}

// IsValidDataClassification checks if dc is a valid data classification label.
func IsValidDataClassification(dc string) bool {
	return containsStr(catalog.MustDefault().DataClassifications(), dc)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func containsStr(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}
