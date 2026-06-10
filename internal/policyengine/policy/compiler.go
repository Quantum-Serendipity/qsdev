package policy

import (
	"cmp"
	"slices"
)

type CompiledRule struct {
	Rule      PolicyRule
	Condition CompiledCondition
	Action    ActionHandler
}

type CompiledPolicySet struct {
	Rules     []CompiledRule
	ToolIndex map[string][]*CompiledRule
	DenyRules []DenyRule
}

func Compile(policy *SecurityPolicy) (*CompiledPolicySet, error) {
	compiled := make([]CompiledRule, 0, len(policy.Rules))

	for _, r := range policy.Rules {
		if !r.IsEnabled() {
			continue
		}

		cond, err := CompileCondition(r.Conditions)
		if err != nil {
			return nil, err
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

func extractDenyRules(rules []CompiledRule) []DenyRule {
	var result []DenyRule
	for _, r := range rules {
		collectDenyRules(r.Rule.Conditions, &result)
	}
	return result
}

func collectDenyRules(cond Condition, out *[]DenyRule) {
	switch cond.Type {
	case DeniedPathCheck:
		*out = append(*out, DenyRule{Pattern: cond.Pattern, Type: "denied_path"})
	case PathGlob:
		*out = append(*out, DenyRule{Pattern: cond.Pattern, Type: "path_glob"})
	case All, Any:
		for _, child := range cond.Conditions {
			collectDenyRules(child, out)
		}
	case Not:
		if cond.Condition != nil {
			collectDenyRules(*cond.Condition, out)
		}
	}
}
