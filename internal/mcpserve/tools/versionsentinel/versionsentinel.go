// Package versionsentinel implements the version-sentinel MCP tool module for
// the universal qsdev server: check_versions (the dependency versions the
// project's manifests declare), detect_drift (lock-file entries that disagree
// with their manifest), manifest_coverage (which ecosystems' manifests
// Version-Sentinel tracks) and version_history (the recorded version-change
// events).
//
// The tools run behind the universal server's middleware chain (guardrail,
// rate limiting, content safety, audit) like every other mounted tool, and read
// only inside the server's resolved project root: a caller-supplied path that
// escapes it (via "..", an absolute path elsewhere, or a symlink) is refused.
package versionsentinel

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/Quantum-Serendipity/qsdev/internal/answers"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/middleware"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools/toolutil"
	"github.com/Quantum-Serendipity/qsdev/internal/vsentinel"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// ModuleName names this tool module. It is the value `qsdev mcp serve
// --module` takes and the key of the catalog's version-sentinel MCP server.
const ModuleName = "version-sentinel"

// tierStandard mirrors the other tool modules' standard tier.
const tierStandard = 1

// module holds the project root every tool is confined to.
type module struct {
	projectRoot string
}

// Tools returns the version-sentinel tool registrations bound to projectRoot.
func Tools(projectRoot string) []spi.ToolRegistration {
	m := &module{projectRoot: projectRoot}
	return []spi.ToolRegistration{
		{
			Name:        "check_versions",
			Description: "Scan manifest files and report current dependency versions.",
			InputSchema: rootArgSchema(),
			Category:    middleware.CategoryStatus,
			Tier:        tierStandard,
			Annotations: spi.ReadOnlyAnnotations(false),
			Handler:     m.checkVersions,
		},
		{
			Name:        "detect_drift",
			Description: "Compare lockfile entries against manifest declarations to detect version drift.",
			InputSchema: rootArgSchema(),
			Category:    middleware.CategoryStatus,
			Tier:        tierStandard,
			Annotations: spi.ReadOnlyAnnotations(false),
			Handler:     m.detectDrift,
		},
		{
			Name:        "manifest_coverage",
			Description: "Report which ecosystems have version tracking coverage and which are uncovered.",
			InputSchema: toolutil.EmptyObjectSchema(),
			Category:    middleware.CategoryStatus,
			Tier:        tierStandard,
			Annotations: spi.ReadOnlyAnnotations(false),
			Handler:     m.manifestCoverage,
		},
		{
			Name:        "version_history",
			Description: "Read the version-sentinel event log showing a timeline of version changes.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"log_path": map[string]any{
						"type":        "string",
						"description": "Path to events.jsonl within the project root (defaults to .version-sentinel/events.jsonl)",
					},
				},
			},
			Category:    middleware.CategoryStatus,
			Tier:        tierStandard,
			Annotations: spi.ReadOnlyAnnotations(false),
			Handler:     m.versionHistory,
		},
	}
}

// rootArgSchema is the input schema of the tools that scan a directory.
func rootArgSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"project_root": map[string]any{
				"type":        "string",
				"description": "Directory to scan, within the project root (defaults to the project root)",
			},
		},
	}
}

// confinedPath resolves the caller-supplied path argument key against the
// project root. It returns "" when the argument is absent, and a denial result
// when the path escapes the root.
func (m *module) confinedPath(args map[string]any, key string) (string, *spi.ToolResult) {
	arg := toolutil.StringArgOr(args, key, "")
	if arg == "" {
		return "", nil
	}
	resolved, ok := toolutil.ConfineToRoot(m.projectRoot, arg)
	if !ok {
		return "", toolutil.Denied(fmt.Sprintf("%s %q is outside the project root %q", key, arg, m.projectRoot), map[string]any{key: arg})
	}
	return resolved, nil
}

// scanRoot returns the directory to scan: the caller's project_root, confined
// to the project root, or the project root itself.
func (m *module) scanRoot(args map[string]any) (string, *spi.ToolResult) {
	dir, denied := m.confinedPath(args, "project_root")
	if denied != nil || dir != "" {
		return dir, denied
	}
	return m.projectRoot, nil
}

func (m *module) checkVersions(_ context.Context, _ *spi.ToolCallContext, req *spi.ToolRequest) (*spi.ToolResult, error) {
	root, denied := m.scanRoot(req.Arguments)
	if denied != nil {
		return denied, nil
	}
	report, err := vsentinel.CheckVersions(root)
	if err != nil {
		return toolutil.ErrorResult(err.Error(), nil), nil
	}
	return toolutil.Result(toolutil.MarshalText(report, "version report"), report), nil
}

func (m *module) detectDrift(_ context.Context, _ *spi.ToolCallContext, req *spi.ToolRequest) (*spi.ToolResult, error) {
	root, denied := m.scanRoot(req.Arguments)
	if denied != nil {
		return denied, nil
	}
	report, err := vsentinel.DetectDrift(root)
	if err != nil {
		return toolutil.ErrorResult(err.Error(), nil), nil
	}
	return toolutil.Result(toolutil.MarshalText(report, "drift report"), report), nil
}

// manifestCoverage reports the coverage of the languages recorded in the
// project's primary qsdev answers file.
func (m *module) manifestCoverage(_ context.Context, _ *spi.ToolCallContext, _ *spi.ToolRequest) (*spi.ToolResult, error) {
	a, err := answers.RequirePrimary(m.projectRoot)
	if err != nil {
		return toolutil.NotConfigured(fmt.Sprintf("loading qsdev answers: %v", err), nil), nil
	}
	report := ecosystem.LanguageManifestCoverage(a.Languages, ecosystem.DefaultRegistry())
	return toolutil.Result(toolutil.MarshalText(report, "manifest coverage"), report), nil
}

func (m *module) versionHistory(_ context.Context, _ *spi.ToolCallContext, req *spi.ToolRequest) (*spi.ToolResult, error) {
	logPath, denied := m.confinedPath(req.Arguments, "log_path")
	if denied != nil {
		return denied, nil
	}
	if logPath == "" {
		logPath = filepath.Join(m.projectRoot, ".version-sentinel", "events.jsonl")
	}
	events, err := vsentinel.ReadVersionHistory(logPath)
	if err != nil {
		return toolutil.ErrorResult(err.Error(), nil), nil
	}
	if events == nil {
		events = []vsentinel.VersionEvent{}
	}
	return toolutil.Result(toolutil.MarshalText(events, "[]"), map[string]any{"events": events}), nil
}
