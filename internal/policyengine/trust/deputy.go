package trust

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/policyengine/policy"
)

// ToolEquivalence maps an MCP tool that touches the local filesystem to the
// first-party tool it stands in for and the local paths a call touches.
type ToolEquivalence struct {
	FirstPartyTool string
	// PathArgFields are the argument fields that carry the call's local paths.
	// Each field may hold a single path string or an array of paths.
	PathArgFields []string
	// WrittenPaths are files the tool writes on its own, whatever its arguments
	// say. They are project-relative and resolve against the hook's working
	// directory (the project root).
	WrittenPaths []string
	// WriteFlagField, when set, names the argument that switches the
	// WrittenPaths write on: an absent, null or false value means the call does
	// not write them, and any other value is treated as a write. When empty,
	// every call writes them.
	WriteFlagField string
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

// qsdevServerEquivalence lists tools of qsdev's own MCP server (`qsdev mcp
// serve`) that write local files. The server can be registered in .mcp.json
// under any name (qsdev, qsdev-universal, ...), so these match on the tool
// segment of mcp__<server>__<tool>; qsdev tool names are qsdev_-prefixed and
// contain no "__".
//
// qsdev_cc_config_render is dry-run only, but a qsdev binary that predates that
// change materializes .claude/settings.json and .mcp.json when called with
// write=true, and those files are the agent's own guardrail configuration. A
// write=true call is therefore checked as an Edit of both files.
var qsdevServerEquivalence = map[string]ToolEquivalence{
	"qsdev_cc_config_render": {
		FirstPartyTool: "Edit",
		WrittenPaths:   []string{".claude/settings.json", ".mcp.json"},
		WriteFlagField: "write",
	},
}

// equivalenceFor returns the equivalence entry for an MCP tool name, if any.
func equivalenceFor(toolName string) (ToolEquivalence, bool) {
	if equiv, ok := crossToolEquivalence[toolName]; ok {
		return equiv, true
	}
	if !strings.HasPrefix(toolName, "mcp__") {
		return ToolEquivalence{}, false
	}
	tool := toolName[strings.LastIndex(toolName, "__")+len("__"):]
	equiv, ok := qsdevServerEquivalence[tool]
	return equiv, ok
}

// CheckAccess blocks a known path-bearing MCP tool call when any local path it
// touches, from its path arguments or the files it writes on its own, matches
// a "path" deny rule. Patterns are matched with the policy engine's own path
// matcher, so a pattern such as `**/.ssh/*` means the same thing here as in a
// path_glob condition. The check fails closed: tool arguments that cannot be
// parsed, or a deny pattern that cannot be compiled, block the call rather
// than let an unchecked path through.
func CheckAccess(toolName string, toolArgs json.RawMessage, denyRules []policy.DenyRule) (blocked bool, reason string) {
	equiv, ok := equivalenceFor(toolName)
	if !ok {
		return false, ""
	}

	paths, err := callPaths(toolArgs, equiv)
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

// callPaths returns the local paths a call touches: those held by the
// equivalence's path argument fields plus, when the call writes them, its
// WrittenPaths. Empty arguments carry no paths and no write flag; malformed
// JSON or a path field of the wrong shape is an error.
func callPaths(args json.RawMessage, equiv ToolEquivalence) ([]string, error) {
	var m map[string]json.RawMessage
	if len(args) > 0 {
		if err := json.Unmarshal(args, &m); err != nil {
			return nil, fmt.Errorf("unmarshaling tool args: %w", err)
		}
	}

	paths, err := policy.PathArgs(m, equiv.PathArgFields)
	if err != nil {
		return nil, fmt.Errorf("reading path fields: %w", err)
	}
	if writesOwnPaths(m, equiv.WriteFlagField) {
		paths = append(paths, equiv.WrittenPaths...)
	}
	return paths, nil
}

// writesOwnPaths reports whether a call writes its equivalence's WrittenPaths.
// With no flag field every call does. Otherwise only an absent, null or false
// flag means no write; any other value, including a non-boolean one, counts as
// a write so a malformed flag cannot slip past the check.
func writesOwnPaths(args map[string]json.RawMessage, flagField string) bool {
	if flagField == "" {
		return true
	}
	raw, ok := args[flagField]
	if !ok {
		return false
	}
	switch strings.TrimSpace(string(raw)) {
	case "null", "false":
		return false
	default:
		return true
	}
}
