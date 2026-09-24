package rules

import (
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/canon"
)

// reMcpDangerousCmd matches a server command that fetches and runs remote code.
var reMcpDangerousCmd = regexp.MustCompile(`(?i)\b(curl|wget|fetch)\b[^|]*\|\s*(sh|bash|zsh|source)\b|\bnpx?\s+(-y\s+)?https?://`)

// mcpConfigNames are the basenames of MCP server configuration files, mapped to
// whether the file is an MCP config wherever it lives (true) or only inside an
// editor config directory (false: .cursor/mcp.json, .vscode/mcp.json).
// .claude.json is Claude Code's user config, which holds the user- and
// local-scope mcpServers.
var mcpConfigNames = map[string]bool{".mcp.json": true, ".claude.json": true, "mcp.json": false}

// mcpEditorDirs are the editor config directories whose mcp.json is an MCP
// server configuration.
var mcpEditorDirs = []string{".cursor", ".vscode"}

// isMCPConfigPath reports whether p names an MCP server configuration file: a
// project .mcp.json, Claude Code's .claude.json (~/.claude.json, or under
// CLAUDE_CONFIG_DIR), or an editor's .cursor/mcp.json or .vscode/mcp.json.
func isMCPConfigPath(p string) bool {
	normalized := filepath.ToSlash(p)
	base := path.Base(normalized)
	if strings.HasSuffix(base, ".mcp.json") || base == ".claude.json" {
		return true
	}
	for _, dir := range mcpEditorDirs {
		if strings.HasSuffix(normalized, dir+"/mcp.json") {
			return true
		}
	}
	return false
}

// writesMCPConfig reports whether a Write/Edit call targets an MCP config.
func writesMCPConfig(ctx *EvalContext) bool {
	return isMCPConfigPath(ctx.CanonicalPath) || isMCPConfigPath(ctx.FilePath)
}

// commandMentionsMcpConfig reports whether command text references an MCP
// config path anywhere (the cheap deny trigger).
func commandMentionsMcpConfig(command string) bool {
	s := filepath.ToSlash(command)
	if strings.Contains(s, ".mcp.json") || strings.Contains(s, ".claude.json") {
		return true
	}
	for _, dir := range mcpEditorDirs {
		if strings.Contains(s, dir+"/mcp.json") {
			return true
		}
	}
	return false
}

// mcpGlobMatches reports whether a glob word can expand to an MCP config file:
// its last segment matches a config basename (for mcp.json, only when the
// parent segment can match an editor config directory).
func mcpGlobMatches(p string) bool {
	if !hasGlobMeta(p) {
		return false
	}
	p = filepath.ToSlash(p)
	base, parent := path.Base(p), path.Base(path.Dir(p))
	for name, anywhere := range mcpConfigNames {
		if !shellSegMatch(base, name) {
			continue
		}
		if anywhere {
			return true
		}
		for _, dir := range mcpEditorDirs {
			if shellSegMatch(parent, dir) {
				return true
			}
		}
	}
	return false
}

// wordNamesMcpConfig reports whether word p, used by command sc, can name an
// MCP config file: as written, through brace or glob expansion, or resolved
// against the directory sc runs in (`cd .cursor && echo x > mcp.json`).
func wordNamesMcpConfig(sc scannedCommand, p string) bool {
	variants, ok := expandBraces(p)
	if !ok {
		return true
	}
	for _, v := range variants {
		if isMCPConfigPath(v) || mcpGlobMatches(v) {
			return true
		}
		if resolved, known := resolveWord(sc, v); known && isMCPConfigPath(resolved) {
			return true
		}
	}
	return false
}

func anyWordNamesMcpConfig(sc scannedCommand, words []string) bool {
	for _, w := range words {
		if isFlag(w) && w != "-" {
			w = flagValue(w)
		}
		if w != "" && wordNamesMcpConfig(sc, w) {
			return true
		}
	}
	return false
}

// lineMentionsMcpConfig reports whether a Bash command references an MCP
// config file in its raw or quote-stripped text or in any parsed word.
func lineMentionsMcpConfig(ctx *EvalContext) bool {
	if commandMentionsMcpConfig(ctx.Command) || commandMentionsMcpConfig(looseText(ctx.Command)) {
		return true
	}
	scs, err := ctx.scannedCommands()
	if err != nil {
		return false
	}
	for _, sc := range scs {
		if anyWordNamesMcpConfig(sc, sc.Args) || anyWordNamesMcpConfig(sc, sc.WriteRedirects) ||
			anyWordNamesMcpConfig(sc, sc.ReadRedirects) {
			return true
		}
	}
	return false
}

