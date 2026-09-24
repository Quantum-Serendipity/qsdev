package config

import (
	"slices"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// ResolvedConfig is the result of merging all configuration layers.
type ResolvedConfig struct {
	Config     *types.QsdevConfig
	Traces     []ResolutionTrace
	Violations []FloorViolation
}

// FloorViolation records a case where a local or project override attempted
// to weaken a security setting below the enforced floor.
type FloorViolation struct {
	Field     string
	Attempted any
	Enforced  any
	Reason    string
}

// ResolveConfig performs five-layer configuration resolution:
//  1. Organization defaults (orgDefaults)
//  2. Profile overlay (if profile is not nil)
//  3. Compliance level overlay (if project has Client.SecurityLevel)
//  4. Project overrides (project)
//  5. Local developer overrides (local, converted to QsdevConfig)
//
// After merging, enforceSecurityFloor ensures security settings cannot be
// weakened below the project's declared floor.
func ResolveConfig(orgDefaults, profile, project *types.QsdevConfig, local *LocalConfig, verbose bool) (*ResolvedConfig, error) {
	tracer := NewTracer(verbose)

	// Layer 1: Start from org defaults.
	resolved := cloneQsdevConfig(orgDefaults)
	if resolved == nil {
		resolved = &types.QsdevConfig{}
	}
	tracer.Record("*", "org-defaults", "layer-1", nil, "base layer")

	// Layer 2: Merge profile overlay.
	if profile != nil {
		resolved = deepMerge(resolved, profile)
		tracer.Record("*", "profile", "layer-2", nil, "profile overlay applied")
	}

	// Layer 3: Compliance level overlay from client config.
	if project != nil && project.Client != nil && project.Client.SecurityLevel != "" {
		complianceOverlay := ComplianceLevelToConfig(project.Client.SecurityLevel)
		if complianceOverlay != nil {
			resolved = deepMerge(resolved, complianceOverlay)
			tracer.Record("security.level", project.Client.SecurityLevel, "layer-3", nil, "client compliance overlay")
		}
	}

	// Layer 4: Merge project overrides.
	if project != nil {
		resolved = deepMerge(resolved, project)
		tracer.Record("*", "project", "layer-4", nil, "project overrides applied")
	}

	// Layer 5: Merge local overrides.
	if local != nil {
		localCfg := localToQsdevConfig(local)
		resolved = deepMerge(resolved, localCfg)
		tracer.Record("*", "local", "layer-5", nil, "local overrides applied")
	}

	// Post-merge: enforce security floor.
	violations := enforceSecurityFloor(resolved, project)

	return &ResolvedConfig{
		Config:     resolved,
		Traces:     tracer.Traces(),
		Violations: violations,
	}, nil
}

// deepMerge applies per-field merge semantics to combine a base and overlay
// QsdevConfig. The result is a new QsdevConfig; neither input is modified.
func deepMerge(base, overlay *types.QsdevConfig) *types.QsdevConfig {
	if base == nil && overlay == nil {
		return &types.QsdevConfig{}
	}
	if base == nil {
		return cloneQsdevConfig(overlay)
	}
	if overlay == nil {
		return cloneQsdevConfig(base)
	}

	result := cloneQsdevConfig(base)

	// Languages: replacement semantics.
	result.Languages = mergeReplaceLanguages(base.Languages, overlay.Languages)

	// Services: replacement semantics.
	result.Services = mergeReplaceServices(base.Services, overlay.Services)

	// Packages and overlays: union.
	result.Packages = mergeUnionStrings(base.Packages, overlay.Packages)
	result.Overlays = mergeUnionStrings(base.Overlays, overlay.Overlays)

	// Security.Level: last-wins scalar.
	if overlay.Security.Level != "" {
		result.Security.Level = overlay.Security.Level
	}

	// Security boolean pointers: pointer bool merge.
	result.Security.AgeGating = mergePointerBool(base.Security.AgeGating, overlay.Security.AgeGating)
	result.Security.ScriptBlocking = mergePointerBool(base.Security.ScriptBlocking, overlay.Security.ScriptBlocking)
	result.Security.LockEnforcement = mergePointerBool(base.Security.LockEnforcement, overlay.Security.LockEnforcement)
	result.Security.VulnScanning = mergePointerBool(base.Security.VulnScanning, overlay.Security.VulnScanning)

	// Tools.Enabled: union.
	result.Tools.Enabled = mergeUnionStrings(base.Tools.Enabled, overlay.Tools.Enabled)

	// Tools.Disabled: union.
	result.Tools.Disabled = mergeUnionStrings(base.Tools.Disabled, overlay.Tools.Disabled)

	// Tools.Config: recursive map merge.
	result.Tools.Config = mergeMapStringAny(base.Tools.Config, overlay.Tools.Config)

	// ClaudeCode.Enabled: pointer bool merge.
	result.ClaudeCode.Enabled = mergePointerBool(base.ClaudeCode.Enabled, overlay.ClaudeCode.Enabled)

	// ClaudeCode.PermissionLevel: last-wins scalar.
	if overlay.ClaudeCode.PermissionLevel != "" {
		result.ClaudeCode.PermissionLevel = overlay.ClaudeCode.PermissionLevel
	}

	// ClaudeCode.Skills: union.
	result.ClaudeCode.Skills = mergeUnionStrings(base.ClaudeCode.Skills, overlay.ClaudeCode.Skills)

	// ClaudeCode.MCPServers: union.
	result.ClaudeCode.MCPServers = mergeUnionStrings(base.ClaudeCode.MCPServers, overlay.ClaudeCode.MCPServers)

	// Infrastructure: last-wins scalars, map merge for overrides.
	if overlay.Infrastructure.RegistryProxy != "" {
		result.Infrastructure.RegistryProxy = overlay.Infrastructure.RegistryProxy
	}
	if len(overlay.Infrastructure.RegistryProxyOverrides) > 0 {
		if result.Infrastructure.RegistryProxyOverrides == nil {
			result.Infrastructure.RegistryProxyOverrides = make(map[string]string)
		}
		for k, v := range overlay.Infrastructure.RegistryProxyOverrides {
			result.Infrastructure.RegistryProxyOverrides[k] = v
		}
	}
	if len(overlay.Infrastructure.RegistryProxyPaths) > 0 {
		if result.Infrastructure.RegistryProxyPaths == nil {
			result.Infrastructure.RegistryProxyPaths = make(map[string]string)
		}
		for k, v := range overlay.Infrastructure.RegistryProxyPaths {
			result.Infrastructure.RegistryProxyPaths[k] = v
		}
	}
	if overlay.Infrastructure.NixCache != "" {
		result.Infrastructure.NixCache = overlay.Infrastructure.NixCache
	}
	if overlay.Infrastructure.BuildCache != "" {
		result.Infrastructure.BuildCache = overlay.Infrastructure.BuildCache
	}

	// Git: last-wins scalar.
	if overlay.Git.BranchPattern != "" {
		result.Git.BranchPattern = overlay.Git.BranchPattern
	}

	// Client: NOT merged, only from project config.
	// The overlay's Client replaces the base's if present (as a deep copy,
	// so later edits to the resolved config never reach the caller's input).
	if overlay.Client != nil {
		result.Client = cloneClient(overlay.Client)
	}

	// Tier: last-wins scalar.
	if overlay.Tier != "" {
		result.Tier = overlay.Tier
	}

	// Profile: NOT merged, only from project.
	if overlay.Profile != "" {
		result.Profile = overlay.Profile
	}

	// InfraProfile: NOT merged, only from project.
	if overlay.InfraProfile != "" {
		result.InfraProfile = overlay.InfraProfile
	}

	// Version: NOT merged, only from project.
	if overlay.Version != 0 {
		result.Version = overlay.Version
	}

	// QsdevVersion: NOT merged, only from project.
	if overlay.QsdevVersion != "" {
		result.QsdevVersion = overlay.QsdevVersion
	}

	return result
}

// enforceSecurityFloor ensures the resolved config cannot have security
// settings weaker than the project's declared floor.
func enforceSecurityFloor(resolved, project *types.QsdevConfig) []FloorViolation {
	if project == nil {
		return nil
	}

	var violations []FloorViolation

	// Determine effective floor level.
	floorLevel := project.Security.Level
	if project.Client != nil && project.Client.SecurityLevel != "" {
		// Client security level acts as additional floor.
		if CompareComplianceLevels(project.Client.SecurityLevel, floorLevel) > 0 {
			floorLevel = project.Client.SecurityLevel
		}
	}

	// Cannot lower security.level below floor.
	if floorLevel != "" && resolved.Security.Level != "" {
		if CompareComplianceLevels(resolved.Security.Level, floorLevel) < 0 {
			violations = append(violations, FloorViolation{
				Field:     "security.level",
				Attempted: resolved.Security.Level,
				Enforced:  floorLevel,
				Reason:    "cannot lower security level below project floor",
			})
			resolved.Security.Level = floorLevel
		}
	}

	// Compute the compliance-mandated bool floor from the same effective level
	// used to floor security.level above. A project that declares a client
	// compliance level (or a direct security.level) inherits the security
	// controls that level mandates, so a local override can never silently
	// disable a compliance-required control while security.level still reports
	// at that level. This is an ADDITIONAL floor stacked on the project's own
	// declared bool floor; the stronger of the two wins (fail-closed).
	var complianceFloor types.SecurityConfig
	if overlay := ComplianceLevelToConfig(floorLevel); overlay != nil {
		complianceFloor = overlay.Security
	}

	// Cannot set security bools to false when the project or the effective
	// compliance level requires them to be true.
	violations = append(violations, enforceBoolFloor(&resolved.Security.AgeGating,
		strongerBoolFloor(project.Security.AgeGating, complianceFloor.AgeGating), "security.age_gating")...)
	violations = append(violations, enforceBoolFloor(&resolved.Security.ScriptBlocking,
		strongerBoolFloor(project.Security.ScriptBlocking, complianceFloor.ScriptBlocking), "security.script_blocking")...)
	violations = append(violations, enforceBoolFloor(&resolved.Security.LockEnforcement,
		strongerBoolFloor(project.Security.LockEnforcement, complianceFloor.LockEnforcement), "security.lock_enforcement")...)
	violations = append(violations, enforceBoolFloor(&resolved.Security.VulnScanning,
		strongerBoolFloor(project.Security.VulnScanning, complianceFloor.VulnScanning), "security.vuln_scanning")...)

	// Client MCP policy: only the project's client block sets it, and no
	// layer can re-add a server it blocks.
	filterBlockedMCP(resolved, ClientMCPPolicy(project))

	return violations
}

// strongerBoolFloor combines two security-bool floors and reports whether a
// floor is in effect (fail-closed). A floor of true (must be enabled)
// dominates: if either the project's declared floor or the compliance-mandated
// floor is true, the effective floor is true. This is used to stack the
// compliance-level requirement on top of the project's own bool floor without
// weakening either.
func strongerBoolFloor(a, b *bool) bool {
	return (a != nil && *a) || (b != nil && *b)
}

// enforceBoolFloor ensures a resolved *bool cannot be weaker than the floor.
// When the floor requires the setting to be true, any resolved value below it
// is enforced back up to true — a compliance- or project-mandated control must
// never be locally disabled. Only an explicit false is recorded as a
// FloorViolation: an unset value (which downstream treats as disabled) is
// raised silently, because nothing tried to weaken it. Without that
// distinction every project that declares a security level but leaves the
// individual bools unset (the file init writes) would report violations.
func enforceBoolFloor(resolved **bool, floor bool, field string) []FloorViolation {
	if !floor {
		return nil
	}
	if *resolved != nil && **resolved {
		return nil
	}

	attempted := *resolved
	t := true
	*resolved = &t
	if attempted == nil {
		return nil
	}
	return []FloorViolation{{
		Field:     field,
		Attempted: false,
		Enforced:  true,
		Reason:    "cannot disable security setting that project or compliance level requires",
	}}
}

// filterBlockedMCP removes the servers the client MCP policy does not permit
// from the resolved claude_code.mcp_servers.
func filterBlockedMCP(resolved *types.QsdevConfig, policy types.MCPPolicy) {
	resolved.ClaudeCode.MCPServers = policy.Filter(resolved.ClaudeCode.MCPServers)
}

// cloneQsdevConfig creates a deep copy of a QsdevConfig.
func cloneQsdevConfig(cfg *types.QsdevConfig) *types.QsdevConfig {
	if cfg == nil {
		return nil
	}

	result := &types.QsdevConfig{
		Version:      cfg.Version,
		QsdevVersion: cfg.QsdevVersion,
		Tier:         cfg.Tier,
		Profile:      cfg.Profile,
		InfraProfile: cfg.InfraProfile,
		Security: types.SecurityConfig{
			Level:           cfg.Security.Level,
			AgeGating:       cloneBoolPtr(cfg.Security.AgeGating),
			ScriptBlocking:  cloneBoolPtr(cfg.Security.ScriptBlocking),
			LockEnforcement: cloneBoolPtr(cfg.Security.LockEnforcement),
			VulnScanning:    cloneBoolPtr(cfg.Security.VulnScanning),
		},
		ClaudeCode: types.ClaudeCodeConfig{
			Enabled:         cloneBoolPtr(cfg.ClaudeCode.Enabled),
			PermissionLevel: cfg.ClaudeCode.PermissionLevel,
		},
		Infrastructure: types.InfraConfig{
			RegistryProxy: cfg.Infrastructure.RegistryProxy,
			NixCache:      cfg.Infrastructure.NixCache,
			BuildCache:    cfg.Infrastructure.BuildCache,
		},
		Git:      cfg.Git,
		Packages: slices.Clone(cfg.Packages),
		Overlays: slices.Clone(cfg.Overlays),
	}

	if len(cfg.Infrastructure.RegistryProxyOverrides) > 0 {
		result.Infrastructure.RegistryProxyOverrides = make(map[string]string, len(cfg.Infrastructure.RegistryProxyOverrides))
		for k, v := range cfg.Infrastructure.RegistryProxyOverrides {
			result.Infrastructure.RegistryProxyOverrides[k] = v
		}
	}

	if len(cfg.Infrastructure.RegistryProxyPaths) > 0 {
		result.Infrastructure.RegistryProxyPaths = make(map[string]string, len(cfg.Infrastructure.RegistryProxyPaths))
		for k, v := range cfg.Infrastructure.RegistryProxyPaths {
			result.Infrastructure.RegistryProxyPaths[k] = v
		}
	}

	// Clone Languages.
	if len(cfg.Languages) > 0 {
		result.Languages = make([]types.LanguageConfig, len(cfg.Languages))
		copy(result.Languages, cfg.Languages)
	}

	// Clone Services.
	if len(cfg.Services) > 0 {
		result.Services = make([]types.ServiceConfig, len(cfg.Services))
		for i, s := range cfg.Services {
			result.Services[i] = types.ServiceConfig{
				Name:    s.Name,
				Version: s.Version,
			}
			if s.Options != nil {
				result.Services[i].Options = make(map[string]string, len(s.Options))
				for k, v := range s.Options {
					result.Services[i].Options[k] = v
				}
			}
		}
	}

	// Clone Tools.
	if len(cfg.Tools.Enabled) > 0 {
		result.Tools.Enabled = make([]string, len(cfg.Tools.Enabled))
		copy(result.Tools.Enabled, cfg.Tools.Enabled)
	}
	if len(cfg.Tools.Disabled) > 0 {
		result.Tools.Disabled = make([]string, len(cfg.Tools.Disabled))
		copy(result.Tools.Disabled, cfg.Tools.Disabled)
	}
	if len(cfg.Tools.Config) > 0 {
		result.Tools.Config = make(map[string]map[string]any, len(cfg.Tools.Config))
		for k, v := range cfg.Tools.Config {
			inner := make(map[string]any, len(v))
			for ik, iv := range v {
				inner[ik] = iv
			}
			result.Tools.Config[k] = inner
		}
	}

	// Clone ClaudeCode slices.
	if len(cfg.ClaudeCode.Skills) > 0 {
		result.ClaudeCode.Skills = make([]string, len(cfg.ClaudeCode.Skills))
		copy(result.ClaudeCode.Skills, cfg.ClaudeCode.Skills)
	}
	if len(cfg.ClaudeCode.MCPServers) > 0 {
		result.ClaudeCode.MCPServers = make([]string, len(cfg.ClaudeCode.MCPServers))
		copy(result.ClaudeCode.MCPServers, cfg.ClaudeCode.MCPServers)
	}

	result.Client = cloneClient(cfg.Client)

	return result
}

// cloneClient returns a deep copy of a *ClientConfig, or nil for nil.
func cloneClient(c *types.ClientConfig) *types.ClientConfig {
	if c == nil {
		return nil
	}
	client := *c
	client.Compliance = slices.Clone(c.Compliance)
	client.AllowedMCP = slices.Clone(c.AllowedMCP)
	client.BlockedMCP = slices.Clone(c.BlockedMCP)
	return &client
}

// cloneBoolPtr returns a copy of a *bool value.
func cloneBoolPtr(p *bool) *bool {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}
