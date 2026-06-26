package security

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/config"
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
type policyChecker struct {
	projectRoot string

	mu            sync.Mutex
	projectPath   string
	projectMtime  time.Time
	cachedProject *types.QsdevConfig
	localPath     string
	localMtime    time.Time
	cachedLocal   *config.LocalConfig
}

func newPolicyChecker(projectRoot string) *policyChecker {
	return &policyChecker{
		projectRoot: projectRoot,
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
	requestedPath := toolutil.StringArgOr(req.Arguments, "policy_path",
		filepath.Join(pc.projectRoot, branding.Get().ConfigFile))

	// Containment: a caller-supplied policy_path must resolve to a location
	// within the project root. Without this an attacker could point the tool at
	// any readable file on the host (path traversal). An escaping path degrades to
	// not_configured rather than reading the file.
	policyPath, ok := pc.resolvePolicyPath(requestedPath)
	if !ok {
		return toolutil.NotConfigured("policy_path escapes the project root",
			map[string]any{"policy_path": requestedPath, "project_root": pc.projectRoot}), nil
	}

	project, local, err := pc.loadPolicy(policyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return toolutil.NotConfigured("project policy not found: "+branding.Get().ConfigFile+" is absent",
				map[string]any{"expected_path": policyPath}), nil
		}
		return toolutil.NotConfigured("could not parse project policy",
			map[string]any{"path": policyPath, "error": err.Error()}), nil
	}

	defaultDecision := defaultDecisionFor(project, local)

	if toolName, ok := toolutil.StringArg(req.Arguments, "tool_name"); ok && toolName != "" {
		dec := evaluateTool(toolName, project, local, defaultDecision)
		text := fmt.Sprintf("policy: tool %q is %s (source=%s, rule=%s)", dec.Tool, dec.Decision, dec.Source, dec.Rule)
		return toolutil.Result(text, map[string]any{
			"policy_path":      policyPath,
			"default_decision": defaultDecision,
			"evaluation":       dec,
		}), nil
	}

	rules := inventoryRules(project, local)
	structured := map[string]any{
		"policy_path":      policyPath,
		"default_decision": defaultDecision,
		"rules":            rules,
		"rule_count":       len(rules),
	}
	text := fmt.Sprintf("policy: %d explicit rules; default decision is %q", len(rules), defaultDecision)
	return toolutil.Result(text, structured), nil
}

// loadPolicy returns the parsed project config and optional local overlay,
// reusing the cached parse while the files are unchanged. It returns a
// not-exist error (detectable via os.IsNotExist) when the project policy file is
// absent so the caller can degrade to not_configured.
func (pc *policyChecker) loadPolicy(policyPath string) (*types.QsdevConfig, *config.LocalConfig, error) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	info, err := os.Stat(policyPath)
	if err != nil {
		return nil, nil, err
	}

	if pc.cachedProject != nil && pc.projectPath == policyPath && pc.projectMtime.Equal(info.ModTime()) {
		// Project parse is still valid; only re-check the local overlay below.
	} else {
		project, perr := config.ParseQsdevConfig(policyPath)
		if perr != nil {
			return nil, nil, perr
		}
		pc.cachedProject = project
		pc.projectPath = policyPath
		pc.projectMtime = info.ModTime()
	}

	pc.refreshLocalLocked()
	return pc.cachedProject, pc.cachedLocal, nil
}

// refreshLocalLocked reloads the .qsdev.local.yaml overlay when its mtime
// changed. An absent overlay yields a nil cachedLocal. The caller holds pc.mu.
func (pc *policyChecker) refreshLocalLocked() {
	info, err := os.Stat(pc.localPath)
	if err != nil {
		pc.cachedLocal = nil
		pc.localMtime = time.Time{}
		return
	}
	if pc.cachedLocal != nil && pc.localMtime.Equal(info.ModTime()) {
		return
	}
	// ParseLocalConfig returns (nil, nil) when the file is simply absent; a parse
	// error degrades to no overlay rather than failing the whole evaluation.
	local, perr := config.ParseLocalConfig(pc.localPath)
	if perr != nil {
		pc.cachedLocal = nil
		return
	}
	pc.cachedLocal = local
	pc.localMtime = info.ModTime()
}

// evaluateTool resolves the verdict for tool through the cascade
// local > project > default; within a level a deny (tools.disabled) takes
// precedence over an allow (tools.enabled), so an explicit deny is never
// silently overridden at the same level.
func evaluateTool(tool string, project *types.QsdevConfig, local *config.LocalConfig, defaultDecision string) policyDecision {
	if local != nil {
		if containsStr(local.Tools.Disabled, tool) {
			return policyDecision{Tool: tool, Decision: decisionDenied, Source: sourceLocal, Rule: "tools.disabled"}
		}
		if containsStr(local.Tools.Enabled, tool) {
			return policyDecision{Tool: tool, Decision: decisionAllowed, Source: sourceLocal, Rule: "tools.enabled"}
		}
	}
	if containsStr(project.Tools.Disabled, tool) {
		return policyDecision{Tool: tool, Decision: decisionDenied, Source: sourceProject, Rule: "tools.disabled"}
	}
	if containsStr(project.Tools.Enabled, tool) {
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
// explicit rule mentions. A restrictive Claude Code permission level or a high
// security level escalates the default from "allowed" to "ask"; the local
// overlay (when present) takes precedence over the project value.
func defaultDecisionFor(project *types.QsdevConfig, local *config.LocalConfig) string {
	level := project.ClaudeCode.PermissionLevel
	secLevel := project.Security.Level
	if local != nil {
		if local.ClaudeCode.PermissionLevel != "" {
			level = local.ClaudeCode.PermissionLevel
		}
		if local.Security.Level != "" {
			secLevel = local.Security.Level
		}
	}
	switch level {
	case "strict", "ask", "plan":
		return decisionAsk
	}
	switch secLevel {
	case "high", "strict", "maximum":
		return decisionAsk
	}
	return decisionAllowed
}

func containsStr(list []string, v string) bool {
	for _, e := range list {
		if e == v {
			return true
		}
	}
	return false
}
