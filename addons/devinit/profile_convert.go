package devinit

import (
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

// ProfileToAnswers converts a project-type Profile into a fully populated
// WizardAnswers struct. The projectRoot and projectName are passed through
// directly since profiles do not encode per-project paths.
func ProfileToAnswers(p Profile, projectRoot, projectName string) types.WizardAnswers {
	answers := types.WizardAnswers{
		ProjectName:    projectName,
		ProjectRoot:    projectRoot,
		Direnv:         p.Direnv,
		ClaudeCode:     p.ClaudeCode,
		PermissionLevel: p.PermissionLevel,
		Skills:         copyStrings(p.Skills),
		ExtraPackages:  copyStrings(p.ExtraPackages),
		MCPServers:     copyStrings(p.MCPServers),
		GitHooks:       copyStrings(p.GitHooks),
		ProfileName:    p.InfraProfile,
		Confirmed:      true,
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
	answers.Hooks = hooksFromStrings(p.Hooks)

	return answers
}

// hooksFromStrings maps a slice of hook name strings to HookChoices boolean fields.
func hooksFromStrings(hooks []string) types.HookChoices {
	var hc types.HookChoices
	for _, h := range hooks {
		switch h {
		case "auto-format":
			hc.AutoFormat = true
		case "safety-block":
			hc.SafetyBlock = true
		case "pre-commit":
			hc.PreCommit = true
		case "audit-log":
			hc.AuditLog = true
		}
	}
	return hc
}

// MergeProfileWithFlags merges a profile-derived WizardAnswers (base) with
// flag-derived overrides. The changed map tracks which flag fields were
// explicitly set by the user; only those fields override the base.
//
// Language overrides REPLACE the base languages entirely.
// Service overrides APPEND to the base services (deduplicating by name).
// All other fields use simple replacement when the key is present in changed.
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

	return result
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
