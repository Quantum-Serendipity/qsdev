package projectctx

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/internal/detect"
	"github.com/Quantum-Serendipity/qsdev/internal/doctor"
	"github.com/Quantum-Serendipity/qsdev/internal/mcphealth"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpregistry"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/middleware"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools/toolutil"
	"github.com/Quantum-Serendipity/qsdev/internal/sysinfo"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Tool names exposed by the generic project context surface. They use the flat
// namespace with the qsdev_ prefix and underscore_case established by the
// universal-server design.
const (
	toolProjectInfo = "qsdev_project_info"
	toolDoctor      = "qsdev_doctor"
	toolConfigShow  = "qsdev_config_show"
	toolMCPList     = "qsdev_mcp_list"
	toolToolList    = "qsdev_tool_list"
	toolDetect      = "qsdev_detect"
)

// optionalBoolSchema builds an object schema with a single optional boolean
// property of the given name and description.
func optionalBoolSchema(name, desc string) map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			name: map[string]any{"type": "boolean", "description": desc},
		},
	}
}

// Tools returns the six generic project context tool registrations. Each is a
// fully-implemented handler that delegates to an existing qsdev package; none is
// a stub.
func (pc *ProjectContext) Tools() []spi.ToolRegistration {
	return []spi.ToolRegistration{
		{
			Name:        toolProjectInfo,
			Description: "Summarize the detected project: languages, frameworks, ecosystems, container runtime, and AI-framework integration captured at server startup.",
			InputSchema: toolutil.EmptyObjectSchema(),
			Category:    middleware.CategoryStatus,
			Tier:        int(TierCritical),
			Handler:     pc.handleProjectInfo,
		},
		{
			Name:        toolDoctor,
			Description: "Run qsdev system-prerequisite diagnostics (git, go, node, nix, devenv, ...) and report which tools are installed, their versions, and whether required tools are missing.",
			InputSchema: optionalBoolSchema("verbose", "Include the full per-tool detail for every checked tool, not just failures."),
			Category:    middleware.CategoryDiagnostics,
			Tier:        int(TierStandard),
			Handler:     pc.handleDoctor,
		},
		{
			Name:        toolConfigShow,
			Description: "Show the project's .qsdev.yaml configuration as YAML, optionally merged with the developer's .qsdev.local.yaml overrides.",
			InputSchema: optionalBoolSchema("include_local", "Merge .qsdev.local.yaml developer overrides into the returned configuration."),
			Category:    middleware.CategoryStatus,
			Tier:        int(TierStandard),
			Handler:     pc.handleConfigShow,
		},
		{
			Name:        toolMCPList,
			Description: "List the MCP servers known to qsdev with their category, transport, and compliance grade. Optionally probe each server's live health.",
			InputSchema: optionalBoolSchema("health", "Probe each registered MCP server for live health (starts each server; slower)."),
			Category:    middleware.CategoryStatus,
			Tier:        int(TierExtended),
			Handler:     pc.handleMCPList,
		},
		{
			Name:        toolToolList,
			Description: "List the qsdev-managed tools (security, AI-agent, devex, infrastructure) with their category and enabled/disabled status for this project.",
			InputSchema: toolutil.EmptyObjectSchema(),
			Category:    middleware.CategoryStatus,
			Tier:        int(TierStandard),
			Handler:     pc.handleToolList,
		},
		{
			Name:        toolDetect,
			Description: "Re-run ecosystem detection against the project root and return the refreshed summary. Useful after files have been added or removed.",
			InputSchema: optionalBoolSchema("force", "Bypass any cached detection result and re-scan from scratch (detection always re-scans)."),
			Category:    middleware.CategoryStatus,
			Tier:        int(TierCritical),
			Handler:     pc.handleDetect,
		},
	}
}

// handleProjectInfo formats the startup detection snapshot.
func (pc *ProjectContext) handleProjectInfo(_ context.Context, _ *spi.ToolCallContext, _ *spi.ToolRequest) (*spi.ToolResult, error) {
	text, structured := detectionSummary(pc.projectRoot, pc.detection)
	return &spi.ToolResult{Text: text, Structured: structured}, nil
}

