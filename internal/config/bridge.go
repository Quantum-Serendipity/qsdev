package config

import (
	"maps"
	"path/filepath"
	"slices"

	"github.com/Quantum-Serendipity/qsdev/internal/tier"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// ProjectNameFor returns the project name cfg records, or, for a config
// written before it recorded one, the name of the directory at projectRoot.
func ProjectNameFor(cfg *types.QsdevConfig, projectRoot string) string {
	if cfg != nil && cfg.ProjectName != "" {
		return cfg.ProjectName
	}
	return filepath.Base(projectRoot)
}

// ConfigToAnswers maps a QsdevConfig to WizardAnswers for downstream generators.
// It is the single config-to-answers converter: `qsdev init` join mode uses it
// to rebuild a teammate's answers from the committed .qsdev.yaml, so every
// field devinit persists there must be read back here. Config-level types
// (LanguageConfig, ServiceConfig) become their wizard equivalents
// (LanguageChoice, ServiceChoice), and fields without a config equivalent get
// the same defaults the create path applies.
func ConfigToAnswers(cfg *types.QsdevConfig, detected types.DetectedProject, projectRoot string) types.WizardAnswers {
	answers := types.WizardAnswers{
		ProjectRoot: projectRoot,
		ProjectName: ProjectNameFor(cfg, projectRoot),
		Detected:    detected,
		Direnv:      true, // Default: direnv is always on.
		Confirmed:   true,
	}

	// Map LanguageConfig -> LanguageChoice.
	for _, lang := range cfg.Languages {
		answers.Languages = append(answers.Languages, types.LanguageChoice{
			Name:           lang.Name,
			Version:        lang.Version,
			PackageManager: lang.PackageManager,
		})
	}

	// Map ServiceConfig -> ServiceChoice.
	for _, svc := range cfg.Services {
		answers.Services = append(answers.Services, types.ServiceChoice{
			Name:     svc.Name,
			Version:  svc.Version,
			Settings: maps.Clone(svc.Options),
		})
	}

	answers.ExtraPackages = slices.Clone(cfg.Packages)
	answers.Overlays = slices.Clone(cfg.Overlays)

	mapClaudeCode(cfg, &answers)

	// Map Tools to EnabledTools.
	if len(cfg.Tools.Enabled) > 0 || len(cfg.Tools.Disabled) > 0 {
		answers.EnabledTools = make(map[string]bool)
		for _, t := range cfg.Tools.Enabled {
			answers.EnabledTools[t] = true
		}
		for _, t := range cfg.Tools.Disabled {
			answers.EnabledTools[t] = false
		}
	}

	// Security level: the stricter of the project and client levels.
	level := EffectiveSecurityLevel(cfg)
	answers.ComplianceLevel = level
	answers.HookTier = level

	answers.Tier = ConfigTier(cfg)

	// `profile` is the project-type profile and `infra_profile` the
	// infrastructure profile. A version 1 file that held the infra profile
	// under `profile` has already been split by the v1->v2 migration.
	answers.ProjectTypeProfile = cfg.Profile
	answers.ProfileName = cfg.InfraProfile

	answers.Infrastructure = cloneInfra(cfg.Infrastructure)
	answers.BranchPattern = cfg.Git.BranchPattern
	answers.HookPolicy = cfg.Hooks.Clone()
	answers.Java = cloneJava(cfg.Java)
	answers.Cloud = cfg.Cloud

	return answers
}

// ConfigTier returns cfg's tier, inferred from its legacy fields when the
// file predates the always-persisted tier field.
func ConfigTier(cfg *types.QsdevConfig) string {
	if cfg.Tier != "" {
		return cfg.Tier
	}
	return tier.Infer(cfg.ClaudeCode.PermissionLevel, cfg.ClaudeCode.MCPServers).String()
}

// cloneJava returns a deep copy of a JavaConfig.
func cloneJava(in types.JavaConfig) types.JavaConfig {
	return types.JavaConfig{RepositoryAllowlist: slices.Clone(in.RepositoryAllowlist)}
}

// mapClaudeCode maps the claude_code block and the hook choices it implies.
func mapClaudeCode(cfg *types.QsdevConfig, answers *types.WizardAnswers) {
	// An absent claude_code.enabled predates the key always being written,
	// when Claude Code was on by default; an explicit false is honoured.
	answers.ClaudeCode = ClaudeCodeEnabled(cfg)
	answers.PermissionLevel = cfg.ClaudeCode.PermissionLevel
	// Like FillDefaults on the create path, default the permission level only
	// when no tier is set: an explicit tier supplies its own preset.
	if answers.PermissionLevel == "" && answers.ClaudeCode && cfg.Tier == "" {
		answers.PermissionLevel = "standard"
	}
	answers.Skills = slices.Clone(cfg.ClaudeCode.Skills)
	answers.MCPServers = slices.Clone(cfg.ClaudeCode.MCPServers)

	answers.Hooks = securityToHookChoices(cfg)
	// Self-protection is always on when Claude Code is enabled, matching the
	// invariant FillDefaults enforces on the create path.
	answers.Hooks.SelfProtection = answers.ClaudeCode
}

// EffectiveSecurityLevel returns the stricter of security.level and
// client.security_level (a client can raise but never lower the floor): the
// compliance level the project's generated hardening must meet.
func EffectiveSecurityLevel(cfg *types.QsdevConfig) string {
	level := cfg.Security.Level
	if cfg.Client != nil && cfg.Client.SecurityLevel != "" {
		if CompareComplianceLevels(cfg.Client.SecurityLevel, level) > 0 {
			level = cfg.Client.SecurityLevel
		}
	}
	return level
}

// securityToHookChoices maps a QsdevConfig's security settings to HookChoices:
// the always-on safety block plus the hooks its effective level implies.
func securityToHookChoices(cfg *types.QsdevConfig) types.HookChoices {
	hc := levelHookChoices(EffectiveSecurityLevel(cfg))
	hc.SafetyBlock = true // Always on.
	return hc
}

// levelHookChoices returns the Claude Code hooks a compliance level requires:
// audit-log for strict, none beyond the safety block otherwise. No level
// implies auto-format or pre-commit: neither registers a Claude Code hook,
// and the commit-time checks a level requires are devenv git hooks (the
// compliance level's required_pre_commit_hooks).
func levelHookChoices(level string) types.HookChoices {
	var hc types.HookChoices
	if level == "strict" {
		hc.AuditLog = true
	}
	return hc
}

// cloneInfra returns a deep copy of an InfraConfig.
func cloneInfra(in types.InfraConfig) types.InfraConfig {
	out := in
	out.RegistryProxyOverrides = maps.Clone(in.RegistryProxyOverrides)
	out.RegistryProxyPaths = maps.Clone(in.RegistryProxyPaths)
	return out
}
