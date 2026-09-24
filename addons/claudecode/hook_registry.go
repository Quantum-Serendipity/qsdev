package claudecode

import (
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"
	"github.com/Quantum-Serendipity/qsdev/internal/tier"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// HookDefinition describes a hook that can be registered with the HookRegistry.
// Each definition maps to one HookMatcher entry in the project's generated
// .claude/settings.json; qsdev does not generate user-level or managed
// (organisation) settings.
type HookDefinition struct {
	Owner           string
	Event           string
	Matcher         string
	Command         string
	Timeout         int
	StatusMessage   string
	SandboxCategory string // sandbox permission profile (e.g., "linter", "generator")
	EnabledFunc     func(types.WizardAnswers) bool
	// CommandFunc, when set, derives the emitted command from the answers
	// (e.g. to bake a configured setting into it). Command stays the base
	// command used for display and template lookup.
	CommandFunc func(types.WizardAnswers) string
	// HasPolicy, when set, reports whether the answers carry the policy the
	// hook enforces. Such a hook, enabled without one, runs but restricts
	// nothing, and status reports show it as "enabled (no policy)".
	// PolicyKey names the .qsdev.yaml key that sets the policy.
	HasPolicy func(types.WizardAnswers) bool
	PolicyKey string
}

// lacksPolicy reports whether the hook is enabled by answers but has no
// policy to enforce.
func (h HookDefinition) lacksPolicy(answers types.WizardAnswers) bool {
	enabled := h.EnabledFunc == nil || h.EnabledFunc(answers)
	return enabled && h.HasPolicy != nil && !h.HasPolicy(answers)
}

// commandFor returns the command emitted into settings.json for answers.
func (h HookDefinition) commandFor(answers types.WizardAnswers) string {
	if h.CommandFunc != nil {
		return h.CommandFunc(answers)
	}
	return h.Command
}

// HookRegistry collects hook definitions and produces the hooks map for
// settings.json generation. Hooks are evaluated in registration order.
type HookRegistry struct {
	hooks []HookDefinition
}

// NewHookRegistry returns an empty registry.
func NewHookRegistry() *HookRegistry {
	return &HookRegistry{}
}

// Register adds a hook definition to the registry.
func (r *HookRegistry) Register(h HookDefinition) {
	r.hooks = append(r.hooks, h)
}

// HooksForEvent returns all HookMatcher entries for the given event, filtering
// to only those whose EnabledFunc returns true for the provided answers. If
// EnabledFunc is nil the hook is always enabled.
func (r *HookRegistry) HooksForEvent(event string, answers types.WizardAnswers) []HookMatcher {
	var matchers []HookMatcher
	for _, h := range r.hooks {
		if h.Event != event {
			continue
		}
		if h.EnabledFunc != nil && !h.EnabledFunc(answers) {
			continue
		}
		matchers = append(matchers, HookMatcher{
			Matcher: h.Matcher,
			Hooks: []HookEntry{{
				Type:          "command",
				Command:       h.commandFor(answers),
				Timeout:       h.Timeout,
				StatusMessage: h.StatusMessage,
			}},
		})
	}
	return matchers
}

// HookWithoutPolicy names a hook the answers enable without the policy it
// enforces, and the .qsdev.yaml key that sets that policy.
type HookWithoutPolicy struct {
	Name      string
	PolicyKey string
}

// HooksWithoutPolicy returns, in registration order, each hook the answers
// enable without the policy it enforces (e.g. tool-gates with neither an
// allow nor a deny list), which therefore restricts nothing.
func HooksWithoutPolicy(answers types.WizardAnswers) []HookWithoutPolicy {
	var out []HookWithoutPolicy
	seen := make(map[string]bool)
	for _, h := range defaultHookRegistry().Definitions() {
		if seen[h.Owner] || !h.lacksPolicy(answers) {
			continue
		}
		seen[h.Owner] = true
		out = append(out, HookWithoutPolicy{Name: h.Owner, PolicyKey: h.PolicyKey})
	}
	return out
}

// BuildHooksMap evaluates all registered hooks against the provided answers and
// returns the complete hooks map keyed by event name, ready for SettingsJSON.
// Returns nil when no hooks are enabled.
func (r *HookRegistry) BuildHooksMap(answers types.WizardAnswers) map[string][]HookMatcher {
	hooks := make(map[string][]HookMatcher)
	seen := make(map[string]bool)
	for _, h := range r.hooks {
		if !seen[h.Event] {
			seen[h.Event] = true
		}
	}
	for event := range seen {
		if matchers := r.HooksForEvent(event, answers); len(matchers) > 0 {
			hooks[event] = matchers
		}
	}
	if len(hooks) == 0 {
		return nil
	}
	return hooks
}

// Definitions returns all registered hook definitions. This is used by the
// hooks list command to display hook metadata.
func (r *HookRegistry) Definitions() []HookDefinition {
	result := make([]HookDefinition, len(r.hooks))
	copy(result, r.hooks)
	return result
}

// shellToolMatcher matches every tool that runs a shell command, for the hooks
// that inspect tool_input.command. A hook matching only "Bash" never sees the
// same command sent through PowerShell or Monitor.
var shellToolMatcher = strings.Join(cmdscan.ShellTools, "|")

// defaultHookRegistry returns a registry pre-populated with the built-in hooks
// (package-guard and audit-log).
func defaultHookRegistry() *HookRegistry {
	r := NewHookRegistry()

	selfprotectApp := branding.Get().AppName

	r.Register(HookDefinition{
		Owner:         "self-protection",
		Event:         "PreToolUse",
		Matcher:       "*",
		Command:       selfprotectApp + " selfprotect",
		Timeout:       10,
		StatusMessage: "Checking self-protection rules...",
		EnabledFunc:   func(a types.WizardAnswers) bool { return a.Hooks.SelfProtection },
	})

	guardCmd := `"${CLAUDE_PROJECT_DIR}"/.claude/hooks/package-guard.py`

	r.Register(HookDefinition{
		Owner:           "package-guard",
		Event:           "PreToolUse",
		Matcher:         shellToolMatcher,
		Command:         guardCmd,
		Timeout:         30,
		StatusMessage:   "Checking package install safety...",
		SandboxCategory: "linter",
		EnabledFunc:     func(a types.WizardAnswers) bool { return a.Hooks.SafetyBlock },
	})

	r.Register(HookDefinition{
		Owner:           "credential-scan",
		Event:           "PreToolUse",
		Matcher:         "Write|Edit|MultiEdit|NotebookEdit",
		Command:         `"${CLAUDE_PROJECT_DIR}"/.claude/hooks/scan-secrets.py`,
		Timeout:         10,
		StatusMessage:   "Scanning for credentials...",
		SandboxCategory: "linter",
		EnabledFunc:     func(a types.WizardAnswers) bool { return a.Hooks.CredentialScan },
	})

	r.Register(HookDefinition{
		Owner:           "destructive-prevention",
		Event:           "PreToolUse",
		Matcher:         shellToolMatcher,
		Command:         `"${CLAUDE_PROJECT_DIR}"/.claude/hooks/block-destructive.py`,
		Timeout:         5,
		StatusMessage:   "Checking command safety...",
		SandboxCategory: "linter",
		EnabledFunc:     func(a types.WizardAnswers) bool { return a.Hooks.DestructivePrevention },
	})

	r.Register(HookDefinition{
		Owner:           "file-boundary",
		Event:           "PreToolUse",
		Matcher:         "Write|Edit|MultiEdit|NotebookEdit|Read|Grep|Glob",
		Command:         `"${CLAUDE_PROJECT_DIR}"/.claude/hooks/file-boundary.py`,
		Timeout:         5,
		StatusMessage:   "Checking file boundary...",
		SandboxCategory: "linter",
		EnabledFunc:     func(a types.WizardAnswers) bool { return a.Hooks.FileBoundary },
	})

	r.Register(HookDefinition{
		Owner:           "tool-gates",
		Event:           "PreToolUse",
		Matcher:         "*",
		Command:         `"${CLAUDE_PROJECT_DIR}"/.claude/hooks/tool-gates.py`,
		Timeout:         3,
		StatusMessage:   "Checking tool policy...",
		SandboxCategory: "linter",
		EnabledFunc:     func(a types.WizardAnswers) bool { return a.Hooks.ToolGates },
		HasPolicy:       func(a types.WizardAnswers) bool { return a.HookPolicy.ToolGates.HasPolicy() },
		PolicyKey:       "hooks.tool_gates",
	})

	soc2Cmd := `"${CLAUDE_PROJECT_DIR}"/.claude/hooks/soc2-audit-log.py`
	soc2Enabled := func(a types.WizardAnswers) bool { return a.Hooks.SOC2Audit }

	// No matcher: sessions started by /clear, compaction or a fork need a
	// start record too, or their tool activity has no attributable user.
	r.Register(HookDefinition{
		Owner:           "soc2-audit",
		Event:           "SessionStart",
		Command:         soc2Cmd + " session_start",
		Timeout:         5,
		StatusMessage:   "Logging session start...",
		SandboxCategory: "generator",
		EnabledFunc:     soc2Enabled,
	})

	r.Register(HookDefinition{
		Owner:           "soc2-audit",
		Event:           "PostToolUse",
		Matcher:         "*",
		Command:         soc2Cmd + " tool_use",
		Timeout:         3,
		StatusMessage:   "Logging tool action...",
		SandboxCategory: "generator",
		EnabledFunc:     soc2Enabled,
	})

	// Failed and refused tool calls are the attempts an access-control audit
	// (CC6.x) cares about most; PostToolUse sees neither.
	r.Register(HookDefinition{
		Owner:           "soc2-audit",
		Event:           "PostToolUseFailure",
		Matcher:         "*",
		Command:         soc2Cmd + " tool_failure",
		Timeout:         3,
		StatusMessage:   "Logging failed tool action...",
		SandboxCategory: "generator",
		EnabledFunc:     soc2Enabled,
	})

	r.Register(HookDefinition{
		Owner:           "soc2-audit",
		Event:           "PermissionDenied",
		Matcher:         "*",
		Command:         soc2Cmd + " permission_denied",
		Timeout:         3,
		StatusMessage:   "Logging denied tool action...",
		SandboxCategory: "generator",
		EnabledFunc:     soc2Enabled,
	})

	r.Register(HookDefinition{
		Owner:           "soc2-audit",
		Event:           "Stop",
		Command:         soc2Cmd + " session_checkpoint",
		Timeout:         5,
		StatusMessage:   "Logging session checkpoint...",
		SandboxCategory: "generator",
		EnabledFunc:     soc2Enabled,
	})

	r.Register(HookDefinition{
		Owner:           "soc2-audit",
		Event:           "SessionEnd",
		Command:         soc2Cmd + " session_end",
		Timeout:         5,
		StatusMessage:   "Logging session end...",
		SandboxCategory: "generator",
		EnabledFunc:     soc2Enabled,
	})

	r.Register(HookDefinition{
		Owner:           "semble",
		Event:           "PostToolUse",
		Matcher:         "mcp__semble__*",
		Command:         `"${CLAUDE_PROJECT_DIR}"/.claude/hooks/semble-analytics.sh`,
		Timeout:         5,
		StatusMessage:   "Logging search analytics...",
		SandboxCategory: "generator",
		EnabledFunc:     func(a types.WizardAnswers) bool { return a.AgentTools.SembleEnabled },
	})

	r.Register(HookDefinition{
		Owner:           "audit-log",
		Event:           "PostToolUse",
		Matcher:         "*",
		Command:         `"${CLAUDE_PROJECT_DIR}"/.claude/hooks/audit-log.sh`,
		Timeout:         5,
		StatusMessage:   "Logging tool action...",
		SandboxCategory: "generator",
		EnabledFunc:     func(a types.WizardAnswers) bool { return a.Hooks.AuditLog && !a.Hooks.SOC2Audit },
	})

	enforceApp := branding.Get().AppName

	r.Register(HookDefinition{
		Owner:         "security-enforcement",
		Event:         "PreToolUse",
		Matcher:       "*",
		Command:       enforceApp + " enforce --hook PreToolUse",
		Timeout:       5,
		StatusMessage: "Evaluating security policy...",
		EnabledFunc:   func(a types.WizardAnswers) bool { return a.Hooks.SecurityEnforcement },
	})

	r.Register(HookDefinition{
		Owner:         "security-enforcement",
		Event:         "PostToolUse",
		Matcher:       "mcp__*",
		Command:       enforceApp + " enforce --hook PostToolUse",
		Timeout:       5,
		StatusMessage: "Applying MCP security hardening...",
		EnabledFunc:   func(a types.WizardAnswers) bool { return a.Hooks.SecurityEnforcement },
	})

	// lsp-first-guard redirects code-symbol Grep searches to Claude Code's LSP
	// tool (Phase 31). Registered last so it does not shift the positions of
	// the preceding security hooks. See lspGuardEnabled and lspGuardTier.
	lspGuardCmd := `"${CLAUDE_PROJECT_DIR}"/.claude/hooks/lsp-first-guard.sh`
	r.Register(HookDefinition{
		Owner:           "lsp-guard",
		Event:           "PreToolUse",
		Matcher:         "Grep",
		Command:         lspGuardCmd,
		CommandFunc:     func(a types.WizardAnswers) string { return lspGuardCmd + " " + lspGuardTier(a) },
		Timeout:         5,
		StatusMessage:   "Checking for LSP-navigable symbols...",
		SandboxCategory: "linter",
		EnabledFunc:     lspGuardEnabled,
	})

	return r
}

// lspGuardEnabled reports whether the lsp-first-guard hook is installed: LSP
// enforcement is not "off" and the tier generates the LSP plugin the guard
// redirects to (Standard+; see Generate). Below that tier the guard would deny
// Grep in favour of an LSP tool that has no servers configured.
func lspGuardEnabled(a types.WizardAnswers) bool {
	return a.LSP.EnforcementTier() != "off" && resolveTier(a) >= tier.Standard
}

// lspGuardTier is the enforcement tier passed to the hook script as its first
// argument, so the configured tier applies even when Claude runs outside the
// devenv shell that exports QSDEV_LSP_ENFORCEMENT. It is an argument rather
// than an environment prefix so the command also works when wrapped by
// "<app> sandbox exec --", which execs its arguments directly. Any value other
// than "warn" maps to "block", the script's own default, so arbitrary config
// text never reaches the shell.
func lspGuardTier(a types.WizardAnswers) string {
	if a.LSP.EnforcementTier() == "warn" {
		return "warn"
	}
	return "block"
}
