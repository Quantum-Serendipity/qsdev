package config

import (
	"path/filepath"

	"github.com/Quantum-Serendipity/qsdev/internal/tier"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// ConfigToAnswers maps a QsdevConfig to WizardAnswers for downstream generators.
// It converts config-level types (LanguageConfig, ServiceConfig) into their
// wizard equivalents (LanguageChoice, ServiceChoice) and sets reasonable
// defaults for fields that don't have config equivalents.
func ConfigToAnswers(cfg *types.QsdevConfig, detected types.DetectedProject, projectRoot string) types.WizardAnswers {
	answers := types.WizardAnswers{
		ProjectRoot: projectRoot,
		ProjectName: filepath.Base(projectRoot),
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
			Settings: svc.Options,
		})
	}

	// Map Security -> HookChoices.
	answers.Hooks = securityToHookChoices(cfg)

	// Map ClaudeCode.
	if cfg.ClaudeCode.Enabled != nil && *cfg.ClaudeCode.Enabled {
		answers.ClaudeCode = true
	}
	answers.PermissionLevel = cfg.ClaudeCode.PermissionLevel
	answers.Skills = copyStrings(cfg.ClaudeCode.Skills)
	answers.MCPServers = copyStrings(cfg.ClaudeCode.MCPServers)

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

	// Set compliance level from security config or client config.
	if cfg.Client != nil && cfg.Client.SecurityLevel != "" {
		answers.HookTier = cfg.Client.SecurityLevel
	} else if cfg.Security.Level != "" {
		answers.HookTier = cfg.Security.Level
	}

	// Set tier (infer from legacy fields if not explicit).
	if cfg.Tier != "" {
		answers.Tier = cfg.Tier
	} else {
		answers.Tier = tier.Infer(cfg.ClaudeCode.PermissionLevel, cfg.ClaudeCode.MCPServers).String()
	}

	// Set profile.
	if cfg.Profile != "" {
		answers.ProfileName = cfg.Profile
	}

	return answers
}

// securityToHookChoices maps a QsdevConfig's security settings to HookChoices.
// The mapping is based on compliance level:
//   - baseline: safety-block
//   - enhanced: safety-block + pre-commit
//   - strict: safety-block + pre-commit + audit-log + auto-format
func securityToHookChoices(cfg *types.QsdevConfig) types.HookChoices {
	hc := types.HookChoices{
		SafetyBlock: true, // Always on.
	}

	level := cfg.Security.Level
	if cfg.Client != nil && cfg.Client.SecurityLevel != "" {
		if CompareComplianceLevels(cfg.Client.SecurityLevel, level) > 0 {
			level = cfg.Client.SecurityLevel
		}
	}

	switch level {
	case "enhanced":
		hc.PreCommit = true
	case "strict":
		hc.PreCommit = true
		hc.AuditLog = true
		hc.AutoFormat = true
	}

	return hc
}

// copyStrings returns a copy of a string slice, or nil if input is nil.
func copyStrings(s []string) []string {
	if s == nil {
		return nil
	}
	out := make([]string, len(s))
	copy(out, s)
	return out
}
