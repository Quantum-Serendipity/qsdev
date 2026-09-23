package claudecode

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/Quantum-Serendipity/qsdev/internal/sliceutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// EnforcementTier reports that Claude Code enforces security policy via
// PreToolUse hooks rather than a kernel-level sandbox.
func (a *Adapter) EnforcementTier() aiframework.EnforcementTier {
	return aiframework.TierHook
}

// TranslatePermissions renders the framework-agnostic permission policy into a
// real Claude Code .claude/settings.json via GenerateSettings (which applies
// the preset, ecosystem, and base deny rules and the always-on hooks), then
// merges the policy's explicit allow/deny/ask rules onto the rendered
// permissions block so a translated deny policy is actually enforced. A nil
// policy renders the configured defaults.
func (a *Adapter) TranslatePermissions(ctx context.Context, policy *aiframework.PermissionPolicy) (*aiframework.PermissionArtifacts, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if policy == nil {
		policy = &aiframework.PermissionPolicy{}
	}

	cfg := a.cfg
	answers := baseAnswers()
	if policy.Preset != "" {
		cfg.DefaultPermissions = PermissionPreset(policy.Preset)
		answers.PermissionLevel = policy.Preset
	}

	base, err := GenerateSettings(answers, a.registry, cfg)
	if err != nil {
		return nil, fmt.Errorf("rendering claude code permissions: %w", err)
	}

	merged, err := mergeRulesIntoSettings(base,
		aiframework.RulePatterns(policy.AllowRules),
		aiframework.RulePatterns(policy.DenyRules),
		aiframework.RulePatterns(policy.AskRules))
	if err != nil {
		return nil, err
	}

	return &aiframework.PermissionArtifacts{
		GeneratedFiles: []types.GeneratedFile{*merged},
		ActiveTier:     aiframework.TierHook,
	}, nil
}

// TranslateIgnorePatterns expresses exclusion patterns the only way Claude Code
// supports them: as Read(<pattern>) deny rules inside settings.json. It renders
// the real base settings via GenerateSettings, then merges one Read() deny rule
// per pattern.
func (a *Adapter) TranslateIgnorePatterns(ctx context.Context, patterns []aiframework.IgnorePattern) ([]types.GeneratedFile, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	base, err := GenerateSettings(baseAnswers(), a.registry, a.cfg)
	if err != nil {
		return nil, fmt.Errorf("rendering claude code settings for ignore patterns: %w", err)
	}

	reads := make([]string, 0, len(patterns))
	for _, p := range patterns {
		if p.Pattern != "" {
			reads = append(reads, fmt.Sprintf("Read(%s)", p.Pattern))
		}
	}

	merged, err := mergeRulesIntoSettings(base, nil, reads, nil)
	if err != nil {
		return nil, err
	}
	return []types.GeneratedFile{*merged}, nil
}

// InjectCredentials translates a credential scope into concrete sandbox
// exclusions: a set of credential file globs to keep out of the agent's reach
// and a passthrough map of required API-key environment variables. It never
// writes secret values, and it emits no file that embeds the scope's sandbox
// filter globs, so credentials cannot leak into generated artifacts.
func (a *Adapter) InjectCredentials(ctx context.Context, scope *aiframework.CredentialScope) (*aiframework.CredentialArtifacts, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if scope == nil {
		scope = &aiframework.CredentialScope{}
	}
	return &aiframework.CredentialArtifacts{
		EnvVars:      credentialEnvVars(scope.APIKeys),
		ExcludePaths: credentialExcludePaths(scope),
	}, nil
}

// ReportGaps enumerates, for each requested deny rule, the gap between the
// kernel-level isolation the rule ideally wants and the hook-level enforcement
// Claude Code actually provides. A nil policy has no gaps.
func (a *Adapter) ReportGaps(ctx context.Context, policy *aiframework.PermissionPolicy) []aiframework.EnforcementGap {
	if ctx.Err() != nil || policy == nil {
		return nil
	}
	var gaps []aiframework.EnforcementGap
	for _, rule := range policy.DenyRules {
		gaps = append(gaps, aiframework.EnforcementGap{
			Rule:         rule,
			RequiredTier: aiframework.TierKernel,
			ActualTier:   aiframework.TierHook,
			Description:  "Claude Code enforces via PreToolUse hooks, not kernel-level sandboxing",
			Mitigation:   "enable external bubblewrap wrapping via qsdev sandbox exec",
		})
	}
	return gaps
}

// baseAnswers returns the answers for a standalone settings.json render, with
// the always-on Claude Code hooks applied so the rendered file never lacks
// self-protection.
func baseAnswers() types.WizardAnswers {
	answers := types.WizardAnswers{ClaudeCode: true}
	answers.ApplyClaudeHookDefaults()
	return answers
}

