package devinit

import (
	"fmt"
	"maps"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/validation"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// ProfileToAnswers converts a project-type Profile into a fully populated
// WizardAnswers struct. The projectRoot and projectName are passed through
// directly since profiles do not encode per-project paths. It fails when the
// profile names a hook preset that does not exist.
func ProfileToAnswers(p Profile, projectRoot, projectName string) (types.WizardAnswers, error) {
	answers := types.WizardAnswers{
		ProjectName:     projectName,
		ProjectRoot:     projectRoot,
		Direnv:          p.Direnv,
		ClaudeCode:      p.ClaudeCode,
		PermissionLevel: p.PermissionLevel,
		Tier:            p.Tier,
		Skills:          copyStrings(p.Skills),
		ExtraPackages:   copyStrings(p.ExtraPackages),
		MCPServers:      copyStrings(p.MCPServers),
		GitHooks:        copyStrings(p.GitHooks),
		ProfileName:     p.InfraProfile,
		Confirmed:       true,
	}

	// Convert LanguageSpec -> LanguageChoice.
	for _, ls := range p.Languages {
		answers.Languages = append(answers.Languages, types.LanguageChoice{
			Name:           ls.Name,
			Version:        ls.Version,
			PackageManager: ls.PackageManager,
		})
	}

	// Convert service name strings -> ServiceChoice.
	for _, svc := range p.Services {
		answers.Services = append(answers.Services, types.ServiceChoice{
			Name: svc,
		})
	}

	// Convert hook name strings -> HookChoices booleans.
	hooks, err := hooksFromStrings(p.Hooks)
	if err != nil {
		return types.WizardAnswers{}, fmt.Errorf("profile hooks: %w", err)
	}
	answers.Hooks = hooks
	applyAnswerInvariants(&answers)

	return answers, nil
}

// hooksFromStrings maps hook preset names to HookChoices. The selectable names
// are the catalog's hook presets; an unknown or misspelled name is an error
// rather than a silently disabled hook.
func hooksFromStrings(hooks []string) (types.HookChoices, error) {
	var hc types.HookChoices
	for _, h := range hooks {
		if !validation.IsValidHookPreset(h) {
			return types.HookChoices{}, fmt.Errorf("unknown hook preset %q; valid presets: %s",
				h, strings.Join(validation.HookPresets(), ", "))
		}
		if err := hc.EnableHook(h); err != nil {
			return types.HookChoices{}, fmt.Errorf("hook preset %q: %w", h, err)
		}
	}
	return hc, nil
}

// applyAnswerInvariants enforces settings that hold for every answer source.
// Self-protection is always on when Claude Code is enabled: without it the
// agent could edit its own settings and hook scripts.
func applyAnswerInvariants(a *types.WizardAnswers) {
	if a.ClaudeCode {
		a.Hooks.SelfProtection = true
	}
}

// MergeProfileWithFlags merges a profile-derived WizardAnswers (base) with
// flag-derived overrides. The changed map tracks which flag fields were
// explicitly set by the user; only those fields override the base.
//
// Language overrides REPLACE the base languages entirely.
// Service overrides APPEND to the base services (deduplicating by name).
// Environment variables merge per key, and agent tools per setting.
// All other fields use simple replacement when the key is present in changed.
// The result satisfies applyAnswerInvariants.
func MergeProfileWithFlags(base types.WizardAnswers, overrides types.WizardAnswers, changed map[string]bool) types.WizardAnswers {
	result := base

	if changed["project_name"] {
		result.ProjectName = overrides.ProjectName
	}
	if changed["project_root"] {
		result.ProjectRoot = overrides.ProjectRoot
	}
	if changed["languages"] {
		// Language overrides REPLACE entirely.
		result.Languages = overrides.Languages
	}
	if changed["services"] {
		// Service overrides APPEND (deduplicate by name).
		result.Services = appendServicesUnique(base.Services, overrides.Services)
	}
	if changed["direnv"] {
		result.Direnv = overrides.Direnv
	}
	if changed["claude_code"] {
		result.ClaudeCode = overrides.ClaudeCode
	}
	if changed["permission_level"] {
		result.PermissionLevel = overrides.PermissionLevel
	}
	if changed["skills"] {
		result.Skills = overrides.Skills
	}
	if changed["hooks"] {
		result.Hooks = overrides.Hooks
	}
	if changed["git_hooks"] {
		result.GitHooks = overrides.GitHooks
	}
	if changed["extra_packages"] {
		result.ExtraPackages = overrides.ExtraPackages
	}
	if changed["mcp_servers"] {
		result.MCPServers = overrides.MCPServers
	}
	if changed["profile_name"] {
		result.ProfileName = overrides.ProfileName
	}
	if changed["confirmed"] {
		result.Confirmed = overrides.Confirmed
	}
	if changed["tier"] {
		result.Tier = overrides.Tier
	}
	if changed["env_vars"] {
		// Flag values win per key; other base variables are kept.
		result.EnvVars = make(map[string]string, len(base.EnvVars)+len(overrides.EnvVars))
		maps.Copy(result.EnvVars, base.EnvVars)
		maps.Copy(result.EnvVars, overrides.EnvVars)
	}
	if changed["nix_hardening_guide"] {
		result.NixHardeningGuide = overrides.NixHardeningGuide
	}
	if changed["project_type_profile"] {
		result.ProjectTypeProfile = overrides.ProjectTypeProfile
	}
	if changed["agent_tools"] {
		result.AgentTools = overrides.AgentTools
	}
	mergeAgentToolFlags(&result.AgentTools, overrides.AgentTools, changed)

	applyAnswerInvariants(&result)
	return result
}

// mergeAgentToolFlags overrides individual agent-tool settings whose flags were
// set, so one --agent-* flag does not reset the others to their flag defaults.
func mergeAgentToolFlags(dst *types.AgentToolsAnswers, src types.AgentToolsAnswers, changed map[string]bool) {
	if changed["agent_postmortem"] {
		dst.PostmortemEnabled = src.PostmortemEnabled
	}
	if changed["agent_version_sentinel"] {
		dst.VersionSentinel = src.VersionSentinel
	}
	if changed["agent_semble"] {
		dst.SembleEnabled = src.SembleEnabled
	}
	if changed["agent_semble_mode"] {
		dst.SembleMode = src.SembleMode
	}
	if changed["agent_semble_text_files"] {
		dst.SembleTextFiles = src.SembleTextFiles
	}
}

// appendServicesUnique appends src services to dst, skipping any that already
// exist (by name) in dst.
func appendServicesUnique(dst, src []types.ServiceChoice) []types.ServiceChoice {
	existing := make(map[string]bool, len(dst))
	for _, svc := range dst {
		existing[svc.Name] = true
	}

	merged := make([]types.ServiceChoice, len(dst))
	copy(merged, dst)

	for _, svc := range src {
		if !existing[svc.Name] {
			merged = append(merged, svc)
			existing[svc.Name] = true
		}
	}
	return merged
}

// copyStrings returns a shallow copy of a string slice, or nil if the input is nil.
func copyStrings(s []string) []string {
	if s == nil {
		return nil
	}
	c := make([]string, len(s))
	copy(c, s)
	return c
}
