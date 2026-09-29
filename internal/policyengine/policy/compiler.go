package policy

import (
	"cmp"
	"fmt"
	"slices"
)

type CompiledRule struct {
	Rule      PolicyRule
	Condition CompiledCondition
	Action    ActionHandler
}

// CompiledPolicySet is the evaluation-ready form of a SecurityPolicy. Rules
// holds every loaded rule, including disabled ones, so posture and listing
// reflect the whole policy; only enabled rules are reachable through ToolIndex
// (and therefore Evaluate) or contribute DenyRules.
type CompiledPolicySet struct {
	Rules     []CompiledRule
	ToolIndex map[string][]*CompiledRule
	DenyRules []DenyRule
}

func Compile(policy *SecurityPolicy) (*CompiledPolicySet, error) {
	compiled := make([]CompiledRule, 0, len(policy.Rules))

	for _, r := range policy.Rules {
		cond, err := CompileCondition(r.Conditions)
		if err != nil {
			return nil, fmt.Errorf("compiling rule %q: %w", r.ID, err)
		}

		handler := ResolveAction(r.Action.Type)
		compiled = append(compiled, CompiledRule{
			Rule:      r,
			Condition: cond,
			Action:    handler,
		})
	}

	slices.SortStableFunc(compiled, func(a, b CompiledRule) int {
		if c := cmp.Compare(int(a.Rule.BypassTier), int(b.Rule.BypassTier)); c != 0 {
			return c
		}
		return cmp.Compare(int(a.Rule.Severity), int(b.Rule.Severity))
	})

	toolIndex := buildToolIndex(compiled)
	denyRules := extractDenyRules(compiled)

	return &CompiledPolicySet{
		Rules:     compiled,
		ToolIndex: toolIndex,
		DenyRules: denyRules,
	}, nil
}

func RulesForTool(set *CompiledPolicySet, toolName string) []*CompiledRule {
	specific := set.ToolIndex[toolName]
	wildcard := set.ToolIndex["*"]

	if len(specific) == 0 {
		return wildcard
	}
	if len(wildcard) == 0 {
		return specific
	}

	// Both slices are already sorted; merge them maintaining sort order.
	merged := make([]*CompiledRule, 0, len(specific)+len(wildcard))
	i, j := 0, 0
	for i < len(specific) && j < len(wildcard) {
		if compareRules(specific[i], wildcard[j]) <= 0 {
			merged = append(merged, specific[i])
			i++
		} else {
			merged = append(merged, wildcard[j])
			j++
		}
	}
	merged = append(merged, specific[i:]...)
	merged = append(merged, wildcard[j:]...)
	return merged
}

func compareRules(a, b *CompiledRule) int {
	if c := cmp.Compare(int(a.Rule.BypassTier), int(b.Rule.BypassTier)); c != 0 {
		return c
	}
	return cmp.Compare(int(a.Rule.Severity), int(b.Rule.Severity))
}

func buildToolIndex(rules []CompiledRule) map[string][]*CompiledRule {
	idx := make(map[string][]*CompiledRule)

	for i := range rules {
		r := &rules[i]
		if !r.Rule.IsEnabled() {
			continue
		}
		cond := r.Rule.Conditions

		toolName := indexKeyForCondition(cond)
		idx[toolName] = append(idx[toolName], r)
	}

	return idx
}

func indexKeyForCondition(cond Condition) string {
	if cond.Type == ToolMatch {
		return cond.ToolName
	}
	if cond.Type == All && len(cond.Conditions) > 0 && cond.Conditions[0].Type == ToolMatch {
		return cond.Conditions[0].ToolName
	}
	return "*"
}

// extractDenyRules projects the path patterns of every rule that actually
// blocks onto DenyRules for the MCP confused-deputy check. A rule contributes
// only when it is enabled, not in monitor mode, and its action blocks (block,
// or prompt with a fail-closed default): a warn, audit, allow-default prompt or
// monitor-only rule never blocks the first-party call, so projecting its
// pattern as a hard deny would block MCP access the policy allows. Each
// DenyRule carries its rule ID and bypass tier so the deputy check can honor
// session overrides exactly as Evaluate does.
func extractDenyRules(rules []CompiledRule) []DenyRule {
	var result []DenyRule
	for i := range rules {
		rule := &rules[i].Rule
		if !ruleBlocks(rule) {
			continue
		}
		collectDenyRules(rule.Conditions, rule, &result)
	}
	return result
}

// ruleBlocks reports whether a matching rule denies the tool call.
func ruleBlocks(rule *PolicyRule) bool {
	if !rule.IsEnabled() || rule.MonitorMode {
		return false
	}
	switch rule.Action.Type {
	case Block:
		return true
	case Prompt:
		return !promptDefaultAllows(rule.Action.DefaultOnTimeout)
	default:
		return false
	}
}

// collectDenyRules gathers path patterns from positive condition positions
// only. A pattern under `not` means "anything except this path", so denying
// that path would invert the rule's meaning; negated subtrees are skipped.
func collectDenyRules(cond Condition, rule *PolicyRule, out *[]DenyRule) {
	switch cond.Type {
	case DeniedPathCheck:
		*out = append(*out, DenyRule{Pattern: cond.Pattern, Type: "denied_path", RuleID: rule.ID, BypassTier: rule.BypassTier})
	case PathGlob:
		*out = append(*out, DenyRule{Pattern: cond.Pattern, Type: "path_glob", RuleID: rule.ID, BypassTier: rule.BypassTier})
	case All, Any:
		for _, child := range cond.Conditions {
			collectDenyRules(child, rule, out)
		}
	}
}
