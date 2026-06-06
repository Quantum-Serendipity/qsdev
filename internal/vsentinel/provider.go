package vsentinel

import (
	"context"
	"encoding/json"
	"path/filepath"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserver"
)

type MCPProvider struct {
	ProjectRootFunc      func() (string, error)
	ManifestCoverageFunc func() (any, error)
}

func (p *MCPProvider) Name() string        { return "version-sentinel" }
func (p *MCPProvider) Description() string { return "Run the version-sentinel MCP server" }

func (p *MCPProvider) Tools() []mcpserver.ToolDef {
	return []mcpserver.ToolDef{
		{
			Name:        "check_versions",
			Description: "Scan manifest files and report current dependency versions.",
			Params: []mcpserver.ParamDef{
				{Name: "project_root", Description: "Project root directory (defaults to current directory)"},
			},
			Handler: p.handleCheckVersions,
		},
		{
			Name:        "detect_drift",
			Description: "Compare lockfile entries against manifest declarations to detect version drift.",
			Params: []mcpserver.ParamDef{
				{Name: "project_root", Description: "Project root directory (defaults to current directory)"},
			},
			Handler: p.handleDetectDrift,
		},
		{
			Name:        "manifest_coverage",
			Description: "Report which ecosystems have version tracking coverage and which are uncovered.",
			Handler:     p.handleManifestCoverage,
		},
		{
			Name:        "version_history",
			Description: "Read the version-sentinel event log showing a timeline of version changes.",
			Params: []mcpserver.ParamDef{
				{Name: "log_path", Description: "Path to events.jsonl (defaults to .version-sentinel/events.jsonl)"},
			},
			Handler: p.handleVersionHistory,
		},
	}
}

func (p *MCPProvider) resolveRoot(args map[string]any) (string, error) {
	root, _ := args["project_root"].(string)
	if root == "" && p.ProjectRootFunc != nil {
		return p.ProjectRootFunc()
	}
	if root == "" {
		return ".", nil
	}
	return root, nil
}

func (p *MCPProvider) handleCheckVersions(_ context.Context, args map[string]any) (string, error) {
	root, err := p.resolveRoot(args)
	if err != nil {
		return "", &toolError{err.Error()}
	}

	report, err := CheckVersions(root)
	if err != nil {
		return "", &toolError{err.Error()}
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", &toolError{err.Error()}
	}

	return string(data), nil
}

func (p *MCPProvider) handleDetectDrift(_ context.Context, args map[string]any) (string, error) {
	root, err := p.resolveRoot(args)
	if err != nil {
		return "", &toolError{err.Error()}
	}

	report, err := DetectDrift(root)
	if err != nil {
		return "", &toolError{err.Error()}
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", &toolError{err.Error()}
	}

	return string(data), nil
}

func (p *MCPProvider) handleManifestCoverage(_ context.Context, _ map[string]any) (string, error) {
	if p.ManifestCoverageFunc == nil {
		return "[]", nil
	}

	report, err := p.ManifestCoverageFunc()
	if err != nil {
		return "", &toolError{err.Error()}
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", &toolError{err.Error()}
	}

	return string(data), nil
}

func (p *MCPProvider) handleVersionHistory(_ context.Context, args map[string]any) (string, error) {
	logPath, _ := args["log_path"].(string)
	if logPath == "" {
		root, err := p.resolveRoot(args)
		if err != nil {
			return "", &toolError{err.Error()}
		}
		logPath = filepath.Join(root, ".version-sentinel", "events.jsonl")
	}

	events, err := ReadVersionHistory(logPath)
	if err != nil {
		return "", &toolError{err.Error()}
	}

	data, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		return "", &toolError{err.Error()}
	}

	return string(data), nil
}

type toolError struct{ msg string }

func (e *toolError) Error() string { return e.msg }
