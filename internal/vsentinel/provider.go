package vsentinel

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools/toolutil"
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
				{Name: "project_root", Description: "Directory to scan, within the project root (defaults to the project root)"},
			},
			Handler: p.handleCheckVersions,
		},
		{
			Name:        "detect_drift",
			Description: "Compare lockfile entries against manifest declarations to detect version drift.",
			Params: []mcpserver.ParamDef{
				{Name: "project_root", Description: "Directory to scan, within the project root (defaults to the project root)"},
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
				{Name: "log_path", Description: "Path to events.jsonl within the project root (defaults to .version-sentinel/events.jsonl)"},
			},
			Handler: p.handleVersionHistory,
		},
	}
}

// projectRoot returns the root every tool is confined to: the one
// ProjectRootFunc reports, or the working directory when none is configured.
func (p *MCPProvider) projectRoot() (string, error) {
	if p.ProjectRootFunc != nil {
		root, err := p.ProjectRootFunc()
		if err != nil {
			return "", fmt.Errorf("resolving project root: %w", err)
		}
		return root, nil
	}
	root, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolving working directory: %w", err)
	}
	return root, nil
}

// confinedPath resolves a caller-supplied path argument against the project
// root and rejects one that escapes it (via "..", an absolute path elsewhere,
// or a symlink), so the tools cannot be pointed at manifests or files outside
// the project. An empty argument yields "".
func (p *MCPProvider) confinedPath(args map[string]any, key string) (string, error) {
	root, err := p.projectRoot()
	if err != nil {
		return "", err
	}
	arg, _ := args[key].(string)
	if arg == "" {
		return "", nil
	}
	resolved, ok := toolutil.ConfineToRoot(root, arg)
	if !ok {
		return "", fmt.Errorf("%s %q is outside the project root %q", key, arg, root)
	}
	return resolved, nil
}

// resolveRoot returns the directory to scan: the caller's project_root,
// confined to the project root, or the project root itself.
func (p *MCPProvider) resolveRoot(args map[string]any) (string, error) {
	dir, err := p.confinedPath(args, "project_root")
	if err != nil || dir != "" {
		return dir, err
	}
	return p.projectRoot()
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
	logPath, err := p.confinedPath(args, "log_path")
	if err != nil {
		return "", &toolError{err.Error()}
	}
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
