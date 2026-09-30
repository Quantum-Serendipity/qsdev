package claudesettings

import (
	"encoding/json"
	"regexp"
	"slices"
)

// Project-relative, slash-separated paths of the settings files Claude Code
// reads for a project: the committed one and the per-machine local override.
const (
	ProjectRelPath = ".claude/settings.json"
	LocalRelPath   = ".claude/settings.local.json"
)

// Settings keys, as Claude Code spells them. KeyDefaultMode, KeyDeny and
// KeyDisableBypassPermissionsMode live under KeyPermissions.
const (
	KeyPermissions                  = "permissions"
	KeyDeny                         = "deny"
	KeyDefaultMode                  = "defaultMode"
	KeyDisableBypassPermissionsMode = "disableBypassPermissionsMode"
	KeyDisableAllHooks              = "disableAllHooks"
	KeyHooks                        = "hooks"
	KeyEnv                          = "env"
)

// ModeBypassPermissions is the permission mode that skips every permission
// prompt.
const ModeBypassPermissions = "bypassPermissions"

// EventPreToolUse is the hook event the guard hooks (self-protection,
// package guard) are registered under.
const EventPreToolUse = "PreToolUse"

// ScriptRe matches a project hook script referenced by a hook command,
// e.g. "${CLAUDE_PROJECT_DIR}"/.claude/hooks/package-guard.py.
var ScriptRe = regexp.MustCompile(`\.claude/hooks/[A-Za-z0-9._-]+(?:/[A-Za-z0-9._-]+)*`)

// Settings is the part of a settings file that decides whether the
// generated guardrails are in force.
type Settings struct {
	Deny                         []string
	DefaultMode                  string
	DisableBypassPermissionsMode string
	DisableAllHooks              bool
	Hooks                        map[string][]Matcher
	// Env is the "env" object; a value that is not a string is kept as
	// absent, since it cannot be the policy qsdev generated.
	Env map[string]string
}

// Matcher is one matcher entry of a hook event.
type Matcher struct {
	Matcher string
	Hooks   []Hook
}

// Hook is one registered hook. If is the optional permission-rule condition
// that limits when the hook runs.
type Hook struct {
	Type    string
	Command string
	If      string
}

// Parse reads the posture keys of a settings document. Keys are matched
// exactly, as Claude Code reads them: encoding/json matches struct fields
// case-insensitively, so a decoy "Hooks" or "Permissions" key next to the
// real one could otherwise decide what qsdev sees. Duplicate keys resolve to
// the last value, as in JavaScript's JSON.parse. A syntax error is returned
// as encoding/json reports it; callers add the file name.
func Parse(data []byte) (Settings, error) {
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return Settings{}, err
	}
	var s Settings
	perms, _ := root[KeyPermissions].(map[string]any)
	s.DefaultMode, _ = perms[KeyDefaultMode].(string)
	s.DisableBypassPermissionsMode, _ = perms[KeyDisableBypassPermissionsMode].(string)
	s.DisableAllHooks, _ = root[KeyDisableAllHooks].(bool)
	deny, _ := perms[KeyDeny].([]any)
	for _, rule := range deny {
		if r, ok := rule.(string); ok {
			s.Deny = append(s.Deny, r)
		}
	}

	env, _ := root[KeyEnv].(map[string]any)
	s.Env = make(map[string]string, len(env))
	for k, v := range env {
		if str, ok := v.(string); ok {
			s.Env[k] = str
		}
	}

	events, _ := root[KeyHooks].(map[string]any)
	s.Hooks = make(map[string][]Matcher, len(events))
	for event, v := range events {
		entries, _ := v.([]any)
		for _, e := range entries {
			s.Hooks[event] = append(s.Hooks[event], parseMatcher(e))
		}
	}
	return s, nil
}

func parseMatcher(v any) Matcher {
	em, _ := v.(map[string]any)
	var m Matcher
	m.Matcher, _ = em["matcher"].(string)
	hooks, _ := em["hooks"].([]any)
	for _, h := range hooks {
		hm, _ := h.(map[string]any)
		var he Hook
		he.Type, _ = hm["type"].(string)
		he.Command, _ = hm["command"].(string)
		he.If, _ = hm["if"].(string)
		m.Hooks = append(m.Hooks, he)
	}
	return m
}

// Registered reports whether want is registered for event under matcher with
// the same type and run condition (an added "if" narrows when a guard runs).
func (s Settings) Registered(event, matcher string, want Hook) bool {
	for _, m := range s.Hooks[event] {
		if m.Matcher == matcher && slices.Contains(m.Hooks, want) {
			return true
		}
	}
	return false
}

// RunsScript reports whether a hook registered for event runs the project
// hook script at the slash-separated path script. The script must be a whole
// path token of the command (see ScriptRe), not merely a substring of it.
func (s Settings) RunsScript(event, script string) bool {
	for _, m := range s.Hooks[event] {
		for _, h := range m.Hooks {
			if slices.Contains(ScriptRe.FindAllString(h.Command, -1), script) {
				return true
			}
		}
	}
	return false
}
