// Package frameworkstub holds the single, manifest-driven framework stub adapter
// shared by the Cursor, Windsurf, Cline, and Codex adapters (Phase 32, Unit
// 32.5). Those four frameworks are honest stubs with near-identical behavior:
// project-marker detection, client-identity matching, a real project summary, a
// real validated-capabilities report, and one honestly research-gated config
// renderer. Rather than cloning ~280 lines per framework, each framework supplies
// a Descriptor (the variation points: id, label, detection marker, client hints,
// config convention, tool descriptions, capability profile, and gated-config
// metadata) and the shared Adapter here supplies every behavior.
//
// Honest-stub policy: everything that can be real IS real. The only gated piece
// is deep framework config rendering, which has no P19 reference adapter yet, so
// the config tool returns an honest structured research_gated result rather than
// a fake success or an empty no-op.
//
// The shared Adapter is stateless beyond its immutable Descriptor. Each framework
// package's New() singleton is registered into the adapter registry explicitly
// from cmd/qsdev/main.go, never from the mcpserve server root, so no import cycle
// is possible. Every handler reads the
// resolved project root from its *spi.ToolCallContext at call time; the adapter
// captures no root.
package frameworkstub

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/detect"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/middleware"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools/toolutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
)

// Tool tier hints (lower is more core).
const (
	tierStandard = 1
	tierExtended = 2
)

// pendingSpike is the research spike that gates deep config rendering for every
// stubbed framework; it is shared because it is the same spike for all of them.
const pendingSpike = "gdev-universal-mcp-server-design"

// Descriptor is the per-framework manifest that parameterizes the shared stub
// adapter. Every field is a variation point distinguishing one near-identical
// framework stub from another; the shared Adapter supplies all behavior.
type Descriptor struct {
	// ID is the framework identity the adapter registers under.
	ID aiframework.FrameworkID
	// Label is the lowercase framework name used in the qsdev_<label>_ tool
	// infix and the human-readable result text (e.g. "cursor", "cline").
	Label string

	// Marker is the project-root-relative path whose presence marks the
	// framework, and MarkerIsDir selects whether it must be a directory (true)
	// or a non-directory file (false).
	Marker      string
	MarkerIsDir bool

	// ClientHints are the lowercase substrings matched (case-insensitively)
	// against an MCP client's reported name in MatchesClient.
	ClientHints []string

	// ConfigConvention names where the framework stores its configuration; it
	// frames the info result's structured payload and human text.
	ConfigConvention string

	// InfoDescription, ConfigDescription, and CapabilitiesDescription are the
	// three tools/list descriptions, stored verbatim because each is genuine
	// per-framework prose.
	InfoDescription         string
	ConfigDescription       string
	CapabilitiesDescription string

	// ConfigFormat, ConfigPaths, GatedReason, and PendingP19 populate the
	// honest research_gated config result.
	ConfigFormat string
	ConfigPaths  []string
	GatedReason  string
	PendingP19   string

	// Capabilities is the framework's validated capability profile, returned
	// verbatim as the capabilities tool's structured result. CapabilitiesText
	// is its static human-readable summary.
	Capabilities     map[string]any
	CapabilitiesText string
}

// Adapter is the shared, stateless framework stub adapter. It holds only its
// immutable Descriptor; a single instance serves every project root.
type Adapter struct {
	desc Descriptor
}

// compile-time assertions that Adapter satisfies the base contract and the
// optional client-matching seam DetectFrameworks consumes.
var (
	_ spi.FrameworkAdapter = (*Adapter)(nil)
	_ spi.ClientMatcher    = (*Adapter)(nil)
)

// New constructs the shared stub adapter for the given framework descriptor.
func New(d Descriptor) *Adapter { return &Adapter{desc: d} }

// ID identifies this adapter as its descriptor's framework.
func (a *Adapter) ID() aiframework.FrameworkID { return a.desc.ID }

// toolName builds a tool name in the flat qsdev_ namespace with the framework's
// label infix (e.g. "qsdev_cursor_info").
func (a *Adapter) toolName(suffix string) string {
	return "qsdev_" + a.desc.Label + "_" + suffix
}

// Applies reports whether projectRoot carries the framework's project marker.
// The marker must exist and match the descriptor's directory-vs-file expectation.
func (a *Adapter) Applies(_ context.Context, projectRoot string) bool {
	if projectRoot == "" {
		return false
	}
	info, err := os.Stat(filepath.Join(projectRoot, filepath.FromSlash(a.desc.Marker)))
	if err != nil {
		return false
	}
	return info.IsDir() == a.desc.MarkerIsDir
}

