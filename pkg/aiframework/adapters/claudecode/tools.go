package claudecode

import (
	"context"

	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// EnforcementTier reports that Claude Code enforces security policy via
// PreToolUse hooks rather than a kernel-level sandbox.
func (a *Adapter) EnforcementTier() aiframework.EnforcementTier { return a.addon.EnforcementTier() }

// TranslatePermissions renders the agnostic permission policy into a real
// Claude Code .claude/settings.json with the policy's explicit allow/deny/ask
// rules merged onto the preset-derived permissions block. A nil policy
// renders the configured defaults.
func (a *Adapter) TranslatePermissions(ctx context.Context, policy *aiframework.PermissionPolicy) (*aiframework.PermissionArtifacts, error) {
	return a.addon.TranslatePermissions(ctx, policy)
}

// TranslateIgnorePatterns expresses exclusion patterns the only way Claude Code
// supports them: as Read(<pattern>) deny rules inside settings.json.
func (a *Adapter) TranslateIgnorePatterns(ctx context.Context, patterns []aiframework.IgnorePattern) ([]types.GeneratedFile, error) {
	return a.addon.TranslateIgnorePatterns(ctx, patterns)
}

// InjectCredentials translates a credential scope into credential file globs
// to exclude from the agent's reach and passthrough references for required
// API-key environment variables, never embedding secret values.
func (a *Adapter) InjectCredentials(ctx context.Context, scope *aiframework.CredentialScope) (*aiframework.CredentialArtifacts, error) {
	return a.addon.InjectCredentials(ctx, scope)
}

// ReportGaps enumerates, for each requested deny rule, the gap between the
// kernel-level isolation the rule ideally wants and the hook-level enforcement
// Claude Code actually provides. A nil policy has no gaps.
func (a *Adapter) ReportGaps(ctx context.Context, policy *aiframework.PermissionPolicy) []aiframework.EnforcementGap {
	return a.addon.ReportGaps(ctx, policy)
}
