package postmortem

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserver"
)

// maxSessionFiles bounds how many transcripts one list_failure_patterns call
// parses, so a huge sessions directory cannot stall the MCP server.
const maxSessionFiles = 5000

type MCPProvider struct {
	ChecklistFunc func() ([]string, error)
	// SessionsRoot confines the paths the tools may read. Empty means the
	// Claude Code projects directory ($CLAUDE_CONFIG_DIR/projects, or
	// ~/.claude/projects).
	SessionsRoot string
}

func (p *MCPProvider) Name() string        { return "agent-postmortem" }
func (p *MCPProvider) Description() string { return "Run the agent-postmortem MCP server" }

func (p *MCPProvider) Tools() []mcpserver.ToolDef {
	return []mcpserver.ToolDef{
		{
			Name:        "analyze_session",
			Description: "Parse a Claude session JSONL file and return structured analysis including tool calls and errors.",
			Params: []mcpserver.ParamDef{
				{Name: "session_path", Description: "Path to the session .jsonl file (must be under the Claude projects directory)", Required: true},
			},
			Handler: p.handleAnalyzeSession,
		},
		{
			Name:        "list_failure_patterns",
			Description: "Walk a directory of Claude session files, parse each, and aggregate failure patterns across sessions.",
			Params: []mcpserver.ParamDef{
				{Name: "sessions_dir", Description: "Directory to search for .jsonl session files (defaults to ~/.claude/projects/; must be inside it)"},
			},
			Handler: p.handleListFailurePatterns,
		},
		{
			Name:        "generate_verification_checklist",
			Description: "Generate a project-specific verification checklist based on detected ecosystems and languages.",
			Handler:     p.handleGenerateChecklist,
		},
	}
}

func (p *MCPProvider) handleAnalyzeSession(_ context.Context, args map[string]any) (string, error) {
	sessionPath, _ := args["session_path"].(string)
	if sessionPath == "" {
		return "", &toolError{"missing required parameter: session_path"}
	}
	sessionPath, err := p.confine(sessionPath)
	if err != nil {
		return "", &toolError{err.Error()}
	}

	analysis, err := ParseSessionJSONL(sessionPath)
	if err != nil {
		return "", &toolError{err.Error()}
	}

	data, err := json.MarshalIndent(analysis, "", "  ")
	if err != nil {
		return "", &toolError{err.Error()}
	}

	return string(data), nil
}

func (p *MCPProvider) handleListFailurePatterns(_ context.Context, args map[string]any) (string, error) {
	sessionsDir, _ := args["sessions_dir"].(string)
	if sessionsDir == "" {
		root, err := p.sessionsRoot()
		if err != nil {
			return "", &toolError{err.Error()}
		}
		sessionsDir = root
	}
	sessionsDir, err := p.confine(sessionsDir)
	if err != nil {
		return "", &toolError{err.Error()}
	}

	scan, err := FindSessionFiles(sessionsDir, maxSessionFiles)
	if err != nil {
		return "", &toolError{err.Error()}
	}

	var analyses []*SessionAnalysis
	skipped := scan.Unreadable
	for _, path := range scan.Paths {
		a, parseErr := ParseSessionJSONL(path)
		if parseErr != nil {
			skipped++
			continue
		}
		analyses = append(analyses, a)
	}

	report := AggregateFailures(analyses)
	report.SkippedFiles = skipped
	report.Truncated = scan.Truncated

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", &toolError{err.Error()}
	}

	return string(data), nil
}

func (p *MCPProvider) handleGenerateChecklist(_ context.Context, _ map[string]any) (string, error) {
	if p.ChecklistFunc == nil {
		return "", &toolError{"verification checklist is not available: no checklist source is configured"}
	}
	cmds, err := p.ChecklistFunc()
	if err != nil {
		// Never substitute another ecosystem's commands: a Go checklist for a
		// Node project would be presented as project-specific and "pass".
		return "", &toolError{fmt.Sprintf("generating verification checklist: %v", err)}
	}
	if cmds == nil {
		cmds = []string{}
	}

	data, err := json.MarshalIndent(cmds, "", "  ")
	if err != nil {
		return "", &toolError{err.Error()}
	}

	return string(data), nil
}

// sessionsRoot returns the directory the tools are confined to.
func (p *MCPProvider) sessionsRoot() (string, error) {
	if p.SessionsRoot != "" {
		return p.SessionsRoot, nil
	}
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, "projects"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locating the Claude projects directory: %w", err)
	}
	return filepath.Join(home, ".claude", "projects"), nil
}

// confine resolves path (following symlinks) and verifies it lies inside the
// sessions root, so an agent-supplied path cannot direct the tools to read or
// walk arbitrary parts of the filesystem.
func (p *MCPProvider) confine(path string) (string, error) {
	root, err := p.sessionsRoot()
	if err != nil {
		return "", err
	}
	realRoot, err := resolvePath(root)
	if err != nil {
		return "", fmt.Errorf("resolving sessions root %s: %w", root, err)
	}
	realPath, err := resolvePath(path)
	if err != nil {
		return "", fmt.Errorf("resolving %s: %w", path, err)
	}
	rel, err := filepath.Rel(realRoot, realPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("path %s is outside the Claude sessions directory %s", path, root)
	}
	return realPath, nil
}

// resolvePath returns the absolute, symlink-free form of path.
func resolvePath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(abs)
}

type toolError struct{ msg string }

func (e *toolError) Error() string { return e.msg }