// handleDetect re-runs detection so the result reflects the current on-disk
// project. The force flag is accepted for forward compatibility; detection is
// uncached and always re-scans.
func (pc *ProjectContext) handleDetect(_ context.Context, _ *spi.ToolCallContext, req *spi.ToolRequest) (*spi.ToolResult, error) {
	fresh := detect.Detect(pc.projectRoot)
	text, structured := detectionSummary(pc.projectRoot, fresh)
	structured["forced"] = boolArg(req.Arguments, "force")
	return &spi.ToolResult{Text: text, Structured: structured}, nil
}

// handleDoctor delegates to internal/doctor's prerequisite checks.
func (pc *ProjectContext) handleDoctor(ctx context.Context, _ *spi.ToolCallContext, req *spi.ToolRequest) (*spi.ToolResult, error) {
	verbose := boolArg(req.Arguments, "verbose")
	results := doctor.RunAllChecks(ctx, sysinfo.DetectOS())

	var missingRequired, missingOptional []string
	checks := make([]map[string]any, 0, len(results))
	for _, r := range results {
		ok := r.Installed && r.VersionOK
		if !ok {
			if r.Required {
				missingRequired = append(missingRequired, r.Name)
			} else {
				missingOptional = append(missingOptional, r.Name)
			}
		}
		if verbose || !ok {
			checks = append(checks, map[string]any{
				"name": r.Name, "required": r.Required, "installed": r.Installed,
				"version": r.Version, "version_ok": r.VersionOK, "notes": r.Notes,
			})
		}
	}

	pass := len(missingRequired) == 0
	structured := map[string]any{
		"pass": pass, "total": len(results),
		"missing_required": missingRequired, "missing_optional": missingOptional,
		"checks": checks,
	}
	verdict := "PASS"
	if !pass {
		verdict = "FAIL"
	}
	text := fmt.Sprintf("doctor: %s — %d tools checked; %d required missing, %d optional missing",
		verdict, len(results), len(missingRequired), len(missingOptional))
	if len(missingRequired) > 0 {
		text += "\nrequired missing: " + strings.Join(missingRequired, ", ")
	}
	return &spi.ToolResult{Text: text, Structured: structured}, nil
}

// handleConfigShow reads .qsdev.yaml (and optionally merges .qsdev.local.yaml)
// and returns the result as YAML. A missing .qsdev.yaml degrades gracefully.
func (pc *ProjectContext) handleConfigShow(_ context.Context, _ *spi.ToolCallContext, req *spi.ToolRequest) (*spi.ToolResult, error) {
	cfgPath := pc.configFile()
	if _, err := os.Stat(cfgPath); err != nil {
		return toolutil.NotConfigured("project is not initialized: .qsdev.yaml not found",
			map[string]any{"expected_path": cfgPath}), nil
	}

	project, err := config.ParseQsdevConfig(cfgPath)
	if err != nil {
		return toolutil.NotConfigured("could not parse .qsdev.yaml",
			map[string]any{"path": cfgPath, "error": err.Error()}), nil
	}

	var local *config.LocalConfig
	includeLocal := boolArg(req.Arguments, "include_local")
	if includeLocal {
		// ParseLocalConfig returns (nil, nil) when the file is simply absent.
		local, err = config.ParseLocalConfig(pc.localConfigFile())
		if err != nil {
			return toolutil.NotConfigured("could not parse .qsdev.local.yaml",
				map[string]any{"path": pc.localConfigFile(), "error": err.Error()}), nil
		}
	}

	// Merge project over an empty base (no org defaults injected) then merge the
	// optional local overlay, reusing the canonical resolver. The project config
	// itself is passed as the security floor so local cannot weaken it.
	resolved, err := config.ResolveConfig(&types.QsdevConfig{}, nil, project, local, false)
	if err != nil {
		return nil, fmt.Errorf("resolving config for display: %w", err)
	}

	out, err := yaml.Marshal(resolved.Config)
	if err != nil {
		return nil, fmt.Errorf("marshaling merged config: %w", err)
	}
	return &spi.ToolResult{Text: string(out)}, nil
}

