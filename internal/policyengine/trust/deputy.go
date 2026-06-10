package trust

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

type ToolEquivalence struct {
	FirstPartyTool string
	PathArgField   string
}

var crossToolEquivalence = map[string]ToolEquivalence{
	"mcp__github__create_or_update_file": {FirstPartyTool: "Edit", PathArgField: "path"},
	"mcp__github__get_file_contents":     {FirstPartyTool: "Read", PathArgField: "path"},
	"mcp__filesystem__read_file":         {FirstPartyTool: "Read", PathArgField: "path"},
	"mcp__filesystem__write_file":        {FirstPartyTool: "Edit", PathArgField: "path"},
	"mcp__filesystem__edit_file":         {FirstPartyTool: "Edit", PathArgField: "path"},
}

func CheckAccess(toolName string, toolArgs json.RawMessage, denyRules []DenyRule) (blocked bool, reason string) {
	equiv, ok := crossToolEquivalence[toolName]
	if !ok {
		return false, ""
	}

	path, err := extractPath(toolArgs, equiv.PathArgField)
	if err != nil || path == "" {
		return false, ""
	}

	canonical := canonicalizePath(path)

	for _, rule := range denyRules {
		if rule.Type != "path" {
			continue
		}
		if matchDenyPattern(canonical, rule.Pattern) {
			return true, fmt.Sprintf("confused deputy: %s accessing denied path %s", toolName, canonical)
		}
	}

	return false, ""
}

func extractPath(args json.RawMessage, field string) (string, error) {
	if len(args) == 0 {
		return "", nil
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(args, &m); err != nil {
		return "", fmt.Errorf("unmarshaling tool args: %w", err)
	}

	raw, ok := m[field]
	if !ok {
		return "", nil
	}

	var path string
	if err := json.Unmarshal(raw, &path); err != nil {
		return "", fmt.Errorf("unmarshaling path field %q: %w", field, err)
	}

	return path, nil
}

func canonicalizePath(path string) string {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		resolved = filepath.Clean(path)
	}

	abs, err := filepath.Abs(resolved)
	if err != nil {
		return resolved
	}

	return abs
}

func matchDenyPattern(path, pattern string) bool {
	if prefix, ok := strings.CutSuffix(pattern, "/*"); ok {
		prefix = canonicalizePath(prefix)
		return strings.HasPrefix(path, prefix+"/") || path == prefix
	}

	canonPattern := canonicalizePath(pattern)
	matched, err := filepath.Match(canonPattern, path)
	if err != nil {
		return path == canonPattern
	}
	return matched
}
