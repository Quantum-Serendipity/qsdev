package security

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/middleware"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools/toolutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// policyChecker evaluates the project's security policy entirely in memory from
// the loaded .qsdev.yaml (and optional .qsdev.local.yaml overlay). It is a
// fast-path tool: parsed configuration is cached and reused while the policy
// files are unchanged (compared by modification time), so repeated calls return
// in well under the ≤5ms target without re-parsing YAML.
//
// Verdicts are computed from the canonical resolved configuration
// (config.ResolveConfig), so the overlay semantics — union of the tools lists,
// the security-level floor — are exactly those every other qsdev surface applies.
type policyChecker struct {
	projectRoot string
	defaultPath string
	// enforced is the Guardrail policy the running server installed at startup
	// (nil when nothing is narrowed). mcp_enforced_deny is read from it, not
	// re-derived from the file on disk, so an edit made after startup is never
	// reported as enforced before a restart actually enforces it.
	enforced *middleware.Policy

	mu            sync.Mutex
	projectPath   string
	projectMtime  time.Time
	cachedProject *types.QsdevConfig
	localPath     string
	localMtime    time.Time
	cachedLocal   *config.LocalConfig
}

func newPolicyChecker(projectRoot string, enforced *middleware.Policy) *policyChecker {
	return &policyChecker{
		projectRoot: projectRoot,
		defaultPath: filepath.Join(projectRoot, branding.Get().ConfigFile),
		enforced:    enforced,
		localPath:   filepath.Join(projectRoot, branding.Get().LocalConfig),
	}
}

// resolvePolicyPath confines the requested policy path to the project root,
// delegating to the shared toolutil.ConfineToRoot primitive so policy_check and
// security_scan apply identical containment. It returns the cleaned, absolute
// path and ok=true when the target stays inside the root, or ok=false when the
// path escapes (path traversal).
func (pc *policyChecker) resolvePolicyPath(policyPath string) (string, bool) {
	return toolutil.ConfineToRoot(pc.projectRoot, policyPath)
}

// policyDecision is the evaluated verdict for a single tool.
type policyDecision struct {
	Tool     string `json:"tool"`
	Decision string `json:"decision"` // allowed | denied | ask
	Source   string `json:"source"`   // project | local | default
	Rule     string `json:"rule"`     // tools.disabled | tools.enabled | default
}

// floorViolation is the JSON-facing form of a config.FloorViolation: an overlay
// setting the canonical resolver refused because it would weaken the project's
// security floor.
type floorViolation struct {
	Field     string `json:"field"`
	Attempted any    `json:"attempted"`
	Enforced  any    `json:"enforced"`
	Reason    string `json:"reason"`
}

const (
	decisionAllowed = "allowed"
	decisionDenied  = "denied"
	decisionAsk     = "ask"

	sourceProject = "project"
	sourceLocal   = "local"
	sourceDefault = "default"
)

// handle evaluates the in-memory policy. With tool_name set it returns the
// single decision for that tool; without it, it returns the full rule inventory.
func (pc *policyChecker) handle(_ context.Context, _ *spi.ToolCallContext, req *spi.ToolRequest) (*spi.ToolResult, error) {
	requestedPath := toolutil.StringArgOr(req.Arguments, "policy_path", pc.defaultPath)

	// Containment: a caller-supplied policy_path must resolve to a location
	// within the project root. Without this an attacker could point the tool at
	// any readable file on the host (path traversal). An escaping path degrades to
	// not_configured rather than reading the file.
	policyPath, ok := pc.resolvePolicyPath(requestedPath)
	if !ok {
		return toolutil.NotConfigured("policy_path escapes the project root",
			map[string]any{"policy_path": requestedPath, "project_root": pc.projectRoot}), nil
	}

	project, local, localWarn, err := pc.loadPolicy(policyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return toolutil.NotConfigured("project policy not found: "+branding.Get().ConfigFile+" is absent",
				map[string]any{"expected_path": policyPath}), nil
		}
		return toolutil.NotConfigured("could not parse project policy",
			map[string]any{"path": policyPath, "error": err.Error()}), nil
	}

	// The effective config comes from the canonical resolver (as
	// qsdev_config_show uses), so the local overlay cannot lower the security
	// floor here any more than it can anywhere else.
	resolved, err := config.ResolveConfig(&types.QsdevConfig{}, nil, project, local, false)
	if err != nil {
		return toolutil.NotConfigured("could not resolve effective policy",
			map[string]any{"path": policyPath, "error": err.Error()}), nil
	}
	defaultDecision := defaultDecisionFor(resolved.Config)

	// mcpEnforcedDeny is the exact tool-name deny set the MCP Guardrail enforces on
	// tool calls, read from the Policy object the running server installed at
	// startup — never re-derived from the file (which may have changed since) or
	// from a caller-chosen policy_path. It reflects the project config's
	// tools.disabled only; the local overlay drives advisory Claude Code semantics.
	mcpEnforcedDeny := pc.enforced.DenyToolSet()

	structured := map[string]any{
		"policy_path":       policyPath,
		"default_decision":  defaultDecision,
		"mcp_enforced_deny": mcpEnforcedDeny,
	}
	if len(resolved.Violations) > 0 {
		structured["floor_violations"] = toFloorViolations(resolved.Violations)
	}
	addWarning(structured, localWarn)
	addWarning(structured, pc.enforcementDrift(policyPath, project, mcpEnforcedDeny))

	if toolName, ok := toolutil.StringArg(req.Arguments, "tool_name"); ok && toolName != "" {
		dec := evaluateTool(toolName, project, local, defaultDecision)
		structured["evaluation"] = dec
		text := fmt.Sprintf("policy: tool %q is %s (source=%s, rule=%s)", dec.Tool, dec.Decision, dec.Source, dec.Rule)
		return toolutil.Result(text, structured), nil
	}

	rules := inventoryRules(project, local)
	structured["rules"] = rules
	structured["rule_count"] = len(rules)
	text := fmt.Sprintf("policy: %d explicit rules; default decision is %q", len(rules), defaultDecision)
	return toolutil.Result(text, structured), nil
}