// handleMCPList delegates to the MCP server registry.
func (pc *ProjectContext) handleMCPList(ctx context.Context, _ *spi.ToolCallContext, req *spi.ToolRequest) (*spi.ToolResult, error) {
	defs := pc.mcpReg.All()
	withHealth := boolArg(req.Arguments, "health")

	servers := make([]map[string]any, 0, len(defs))
	for _, d := range defs {
		entry := map[string]any{
			"name": d.Name, "display_name": d.DisplayName,
			"category": string(d.Category), "transport": string(d.Transport),
			"grade": d.ComplianceGrade.String(), "source": string(d.Source),
		}
		if withHealth {
			entry["health"] = pc.probeHealth(ctx, d)
		}
		servers = append(servers, entry)
	}

	structured := map[string]any{"count": len(servers), "servers": servers, "health_probed": withHealth}
	text := fmt.Sprintf("%d MCP servers registered", len(servers))
	return &spi.ToolResult{Text: text, Structured: structured}, nil
}

// probeHealth runs a bounded live health check for a single server. It is only
// reached behind the health flag because it starts the server process.
func (pc *ProjectContext) probeHealth(ctx context.Context, d *mcpregistry.McpServerDefinition) map[string]any {
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	h := mcphealth.CheckServer(cctx, mcphealth.ServerConfig{
		Name: d.Name, Command: d.Command, Args: d.Args, URL: d.URL,
		Env: d.Env, RequiredEnv: d.RequiredEnv,
	})
	return map[string]any{"status": h.Status, "tool_count": h.ToolCount, "error": h.Error}
}

// handleToolList delegates to the tool lifecycle registry, layering on the
// enabled/disabled state recorded in the project's state ledger.
func (pc *ProjectContext) handleToolList(_ context.Context, _ *spi.ToolCallContext, _ *spi.ToolRequest) (*spi.ToolResult, error) {
	all := pc.toolReg.All()
	enabledMap := pc.state.EnabledTools

	tools := make([]map[string]any, 0, len(all))
	enabledCount := 0
	for _, t := range all {
		enabled := enabledMap[t.Name]
		if enabled {
			enabledCount++
		}
		tools = append(tools, map[string]any{
			"name": t.Name, "display_name": t.DisplayName,
			"category": string(t.Category), "default_policy": t.Default.String(),
			"enabled": enabled, "description": t.Description,
		})
	}

	structured := map[string]any{"count": len(tools), "enabled": enabledCount, "tools": tools}
	text := fmt.Sprintf("%d managed tools (%d enabled)", len(tools), enabledCount)
	return &spi.ToolResult{Text: text, Structured: structured}, nil
}

// detectionSummary renders a DetectedProject into a human-readable text summary
// and a JSON-serializable structured payload.
func detectionSummary(root string, d types.DetectedProject) (string, map[string]any) {
	langs := toolutil.DetectedLanguages(d)
	frameworks := detectedFrameworks(d)
	ai := detectedAIFrameworks(d)
	ecosystems := toolutil.SortedTrueKeys(d.Ecosystems)

	structured := map[string]any{
		"project_root":      root,
		"languages":         langs,
		"frameworks":        frameworks,
		"ai_frameworks":     ai,
		"ecosystems":        ecosystems,
		"container_runtime": d.ContainerRuntime,
		"os_family":         d.OSFamily,
		"is_git_repo":       d.IsGitRepo,
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Project: %s\n", root)
	fmt.Fprintf(&b, "Languages: %s\n", toolutil.JoinOrNone(langs))
	fmt.Fprintf(&b, "Frameworks/tools: %s\n", toolutil.JoinOrNone(frameworks))
	fmt.Fprintf(&b, "AI frameworks: %s\n", toolutil.JoinOrNone(ai))
	fmt.Fprintf(&b, "Ecosystems: %s\n", toolutil.JoinOrNone(ecosystems))
	if d.ContainerRuntime != "" {
		fmt.Fprintf(&b, "Container runtime: %s\n", d.ContainerRuntime)
	}
	fmt.Fprintf(&b, "Git repository: %t", d.IsGitRepo)
	return b.String(), structured
}

func detectedFrameworks(d types.DetectedProject) []string {
	var out []string
	if d.HasDockerfile {
		out = append(out, "docker")
	}
	if d.HasTerraform {
		out = append(out, "terraform")
	}
	if d.HasDevenvNix {
		out = append(out, "devenv")
	}
	return out
}

func detectedAIFrameworks(d types.DetectedProject) []string {
	var out []string
	if d.HasClaudeDir || d.HasClaudeMd || d.HasClaudeSettings {
		out = append(out, "claude-code")
	}
	if d.HasMcpJson {
		out = append(out, "mcp")
	}
	return out
}
