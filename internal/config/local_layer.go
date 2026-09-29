package config

import (
	"maps"
	"slices"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Reasons recorded on the FloorViolations of a dropped local override.
const (
	reasonLocalToolDisable   = "a local override can enable tools but not disable them"
	reasonLocalToolConfig    = "tool configuration can only be set in the committed config"
	reasonLocalClaudeEnabled = "claude_code.enabled can only be set in the committed config"
	reasonLocalPackageMgr    = "a local override cannot change a language's committed package manager"
	reasonLocalServiceOption = "a local override can add service options but not change committed ones"
	reasonLocalPermission    = "a local override can only tighten the permission level"
	reasonLocalCredVend      = "credential vending can only be configured in the committed config"
)

// sanitizeLocal returns the part of the developer's local layer that only
// adds to or tightens base, the configuration resolved from the committed
// layers, and a FloorViolation for each setting it drops:
//
//   - languages, services, extra_packages, tools.enabled, claude_code.skills
//     and claude_code.mcp_servers are additions and are kept (the client MCP
//     policy still filters servers afterwards);
//   - a language's or service's version may be overridden; a language's
//     committed package manager and a service's committed options may not;
//   - claude_code.permission_level is kept only when it is at least as strict
//     as base's effective level (see ComparePermissionLevels);
//   - tools.disabled, tools.config and a claude_code.enabled that differs from
//     base are dropped;
//   - security settings are kept: enforceSecurityFloor floors them after the
//     merge, so they can only raise the project's floor. The exception is
//     security.credential_vend, an opt-in to hand out cloud credentials,
//     which is dropped.
func sanitizeLocal(base *types.QsdevConfig, local *LocalConfig) (*types.QsdevConfig, []FloorViolation) {
	applied := &types.QsdevConfig{
		Security: types.SecurityConfig{
			Level:           local.Security.Level,
			AgeGating:       cloneBoolPtr(local.Security.AgeGating),
			ScriptBlocking:  cloneBoolPtr(local.Security.ScriptBlocking),
			LockEnforcement: cloneBoolPtr(local.Security.LockEnforcement),
			VulnScanning:    cloneBoolPtr(local.Security.VulnScanning),
		},
		Packages: slices.Clone(local.ExtraPackages),
		Tools:    types.ToolsConfig{Enabled: slices.Clone(local.Tools.Enabled)},
		ClaudeCode: types.ClaudeCodeConfig{
			Skills:     slices.Clone(local.ClaudeCode.Skills),
			MCPServers: slices.Clone(local.ClaudeCode.MCPServers),
		},
	}

	var violations, v []FloorViolation
	applied.Languages, violations = sanitizeLocalLanguages(base.Languages, local.Languages)
	applied.Services, v = sanitizeLocalServices(base.Services, local.Services)
	violations = append(violations, v...)

	for _, name := range local.Tools.Disabled {
		violations = append(violations, FloorViolation{Field: "tools.disabled", Attempted: name, Reason: reasonLocalToolDisable})
	}
	for _, name := range slices.Sorted(maps.Keys(local.Tools.Config)) {
		violations = append(violations, FloorViolation{Field: "tools.config", Attempted: name, Reason: reasonLocalToolConfig})
	}

	if !local.Security.CredentialVend.IsZero() {
		violations = append(violations, FloorViolation{
			Field:     "security.credential_vend",
			Attempted: local.Security.CredentialVend,
			Reason:    reasonLocalCredVend,
		})
	}

	if local.ClaudeCode.Enabled != nil && *local.ClaudeCode.Enabled != claudeCodeEnabled(base) {
		violations = append(violations, FloorViolation{
			Field:     "claude_code.enabled",
			Attempted: *local.ClaudeCode.Enabled,
			Reason:    reasonLocalClaudeEnabled,
		})
	}

	if level := local.ClaudeCode.PermissionLevel; level != "" {
		floor := EffectivePermissionLevel(base.ClaudeCode.PermissionLevel, base.Tier, base.ClaudeCode.MCPServers)
		if permissionAtLeastAsStrict(level, floor) {
			applied.ClaudeCode.PermissionLevel = level
		} else {
			violations = append(violations, permissionViolation(level, floor))
		}
	}

	return applied, violations
}

// permissionViolation records a local permission level that is not at least
// as strict as floor: raised to floor when the two are comparable, ignored
// otherwise.
func permissionViolation(attempted, floor string) FloorViolation {
	v := FloorViolation{Field: "claude_code.permission_level", Attempted: attempted, Reason: reasonLocalPermission}
	if _, comparable := ComparePermissionLevels(attempted, floor); comparable {
		v.Enforced = floor
	} else {
		v.Reason += " (" + attempted + " is not comparable to " + floor + ")"
	}
	return v
}

// sanitizeLocalLanguages copies the local languages, clearing a package
// manager that would replace the one base commits for the same language.
func sanitizeLocalLanguages(base, local []types.LanguageConfig) ([]types.LanguageConfig, []FloorViolation) {
	out := slices.Clone(local)
	var violations []FloorViolation
	for i, lang := range out {
		j := slices.IndexFunc(base, func(b types.LanguageConfig) bool { return b.Name == lang.Name })
		if j < 0 || lang.PackageManager == "" || base[j].PackageManager == "" || lang.PackageManager == base[j].PackageManager {
			continue
		}
		violations = append(violations, FloorViolation{
			Field:     "languages." + lang.Name + ".package_manager",
			Attempted: lang.PackageManager,
			Reason:    reasonLocalPackageMgr + " (" + base[j].PackageManager + ")",
		})
		out[i].PackageManager = ""
	}
	return out, violations
}

// sanitizeLocalServices copies the local services, dropping each option that
// would change a value base commits for the same service.
func sanitizeLocalServices(base, local []types.ServiceConfig) ([]types.ServiceConfig, []FloorViolation) {
	out := make([]types.ServiceConfig, 0, len(local))
	var violations []FloorViolation
	for _, svc := range local {
		svc.Options = maps.Clone(svc.Options)
		if j := slices.IndexFunc(base, func(b types.ServiceConfig) bool { return b.Name == svc.Name }); j >= 0 {
			for _, key := range slices.Sorted(maps.Keys(svc.Options)) {
				if committed, ok := base[j].Options[key]; !ok || committed == svc.Options[key] {
					continue
				}
				violations = append(violations, FloorViolation{
					Field:     "services." + svc.Name + ".options." + key,
					Attempted: svc.Options[key],
					Reason:    reasonLocalServiceOption,
				})
				delete(svc.Options, key)
			}
		}
		out = append(out, svc)
	}
	if len(out) == 0 {
		return nil, violations
	}
	return out, violations
}

// claudeCodeEnabled reports whether cfg enables Claude Code; an absent key is
// enabled (the legacy default ConfigToAnswers also applies).
func claudeCodeEnabled(cfg *types.QsdevConfig) bool {
	return cfg.ClaudeCode.Enabled == nil || *cfg.ClaudeCode.Enabled
}

// mergeLocal merges the sanitized local layer into base. Unlike deepMerge,
// which replaces the language and service lists, it merges them by name, so
// a local entry adds a language or service or overrides the version of a
// committed one but never removes one. A tool the local layer enables is also
// taken out of tools.disabled, so no tool ends up in both lists.
func mergeLocal(base, applied *types.QsdevConfig) *types.QsdevConfig {
	overlay := cloneQsdevConfig(applied)
	overlay.Languages, overlay.Services = nil, nil
	result := deepMerge(base, overlay)
	result.Languages = mergeLanguagesByName(base.Languages, applied.Languages)
	result.Services = mergeServicesByName(base.Services, applied.Services)
	result.Tools.Disabled = slices.DeleteFunc(result.Tools.Disabled, func(name string) bool {
		return slices.Contains(applied.Tools.Enabled, name)
	})
	return result
}

// mergeLanguagesByName returns base with each overlay language merged in:
// a new language is appended, and a known one takes the overlay's non-empty
// version and package manager.
func mergeLanguagesByName(base, overlay []types.LanguageConfig) []types.LanguageConfig {
	out := slices.Clone(base)
	for _, lang := range overlay {
		i := slices.IndexFunc(out, func(b types.LanguageConfig) bool { return b.Name == lang.Name })
		if i < 0 {
			out = append(out, lang)
			continue
		}
		if lang.Version != "" {
			out[i].Version = lang.Version
		}
		if lang.PackageManager != "" {
			out[i].PackageManager = lang.PackageManager
		}
	}
	return out
}

// mergeServicesByName returns base with each overlay service merged in: a new
// service is appended, and a known one takes the overlay's non-empty version
// and its options on top of the committed ones.
func mergeServicesByName(base, overlay []types.ServiceConfig) []types.ServiceConfig {
	out := make([]types.ServiceConfig, 0, max(len(base), len(overlay)))
	for _, s := range base {
		s.Options = maps.Clone(s.Options)
		out = append(out, s)
	}
	for _, svc := range overlay {
		i := slices.IndexFunc(out, func(b types.ServiceConfig) bool { return b.Name == svc.Name })
		if i < 0 {
			svc.Options = maps.Clone(svc.Options)
			out = append(out, svc)
			continue
		}
		if svc.Version != "" {
			out[i].Version = svc.Version
		}
		if len(svc.Options) > 0 {
			if out[i].Options == nil {
				out[i].Options = make(map[string]string, len(svc.Options))
			}
			maps.Copy(out[i].Options, svc.Options)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
