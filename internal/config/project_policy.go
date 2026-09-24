package config

import (
	"fmt"
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
	committed, err := ResolveConfig(nil, nil, project, nil, false)
	if err != nil {
		return nil, fmt.Errorf("resolving project config: %w", err)
	}
	effective, err := ResolveConfig(nil, nil, project, local, false)
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
// client MCP policy, which replaces any earlier one because .qsdev.yaml is
// authoritative for it.
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
}

// clientComplianceOverlay returns the layer-3 overlay for the committed
// client security level, or nil when the project declares none.
func (p *ProjectPolicy) clientComplianceOverlay() *types.QsdevConfig {
	if p.Committed.Client == nil || p.Committed.Client.SecurityLevel == "" {
		return nil
	}
	return ComplianceLevelToConfig(p.Committed.Client.SecurityLevel)
}

// Warnings describes each setting the security floor raised: a local
// override (or a project setting) weaker than what the project or its client
// compliance level requires.
func (p *ProjectPolicy) Warnings() []string {
	warnings := make([]string, 0, len(p.Effective.Violations))
	for _, v := range p.Effective.Violations {
		attempted := "unset"
		if v.Attempted != nil {
			attempted = fmt.Sprint(v.Attempted)
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
