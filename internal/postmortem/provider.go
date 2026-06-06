package postmortem

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserver"
)

type MCPProvider struct {
	ChecklistFunc func() ([]string, error)
}

func (p *MCPProvider) Name() string        { return "agent-postmortem" }
func (p *MCPProvider) Description() string { return "Run the agent-postmortem MCP server" }

func (p *MCPProvider) Tools() []mcpserver.ToolDef {
	return []mcpserver.ToolDef{
		{
			Name:        "analyze_session",
			Description: "Parse a Claude session JSONL file and return structured analysis including tool calls and errors.",
			Params: []mcpserver.ParamDef{
				{Name: "session_path", Description: "Path to the session .jsonl file", Required: true},
			},
			Handler: p.handleAnalyzeSession,
		},
		{
			Name:        "list_failure_patterns",
			Description: "Walk a directory of Claude session files, parse each, and aggregate failure patterns across sessions.",
			Params: []mcpserver.ParamDef{
				{Name: "sessions_dir", Description: "Directory to search for .jsonl session files (defaults to ~/.claude/projects/)"},
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
		home, err := os.UserHomeDir()
		if err != nil {
			return "", &toolError{err.Error()}
		}
		sessionsDir = filepath.Join(home, ".claude", "projects")
	}

	paths, err := FindSessionFiles(sessionsDir)
	if err != nil {
		return "", &toolError{err.Error()}
	}

	var analyses []*SessionAnalysis
	for _, path := range paths {
		a, parseErr := ParseSessionJSONL(path)
		if parseErr != nil {
			continue
		}
		analyses = append(analyses, a)
	}

	report := AggregateFailures(analyses)

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", &toolError{err.Error()}
	}

	return string(data), nil
}

func (p *MCPProvider) handleGenerateChecklist(_ context.Context, _ map[string]any) (string, error) {
	var cmds []string
	if p.ChecklistFunc != nil {
		var err error
		cmds, err = p.ChecklistFunc()
		if err != nil {
			cmds = nil
		}
	}

	if len(cmds) == 0 {
		cmds = []string{"go build ./...", "go test ./...", "go vet ./..."}
	}

	data, err := json.MarshalIndent(cmds, "", "  ")
	if err != nil {
		return "", &toolError{err.Error()}
	}

	return string(data), nil
}

type toolError struct{ msg string }

func (e *toolError) Error() string { return e.msg }
