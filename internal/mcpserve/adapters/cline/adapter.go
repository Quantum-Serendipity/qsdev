// Package cline is the Cline (Continue.dev family) framework adapter for the
// universal qsdev MCP server (Phase 32, Unit 32.5). It contributes Cline-specific
// tools to the server's adapter registry.
//
// RESEARCH-GATED: Full Cline adapter implementation pending gdev-universal-mcp-server-design spike.
//
// Honest-stub policy: everything that can be real IS real. Project-marker
// detection (Applies), client identity matching (MatchesClient), the
// qsdev_cline_info project summary, and the qsdev_cline_capabilities tool
// (validated Cline research data) are fully implemented. The only gated piece is
// deep framework config rendering: there is no P19 reference adapter for the
// Continue.dev family yet (only Claude Code has one — see
// pkg/aiframework/adapters/claudecode), so qsdev_cline_config returns an honest
// structured research_gated result rather than a fake success or an empty no-op.
//
// Identity note: Cline belongs to the Continue.dev family, so this adapter's
// FrameworkID is aiframework.ContinueDev ("continue"). Because the
// DefaultClientMatch token derived from that id ("continue") would not match a
// client reporting itself as "cline", MatchesClient is implemented explicitly to
// match both "cline" and "continue".
//
// The adapter is a stateless singleton. It self-registers into
// spi.DefaultRegistry() from init() and is blank-imported only from
// cmd/qsdev/main.go, never from the mcpserve server root, so it cannot create an
// import cycle. Every handler reads the resolved project root from its
// *spi.ToolCallContext at call time; the adapter captures no root.
package cline

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

// Tool names exposed by the Cline adapter, in the flat qsdev_ namespace with a
// cline_ infix.
const (
	toolInfo         = "qsdev_cline_info"
	toolConfig       = "qsdev_cline_config"
	toolCapabilities = "qsdev_cline_capabilities"
)

// Tool tier hints (lower is more core).
const (
	tierStandard = 1
	tierExtended = 2
)

// configConvention names where Cline (Continue.dev) stores its configuration.
const configConvention = ".continue/ (YAML config, Continue.dev family)"

// Adapter is the stateless Cline framework adapter.
type Adapter struct{}

// compile-time assertions that Adapter satisfies the base contract and the
// optional client-matching seam DetectFrameworks consumes.
var (
	_ spi.FrameworkAdapter = (*Adapter)(nil)
	_ spi.ClientMatcher    = (*Adapter)(nil)
)

// New constructs the stateless Cline adapter.
func New() *Adapter { return &Adapter{} }

// init self-registers the singleton into the default adapter registry. This is
// the sanctioned registry self-registration pattern; the general "avoid init()"
// guidance does not apply to registry wiring. A duplicate-id error can only mean
// the package was linked twice — a build error worth surfacing loudly.
func init() {
	if err := spi.DefaultRegistry().Register(New()); err != nil {
		panic(fmt.Sprintf("registering cline framework adapter: %v", err))
	}
}

// ID identifies this adapter as the Continue.dev family (Cline).
func (a *Adapter) ID() aiframework.FrameworkID { return aiframework.ContinueDev }

// Applies reports whether projectRoot carries the Cline project marker: a
// .continue/ directory (the Continue.dev configuration directory).
func (a *Adapter) Applies(_ context.Context, projectRoot string) bool {
	if projectRoot == "" {
		return false
	}
	info, err := os.Stat(filepath.Join(projectRoot, ".continue"))
	return err == nil && info.IsDir()
}

// MatchesClient reports whether the connected MCP client identifies as Cline or
// Continue. It matches any client whose reported name contains "cline" or
// "continue" (case-insensitive). It is implemented explicitly because the
// FrameworkID token ("continue") alone would not match a "cline" client.
func (a *Adapter) MatchesClient(client spi.ClientInfo) bool {
	name := strings.ToLower(client.Name)
	return strings.Contains(name, "cline") || strings.Contains(name, "continue")
}

// Resources returns no resources: the Cline stub contributes none.
func (a *Adapter) Resources() []spi.ResourceRegistration { return nil }

// Prompts returns no prompts: the Cline stub contributes none.
func (a *Adapter) Prompts() []spi.PromptRegistration { return nil }

// Tools returns the three Cline tool registrations: two fully-real
// (qsdev_cline_info, qsdev_cline_capabilities) and one honestly research-gated
// (qsdev_cline_config).
func (a *Adapter) Tools() []spi.ToolRegistration {
	return []spi.ToolRegistration{
		{
			Name:        toolInfo,
			Description: "Summarize the detected project (languages, ecosystems, container runtime, git state) framed for Cline, including whether Cline project markers are present and Cline's config convention (" + configConvention + ").",
			InputSchema: emptyObjectSchema(),
			Category:    middleware.CategoryStatus,
			Tier:        tierStandard,
			Handler:     a.handleInfo,
		},
		{
			Name:        toolConfig,
			Description: "Render the Cline configuration (.continue/ YAML) from the project's qsdev policy. RESEARCH-GATED: full Cline config rendering is not yet implemented (no Continue.dev P19 reference adapter exists); this returns an honest structured research_gated result describing what is pending.",
			InputSchema: emptyObjectSchema(),
			Category:    middleware.CategoryGeneral,
			Tier:        tierExtended,
			Handler:     a.handleConfig,
		},
		{
			Name:        toolCapabilities,
			Description: "Report Cline's validated capability profile: its (undocumented) tool ceiling, configuration format (YAML), Continue.dev family lineage, native-hook support, and the strongest enforcement tier qsdev can target on Cline.",
			InputSchema: emptyObjectSchema(),
			Category:    middleware.CategoryStatus,
			Tier:        tierStandard,
			Handler:     a.handleCapabilities,
		},
	}
}

