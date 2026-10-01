package claudesettings

import (
	"encoding/json"
	"regexp"
	"slices"
	"strings"
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

// HookTypeCommand is the hook type that runs its command through a shell.
const HookTypeCommand = "command"

// GuardHookTimeout is the timeout, in seconds, the generator emits for the
// package guard's hook: the budget its registry lookups need. Claude Code
// cancels a hook at its timeout and a cancelled hook decides nothing, so a
// registration with a shorter one is not credited as running the guard.
const GuardHookTimeout = 30

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
// that limits when the hook runs. Async is set by "async" or "asyncRewake":
// such a hook runs in the background and cannot block the tool call. Timeout
// is the "timeout" in seconds, 0 when absent (Claude Code's default then
// applies); a value that is not a positive number reads as -1.
type Hook struct {
	Type    string
	Command string
	If      string
	Async   bool
	Timeout float64
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
		he.Async = isSet(hm, "async") || isSet(hm, "asyncRewake")
		if v, ok := hm["timeout"]; ok {
			he.Timeout = -1
			if n, isNum := v.(float64); isNum && n > 0 {
				he.Timeout = n
			}
		}
		m.Hooks = append(m.Hooks, he)
	}
	return m
}

// isSet reports whether key is present in m with any value other than false.
func isSet(m map[string]any, key string) bool {
	v, ok := m[key]
	return ok && v != false
}

// Registered reports whether want is registered for event under matcher with
// the same type, run condition (an added "if" narrows when a guard runs),
// async mode and timeout.
func (s Settings) Registered(event, matcher string, want Hook) bool {
	for _, m := range s.Hooks[event] {
		if m.Matcher == matcher && slices.Contains(m.Hooks, want) {
			return true
		}
	}
	return false
}

// ScriptRef is one project hook script (see ScriptRe) referenced by a
// registered hook command.
type ScriptRef struct {
	Command string
	Script  string
}

// ScriptRefs returns every project hook script that a hook command registered
// for event references, with that command, in registration order.
func (s Settings) ScriptRefs(event string) []ScriptRef {
	var refs []ScriptRef
	for _, m := range s.Hooks[event] {
		for _, h := range m.Hooks {
			for _, script := range ScriptRe.FindAllString(h.Command, -1) {
				refs = append(refs, ScriptRef{Command: h.Command, Script: script})
			}
		}
	}
	return refs
}

// Scripts returns the project hook scripts that the hook commands registered
// for event reference, each once, in registration order.
func (s Settings) Scripts(event string) []string {
	var scripts []string
	for _, ref := range s.ScriptRefs(event) {
		if !slices.Contains(scripts, ref.Script) {
			scripts = append(scripts, ref.Script)
		}
	}
	return scripts
}

// FailClosedCommand wraps a hook command so that any exit other than 0 or 2
// becomes 2, a block. Claude Code treats every other code (1 for a crash, 126
// or 127 for a missing interpreter or binary) as a non-blocking error, which
// would let the tool call through unchecked. Exit 0 passes through with its
// stdout, and exit 2 keeps the hook's own stderr reason. The wrapper uses only
// POSIX sh: settings.json is shared across the team's OSes, and Claude Code
// runs hooks through sh (Git Bash on Windows). cmd must be one simple command:
// the exit status of a list or pipeline is only its last part's, so an
// earlier part's failure could not be caught here. owner names the hook in
// the block message.
func FailClosedCommand(owner, cmd string) string {
	return cmd + ` || { rc=$?; [ "$rc" -eq 2 ] || echo "qsdev: ` + owner +
		` hook could not run (exit $rc: interpreter/binary missing or crashed); blocking" >&2; exit 2; }`
}

// failClosedRe matches the suffix FailClosedCommand appends, for any owner
// name.
var failClosedRe = func() string {
	const owner = "\x00owner\x00"
	return strings.Replace(regexp.QuoteMeta(FailClosedCommand(owner, "")), regexp.QuoteMeta(owner), `[A-Za-z0-9._-]+`, 1)
}()

