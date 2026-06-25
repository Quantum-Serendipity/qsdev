package projectctx

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

// Resource URIs and MIME types for the generic project context surface.
const (
	resURIDetection  = "qsdev://project/detection"
	resURIConfig     = "qsdev://project/config"
	resURIState      = "qsdev://project/state"
	resURIMCPServers = "qsdev://project/mcp-servers"
	resURIPackageCtx = "qsdev://project/{package}/context"

	mimeJSON = "application/json"
	mimeYAML = "text/yaml"
)

// Resources returns the five generic project context resources: four backed by
// real data and one (the per-package monorepo context) that degrades to a
// structured not_configured payload until the Unit 32.7 workspace graph lands
// (Task T9).
func (pc *ProjectContext) Resources() []spi.ResourceRegistration {
	return []spi.ResourceRegistration{
		{
			URI:         resURIDetection,
			Name:        "Project detection",
			Description: "Full ecosystem detection result for the project as JSON.",
			MIMEType:    mimeJSON,
			Handler:     pc.readDetection,
		},
		{
			URI:         resURIConfig,
			Name:        "Project config",
			Description: "Raw .qsdev.yaml project configuration.",
			MIMEType:    mimeYAML,
			Handler:     pc.readConfig,
		},
		{
			URI:         resURIState,
			Name:        "Project state",
			Description: "Raw primary qsdev state file (generated-file ledger).",
			MIMEType:    mimeYAML,
			Handler:     pc.readState,
		},
		{
			URI:         resURIMCPServers,
			Name:        "MCP server registry",
			Description: "Snapshot of the MCP servers known to qsdev as JSON.",
			MIMEType:    mimeJSON,
			Handler:     pc.readMCPServers,
		},
		{
			URI:         resURIPackageCtx,
			Name:        "Per-package context",
			Description: "Per-package context for monorepo workspace packages (name, ecosystem, directory, dependencies, sub-detection). Resolves by relative directory or {ecosystem}:{name}; returns not_configured for non-monorepo projects or unknown packages.",
			MIMEType:    mimeJSON,
			Handler:     pc.readPackageContext,
		},
	}
}

// readDetection serializes the startup detection result as JSON.
func (pc *ProjectContext) readDetection(_ context.Context, _ *spi.ToolCallContext, _ *spi.ResourceRequest) (*spi.ResourceResult, error) {
	data, err := json.MarshalIndent(pc.detection, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling detection result: %w", err)
	}
	return jsonResult(resURIDetection, data), nil
}

// readConfig returns the raw .qsdev.yaml bytes, or a YAML not_configured comment
// when the project has not been initialized.
func (pc *ProjectContext) readConfig(_ context.Context, _ *spi.ToolCallContext, _ *spi.ResourceRequest) (*spi.ResourceResult, error) {
	data, err := os.ReadFile(pc.configFile())
	if err != nil {
		if os.IsNotExist(err) {
			return textResult(resURIConfig, mimeYAML,
				"# not_configured: .qsdev.yaml not found at "+pc.configFile()+"\n"), nil
		}
		return nil, fmt.Errorf("reading %s: %w", pc.configFile(), err)
	}
	return textResult(resURIConfig, mimeYAML, string(data)), nil
}

// readState returns the raw primary state file, or a YAML not_configured comment
// when no state file exists yet.
func (pc *ProjectContext) readState(_ context.Context, _ *spi.ToolCallContext, _ *spi.ResourceRequest) (*spi.ResourceResult, error) {
	data, err := os.ReadFile(pc.statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return textResult(resURIState, mimeYAML,
				"# not_configured: no qsdev state file at "+pc.statePath+"\n"), nil
		}
		return nil, fmt.Errorf("reading %s: %w", pc.statePath, err)
	}
	return textResult(resURIState, mimeYAML, string(data)), nil
}

// readMCPServers serializes the MCP registry snapshot as JSON.
func (pc *ProjectContext) readMCPServers(_ context.Context, _ *spi.ToolCallContext, _ *spi.ResourceRequest) (*spi.ResourceResult, error) {
	defs := pc.mcpReg.All()
	snapshot := make([]map[string]any, 0, len(defs))
	for _, d := range defs {
		snapshot = append(snapshot, map[string]any{
			"name": d.Name, "display_name": d.DisplayName,
			"category": string(d.Category), "transport": string(d.Transport),
			"grade": d.ComplianceGrade.String(), "source": string(d.Source),
			"command": d.Command, "args": d.Args, "url": d.URL,
		})
	}
	data, err := json.MarshalIndent(map[string]any{"count": len(snapshot), "servers": snapshot}, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling MCP registry: %w", err)
	}
	return jsonResult(resURIMCPServers, data), nil
}

// readPackageContext renders the per-package monorepo context backed by the
// Unit 32.7 workspace graph. When the project is a monorepo and the requested
// package resolves (by relative directory or {ecosystem}:{name}), it returns the
// package's JSON context. Otherwise it degrades to a structured not_configured
// document rather than failing: distinguishing a non-monorepo project from an
// unknown package within a monorepo.
func (pc *ProjectContext) readPackageContext(_ context.Context, _ *spi.ToolCallContext, req *spi.ResourceRequest) (*spi.ResourceResult, error) {
	reason := "no monorepo workspace configuration detected at the project root"
	if pc.workspace != nil {
		if res, ok := pc.workspace.RenderPackageContext(req.URI); ok {
			return res, nil
		}
		reason = "no workspace package matches the requested identifier"
	}
	payload := map[string]any{
		"status": "not_configured",
		"reason": reason,
		"uri":    req.URI,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling not_configured payload: %w", err)
	}
	return jsonResult(req.URI, data), nil
}

// jsonResult wraps JSON bytes in a single-content ResourceResult.
func jsonResult(uri string, data []byte) *spi.ResourceResult {
	return textResult(uri, mimeJSON, string(data))
}

// textResult wraps text in a single-content ResourceResult.
func textResult(uri, mime, text string) *spi.ResourceResult {
	return &spi.ResourceResult{Contents: []spi.ResourceContent{{URI: uri, MIMEType: mime, Text: text}}}
}
