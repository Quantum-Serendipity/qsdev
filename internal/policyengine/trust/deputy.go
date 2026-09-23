package trust

import (
	"encoding/json"
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/internal/policyengine/policy"
)

// ToolEquivalence maps an MCP tool that touches the local filesystem to the
// first-party tool it stands in for and the argument fields that carry its
// local paths. Each field may hold a single path string or an array of paths.
type ToolEquivalence struct {
	FirstPartyTool string
	PathArgFields  []string
}

// crossToolEquivalence lists the MCP tools whose path arguments name local
// files, so a deny rule written for first-party tools also applies to them
// (path laundering through an MCP server). It covers the full tool set of the
// reference filesystem server. mcp__github__get_file_contents is deliberately
// absent: its `path` names a file in a remote repository, not a local path, so
// matching it against local deny rules only produces false blocks.
var crossToolEquivalence = map[string]ToolEquivalence{
	"mcp__github__create_or_update_file":         {FirstPartyTool: "Edit", PathArgFields: []string{"path"}},
	"mcp__filesystem__read_file":                 {FirstPartyTool: "Read", PathArgFields: []string{"path"}},
	"mcp__filesystem__read_text_file":            {FirstPartyTool: "Read", PathArgFields: []string{"path"}},
	"mcp__filesystem__read_media_file":           {FirstPartyTool: "Read", PathArgFields: []string{"path"}},
	"mcp__filesystem__read_multiple_files":       {FirstPartyTool: "Read", PathArgFields: []string{"paths"}},
	"mcp__filesystem__write_file":                {FirstPartyTool: "Edit", PathArgFields: []string{"path"}},
	"mcp__filesystem__edit_file":                 {FirstPartyTool: "Edit", PathArgFields: []string{"path"}},
	"mcp__filesystem__create_directory":          {FirstPartyTool: "Edit", PathArgFields: []string{"path"}},
	"mcp__filesystem__move_file":                 {FirstPartyTool: "Edit", PathArgFields: []string{"source", "destination"}},
	"mcp__filesystem__list_directory":            {FirstPartyTool: "Read", PathArgFields: []string{"path"}},
	"mcp__filesystem__list_directory_with_sizes": {FirstPartyTool: "Read", PathArgFields: []string{"path"}},
	"mcp__filesystem__directory_tree":            {FirstPartyTool: "Read", PathArgFields: []string{"path"}},
	"mcp__filesystem__search_files":              {FirstPartyTool: "Read", PathArgFields: []string{"path"}},
	"mcp__filesystem__get_file_info":             {FirstPartyTool: "Read", PathArgFields: []string{"path"}},
}

// CheckAccess blocks a known path-bearing MCP tool call when any of its local
// path arguments matches a "path" deny rule. Patterns are matched with the
// policy engine's own path matcher, so a pattern such as `**/.ssh/*` means the
// same thing here as in a path_glob condition. The check fails closed: tool
// arguments that cannot be parsed, or a deny pattern that cannot be compiled,
// block the call rather than let an unchecked path through.
func CheckAccess(toolName string, toolArgs json.RawMessage, denyRules []policy.DenyRule) (blocked bool, reason string) {
	equiv, ok := crossToolEquivalence[toolName]
	if !ok {
		return false, ""
	}

	paths, err := extractPaths(toolArgs, equiv.PathArgFields)
	if err != nil {
		return true, fmt.Sprintf("confused deputy: %s: cannot read path arguments: %v", toolName, err)
	}
	if len(paths) == 0 {
		return false, ""
	}

	forms := make([][]string, len(paths))
	for i, p := range paths {
		forms[i] = policy.PathForms(p, "")
	}

	for _, rule := range denyRules {
		if rule.Type != "path" {
			continue
		}
		m, err := policy.CompilePathMatcher(rule.Pattern)
		if err != nil {
			return true, fmt.Sprintf("confused deputy: %s: invalid deny pattern %q: %v", toolName, rule.Pattern, err)
		}
		for i, p := range paths {
			if m.MatchForms(forms[i]) {
				return true, fmt.Sprintf("confused deputy: %s accessing denied path %s (pattern %s)", toolName, p, rule.Pattern)
			}
		}
	}

	return false, ""
}

// extractPaths returns the local paths held by fields in the tool arguments.
// Empty arguments carry no paths; malformed JSON or a path field of the wrong
// shape is an error.
func extractPaths(args json.RawMessage, fields []string) ([]string, error) {
	if len(args) == 0 {
		return nil, nil
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(args, &m); err != nil {
		return nil, fmt.Errorf("unmarshaling tool args: %w", err)
	}

	paths, err := policy.PathArgs(m, fields)
	if err != nil {
		return nil, fmt.Errorf("reading path fields: %w", err)
	}
	return paths, nil
}
