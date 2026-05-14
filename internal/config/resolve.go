package config

import (
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

// ResolvedConfig is the result of merging all configuration layers.
type ResolvedConfig struct {
	Config     *types.GdevConfig
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
//  5. Local developer overrides (local, converted to GdevConfig)
//
// After merging, enforceSecurityFloor ensures security settings cannot be
// weakened below the project's declared floor.
func ResolveConfig(orgDefaults, profile, project *types.GdevConfig, local *LocalConfig, verbose bool) (*ResolvedConfig, error) {
	tracer := NewTracer(verbose)

	// Layer 1: Start from org defaults.
	resolved := cloneGdevConfig(orgDefaults)
	if resolved == nil {
		resolved = &types.GdevConfig{}
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
		localCfg := localToGdevConfig(local)
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
// GdevConfig. The result is a new GdevConfig; neither input is modified.
func deepMerge(base, overlay *types.GdevConfig) *types.GdevConfig {
	if base == nil && overlay == nil {
		return &types.GdevConfig{}
	}
	if base == nil {
		return cloneGdevConfig(overlay)
	}
	if overlay == nil {
		return cloneGdevConfig(base)
	}

	result := cloneGdevConfig(base)

	// Languages: replacement semantics.
	result.Languages = mergeReplaceLanguages(base.Languages, overlay.Languages)

	// Services: replacement semantics.
	result.Services = mergeReplaceServices(base.Services, overlay.Services)

	// Security.Level: last-wins scalar.
	if overlay.Security.Level != "" {
		result.Security.Level = overlay.Security.Level
	}

	// Security boolean pointers: pointer bool merge.
	result.Security.AgeGating = mergePointerBool(base.Security.AgeGating, overlay.Security.AgeGating)
	result.Security.ScriptBlocking = mergePointerBool(base.Security.ScriptBlocking, overlay.Security.ScriptBlocking)
	result.Security.LockEnforce = mergePointerBool(base.Security.LockEnforce, overlay.Security.LockEnforce)
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

	// Infrastructure: last-wins scalars.
	if overlay.Infrastructure.RegistryProxy != "" {
		result.Infrastructure.RegistryProxy = overlay.Infrastructure.RegistryProxy
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
	// The overlay's Client is used directly if present.
	if overlay.Client != nil {
		result.Client = overlay.Client
	}

	// Profile: NOT merged, only from project.
	if overlay.Profile != "" {
		result.Profile = overlay.Profile
	}

	// Version: NOT merged, only from project.
	if overlay.Version != 0 {
		result.Version = overlay.Version
	}

	// GdevVersion: NOT merged, only from project.
	if overlay.GdevVersion != "" {
		result.GdevVersion = overlay.GdevVersion
	}

	return result
}

// enforceSecurityFloor ensures the resolved config cannot have security
// settings weaker than the project's declared floor.
func enforceSecurityFloor(resolved, project *types.GdevConfig) []FloorViolation {
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

	// Cannot set security bools to false when project set them to true.
	violations = append(violations, enforceBoolFloor(&resolved.Security.AgeGating, project.Security.AgeGating, "security.age_gating")...)
	violations = append(violations, enforceBoolFloor(&resolved.Security.ScriptBlocking, project.Security.ScriptBlocking, "security.script_blocking")...)
	violations = append(violations, enforceBoolFloor(&resolved.Security.LockEnforce, project.Security.LockEnforce, "security.lock_enforcement")...)
	violations = append(violations, enforceBoolFloor(&resolved.Security.VulnScanning, project.Security.VulnScanning, "security.vuln_scanning")...)

	// Client blocked MCP: union-only, never removed.
	if project.Client != nil {
		// Enforce blocked MCP servers.
		if len(project.Client.BlockedMCP) > 0 {
			filterBlockedMCP(resolved, project.Client.BlockedMCP, project.Client.AllowedMCP)
		}
	}

	return violations
}

// enforceBoolFloor ensures a resolved *bool cannot be set to false when the
// project floor has it set to true.
func enforceBoolFloor(resolved **bool, floor *bool, field string) []FloorViolation {
	if floor == nil || !*floor {
		return nil
	}

	// Floor is true. If resolved is false, enforce.
	if *resolved != nil && !**resolved {
		t := true
		*resolved = &t
		return []FloorViolation{{
			Field:     field,
			Attempted: false,
			Enforced:  true,
			Reason:    "cannot disable security setting that project requires",
		}}
	}

	return nil
}

// filterBlockedMCP removes blocked MCP servers from the resolved config.
// If BlockedMCP contains ["*"], all servers except those in AllowedMCP are removed.
func filterBlockedMCP(resolved *types.GdevConfig, blocked, allowed []string) *types.GdevConfig {
	if len(blocked) == 0 {
		return resolved
	}

	// Check for wildcard block.
	wildcard := false
	for _, b := range blocked {
		if b == "*" {
			wildcard = true
			break
		}
	}

	allowedSet := make(map[string]bool, len(allowed))
	for _, a := range allowed {
		allowedSet[a] = true
	}

	if wildcard {
		// Block all except allowed.
		var filtered []string
		for _, s := range resolved.ClaudeCode.MCPServers {
			if allowedSet[s] {
				filtered = append(filtered, s)
			}
		}
		resolved.ClaudeCode.MCPServers = filtered
	} else {
		// Block specific servers.
		blockedSet := make(map[string]bool, len(blocked))
		for _, b := range blocked {
			blockedSet[b] = true
		}

		var filtered []string
		for _, s := range resolved.ClaudeCode.MCPServers {
			if !blockedSet[s] {
				filtered = append(filtered, s)
			}
		}
		resolved.ClaudeCode.MCPServers = filtered
	}

	return resolved
}

// cloneGdevConfig creates a deep copy of a GdevConfig.
func cloneGdevConfig(cfg *types.GdevConfig) *types.GdevConfig {
	if cfg == nil {
		return nil
	}

	result := &types.GdevConfig{
		Version:     cfg.Version,
		GdevVersion: cfg.GdevVersion,
		Profile:     cfg.Profile,
		Security: types.SecurityConfig{
			Level:          cfg.Security.Level,
			AgeGating:      cloneBoolPtr(cfg.Security.AgeGating),
			ScriptBlocking: cloneBoolPtr(cfg.Security.ScriptBlocking),
			LockEnforce:    cloneBoolPtr(cfg.Security.LockEnforce),
			VulnScanning:   cloneBoolPtr(cfg.Security.VulnScanning),
		},
		ClaudeCode: types.ClaudeCodeConfig{
			Enabled:         cloneBoolPtr(cfg.ClaudeCode.Enabled),
			PermissionLevel: cfg.ClaudeCode.PermissionLevel,
		},
		Infrastructure: cfg.Infrastructure,
		Git:            cfg.Git,
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

	// Clone Client.
	if cfg.Client != nil {
		client := *cfg.Client
		if len(cfg.Client.Compliance) > 0 {
			client.Compliance = make([]string, len(cfg.Client.Compliance))
			copy(client.Compliance, cfg.Client.Compliance)
		}
		if len(cfg.Client.AllowedMCP) > 0 {
			client.AllowedMCP = make([]string, len(cfg.Client.AllowedMCP))
			copy(client.AllowedMCP, cfg.Client.AllowedMCP)
		}
		if len(cfg.Client.BlockedMCP) > 0 {
			client.BlockedMCP = make([]string, len(cfg.Client.BlockedMCP))
			copy(client.BlockedMCP, cfg.Client.BlockedMCP)
		}
		result.Client = &client
	}

	return result
}

// cloneBoolPtr returns a copy of a *bool value.
func cloneBoolPtr(p *bool) *bool {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}
