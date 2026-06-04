package claudecode

import (
	"context"

	claudecodeaddon "github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func (a *Adapter) EnforcementTier() aiframework.EnforcementTier {
	return aiframework.TierHook
}

func (a *Adapter) TranslatePermissions(_ context.Context, policy *aiframework.PermissionPolicy) (*aiframework.PermissionArtifacts, error) {
	denyRules := claudecodeaddon.AllBaseDenyRules()

	for _, r := range policy.DenyRules {
		denyRules = append(denyRules, r.Pattern)
	}
	_ = denyRules

	return &aiframework.PermissionArtifacts{
		ActiveTier: aiframework.TierHook,
	}, nil
}

func (a *Adapter) TranslateIgnorePatterns(_ context.Context, _ []aiframework.IgnorePattern) ([]types.GeneratedFile, error) {
	return nil, nil
}

func (a *Adapter) InjectCredentials(_ context.Context, _ *aiframework.CredentialScope) (*aiframework.CredentialArtifacts, error) {
	return &aiframework.CredentialArtifacts{}, nil
}

func (a *Adapter) ReportGaps(_ context.Context, policy *aiframework.PermissionPolicy) []aiframework.EnforcementGap {
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
