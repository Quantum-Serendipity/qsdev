package config

import (
	"fmt"
	"maps"
	"path/filepath"
	"slices"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// ProjectPolicy is a project's configuration resolved through ResolveConfig,
// the single place the declared security floor, the client compliance overlay
// and the client MCP policy are applied. `qsdev init` (create, join and
// update) loads it so what .qsdev.yaml declares reaches generation.
type ProjectPolicy struct {
	// Committed is the committed .qsdev.yaml resolved on its own: the
	// team-shared configuration with the client compliance overlay applied
	// and the client MCP policy enforced. Join rebuilds its answers from it;
	// it never carries a developer's local overrides, so nothing local is
	// written back into .qsdev.yaml.
	Committed *types.QsdevConfig
	// Effective adds the developer's .qsdev.local.yaml on top of Committed,
	// with the security floor enforced. Its Violations are the settings the
	// floor raised.
	Effective *ResolvedConfig
}

// LoadProjectPolicy parses projectRoot's .qsdev.yaml and .qsdev.local.yaml
// and resolves them through ResolveConfig. qsdev has no organization config
// source, so the organization-defaults layer is empty: a project without a
// security block keeps the settings it was generated with instead of being
// raised to the built-in defaults (DefaultQsdevConfig).
//
// A missing .qsdev.yaml is reported as an error wrapping fs.ErrNotExist; a
// missing local file is not an error.
func LoadProjectPolicy(projectRoot string) (*ProjectPolicy, error) {
	b := branding.Get()
	project, err := ParseQsdevConfig(filepath.Join(projectRoot, b.ConfigFile))
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", b.ConfigFile, err)
	}
	local, err := ParseLocalConfig(filepath.Join(projectRoot, b.LocalConfig))
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", b.LocalConfig, err)
	}
	return ResolveProjectPolicy(project, local)
}

// ResolveProjectPolicy resolves an already-parsed project config and optional
// local overrides with an empty organization-defaults layer.
func ResolveProjectPolicy(project *types.QsdevConfig, local *LocalConfig) (*ProjectPolicy, error) {
	committed, err := ResolveConfig(nil, project, nil)
	if err != nil {
		return nil, fmt.Errorf("resolving project config: %w", err)
	}
	effective, err := ResolveConfig(nil, project, local)
	if err != nil {
		return nil, fmt.Errorf("resolving local overrides: %w", err)
	}
	return &ProjectPolicy{Committed: committed.Config, Effective: effective}, nil
}

// Apply tightens answers to the policy; it never loosens a choice. It raises
// the compliance level (and, with Claude Code, the hooks that level implies)
// to the effective security floor, enables the tools the client compliance
// level requires unless the answers record a decision about them, fills an
// unset permission level from the client compliance level, and installs the
// client MCP policy, the git branch naming pattern and the hook settings (the
// `hooks` block), which replace any earlier ones because .qsdev.yaml is
// authoritative for them.
func (p *ProjectPolicy) Apply(a *types.WizardAnswers) {
	level := effectiveSecurityLevel(p.Effective.Config)
	if CompareComplianceLevels(level, a.ComplianceLevel) > 0 {
		a.ComplianceLevel = level
	}

	overlay := p.clientComplianceOverlay()
	if a.ClaudeCode {
		implied := levelHookChoices(level)
		a.Hooks.PreCommit = a.Hooks.PreCommit || implied.PreCommit
		a.Hooks.AuditLog = a.Hooks.AuditLog || implied.AuditLog
		a.Hooks.AutoFormat = a.Hooks.AutoFormat || implied.AutoFormat
		if a.PermissionLevel == "" && overlay != nil {
			a.PermissionLevel = overlay.ClaudeCode.PermissionLevel
		}
	}
	if overlay != nil {
		for _, name := range overlay.Tools.Enabled {
			if _, decided := a.EnabledTools[name]; decided {
				continue
			}
			if a.EnabledTools == nil {
				a.EnabledTools = make(map[string]bool)
			}
			a.EnabledTools[name] = true
		}
	}

	a.MCPPolicy = ClientMCPPolicy(p.Committed)
	a.MCPServers = a.MCPPolicy.Filter(a.MCPServers)
	a.BranchPattern = p.Committed.Git.BranchPattern
	a.HookPolicy = p.Committed.Hooks.Clone()
}