// programRe returns a pattern matching a hook command whose program is a
// project hook script, capturing the script. Only the forms qsdev emits are
// accepted: the script, addressed through the project-dir variable, may
// follow app's sandbox prefix ("<app> sandbox exec [--category X] --") and a
// python3 interpreter, and be followed by plain arguments (no redirection,
// list, pipeline, substitution or quoting) and the FailClosedCommand suffix.
// Anything else could discard the script's decision or exit status, or, as a
// bare relative path does once the session's directory changes, fail to find
// the script at all.
func programRe(app string) *regexp.Regexp {
	const (
		sp  = `[ \t]+`
		arg = "[^\\s<>&;|`$(){}'\"\\\\#]+"
	)
	sandbox := ""
	if app != "" {
		sandbox = `(?:` + regexp.QuoteMeta(app) + sp + `sandbox` + sp + `exec(?:` + sp + `--category` + sp + `[A-Za-z0-9._-]+)?` + sp + `--` + sp + `)?`
	}
	return regexp.MustCompile(`^[ \t]*` + sandbox +
		`(?:python3` + sp + `)?` +
		`(?:"\$\{CLAUDE_PROJECT_DIR\}"/|"\$CLAUDE_PROJECT_DIR"/|\$\{CLAUDE_PROJECT_DIR\}/|\$CLAUDE_PROJECT_DIR/)` +
		`(` + ScriptRe.String() + `)` +
		`(?:` + sp + arg + `)*` +
		`(?:` + failClosedRe + `)?[ \t]*$`)
}

// RunsScript reports whether a hook registered for event, under a matcher
// that covers tool, unconditionally runs the project hook script at the
// slash-separated path script as its program (see programRe), with app as
// the binary of an optional sandbox prefix. Only a hook that can block is
// credited: of type "command", with no "if" condition narrowing when it
// runs, not async, and with a timeout that is absent or at least minTimeout
// seconds.
func (s Settings) RunsScript(event, tool, script, app string, minTimeout float64) bool {
	re := programRe(app)
	for _, m := range s.Hooks[event] {
		if !matcherCovers(m.Matcher, tool) {
			continue
		}
		for _, h := range m.Hooks {
			if !h.canBlock(minTimeout) {
				continue
			}
			if sm := re.FindStringSubmatch(h.Command); sm != nil && sm[1] == script {
				return true
			}
		}
	}
	return false
}

// canBlock reports whether h runs in the foreground, every time its matcher
// fires, for at least minTimeout seconds: the conditions under which its
// decision reaches Claude Code.
func (h Hook) canBlock(minTimeout float64) bool {
	return h.Type == HookTypeCommand && h.If == "" && !h.Async &&
		(h.Timeout == 0 || h.Timeout >= minTimeout)
}

var (
	// exactMatcherRe matches a matcher Claude Code compares as an exact tool
	// name, or a list of them separated by "|" or ",".
	exactMatcherRe = regexp.MustCompile(`^[A-Za-z0-9_\-, |]+$`)
	// divergentSyntaxRe matches regexp syntax that Go's RE2 accepts but
	// JavaScript rejects or reads differently: flag and named groups, POSIX
	// classes, Unicode classes, Go-only escapes and octal escapes.
	divergentSyntaxRe = regexp.MustCompile(`\(\?|\[\[:|\\[pPzAQC0-9]|\\x\{`)
)

// matcherCovers reports whether a hook matcher selects tool under Claude
// Code's semantics: "" and "*" match every tool; a matcher of only letters,
// digits, "_", "-", spaces, "," and "|" is a list of exact names separated by
// "|" or ","; anything else is a JavaScript regular expression tested
// unanchored. A pattern Go would read differently from JavaScript, or cannot
// compile, matches nothing, so it is never credited.
func matcherCovers(matcher, tool string) bool {
	if matcher == "" || matcher == "*" {
		return true
	}
	if exactMatcherRe.MatchString(matcher) {
		for _, name := range strings.FieldsFunc(matcher, func(r rune) bool { return r == '|' || r == ',' }) {
			if strings.TrimSpace(name) == tool {
				return true
			}
		}
		return false
	}
	if divergentSyntaxRe.MatchString(matcher) {
		return false
	}
	re, err := regexp.Compile(matcher)
	return err == nil && re.MatchString(tool)
}