// MatchesClient reports whether the connected MCP client identifies as this
// framework: its reported name (lowercased) contains any of the descriptor's
// client hints. An empty client name never matches.
func (a *Adapter) MatchesClient(client spi.ClientInfo) bool {
	name := strings.ToLower(client.Name)
	if name == "" {
		return false
	}
	for _, hint := range a.desc.ClientHints {
		if strings.Contains(name, hint) {
			return true
		}
	}
	return false
}

// Resources returns no resources: the framework stubs contribute none.
func (a *Adapter) Resources() []spi.ResourceRegistration { return nil }

// Prompts returns no prompts: the framework stubs contribute none.
func (a *Adapter) Prompts() []spi.PromptRegistration { return nil }

// Tools returns the three tool registrations: two fully-real (info,
// capabilities) and one honestly research-gated (config).
func (a *Adapter) Tools() []spi.ToolRegistration {
	return []spi.ToolRegistration{
		{
			Name:        a.toolName("info"),
			Description: a.desc.InfoDescription,
			InputSchema: toolutil.EmptyObjectSchema(),
			Category:    middleware.CategoryStatus,
			Tier:        tierStandard,
			Handler:     a.handleInfo,
		},
		{
			Name:        a.toolName("config"),
			Description: a.desc.ConfigDescription,
			InputSchema: toolutil.EmptyObjectSchema(),
			Category:    middleware.CategoryGeneral,
			Tier:        tierExtended,
			Handler:     a.handleConfig,
		},
		{
			Name:        a.toolName("capabilities"),
			Description: a.desc.CapabilitiesDescription,
			InputSchema: toolutil.EmptyObjectSchema(),
			Category:    middleware.CategoryStatus,
			Tier:        tierStandard,
			Handler:     a.handleCapabilities,
		},
	}
}

// handleInfo summarizes the project via internal/detect, framed for the
// framework. It is fully real: it delegates detection rather than reinventing it.
func (a *Adapter) handleInfo(ctx context.Context, cc *spi.ToolCallContext, _ *spi.ToolRequest) (*spi.ToolResult, error) {
	if cc.ProjectRoot == "" {
		return toolutil.NotConfigured("no project root resolved for "+a.desc.Label+" info", nil), nil
	}
	det := detect.Detect(cc.ProjectRoot)
	langs := toolutil.DetectedLanguages(det)
	structured := map[string]any{
		"framework":         string(a.desc.ID),
		"project_root":      cc.ProjectRoot,
		"config_convention": a.desc.ConfigConvention,
		"markers_present":   a.Applies(ctx, cc.ProjectRoot),
		"languages":         langs,
		"ecosystems":        toolutil.SortedTrueKeys(det.Ecosystems),
		"container_runtime": det.ContainerRuntime,
		"os_family":         det.OSFamily,
		"is_git_repo":       det.IsGitRepo,
	}
	text := fmt.Sprintf("%s project info for %s: languages %s (config convention: %s)",
		a.desc.Label, cc.ProjectRoot, toolutil.JoinOrNone(langs), a.desc.ConfigConvention)
	return &spi.ToolResult{Text: text, Structured: structured}, nil
}

// handleConfig returns the honest research-gated result for deep config
// rendering. It is intentionally NOT a fake success and NOT an empty no-op: it
// reports IsError with a structured payload (encoded in Text, Structured left
// nil) naming exactly what is pending, so the bridge emits a real protocol error
// result and clients observe IsError=true.
func (a *Adapter) handleConfig(_ context.Context, cc *spi.ToolCallContext, _ *spi.ToolRequest) (*spi.ToolResult, error) {
	payload := map[string]any{
		"status":          "research_gated",
		"framework":       string(a.desc.ID),
		"reason":          a.desc.GatedReason,
		"config_format":   a.desc.ConfigFormat,
		"config_paths":    a.desc.ConfigPaths,
		"project_root":    cc.ProjectRoot,
		"pending_spike":   pendingSpike,
		"pending_p19":     a.desc.PendingP19,
		"available_tools": []string{a.toolName("info"), a.toolName("capabilities")},
	}
	text, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return &spi.ToolResult{Text: "research_gated: " + a.desc.Label + " config rendering not yet implemented", IsError: true}, nil
	}
	return &spi.ToolResult{Text: string(text), IsError: true}, nil
}

// handleCapabilities returns the framework's validated capability profile as
// structured content. It is fully real research data carried on the descriptor.
func (a *Adapter) handleCapabilities(_ context.Context, _ *spi.ToolCallContext, _ *spi.ToolRequest) (*spi.ToolResult, error) {
	return &spi.ToolResult{Text: a.desc.CapabilitiesText, Structured: a.desc.Capabilities}, nil
}