// ApplyLocal adds the developer's .qsdev.local.yaml, as ResolveConfig applied
// it (only additions and tightenings; see ResolvedConfig.Local), to answers:
// extra packages, languages and services (a known one takes a local version,
// and a package manager or service option only where the answers set none),
// enabled tools, Claude Code skills and MCP servers (filtered by the client
// MCP policy), and a permission level stricter than the answers' effective
// one. Call it after Apply, on a copy of the answers used only for
// generation: .qsdev.yaml and the saved answers must keep the committed
// choices, so a local override never reaches the team.
func (p *ProjectPolicy) ApplyLocal(a *types.WizardAnswers) {
	local := p.Effective.Local
	if local == nil {
		return
	}
	a.ExtraPackages = mergeUnionStrings(a.ExtraPackages, local.Packages)
	a.Languages = addLocalLanguages(a.Languages, local.Languages)
	a.Services = addLocalServices(a.Services, local.Services)
	for _, name := range local.Tools.Enabled {
		if a.EnabledTools == nil {
			a.EnabledTools = make(map[string]bool)
		}
		a.EnabledTools[name] = true
	}
	if !a.ClaudeCode {
		return
	}
	a.Skills = mergeUnionStrings(a.Skills, local.ClaudeCode.Skills)
	a.MCPServers = a.MCPPolicy.Filter(mergeUnionStrings(a.MCPServers, local.ClaudeCode.MCPServers))
	if level := local.ClaudeCode.PermissionLevel; level != "" {
		current := EffectivePermissionLevel(a.PermissionLevel, a.Tier, a.MCPServers)
		if c, ok := ComparePermissionLevels(level, current); ok && c > 0 {
			a.PermissionLevel = level
		}
	}
}

// addLocalLanguages merges local languages into the answers' by name.
func addLocalLanguages(langs []types.LanguageChoice, local []types.LanguageConfig) []types.LanguageChoice {
	for _, l := range local {
		i := slices.IndexFunc(langs, func(c types.LanguageChoice) bool { return c.Name == l.Name })
		if i < 0 {
			langs = append(langs, types.LanguageChoice{Name: l.Name, Version: l.Version, PackageManager: l.PackageManager})
			continue
		}
		if l.Version != "" {
			langs[i].Version = l.Version
		}
		if langs[i].PackageManager == "" {
			langs[i].PackageManager = l.PackageManager
		}
	}
	return langs
}

// addLocalServices merges local services into the answers' by name.
func addLocalServices(svcs []types.ServiceChoice, local []types.ServiceConfig) []types.ServiceChoice {
	for _, s := range local {
		i := slices.IndexFunc(svcs, func(c types.ServiceChoice) bool { return c.Name == s.Name })
		if i < 0 {
			svcs = append(svcs, types.ServiceChoice{Name: s.Name, Version: s.Version, Settings: maps.Clone(s.Options)})
			continue
		}
		if s.Version != "" {
			svcs[i].Version = s.Version
		}
		for k, v := range s.Options {
			if _, set := svcs[i].Settings[k]; set {
				continue
			}
			if svcs[i].Settings == nil {
				svcs[i].Settings = make(map[string]string, len(s.Options))
			}
			svcs[i].Settings[k] = v
		}
	}
	return svcs
}

// clientComplianceOverlay returns the layer-2 overlay for the committed
// client security level, or nil when the project declares none.
func (p *ProjectPolicy) clientComplianceOverlay() *types.QsdevConfig {
	if p.Committed.Client == nil || p.Committed.Client.SecurityLevel == "" {
		return nil
	}
	return ComplianceLevelToConfig(p.Committed.Client.SecurityLevel)
}

// Warnings describes each setting the security floor raised or ignored: a
// local override (or a project setting) weaker than what the project or its
// client compliance level requires, or a local override that would remove
// or loosen a committed setting.
func (p *ProjectPolicy) Warnings() []string {
	warnings := make([]string, 0, len(p.Effective.Violations))
	for _, v := range p.Effective.Violations {
		attempted := "unset"
		if v.Attempted != nil {
			attempted = fmt.Sprint(v.Attempted)
		}
		if v.Enforced == nil {
			warnings = append(warnings, fmt.Sprintf("%s: %s ignored (%s)", v.Field, attempted, v.Reason))
			continue
		}
		warnings = append(warnings, fmt.Sprintf("%s: %s raised to %v (%s)", v.Field, attempted, v.Enforced, v.Reason))
	}
	return warnings
}

// ClientMCPPolicy returns the MCP server policy cfg's client block declares.
func ClientMCPPolicy(cfg *types.QsdevConfig) types.MCPPolicy {
	if cfg == nil || cfg.Client == nil {
		return types.MCPPolicy{}
	}
	return types.MCPPolicy{
		Allowed: slices.Clone(cfg.Client.AllowedMCP),
		Blocked: slices.Clone(cfg.Client.BlockedMCP),
	}
}