// mergeRulesIntoSettings parses a rendered settings.json, unions the supplied
// allow/deny/ask patterns into its permissions block (de-duplicated, order
// preserved), and re-encodes it while preserving the file's path, mode, and
// merge strategy.
func mergeRulesIntoSettings(base *types.GeneratedFile, allow, deny, ask []string) (*types.GeneratedFile, error) {
	return editSettings(base, func(settings *SettingsJSON) {
		if len(allow) > 0 {
			settings.Permissions.Allow = sliceutil.Dedup(append(settings.Permissions.Allow, allow...))
		}
		if len(deny) > 0 {
			settings.Permissions.Deny = sliceutil.Dedup(append(settings.Permissions.Deny, deny...))
		}
		if len(ask) > 0 {
			settings.Permissions.Ask = sliceutil.Dedup(append(settings.Permissions.Ask, ask...))
		}
	})
}

// applySandboxPolicy merges a framework-agnostic sandbox policy into the
// rendered settings.json sandbox block (Claude Code's schema): writable paths
// become filesystem.allowWrite, read-only paths filesystem.denyWrite, denied
// paths both filesystem.denyRead and denyWrite, and allowed network
// destinations network.allowedDomains. Claude Code's sandbox expresses
// network access only as an allow list, so a policy with a network deny list
// is rejected rather than silently rendered without it.
func applySandboxPolicy(base *types.GeneratedFile, policy *aiframework.SandboxPolicy) (*types.GeneratedFile, error) {
	if len(policy.NetworkDenied) > 0 {
		return nil, fmt.Errorf("claude code sandbox cannot express network deny rules %v: declare allowed destinations instead", policy.NetworkDenied)
	}
	return editSettings(base, func(settings *SettingsJSON) {
		if settings.Sandbox == nil {
			settings.Sandbox = &SandboxConfig{}
		}
		sb := settings.Sandbox
		if sb.Filesystem == nil {
			sb.Filesystem = &SandboxFilesystem{}
		}
		fs := sb.Filesystem
		fs.AllowWrite = sliceutil.Dedup(append(fs.AllowWrite, policy.WritablePaths...))
		fs.DenyWrite = sliceutil.Dedup(slices.Concat(fs.DenyWrite, policy.ReadOnlyPaths, policy.DeniedPaths))
		fs.DenyRead = sliceutil.Dedup(append(fs.DenyRead, policy.DeniedPaths...))
		if len(policy.NetworkAllowed) > 0 {
			if sb.Network == nil {
				sb.Network = &SandboxNetwork{}
			}
			sb.Network.AllowedDomains = sliceutil.Dedup(append(sb.Network.AllowedDomains, policy.NetworkAllowed...))
		}
	})
}

// editSettings parses a rendered settings.json, applies edit, and re-encodes
// it while preserving the file's path, mode, and merge strategy.
func editSettings(base *types.GeneratedFile, edit func(*SettingsJSON)) (*types.GeneratedFile, error) {
	var settings SettingsJSON
	if err := json.Unmarshal(base.Content, &settings); err != nil {
		return nil, fmt.Errorf("parsing rendered settings.json: %w", err)
	}
	edit(&settings)

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling merged settings.json: %w", err)
	}
	data = append(data, '\n')

	out := *base
	out.Content = data
	return &out, nil
}

// credentialEnvVars maps each required API key onto a ${VAR} passthrough
// reference. It records which variables the agent legitimately needs without
// ever embedding a secret value. Returns nil when there is nothing to pass.
func credentialEnvVars(keys []aiframework.APIKeyRequirement) map[string]string {
	env := make(map[string]string, len(keys))
	for _, k := range keys {
		if k.Required && k.EnvVar != "" {
			env[k.EnvVar] = "${" + k.EnvVar + "}"
		}
	}
	if len(env) == 0 {
		return nil
	}
	return env
}

// credentialExcludePaths builds the set of credential file globs to exclude
// from the agent sandbox: a baseline of well-known secret locations plus any
// cloud-provider credential directories implied by the scope.
func credentialExcludePaths(scope *aiframework.CredentialScope) []string {
	paths := []string{
		"**/.env", "**/.env.*", "**/*.pem", "**/*.key",
		"**/secrets/**", "~/.ssh/**",
	}
	if scope.AWSProfile != "" {
		paths = append(paths, "~/.aws/**")
	}
	if scope.GCPProject != "" {
		paths = append(paths, "~/.config/gcloud/**")
	}
	if scope.AzureSubscription != "" {
		paths = append(paths, "~/.azure/**")
	}
	return sliceutil.Dedup(paths)
}
