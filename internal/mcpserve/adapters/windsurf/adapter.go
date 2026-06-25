// Package windsurf is the Windsurf (Cascade) framework adapter for the universal
// qsdev MCP server (Phase 32, Unit 32.5). It contributes Windsurf-specific tools
// to the server's adapter registry.
//
// RESEARCH-GATED: Full Windsurf adapter implementation pending gdev-universal-mcp-server-design spike.
//
// Honest-stub policy: everything that can be real IS real. Project-marker
// detection (Applies), client identity matching (MatchesClient), the
// qsdev_windsurf_info project summary, and the qsdev_windsurf_capabilities tool
// (validated Windsurf research data) are fully implemented. The only gated piece
// is deep framework config rendering: there is no P19 reference adapter for
// Windsurf yet (only Claude Code has one — see pkg/aiframework/adapters/claudecode),
// so qsdev_windsurf_config returns an honest structured research_gated result
// rather than a fake success or an empty no-op.
//
// The adapter is a stateless singleton. It self-registers into
// spi.DefaultRegistry() from init() and is blank-imported only from
// cmd/qsdev/main.go, never from the mcpserve server root, so it cannot create an
// import cycle. Every handler reads the resolved project root from its
// *spi.ToolCallContext at call time; the adapter captures no root.
package windsurf

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/detect"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/middleware"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Tool names exposed by the Windsurf adapter, in the flat qsdev_ namespace with
// a windsurf_ infix.
const (
	toolInfo         = "qsdev_windsurf_info"
	toolConfig       = "qsdev_windsurf_config"
	toolCapabilities = "qsdev_windsurf_capabilities"
)

// Tool tier hints (lower is more core).
const (
	tierStandard = 1
	tierExtended = 2
)

// configConvention names where Windsurf stores its configuration.
const configConvention = ".windsurfrules (MDC) + Cascade context model"

// Adapter is the stateless Windsurf framework adapter.
type Adapter struct{}

// compile-time assertions that Adapter satisfies the base contract and the
// optional client-matching seam DetectFrameworks consumes.
var (
	_ spi.FrameworkAdapter = (*Adapter)(nil)
	_ spi.ClientMatcher    = (*Adapter)(nil)
)

// New constructs the stateless Windsurf adapter.
func New() *Adapter { return &Adapter{} }

// init self-registers the singleton into the default adapter registry. This is
// the sanctioned registry self-registration pattern; the general "avoid init()"
// guidance does not apply to registry wiring. A duplicate-id error can only mean
// the package was linked twice — a build error worth surfacing loudly.
func init() {
	if err := spi.DefaultRegistry().Register(New()); err != nil {
		panic(fmt.Sprintf("registering windsurf framework adapter: %v", err))
	}
}

// ID identifies this adapter as the Windsurf framework.
func (a *Adapter) ID() aiframework.FrameworkID { return aiframework.Windsurf }

// Applies reports whether projectRoot carries the Windsurf project marker: a
// .windsurfrules file (Windsurf's project rules file).
func (a *Adapter) Applies(_ context.Context, projectRoot string) bool {
	if projectRoot == "" {
		return false
	}
	info, err := os.Stat(filepath.Join(projectRoot, ".windsurfrules"))
	return err == nil && !info.IsDir()
}

// MatchesClient reports whether the connected MCP client identifies as Windsurf.
// It matches any client whose reported name contains "windsurf" or "cascade"
// (Windsurf's agent is named Cascade), case-insensitively.
func (a *Adapter) MatchesClient(client spi.ClientInfo) bool {
	name := strings.ToLower(client.Name)
	return strings.Contains(name, "windsurf") || strings.Contains(name, "cascade")
}

// Resources returns no resources: the Windsurf stub contributes none.
func (a *Adapter) Resources() []spi.ResourceRegistration { return nil }

// Prompts returns no prompts: the Windsurf stub contributes none.
func (a *Adapter) Prompts() []spi.PromptRegistration { return nil }

// Tools returns the three Windsurf tool registrations: two fully-real
// (qsdev_windsurf_info, qsdev_windsurf_capabilities) and one honestly
// research-gated (qsdev_windsurf_config).
func (a *Adapter) Tools() []spi.ToolRegistration {
	return []spi.ToolRegistration{
		{
			Name:        toolInfo,
			Description: "Summarize the detected project (languages, ecosystems, container runtime, git state) framed for Windsurf, including whether Windsurf project markers are present and Windsurf's config convention (" + configConvention + ").",
			InputSchema: emptyObjectSchema(),
			Category:    middleware.CategoryStatus,
			Tier:        tierStandard,
			Handler:     a.handleInfo,
		},
		{
			Name:        toolConfig,
			Description: "Render the Windsurf configuration (.windsurfrules) from the project's qsdev policy. RESEARCH-GATED: full Windsurf config rendering is not yet implemented (no Windsurf P19 reference adapter exists); this returns an honest structured research_gated result describing what is pending.",
			InputSchema: emptyObjectSchema(),
			Category:    middleware.CategoryGeneral,
			Tier:        tierExtended,
			Handler:     a.handleConfig,
		},
		{
			Name:        toolCapabilities,
			Description: "Report Windsurf's validated capability profile: its client-side tool ceiling, configuration format (MDC), Cascade context model, native-hook support, and the strongest enforcement tier qsdev can target on Windsurf.",
			InputSchema: emptyObjectSchema(),
			Category:    middleware.CategoryStatus,
			Tier:        tierStandard,
			Handler:     a.handleCapabilities,
		},
	}
}

