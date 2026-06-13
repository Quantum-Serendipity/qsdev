package trust

import (
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/policyengine/policy"
)

func GenerateDenyRuleProjections(denyRules []policy.DenyRule, tierAssignments map[string]TrustTier) []policy.DenyRule {
	var projected []policy.DenyRule

	for _, rule := range denyRules {
		if rule.Type != "path" {
			projected = append(projected, rule)
			continue
		}

		for toolName, equiv := range crossToolEquivalence {
			serverName := serverForTool(toolName)
			tier, ok := tierAssignments[serverName]
			if !ok {
				tier = Tier3Fallback
			}

			toolRule := policy.DenyRule{
				Pattern: rule.Pattern,
				Type:    "tool",
			}

			if tier == Tier3Fallback {
				projected = append(projected, rule)
				projected = append(projected, policy.DenyRule{
					Pattern: equiv.FirstPartyTool + "(" + rule.Pattern + ")",
					Type:    "tool",
				})
			} else {
				projected = append(projected, toolRule)
			}
		}
	}

	return projected
}

func serverForTool(toolName string) string {
	// MCP tool names follow the pattern mcp__{server}__{tool}
	parts := strings.SplitN(toolName, "__", 3)
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}
