// Package codex is the Codex framework adapter for the universal qsdev MCP
// server (Phase 32, Unit 32.5). It contributes Codex-specific tools to the
// server's adapter registry.
//
// RESEARCH-GATED: Full Codex adapter implementation pending gdev-universal-mcp-server-design spike.
//
// Honest-stub policy: everything that can be real IS real. Project-marker
// detection (Applies), client identity matching (MatchesClient), the
// qsdev_codex_info project summary, and the qsdev_codex_capabilities tool
// (validated Codex research data) are fully implemented. The only gated piece is
// deep framework config rendering: there is no P19 reference adapter for Codex
// yet (only Claude Code has one — see pkg/aiframework/adapters/claudecode), so
// qsdev_codex_config returns an honest structured research_gated result rather
// than a fake success or an empty no-op.
//
// Enforcement note: Codex ships a native sandbox, so its enforcement tier is
// kernel — the strongest qsdev models. That sandbox is a client-side guarantee
// enforced by the Codex runtime, NOT a property of the MCP protocol; the
// capabilities tool records this distinction explicitly.
//
// The adapter is a stateless singleton. It self-registers into
// spi.DefaultRegistry() from init() and is blank-imported only from
// cmd/qsdev/main.go, never from the mcpserve server root, so it cannot create an
// import cycle. Every handler reads the resolved project root from its
// *spi.ToolCallContext at call time; the adapter captures no root.
package codex

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

// Tool names exposed by the Codex adapter, in the flat qsdev_ namespace with a
// codex_ infix.
const (
	toolInfo         = "qsdev_codex_info"
	toolConfig       = "qsdev_codex_config"
	toolCapabilities = "qsdev_codex_capabilities"
)

// Tool tier hints (lower is more core).
const (
	tierStandard = 1
	tierExtended = 2
)

// configConvention names where Codex stores its configuration.
const configConvention = ".codex/ (Starlark hook/policy format, native sandbox)"

// Adapter is the stateless Codex framework adapter.
type Adapter struct{}

// compile-time assertions that Adapter satisfies the base contract and the
// optional client-matching seam DetectFrameworks consumes.
var (
	_ spi.FrameworkAdapter = (*Adapter)(nil)
	_ spi.ClientMatcher    = (*Adapter)(nil)
)

// New constructs the stateless Codex adapter.
func New() *Adapter { return &Adapter{} }

// init self-registers the singleton into the default adapter registry. This is
// the sanctioned registry self-registration pattern; the general "avoid init()"
// guidance does not apply to registry wiring. A duplicate-id error can only mean
// the package was linked twice — a build error worth surfacing loudly.
func init() {
	if err := spi.DefaultRegistry().Register(New()); err != nil {
		panic(fmt.Sprintf("registering codex framework adapter: %v", err))
	}
}

// ID identifies this adapter as the Codex framework.
func (a *Adapter) ID() aiframework.FrameworkID { return aiframework.Codex }

// Applies reports whether projectRoot carries the Codex project marker: a
// .codex/ directory (where Codex stores its hooks and policy).
func (a *Adapter) Applies(_ context.Context, projectRoot string) bool {
	if projectRoot == "" {
		return false
	}
	info, err := os.Stat(filepath.Join(projectRoot, ".codex"))
	return err == nil && info.IsDir()
}

// MatchesClient reports whether the connected MCP client identifies as Codex. It
// matches any client whose reported name contains "codex" (case-insensitive).
func (a *Adapter) MatchesClient(client spi.ClientInfo) bool {
	return strings.Contains(strings.ToLower(client.Name), "codex")
}

// Resources returns no resources: the Codex stub contributes none.
func (a *Adapter) Resources() []spi.ResourceRegistration { return nil }

// Prompts returns no prompts: the Codex stub contributes none.
func (a *Adapter) Prompts() []spi.PromptRegistration { return nil }

// Tools returns the three Codex tool registrations: two fully-real
// (qsdev_codex_info, qsdev_codex_capabilities) and one honestly research-gated
// (qsdev_codex_config).
func (a *Adapter) Tools() []spi.ToolRegistration {
	return []spi.ToolRegistration{
		{
			Name:        toolInfo,
			Description: "Summarize the detected project (languages, ecosystems, container runtime, git state) framed for Codex, including whether Codex project markers are present and Codex's config convention (" + configConvention + ").",
			InputSchema: emptyObjectSchema(),
			Category:    middleware.CategoryStatus,
			Tier:        tierStandard,
			Handler:     a.handleInfo,
		},
		{
			Name:        toolConfig,
			Description: "Render the Codex configuration (.codex/ Starlark hooks/policy) from the project's qsdev policy. RESEARCH-GATED: full Codex config rendering is not yet implemented (no Codex P19 reference adapter exists); this returns an honest structured research_gated result describing what is pending.",
			InputSchema: emptyObjectSchema(),
			Category:    middleware.CategoryGeneral,
			Tier:        tierExtended,
			Handler:     a.handleConfig,
		},
		{
			Name:        toolCapabilities,
			Description: "Report Codex's validated capability profile: its (undocumented) tool ceiling, configuration format (Starlark), native sandbox, and the strongest enforcement tier qsdev can target on Codex (kernel — a client-side sandbox guarantee, not an MCP-protocol one).",
			InputSchema: emptyObjectSchema(),
			Category:    middleware.CategoryStatus,
			Tier:        tierStandard,
			Handler:     a.handleCapabilities,
		},
	}
}