// handleInfo summarizes the project via internal/detect, framed for Windsurf. It
// is fully real: it delegates detection rather than reinventing it.
func (a *Adapter) handleInfo(ctx context.Context, cc *spi.ToolCallContext, _ *spi.ToolRequest) (*spi.ToolResult, error) {
	if cc.ProjectRoot == "" {
		return notConfigured("no project root resolved for windsurf info", nil), nil
	}
	det := detect.Detect(cc.ProjectRoot)
	langs := languages(det)
	structured := map[string]any{
		"framework":         string(aiframework.Windsurf),
		"project_root":      cc.ProjectRoot,
		"config_convention": configConvention,
		"markers_present":   a.Applies(ctx, cc.ProjectRoot),
		"languages":         langs,
		"ecosystems":        sortedTrueKeys(det.Ecosystems),
		"container_runtime": det.ContainerRuntime,
		"os_family":         det.OSFamily,
		"is_git_repo":       det.IsGitRepo,
	}
	text := fmt.Sprintf("windsurf project info for %s: languages %s (config convention: %s)",
		cc.ProjectRoot, joinOrNone(langs), configConvention)
	return &spi.ToolResult{Text: text, Structured: structured}, nil
}

// handleConfig returns the honest research-gated result for deep Windsurf config
// rendering. It is intentionally NOT a fake success and NOT an empty no-op: it
// reports IsError with a structured payload naming exactly what is pending.
func (a *Adapter) handleConfig(_ context.Context, cc *spi.ToolCallContext, _ *spi.ToolRequest) (*spi.ToolResult, error) {
	return researchGated(cc.ProjectRoot), nil
}

// handleCapabilities returns Windsurf's validated capability profile as
// structured content. It is fully real research data.
func (a *Adapter) handleCapabilities(_ context.Context, _ *spi.ToolCallContext, _ *spi.ToolRequest) (*spi.ToolResult, error) {
	caps := capabilities()
	text := fmt.Sprintf("windsurf capabilities: tool ceiling %v (%s), %s config, %s enforcement",
		caps["tool_ceiling"], caps["tool_ceiling_kind"], caps["config_format"], caps["enforcement_tier"])
	return &spi.ToolResult{Text: text, Structured: caps}, nil
}

// capabilities returns Windsurf's validated capability data. Tool ceiling is a
// client-side limit (~100 tools) enforced by the Windsurf client, not an MCP
// protocol guarantee. Windsurf's Cascade agent has no native pre-tool hook, so
// the strongest enforcement qsdev can target is advisory (instructions).
func capabilities() map[string]any {
	return map[string]any{
		"framework":               string(aiframework.Windsurf),
		"tool_ceiling":            100,
		"tool_ceiling_documented": true,
		"tool_ceiling_kind":       "client-side", // enforced by the Windsurf client, not the MCP protocol
		"config_format":           "MDC",
		"config_paths":            []string{".windsurfrules"},
		"context_model":           "Cascade",
		"native_hooks":            false,
		"enforcement_tier":        aiframework.TierAdvisory.String(),
		"enforcement_note":        "Windsurf's Cascade agent has no native pre-tool hook; policy is advisory (instructions only).",
	}
}

// researchGated builds the honest structured research_gated result for the
// config tool: a JSON payload encoded in Text with IsError set, mirroring the
// graceful-degradation convention the generic surface and the Claude Code
// adapter use. Structured is left nil so the mcpserve bridge emits a real
// protocol error result and clients observe IsError=true.
func researchGated(projectRoot string) *spi.ToolResult {
	payload := map[string]any{
		"status":          "research_gated",
		"framework":       string(aiframework.Windsurf),
		"reason":          "Full Windsurf configuration rendering is not yet implemented: it is gated on the gdev-universal-mcp-server-design research spike and a Windsurf P19 reference adapter (pkg/aiframework/adapters/windsurf), neither of which exists yet.",
		"config_format":   "MDC",
		"config_paths":    []string{".windsurfrules"},
		"project_root":    projectRoot,
		"pending_spike":   "gdev-universal-mcp-server-design",
		"pending_p19":     "pkg/aiframework/adapters/windsurf",
		"available_tools": []string{toolInfo, toolCapabilities},
	}
	text, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return &spi.ToolResult{Text: "research_gated: windsurf config rendering not yet implemented", IsError: true}
	}
	return &spi.ToolResult{Text: string(text), IsError: true}
}

// emptyObjectSchema is the JSON Schema for a tool that accepts no arguments.
func emptyObjectSchema() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{}}
}

// notConfigured builds the canonical graceful-degradation result for an edge
// case (e.g. an unresolved project root): a structured not_configured payload in
// Text with IsError set.
func notConfigured(reason string, extra map[string]any) *spi.ToolResult {
	payload := map[string]any{"status": "not_configured", "reason": reason}
	for k, v := range extra {
		payload[k] = v
	}
	text, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return &spi.ToolResult{Text: "not_configured: " + reason, IsError: true}
	}
	return &spi.ToolResult{Text: string(text), IsError: true}
}

// languages returns the project's detected programming languages as labels.
func languages(d types.DetectedProject) []string {
	var out []string
	if d.HasGoMod {
		out = append(out, "go")
	}
	if d.HasPackageJSON {
		out = append(out, "node")
	}
	if d.HasCargoToml {
		out = append(out, "rust")
	}
	if d.HasPyProject {
		out = append(out, "python")
	}
	return out
}

// sortedTrueKeys returns the keys of m whose value is true, sorted.
func sortedTrueKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k, v := range m {
		if v {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// joinOrNone joins s with ", " or returns "none" when empty.
func joinOrNone(s []string) string {
	if len(s) == 0 {
		return "none"
	}
	return strings.Join(s, ", ")
}