// enforcementDrift explains any gap between the evaluated policy file and the
// deny set the running server actually enforces, or returns "" when there is
// none. A non-default policy_path is never what the Guardrail enforces, and the
// default project config may have been edited since startup, in which case its
// tools.disabled is not enforced until the server restarts.
func (pc *policyChecker) enforcementDrift(policyPath string, project *types.QsdevConfig, enforced []string) string {
	if policyPath != pc.defaultPath {
		return fmt.Sprintf("policy_path %s is not the project config the MCP server enforces; "+
			"its verdicts are hypothetical and mcp_enforced_deny reflects the running server's policy", policyPath)
	}
	if onDisk := middleware.PolicyFromConfig(project).DenyToolSet(); !slices.Equal(onDisk, enforced) {
		return fmt.Sprintf("%s tools.disabled %v differs from the deny set the running MCP server enforces %v "+
			"(loaded at startup); restart the server to enforce the current file",
			branding.Get().ConfigFile, onDisk, enforced)
	}
	return ""
}

// toFloorViolations converts the resolver's floor violations to their
// JSON-facing form.
func toFloorViolations(vs []config.FloorViolation) []floorViolation {
	out := make([]floorViolation, len(vs))
	for i, v := range vs {
		out[i] = floorViolation{Field: v.Field, Attempted: v.Attempted, Enforced: v.Enforced, Reason: v.Reason}
	}
	return out
}

// addWarning records a non-empty warning on a structured tool result under the
// "warnings" key so a degraded (but non-fatal) condition — e.g. a dropped
// malformed overlay — is surfaced to the caller rather than silently swallowed.
func addWarning(m map[string]any, warning string) {
	if warning == "" {
		return
	}
	existing, _ := m["warnings"].([]string)
	m["warnings"] = append(existing, warning)
}

// loadPolicy returns the parsed project config and optional local overlay,
// reusing the cached parse while the files are unchanged. It returns a
// not-exist error (detectable via os.IsNotExist) when the project policy file is
// absent so the caller can degrade to not_configured.
func (pc *policyChecker) loadPolicy(policyPath string) (*types.QsdevConfig, *config.LocalConfig, string, error) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	info, err := os.Stat(policyPath)
	if err != nil {
		return nil, nil, "", err
	}

	if pc.cachedProject != nil && pc.projectPath == policyPath && pc.projectMtime.Equal(info.ModTime()) {
		// Project parse is still valid; only re-check the local overlay below.
	} else {
		project, perr := config.ParseQsdevConfig(policyPath)
		if perr != nil {
			return nil, nil, "", perr
		}
		pc.cachedProject = project
		pc.projectPath = policyPath
		pc.projectMtime = info.ModTime()
	}

	localWarn := pc.refreshLocalLocked()
	return pc.cachedProject, pc.cachedLocal, localWarn, nil
}

