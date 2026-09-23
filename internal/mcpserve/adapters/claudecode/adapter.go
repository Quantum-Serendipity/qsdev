// Package claudecode is the Claude Code framework adapter for the universal
// qsdev MCP server (Phase 32, Unit 32.3). It contributes Claude Code-specific
// tools and resources (permission translation, hook inspection, context-budget
// accounting, config rendering, and enforcement-gap reporting) to the server's
// adapter registry.
//
// The adapter is a stateless singleton. It is registered into the adapter
// registry explicitly from cmd/qsdev/main.go, before any project root is
// resolved, so it captures no project root: every tool and resource handler reads the
// resolved root from its *spi.ToolCallContext at call time. The mount-time
// applicability check (Applies) likewise receives the root as an argument.
//
// Delegation: this package performs no generation of its own. It is the
// quarantined leaf that is permitted to import addons/claudecode (it is
// imported only from cmd/qsdev/main.go, never from the mcpserve server
// root, so it cannot create an import cycle). Policy translation, config
// rendering, detection, and gap analysis delegate to the P19 reference adapter
// in pkg/aiframework/adapters/claudecode; context-budget accounting and the
// deployed settings/hook inspection read the addon directly.
package claudecode

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	ccaddon "github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools/toolutil"
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

// loadConfig parses the project's .qsdev.yaml. An absent file is benign and
// yields (nil, nil) so callers fall back to defaults. A present-but-unparseable
// file is returned as an error: silently substituting the default preset would
// report and render a policy the project does not have.
func loadConfig(projectRoot string) (*types.QsdevConfig, error) {
	path := filepath.Join(projectRoot, branding.Get().ConfigFile)
	cfg, err := config.ParseQsdevConfig(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return cfg, nil
}

// presetOf resolves the Claude Code permission preset from a loaded config,
// defaulting to the standard preset when the config is absent or declares none.
func presetOf(cfg *types.QsdevConfig) string {
	if cfg == nil || cfg.ClaudeCode.PermissionLevel == "" {
		return defaultPreset
	}
	return cfg.ClaudeCode.PermissionLevel
}

// presetFor resolves the Claude Code permission preset for projectRoot from its
// .qsdev.yaml. It errors when the config is present but unparseable.
func presetFor(projectRoot string) (string, error) {
	cfg, err := loadConfig(projectRoot)
	if err != nil {
		return "", err
	}
	return presetOf(cfg), nil
}

// configError degrades a present-but-unparseable .qsdev.yaml to a structured
// not_configured result that carries the parse error.
func configError(err error) *spi.ToolResult {
	return toolutil.NotConfigured("project "+branding.Get().ConfigFile+" could not be parsed",
		map[string]any{"error": err.Error()})
}

// policyInputFor builds the framework-agnostic PolicyInput a ConfigRenderer
// consumes, derived from .qsdev.yaml (preset, hooks, declared MCP servers) and
// live detection. Model is intentionally left nil so a render preview never
// fails on the context-budget threshold; budget accounting has its own
// dedicated tool. It also returns the enabled hook choices the framework-agnostic
// hook vocabulary cannot express (so they are absent from the render), letting
// callers surface that gap rather than present the render as complete.
func (a *Adapter) policyInputFor(projectRoot string) (*aiframework.PolicyInput, []string, error) {
	cfg, err := loadConfig(projectRoot)
	if err != nil {
		return nil, nil, err
	}
	specs, unrendered, err := projectHookSpecs(projectRoot, cfg)
	if err != nil {
		return nil, nil, err
	}
	input := &aiframework.PolicyInput{
		ProjectRoot: projectRoot,
		Permissions: &aiframework.PermissionPolicy{Preset: presetOf(cfg)},
		Hooks:       &aiframework.HookConfiguration{Hooks: specs},
	}
	if det, derr := a.ref.Detect(projectRoot); derr == nil {
		input.Detection = det
	}
	if cfg != nil {
		for _, name := range cfg.ClaudeCode.MCPServers {
			if name != "" {
				input.MCPServers = append(input.MCPServers, aiframework.MCPServerSpec{Name: name})
			}
		}
	}
	return input, unrendered, nil
}

// projectHookChoices derives the hook selections `qsdev init` would generate
// for this project: the canonical config-to-answers bridge
// (config.ConfigToAnswers) followed by the catalog-driven FillDefaults, with
// Claude Code enabled (this adapter renders Claude Code config). That yields
// the always-on safety-block and self-protection hooks plus whatever the
// security level adds.
func projectHookChoices(projectRoot string, cfg *types.QsdevConfig) (types.HookChoices, error) {
	if cfg == nil {
		cfg = &types.QsdevConfig{}
	}
	cat, err := catalog.Default()
	if err != nil {
		return types.HookChoices{}, fmt.Errorf("loading catalog for hook defaults: %w", err)
	}
	answers := config.ConfigToAnswers(cfg, types.DetectedProject{}, projectRoot)
	answers.ClaudeCode = true
	answers.FillDefaults(types.DetectedProject{}, cat)
	return answers.Hooks, nil
}

// projectHookSpecs converts the project's hook choices into framework-agnostic
// hook specs keyed by logic ID. Enabled choices with no logic ID in the shared
// vocabulary are returned by name in unrendered.
func projectHookSpecs(projectRoot string, cfg *types.QsdevConfig) (specs []aiframework.HookSpec, unrendered []string, err error) {
	hc, err := projectHookChoices(projectRoot, cfg)
	if err != nil {
		return nil, nil, err
	}
	for _, h := range []struct {
		enabled bool
		logic   aiframework.HookLogicID
		name    string
	}{
		{hc.SafetyBlock, aiframework.LogicPackageGuard, "safety_block"},
		{hc.SelfProtection, aiframework.LogicAgentSelfProtection, "self_protection"},
		{hc.CredentialScan, aiframework.LogicCredentialScan, "credential_scan"},
		{hc.DestructivePrevention, aiframework.LogicDestructiveBlock, "destructive_prevention"},
		{hc.FileBoundary, aiframework.LogicFileBoundary, "file_boundary"},
		{hc.ToolGates, aiframework.LogicToolGates, "tool_gates"},
		{hc.AutoFormat, "", "auto_format"},
		{hc.PreCommit, "", "pre_commit"},
		{hc.AuditLog, "", "audit_log"},
		{hc.SOC2Audit, "", "soc2_audit"},
		{hc.SecurityEnforcement, "", "security_enforcement"},
	} {
		switch {
		case !h.enabled:
		case h.logic == "":
			unrendered = append(unrendered, h.name)
		default:
			specs = append(specs, aiframework.HookSpec{Command: string(h.logic)})
		}
	}
	return specs, unrendered, nil
}
