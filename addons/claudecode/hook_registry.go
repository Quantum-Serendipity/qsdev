package claudecode

import "github.com/Quantum-Serendipity/qsdev/pkg/types"

// HookDeploymentTier controls which settings file a hook is generated into.
type HookDeploymentTier int

const (
	// TierProject deploys to .claude/settings.json (project-level).
	TierProject HookDeploymentTier = iota
	// TierTeam deploys to ~/.claude/settings.json (user-level).
	TierTeam
	// TierOrg deploys to /etc/claude-code/managed-settings.json (org-level, non-overridable).
	TierOrg
)

func (t HookDeploymentTier) String() string {
	switch t {
	case TierProject:
		return "project"
	case TierTeam:
		return "team"
	case TierOrg:
		return "org"
	default:
		return "unknown"
	}
}

// HookDefinition describes a hook that can be registered with the HookRegistry.
// Each definition maps to one HookMatcher entry in the generated settings.json.
type HookDefinition struct {
	Owner           string
	Event           string
	Matcher         string
	Command         string
	Timeout         int
	StatusMessage   string
	Tier            HookDeploymentTier
	SandboxCategory string // sandbox permission profile (e.g., "linter", "generator")
	EnabledFunc     func(types.WizardAnswers) bool
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
	return r.hooksForEventFiltered(event, answers, nil)
}

// hooksForEventFiltered returns matchers for an event, optionally restricted to
// a specific deployment tier. When tier is nil all tiers are included.
func (r *HookRegistry) hooksForEventFiltered(event string, answers types.WizardAnswers, tier *HookDeploymentTier) []HookMatcher {
	var matchers []HookMatcher
	for _, h := range r.hooks {
		if h.Event != event {
			continue
		}
		if tier != nil && h.Tier != *tier {
			continue
		}
		if h.EnabledFunc != nil && !h.EnabledFunc(answers) {
			continue
		}
		matchers = append(matchers, HookMatcher{
			Matcher: h.Matcher,
			Hooks: []HookEntry{{
				Type:          "command",
				Command:       h.Command,
				Timeout:       h.Timeout,
				StatusMessage: h.StatusMessage,
			}},
		})
	}
	return matchers
}

// BuildHooksMap evaluates all registered hooks against the provided answers and
// returns the complete hooks map keyed by event name, ready for SettingsJSON.
// Returns nil when no hooks are enabled.
func (r *HookRegistry) BuildHooksMap(answers types.WizardAnswers) map[string][]HookMatcher {
	return r.BuildHooksMapForTier(answers, nil)
}

// BuildHooksMapForTier evaluates hooks filtered to a specific deployment tier.
// Pass nil to include all tiers.
func (r *HookRegistry) BuildHooksMapForTier(answers types.WizardAnswers, tier *HookDeploymentTier) map[string][]HookMatcher {
	hooks := make(map[string][]HookMatcher)
	seen := make(map[string]bool)
	for _, h := range r.hooks {
		if !seen[h.Event] {
			seen[h.Event] = true
		}
	}
	for event := range seen {
		if matchers := r.hooksForEventFiltered(event, answers, tier); len(matchers) > 0 {
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

// defaultHookRegistry returns a registry pre-populated with the built-in hooks
// (package-guard and audit-log).
func defaultHookRegistry() *HookRegistry {
	r := NewHookRegistry()

	guardCmd := `"${CLAUDE_PROJECT_DIR}"/.claude/hooks/package-guard.py`

	r.Register(HookDefinition{
		Owner:           "package-guard",
		Event:           "PreToolUse",
		Matcher:         "Bash",
		Command:         guardCmd,
		Timeout:         30,
		StatusMessage:   "Checking package install safety...",
		SandboxCategory: "linter",
		EnabledFunc:     func(a types.WizardAnswers) bool { return a.Hooks.SafetyBlock },
	})

	r.Register(HookDefinition{
		Owner:           "credential-scan",
		Event:           "PreToolUse",
		Matcher:         "Write|Edit|MultiEdit",
		Command:         `"${CLAUDE_PROJECT_DIR}"/.claude/hooks/scan-secrets.py`,
		Timeout:         10,
		StatusMessage:   "Scanning for credentials...",
		SandboxCategory: "linter",
		EnabledFunc:     func(a types.WizardAnswers) bool { return a.Hooks.CredentialScan },
	})

	r.Register(HookDefinition{
		Owner:           "destructive-prevention",
		Event:           "PreToolUse",
		Matcher:         "Bash",
		Command:         `"${CLAUDE_PROJECT_DIR}"/.claude/hooks/block-destructive.py`,
		Timeout:         5,
		StatusMessage:   "Checking command safety...",
		SandboxCategory: "linter",
		EnabledFunc:     func(a types.WizardAnswers) bool { return a.Hooks.DestructivePrevention },
	})

	r.Register(HookDefinition{
		Owner:           "file-boundary",
		Event:           "PreToolUse",
		Matcher:         "Write|Edit|Read",
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
	})

	soc2Cmd := `"${CLAUDE_PROJECT_DIR}"/.claude/hooks/soc2-audit-log.py`
	soc2Enabled := func(a types.WizardAnswers) bool { return a.Hooks.SOC2Audit }

	r.Register(HookDefinition{
		Owner:           "soc2-audit",
		Event:           "SessionStart",
		Matcher:         "startup|resume",
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

	return r
}
