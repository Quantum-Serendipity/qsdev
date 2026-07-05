package claudecode

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/internal/sliceutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func (a *Adapter) EnforcementTier() aiframework.EnforcementTier {
	return aiframework.TierHook
}

// TranslatePermissions renders the framework-agnostic permission policy into a
// real Claude Code .claude/settings.json. It emits the full base deny rule set
// unioned with the policy's explicit deny rules (so a translated deny policy is
// actually enforced, not merely reported) alongside the policy's allow and ask
// rules.
func (a *Adapter) TranslatePermissions(ctx context.Context, policy *aiframework.PermissionPolicy) (*aiframework.PermissionArtifacts, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if policy == nil {
		policy = &aiframework.PermissionPolicy{}
	}

	deny := AllBaseDenyRules()
	deny = append(deny, rulePatterns(policy.DenyRules)...)

	perms := Permissions{
		Allow: sliceutil.Dedup(rulePatterns(policy.AllowRules)),
		Deny:  sliceutil.Dedup(deny),
	}
	if ask := sliceutil.Dedup(rulePatterns(policy.AskRules)); len(ask) > 0 {
		perms.Ask = ask
	}

	data, err := json.MarshalIndent(SettingsJSON{Permissions: perms}, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling translated settings.json: %w", err)
	}
	data = append(data, '\n')

	return &aiframework.PermissionArtifacts{
		GeneratedFiles: []types.GeneratedFile{{
			Path:     ".claude/settings.json",
			Content:  data,
			Mode:     fileutil.ModeReadWrite,
			Strategy: types.ThreeWayMerge,
		}},
		ActiveTier: aiframework.TierHook,
	}, nil
}

// rulePatterns extracts the non-empty Pattern strings from a slice of
// permission rules, preserving order.
func rulePatterns(rules []aiframework.PermissionRule) []string {
	out := make([]string, 0, len(rules))
	for _, r := range rules {
		if r.Pattern != "" {
			out = append(out, r.Pattern)
		}
	}
	return out
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