// handleInfo summarizes the project via internal/detect, framed for Codex. It is
// fully real: it delegates detection rather than reinventing it.
func (a *Adapter) handleInfo(ctx context.Context, cc *spi.ToolCallContext, _ *spi.ToolRequest) (*spi.ToolResult, error) {
	if cc.ProjectRoot == "" {
		return notConfigured("no project root resolved for codex info", nil), nil
	}
	det := detect.Detect(cc.ProjectRoot)
	langs := languages(det)
	structured := map[string]any{
		"framework":         string(aiframework.Codex),
		"project_root":      cc.ProjectRoot,
		"config_convention": configConvention,
		"markers_present":   a.Applies(ctx, cc.ProjectRoot),
		"languages":         langs,
		"ecosystems":        sortedTrueKeys(det.Ecosystems),
		"container_runtime": det.ContainerRuntime,
		"os_family":         det.OSFamily,
		"is_git_repo":       det.IsGitRepo,
	}
	text := fmt.Sprintf("codex project info for %s: languages %s (config convention: %s)",
		cc.ProjectRoot, joinOrNone(langs), configConvention)
	return &spi.ToolResult{Text: text, Structured: structured}, nil
}

// handleConfig returns the honest research-gated result for deep Codex config
// rendering. It is intentionally NOT a fake success and NOT an empty no-op: it
// reports IsError with a structured payload naming exactly what is pending.
func (a *Adapter) handleConfig(_ context.Context, cc *spi.ToolCallContext, _ *spi.ToolRequest) (*spi.ToolResult, error) {
	return researchGated(cc.ProjectRoot), nil
}

// handleCapabilities returns Codex's validated capability profile as structured
// content. It is fully real research data.
func (a *Adapter) handleCapabilities(_ context.Context, _ *spi.ToolCallContext, _ *spi.ToolRequest) (*spi.ToolResult, error) {
	caps := capabilities()
	text := fmt.Sprintf("codex capabilities: %s config, %s enforcement (%s)",
		caps["config_format"], caps["enforcement_tier"], caps["enforcement_scope"])
	return &spi.ToolResult{Text: text, Structured: caps}, nil
}

// capabilities returns Codex's validated capability data. There is no documented
// tool ceiling for Codex; configuration uses a Starlark hook/policy format. Codex
// ships a native sandbox, so its enforcement tier is kernel — but that sandbox is
// a client-side guarantee enforced by the Codex runtime, NOT a property of the
// MCP protocol, which enforcement_scope records explicitly.
func capabilities() map[string]any {
	return map[string]any{
		"framework":               string(aiframework.Codex),
		"tool_ceiling":            0,
		"tool_ceiling_documented": false,
		"config_format":           "Starlark",
		"config_paths":            []string{".codex/"},
		"native_hooks":            true,
		"native_sandbox":          true,
		"enforcement_tier":        aiframework.TierKernel.String(),
		"enforcement_scope":       "client-side sandbox enforced by the Codex runtime, not an MCP-protocol guarantee",
		"enforcement_note":        "Codex's native sandbox physically constrains tool actions (kernel tier), but this is the client's sandbox, not an enforcement the MCP protocol itself provides.",
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
		"framework":       string(aiframework.Codex),
		"reason":          "Full Codex configuration rendering is not yet implemented: it is gated on the gdev-universal-mcp-server-design research spike and a Codex P19 reference adapter (pkg/aiframework/adapters/codex), neither of which exists yet.",
		"config_format":   "Starlark",
		"config_paths":    []string{".codex/"},
		"project_root":    projectRoot,
		"pending_spike":   "gdev-universal-mcp-server-design",
		"pending_p19":     "pkg/aiframework/adapters/codex",
		"available_tools": []string{toolInfo, toolCapabilities},
	}
	text, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return &spi.ToolResult{Text: "research_gated: codex config rendering not yet implemented", IsError: true}
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