// bashMutatesMcpConfig reports whether a Bash command writes/redirects to or
// mutates an MCP config file (as opposed to merely reading it). A mention of an
// MCP config is the deny trigger; the command is cleared only when every
// command is a proven reader (or a directory change) whose write redirects do
// not target the config, and nothing uses an expansion. A wrapper or unknown
// command word cannot clear it (fail closed), nor can a parse error. So `sh -c
// 'echo x > .mcp.json'`, `{ echo x; } > .mcp.json`, `sed -i … .mcp.json` and
// `cd .cursor && echo x > mcp.json` deny, while `cat .mcp.json` and
// `jq . .mcp.json` (reads) clear.
func bashMutatesMcpConfig(ctx *EvalContext) bool {
	if !lineMentionsMcpConfig(ctx) {
		return false
	}
	scs, err := ctx.scannedCommands()
	if err != nil {
		return true
	}
	for _, sc := range scs {
		if sc.HasExpansion || anyWordNamesMcpConfig(sc, sc.WriteRedirects) || isMutating(sc) {
			return true
		}
	}
	return false
}

// mcpServerTampering compares the MCP servers an MCP config Write/Edit leaves
// behind with the ones on disk and describes the first server it adds or
// changes ("" when it only reformats, reorders, or removes servers). It fails
// closed: a change that cannot be reconstructed or a result that is not valid
// JSON is reported too.
func mcpServerTampering(ctx *EvalContext) string {
	before, after, err := ctx.FileChange()
	if err != nil {
		return "cannot verify MCP server configuration change: " + err.Error()
	}
	next, err := mcpServers(after)
	if err != nil {
		return "MCP server configuration write is not valid JSON: " + err.Error()
	}
	current, err := mcpServers(before)
	if err != nil {
		current = nil // an unreadable current config vouches for no server
	}
	ids := make([]string, 0, len(next))
	for id := range next {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		def, known := current[id]
		name := id[strings.LastIndexByte(id, 0)+1:]
		switch {
		case !known:
			return fmt.Sprintf("MCP config write adds server %q; add MCP servers with `qsdev enable`", name)
		case def != next[id]:
			return fmt.Sprintf("MCP config write changes server %q; change MCP servers with `qsdev enable`", name)
		}
	}
	return ""
}

// mcpServerKeys are the object keys that hold MCP server definitions:
// mcpServers (Claude Code, Cursor; nested per project in ~/.claude.json) and
// servers (VS Code).
var mcpServerKeys = map[string]bool{"mcpServers": true, "servers": true}

// mcpServers extracts every MCP server definition in a config document,
// keyed by its location (the NUL-joined key path ending in the server name)
// and rendered as canonical JSON, so two documents can be compared
// semantically. An empty document has no servers.
func mcpServers(doc string) (map[string]string, error) {
	servers := make(map[string]string)
	if strings.TrimSpace(doc) == "" {
		return servers, nil
	}
	var root any
	if err := json.Unmarshal([]byte(doc), &root); err != nil {
		return nil, fmt.Errorf("parsing MCP config: %w", err)
	}
	if err := collectMCPServers(root, "", servers); err != nil {
		return nil, err
	}
	return servers, nil
}

func collectMCPServers(node any, prefix string, out map[string]string) error {
	switch v := node.(type) {
	case map[string]any:
		for key, child := range v {
			defs, isServers := child.(map[string]any)
			if !mcpServerKeys[key] || !isServers {
				if err := collectMCPServers(child, prefix+key+"\x00", out); err != nil {
					return err
				}
				continue
			}
			for name, def := range defs {
				canonical, err := json.Marshal(def)
				if err != nil {
					return fmt.Errorf("encoding MCP server %q: %w", name, err)
				}
				out[prefix+key+"\x00"+name] = string(canonical)
			}
		}
	case []any:
		for i, child := range v {
			if err := collectMCPServers(child, fmt.Sprintf("%s%d\x00", prefix, i), out); err != nil {
				return err
			}
		}
	}
	return nil
}

// mcpPathFields are the tool_input fields through which MCP filesystem tools
// name the paths they touch (file_path is covered by CanonicalPath).
var (
	mcpPathFields     = []string{"path", "source", "destination"}
	mcpPathListFields = []string{"paths"}
)

// mcpToolProtectedPath reports the first path an MCP tool call names through
// its path/paths/source/destination arguments that is protected. A relative
// path is resolved against the session directory. A path that cannot be
// canonicalized is reported too (fail closed).
func mcpToolProtectedPath(ctx *EvalContext) (string, bool) {
	for _, p := range mcpToolPaths(ctx.ToolInput) {
		resolved := expandTilde(p)
		if !filepath.IsAbs(resolved) && ctx.CWD != "" {
			resolved = filepath.Join(ctx.CWD, resolved)
		}
		canonical, err := canon.Canonicalize(resolved)
		if err != nil {
			return p, true
		}
		if protected, _ := canon.IsProtected(canonical); protected {
			return canonical, true
		}
	}
	return "", false
}

// mcpToolPaths decodes the path arguments of an MCP tool call. Fields that are
// absent or not strings (or string lists) are ignored.
func mcpToolPaths(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil
	}
	var paths []string
	for _, key := range mcpPathFields {
		var s string
		if json.Unmarshal(fields[key], &s) == nil && s != "" {
			paths = append(paths, s)
		}
	}
	for _, key := range mcpPathListFields {
		var list []string
		if json.Unmarshal(fields[key], &list) == nil {
			for _, s := range list {
				if s != "" {
					paths = append(paths, s)
				}
			}
		}
	}
	return paths
}