// handleInfo summarizes the project via internal/detect, framed for Cline. It is
// fully real: it delegates detection rather than reinventing it.
func (a *Adapter) handleInfo(ctx context.Context, cc *spi.ToolCallContext, _ *spi.ToolRequest) (*spi.ToolResult, error) {
	if cc.ProjectRoot == "" {
		return notConfigured("no project root resolved for cline info", nil), nil
	}
	det := detect.Detect(cc.ProjectRoot)
	langs := languages(det)
	structured := map[string]any{
		"framework":         string(aiframework.ContinueDev),
		"project_root":      cc.ProjectRoot,
		"config_convention": configConvention,
		"markers_present":   a.Applies(ctx, cc.ProjectRoot),
		"languages":         langs,
		"ecosystems":        sortedTrueKeys(det.Ecosystems),
		"container_runtime": det.ContainerRuntime,
		"os_family":         det.OSFamily,
		"is_git_repo":       det.IsGitRepo,
	}
	text := fmt.Sprintf("cline project info for %s: languages %s (config convention: %s)",
		cc.ProjectRoot, joinOrNone(langs), configConvention)
	return &spi.ToolResult{Text: text, Structured: structured}, nil
}

// handleConfig returns the honest research-gated result for deep Cline config
// rendering. It is intentionally NOT a fake success and NOT an empty no-op: it
// reports IsError with a structured payload naming exactly what is pending.
func (a *Adapter) handleConfig(_ context.Context, cc *spi.ToolCallContext, _ *spi.ToolRequest) (*spi.ToolResult, error) {
	return researchGated(cc.ProjectRoot), nil
}

// handleCapabilities returns Cline's validated capability profile as structured
// content. It is fully real research data.
func (a *Adapter) handleCapabilities(_ context.Context, _ *spi.ToolCallContext, _ *spi.ToolRequest) (*spi.ToolResult, error) {
	caps := capabilities()
	text := fmt.Sprintf("cline capabilities: tool ceiling %s, %s config, %s enforcement (Continue.dev family)",
		toolCeilingText(caps), caps["config_format"], caps["enforcement_tier"])
	return &spi.ToolResult{Text: text, Structured: caps}, nil
}

// capabilities returns Cline's validated capability data. There is no documented
// tool ceiling for Cline, configuration is YAML, and the strongest enforcement
// qsdev can target is advisory (instructions). Cline belongs to the Continue.dev
// family.
func capabilities() map[string]any {
	return map[string]any{
		"framework":               string(aiframework.ContinueDev),
		"family":                  "Continue.dev",
		"tool_ceiling":            0,
		"tool_ceiling_documented": false,
		"config_format":           "YAML",
		"config_paths":            []string{".continue/"},
		"native_hooks":            false,
		"enforcement_tier":        aiframework.TierAdvisory.String(),
		"enforcement_note":        "Cline has no native pre-tool hook; policy is advisory (instructions only).",
	}
}

// toolCeilingText renders the tool-ceiling for the human-readable summary,
// reporting "undocumented" when no ceiling is documented.
func toolCeilingText(caps map[string]any) string {
	if documented, _ := caps["tool_ceiling_documented"].(bool); !documented {
		return "undocumented"
	}
	return fmt.Sprintf("%v", caps["tool_ceiling"])
}

// researchGated builds the honest structured research_gated result for the
// config tool: a JSON payload encoded in Text with IsError set, mirroring the
// graceful-degradation convention the generic surface and the Claude Code
// adapter use. Structured is left nil so the mcpserve bridge emits a real
// protocol error result and clients observe IsError=true.
func researchGated(projectRoot string) *spi.ToolResult {
	payload := map[string]any{
		"status":          "research_gated",
		"framework":       string(aiframework.ContinueDev),
		"reason":          "Full Cline configuration rendering is not yet implemented: it is gated on the gdev-universal-mcp-server-design research spike and a Continue.dev P19 reference adapter (pkg/aiframework/adapters/continue), neither of which exists yet.",
		"config_format":   "YAML",
		"config_paths":    []string{".continue/"},
		"project_root":    projectRoot,
		"pending_spike":   "gdev-universal-mcp-server-design",
		"pending_p19":     "pkg/aiframework/adapters/continue",
		"available_tools": []string{toolInfo, toolCapabilities},
	}
	text, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return &spi.ToolResult{Text: "research_gated: cline config rendering not yet implemented", IsError: true}
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
