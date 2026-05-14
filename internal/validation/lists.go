// Package validation provides canonical lists of supported languages, services,
// permission presets, and other enumerated values used across addons. All lists
// are exposed via functions (not package-level vars) so callers cannot mutate
// the source of truth.
package validation

// languages is the full set of supported language/platform ecosystem identifiers
// in canonical order.
var languages = []string{
	"go", "javascript", "python", "rust",
	"java", "dotnet", "docker", "terraform",
	"php", "ruby", "scala",
	"cpp", "helm", "ansible", "shell",
	"elixir", "dart", "swift", "haskell", "clojure", "bazel", "nix",
	"perl", "r", "lua", "zig", "powershell",
}

// coreLanguages is the subset of languages that the devenv addon can generate
// full configuration for.
var coreLanguages = []string{
	"go", "javascript", "python", "rust",
	"java", "dotnet", "docker", "terraform",
}

// services is the set of supported development services.
var services = []string{
	"postgres", "redis", "mysql",
	"mongodb", "elasticsearch", "rabbitmq",
}

// permissionPresets is the set of valid Claude Code permission preset names.
var permissionPresets = []string{
	"minimal", "standard", "permissive", "custom",
}

// hookPresets is the set of valid Claude Code hook preset names.
var hookPresets = []string{
	"auto-format", "safety-block", "pre-commit", "audit-log",
}

// securityLevels is the set of valid security posture levels.
var securityLevels = []string{"baseline", "enhanced", "strict"}

// dataClassifications is the set of valid data classification labels.
var dataClassifications = []string{"public", "internal", "confidential"}

// nodePackageManagers is the set of valid Node.js package manager names.
var nodePackageManagers = []string{"npm", "pnpm", "yarn", "bun"}

// pythonPackageManagers is the set of valid Python package manager names.
var pythonPackageManagers = []string{"pip", "uv", "poetry"}

// ---------------------------------------------------------------------------
// Public accessors — each returns a fresh copy to prevent mutation.
// ---------------------------------------------------------------------------

// Languages returns all supported language/ecosystem identifiers in canonical order.
func Languages() []string { return copyStrings(languages) }

// CoreLanguages returns the subset of languages for which the devenv addon
// can generate full configuration.
func CoreLanguages() []string { return copyStrings(coreLanguages) }

// Services returns all supported development service identifiers.
func Services() []string { return copyStrings(services) }

// PermissionPresets returns all valid Claude Code permission preset names.
func PermissionPresets() []string { return copyStrings(permissionPresets) }

// HookPresets returns all valid Claude Code hook preset names.
func HookPresets() []string { return copyStrings(hookPresets) }

// NodePackageManagers returns valid Node.js package manager names.
func NodePackageManagers() []string { return copyStrings(nodePackageManagers) }

// PythonPackageManagers returns valid Python package manager names.
func PythonPackageManagers() []string { return copyStrings(pythonPackageManagers) }

// SecurityLevels returns all valid security posture levels.
func SecurityLevels() []string { return copyStrings(securityLevels) }

// DataClassifications returns all valid data classification labels.
func DataClassifications() []string { return copyStrings(dataClassifications) }

// ---------------------------------------------------------------------------
// Membership checks
// ---------------------------------------------------------------------------

// IsValidLanguage checks if lang is a supported language identifier.
func IsValidLanguage(lang string) bool { return containsStr(languages, lang) }

// IsValidCoreLanguage checks if lang belongs to the core devenv language set.
func IsValidCoreLanguage(lang string) bool { return containsStr(coreLanguages, lang) }

// IsValidService checks if svc is a supported service identifier.
func IsValidService(svc string) bool { return containsStr(services, svc) }

// IsValidPermissionPreset checks if preset is a valid permission preset name.
func IsValidPermissionPreset(preset string) bool { return containsStr(permissionPresets, preset) }

// IsValidHookPreset checks if preset is a valid hook preset name.
func IsValidHookPreset(preset string) bool { return containsStr(hookPresets, preset) }

// IsValidNodePackageManager checks if pm is a valid Node.js package manager.
func IsValidNodePackageManager(pm string) bool { return containsStr(nodePackageManagers, pm) }

// IsValidPythonPackageManager checks if pm is a valid Python package manager.
func IsValidPythonPackageManager(pm string) bool { return containsStr(pythonPackageManagers, pm) }

// IsValidSecurityLevel checks if level is a valid security posture level.
func IsValidSecurityLevel(level string) bool { return containsStr(securityLevels, level) }

// IsValidDataClassification checks if dc is a valid data classification label.
func IsValidDataClassification(dc string) bool { return containsStr(dataClassifications, dc) }

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func copyStrings(src []string) []string {
	out := make([]string, len(src))
	copy(out, src)
	return out
}

func containsStr(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}