// refreshLocalLocked reloads the .qsdev.local.yaml overlay when its mtime
// changed. An absent overlay yields a nil cachedLocal and no warning. A malformed
// overlay is dropped (evaluation proceeds on the project policy) but returns a
// non-empty warning string: the local overlay is the highest-precedence layer, so
// silently losing its denies would be a fail-open the caller must be told about.
// The caller holds pc.mu.
func (pc *policyChecker) refreshLocalLocked() string {
	info, err := os.Stat(pc.localPath)
	if err != nil {
		pc.cachedLocal = nil
		pc.localMtime = time.Time{}
		return ""
	}
	if pc.cachedLocal != nil && pc.localMtime.Equal(info.ModTime()) {
		return ""
	}
	// ParseLocalConfig returns (nil, nil) when the file is simply absent; a parse
	// error degrades to no overlay rather than failing the whole evaluation.
	local, perr := config.ParseLocalConfig(pc.localPath)
	if perr != nil {
		pc.cachedLocal = nil
		return fmt.Sprintf("ignored malformed %s overlay (highest-precedence rules not applied): %v",
			branding.Get().LocalConfig, perr)
	}
	pc.cachedLocal = local
	pc.localMtime = info.ModTime()
	return ""
}

// evaluateTool resolves the verdict for tool. The canonical resolver unions the
// project and local tools lists, so a deny (tools.disabled) at ANY level wins
// over an allow at any level: a local tools.enabled never re-enables a tool the
// project disables (the MCP Guardrail still enforces that project deny). The
// project deny is reported first because it is the one the Guardrail enforces.
func evaluateTool(tool string, project *types.QsdevConfig, local *config.LocalConfig, defaultDecision string) policyDecision {
	var localTools types.ToolsConfig
	if local != nil {
		localTools = local.Tools
	}
	switch {
	case slices.Contains(project.Tools.Disabled, tool):
		return policyDecision{Tool: tool, Decision: decisionDenied, Source: sourceProject, Rule: "tools.disabled"}
	case slices.Contains(localTools.Disabled, tool):
		return policyDecision{Tool: tool, Decision: decisionDenied, Source: sourceLocal, Rule: "tools.disabled"}
	case slices.Contains(localTools.Enabled, tool):
		return policyDecision{Tool: tool, Decision: decisionAllowed, Source: sourceLocal, Rule: "tools.enabled"}
	case slices.Contains(project.Tools.Enabled, tool):
		return policyDecision{Tool: tool, Decision: decisionAllowed, Source: sourceProject, Rule: "tools.enabled"}
	}
	return policyDecision{Tool: tool, Decision: defaultDecision, Source: sourceDefault, Rule: "default"}
}

// inventoryRules lists every explicit allow/deny rule across project and local
// configuration, each tagged with its source so a caller can see the full
// effective policy without naming a specific tool.
func inventoryRules(project *types.QsdevConfig, local *config.LocalConfig) []policyDecision {
	var rules []policyDecision
	for _, t := range project.Tools.Disabled {
		rules = append(rules, policyDecision{Tool: t, Decision: decisionDenied, Source: sourceProject, Rule: "tools.disabled"})
	}
	for _, t := range project.Tools.Enabled {
		rules = append(rules, policyDecision{Tool: t, Decision: decisionAllowed, Source: sourceProject, Rule: "tools.enabled"})
	}
	if local != nil {
		for _, t := range local.Tools.Disabled {
			rules = append(rules, policyDecision{Tool: t, Decision: decisionDenied, Source: sourceLocal, Rule: "tools.disabled"})
		}
		for _, t := range local.Tools.Enabled {
			rules = append(rules, policyDecision{Tool: t, Decision: decisionAllowed, Source: sourceLocal, Rule: "tools.enabled"})
		}
	}
	return rules
}

// defaultDecisionFor derives the fallback verdict applied to a tool that no
// explicit rule mentions, from the effective (resolved, floor-enforced) config.
// It escalates from "allowed" to "ask" when either:
//   - the Claude Code permission preset starts sessions in plan mode per its
//     catalog definition (e.g. "minimal"), so nothing runs without approval; or
//   - the effective security level is the strictest compliance level.
func defaultDecisionFor(effective *types.QsdevConfig) string {
	if presetRequiresApproval(effective.ClaudeCode.PermissionLevel) {
		return decisionAsk
	}
	if lvl, err := config.ParseComplianceLevel(effective.Security.Level); err == nil && lvl >= config.ComplianceLevelStrict {
		return decisionAsk
	}
	return decisionAllowed
}

// planDefaultMode is the Claude Code defaultMode under which no tool runs
// without the user first approving a plan.
const planDefaultMode = "plan"

// presetRequiresApproval reports whether the named permission preset, as defined
// in the catalog, starts Claude Code in plan mode.
func presetRequiresApproval(preset string) bool {
	if preset == "" {
		return false
	}
	cat, err := catalog.Default()
	if err != nil {
		return false
	}
	def, ok := cat.PermissionPreset(preset)
	return ok && def.DefaultMode == planDefaultMode
}
