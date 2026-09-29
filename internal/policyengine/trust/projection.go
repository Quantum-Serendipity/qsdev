package trust

import (
	"maps"
	"slices"
	"strings"
)

// firstPartyRuleTools lists, for each first-party tool an MCP tool stands in
// for, the Claude Code permission rule names whose path-scoped deny entries
// cover it. Claude Code applies Edit rules to every file-editing tool, and a
// Write(...) rule restricts the same writes.
var firstPartyRuleTools = map[string][]string{
	"Read": {"Read"},
	"Edit": {"Edit", "Write"},
}

// GenerateDenyRuleProjections projects the first-party path deny rules of a
// .claude/settings.json deny list onto the path-bearing MCP tools of untrusted
// servers, returning the extra permission deny entries to add.
//
// A Claude Code permission rule cannot scope an MCP tool by its path argument:
// "mcp__filesystem__read_file" denies every call or none. A path deny rule such
// as Read(./.env) therefore cannot be carried over to the MCP tool that stands
// in for Read. For a server in Tier3Fallback the whole tool is denied instead.
// Tier 1 and Tier 2 servers keep their tools, and the PreToolUse
// confused-deputy check (CheckAccess) blocks a call whose paths a policy deny
// rule covers.
//
// A tool is projected only when firstPartyDeny holds at least one path-scoped
// rule for the first-party tool it stands in for (Read(...) for a read tool,
// Edit(...) or Write(...) for a write tool). tierOf returns a server's trust
// tier; a nil tierOf treats every server as Tier3Fallback. The result is
// sorted and holds no duplicates.
func GenerateDenyRuleProjections(firstPartyDeny []string, tierOf func(server string) TrustTier) []string {
	denied := pathDeniedFirstPartyTools(firstPartyDeny)
	if len(denied) == 0 {
		return nil
	}

	var projected []string
	for _, toolName := range slices.Sorted(maps.Keys(crossToolEquivalence)) {
		if !denied[crossToolEquivalence[toolName].FirstPartyTool] {
			continue
		}
		server, ok := serverForTool(toolName)
		if !ok {
			continue
		}
		if tierOf != nil && tierOf(server) != Tier3Fallback {
			continue
		}
		projected = append(projected, toolName)
	}
	return projected
}

// pathDeniedFirstPartyTools returns the first-party tools (keys of
// firstPartyRuleTools) that at least one path-scoped deny rule covers.
func pathDeniedFirstPartyTools(deny []string) map[string]bool {
	denied := make(map[string]bool, len(firstPartyRuleTools))
	for firstParty, ruleTools := range firstPartyRuleTools {
		for _, rule := range deny {
			if slices.ContainsFunc(ruleTools, func(tool string) bool { return isPathRule(rule, tool) }) {
				denied[firstParty] = true
				break
			}
		}
	}
	return denied
}

// isPathRule reports whether rule is a path-scoped permission rule for tool,
// i.e. "<tool>(<pattern>)" with a non-empty pattern.
func isPathRule(rule, tool string) bool {
	pattern, ok := strings.CutPrefix(rule, tool+"(")
	if !ok {
		return false
	}
	pattern, ok = strings.CutSuffix(pattern, ")")
	return ok && pattern != ""
}

// serverForTool returns the server segment of an MCP tool name, which follows
// the pattern mcp__{server}__{tool}.
func serverForTool(toolName string) (string, bool) {
	rest, ok := strings.CutPrefix(toolName, "mcp__")
	if !ok {
		return "", false
	}
	server, _, ok := strings.Cut(rest, "__")
	if !ok || server == "" {
		return "", false
	}
	return server, true
}
