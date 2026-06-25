// Package claudecode is the Claude Code framework adapter for the universal
// qsdev MCP server (Phase 32, Unit 32.3). It contributes Claude Code-specific
// tools and resources (permission translation, hook inspection, context-budget
// accounting, config rendering, and enforcement-gap reporting) to the server's
// adapter registry.
//
// The adapter is a stateless singleton. It self-registers into
// spi.DefaultRegistry() from init(), long before any project root is resolved,
// so it captures no project root: every tool and resource handler reads the
// resolved root from its *spi.ToolCallContext at call time. The mount-time
// applicability check (Applies) likewise receives the root as an argument.
//
// Delegation: this package performs no generation of its own. It is the
// quarantined leaf that is permitted to import addons/claudecode (it is
// blank-imported only from cmd/qsdev/main.go, never from the mcpserve server
// root, so it cannot create an import cycle). Policy translation, config
// rendering, detection, and gap analysis delegate to the P19 reference adapter
// in pkg/aiframework/adapters/claudecode; context-budget accounting and the
// deployed settings/hook inspection read the addon directly.
package claudecode

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	ccaddon "github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
	refcc "github.com/Quantum-Serendipity/qsdev/pkg/aiframework/adapters/claudecode"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Tool names exposed by the Claude Code adapter. They use the flat qsdev_
// namespace with an underscore-cased cc_ infix, mirroring the generic project
// context tools (qsdev_project_info, qsdev_doctor, ...).
const (
	toolPermissions     = "qsdev_cc_permissions"
	toolHooks           = "qsdev_cc_hooks"
	toolContextBudget   = "qsdev_cc_context_budget"
	toolConfigRender    = "qsdev_cc_config_render"
	toolEnforcementGaps = "qsdev_cc_enforcement_gaps"
)

// Resource URIs and the single MIME type the adapter emits.
const (
	resURISettings = "qsdev://claudecode/settings"
	resURIHooks    = "qsdev://claudecode/hooks"

	mimeJSON = "application/json"
)

// Tool tier hints (lower is more core), mirroring projectctx's Tier bands.
const (
	tierStandard = 1
	tierExtended = 2
)

// defaultPreset is the permission preset applied when the project's .qsdev.yaml
// declares none (or is absent). It matches the addon's GenerateSettings default.
const defaultPreset = string(ccaddon.PermissionPresetStandard)

// Adapter is the stateless Claude Code framework adapter. It holds only the P19
// reference adapter it delegates to (constructed once against the package-level
// ecosystem registry); it captures no project root, so a single instance serves
// every project root the server is mounted against.
type Adapter struct {
	ref *refcc.Adapter
}

// compile-time assertions that Adapter satisfies the contract and the optional
// client-matching seam DetectFrameworks consumes.
var (
	_ spi.FrameworkAdapter = (*Adapter)(nil)
	_ spi.ClientMatcher    = (*Adapter)(nil)
)

// New constructs the stateless Claude Code adapter. The reference adapter is
// wired with the standard-preset default and the package-level ecosystem
// registry; per-call handlers override the preset with the project's own
// .qsdev.yaml value, so the construction-time preset is only a fallback.
func New() *Adapter {
	cfg := ccaddon.Config{DefaultPermissions: ccaddon.PermissionPresetStandard}
	return &Adapter{ref: refcc.New(cfg, ecosystem.DefaultRegistry())}
}

// init self-registers the singleton into the default adapter registry. This is
// the sanctioned registry self-registration pattern (see pkg/ecosystem); the
// general "avoid init()" guidance does not apply to registry wiring. The
// registry rejects duplicate IDs, which can only happen if this package is
// linked twice — a build error worth surfacing loudly.
func init() {
	if err := spi.DefaultRegistry().Register(New()); err != nil {
		panic(fmt.Sprintf("registering claude code framework adapter: %v", err))
	}
}

// ID identifies this adapter as the Claude Code framework.
func (a *Adapter) ID() aiframework.FrameworkID { return aiframework.ClaudeCode }

// Applies reports whether projectRoot carries Claude Code markers (.claude/,
// CLAUDE.md, .mcp.json, or the claude binary), delegating to the reference
// adapter's detection. Client-info-based detection (matching the MCP client's
// reported name) is Task T7's DetectFrameworks responsibility and is
// deliberately not performed here.
func (a *Adapter) Applies(_ context.Context, projectRoot string) bool {
	if projectRoot == "" {
		return false
	}
	det, err := a.ref.Detect(projectRoot)
	if err != nil {
		return false
	}
	return det != nil && det.Detected
}

// MatchesClient reports whether the connected MCP client identifies as Claude
// Code. It matches any client whose reported name contains "claude"
// (case-insensitively) — e.g. "claude-code", "Claude Code", "claude-desktop".
// This is deliberately broader than the DefaultClientMatch rule (which would
// require the full normalized "claudecode" token) so every Claude-family client
// is served. It implements spi.ClientMatcher, consumed by
// AdapterRegistry.DetectFrameworks when filtering tools/list per client.
func (a *Adapter) MatchesClient(client spi.ClientInfo) bool {
	return strings.Contains(strings.ToLower(client.Name), "claude")
}

// Prompts returns no prompts: the Claude Code adapter contributes tools and
// resources only (Unit 32.3 defines no prompts).
func (a *Adapter) Prompts() []spi.PromptRegistration { return nil }

// loadConfig parses the project's .qsdev.yaml best-effort. It returns nil when
// the file is absent or unparseable so callers fall back to defaults rather
// than failing — graceful degradation per the universal-server contract.
func loadConfig(projectRoot string) *types.QsdevConfig {
	path := filepath.Join(projectRoot, branding.Get().ConfigFile)
	cfg, err := config.ParseQsdevConfig(path)
	if err != nil {
		return nil
	}
	return cfg
}

// presetFor resolves the Claude Code permission preset for projectRoot from its
// .qsdev.yaml, defaulting to the standard preset when unset or unavailable.
func presetFor(projectRoot string) string {
	cfg := loadConfig(projectRoot)
	if cfg == nil || cfg.ClaudeCode.PermissionLevel == "" {
		return defaultPreset
	}
	return cfg.ClaudeCode.PermissionLevel
}

// policyFor builds the framework-agnostic permission policy for projectRoot,
// keyed on its configured preset.
func policyFor(projectRoot string) *aiframework.PermissionPolicy {
	return &aiframework.PermissionPolicy{Preset: presetFor(projectRoot)}
}

// policyInputFor builds the framework-agnostic PolicyInput a ConfigRenderer
// consumes, derived from .qsdev.yaml (preset, declared MCP servers) and live
// detection. Model is intentionally left nil so a render preview never fails on
// the context-budget threshold; budget accounting has its own dedicated tool.
func (a *Adapter) policyInputFor(projectRoot string) *aiframework.PolicyInput {
	input := &aiframework.PolicyInput{
		ProjectRoot: projectRoot,
		Permissions: policyFor(projectRoot),
	}
	if det, err := a.ref.Detect(projectRoot); err == nil {
		input.Detection = det
	}
	if cfg := loadConfig(projectRoot); cfg != nil {
		for _, name := range cfg.ClaudeCode.MCPServers {
			if name != "" {
				input.MCPServers = append(input.MCPServers, aiframework.MCPServerSpec{Name: name})
			}
		}
	}
	return input
}
