package claudesettings

import (
	"encoding/json"
	"maps"
	"slices"
	"testing"
)

const guardCommand = `"${CLAUDE_PROJECT_DIR}"/.claude/hooks/package-guard.py`

// generated mirrors the settings.json qsdev generates at the standard tier.
const generated = `{
  "permissions": {"defaultMode": "default", "disableBypassPermissionsMode": "disable", "deny": ["Bash(curl *)", 7]},
  "env": {"TOOL_GATES_DENIED": "WebFetch"},
  "hooks": {"PreToolUse": [
    {"matcher": "*", "hooks": [{"type": "command", "command": "qsdev selfprotect"}]},
    {"matcher": "Bash", "hooks": [{"type": "command", "command": "\"${CLAUDE_PROJECT_DIR}\"/.claude/hooks/package-guard.py", "if": "Bash(npm *)"}]}
  ]}
}`

func TestParse(t *testing.T) {
	t.Parallel()
	s, err := Parse([]byte(generated))
	if err != nil {
		t.Fatal(err)
	}
	if s.DefaultMode != "default" || s.DisableBypassPermissionsMode != "disable" || s.DisableAllHooks {
		t.Errorf("permissions = %q/%q/%v", s.DefaultMode, s.DisableBypassPermissionsMode, s.DisableAllHooks)
	}
	if !slices.Equal(s.Deny, []string{"Bash(curl *)"}) {
		t.Errorf("Deny = %v, want the string rules only", s.Deny)
	}
	want := []Matcher{
		{Matcher: "*", Hooks: []Hook{{Type: "command", Command: "qsdev selfprotect"}}},
		{Matcher: "Bash", Hooks: []Hook{{Type: "command", Command: guardCommand, If: "Bash(npm *)"}}},
	}
	if got := s.Hooks[EventPreToolUse]; !slices.EqualFunc(got, want, func(a, b Matcher) bool {
		return a.Matcher == b.Matcher && slices.Equal(a.Hooks, b.Hooks)
	}) {
		t.Errorf("PreToolUse = %+v, want %+v", got, want)
	}
	if _, err := Parse([]byte(`{"permissions": `)); err == nil {
		t.Error("Parse accepted truncated JSON")
	}
}

// TestParse_DecoyKeysIgnored guards the exact-key reading: encoding/json
// would match a decoy "Hooks" or "Permissions" key case-insensitively, but
// Claude Code reads only the exact spelling.
func TestParse_DecoyKeysIgnored(t *testing.T) {
	t.Parallel()
	s, err := Parse([]byte(`{"permissions": {"defaultMode": "bypassPermissions"}, "hooks": {},
  "Permissions": {"defaultMode": "default", "disableBypassPermissionsMode": "disable"},
  "DisableAllHooks": true,
  "Hooks": {"PreToolUse": [{"matcher": "*", "hooks": [{"type": "command", "command": "qsdev selfprotect"}]}]}}`))
	if err != nil {
		t.Fatal(err)
	}
	if s.DefaultMode != ModeBypassPermissions || s.DisableBypassPermissionsMode != "" || s.DisableAllHooks || len(s.Hooks) != 0 {
		t.Errorf("decoy keys were read: %+v", s)
	}
}

// TestParse_DuplicateKeyLastWins pins JSON.parse semantics: the last of
// duplicate keys is the one Claude Code sees.
func TestParse_DuplicateKeyLastWins(t *testing.T) {
	t.Parallel()
	s, err := Parse([]byte(`{"disableAllHooks": false, "disableAllHooks": true}`))
	if err != nil {
		t.Fatal(err)
	}
	if !s.DisableAllHooks {
		t.Error("DisableAllHooks = false, want the last duplicate (true)")
	}
}

func TestParse_EnvNonStringDropped(t *testing.T) {
	t.Parallel()
	s, err := Parse([]byte(`{"env": {"A": "x", "B": ["y"], "C": 1, "D": ""}}`))
	if err != nil {
		t.Fatal(err)
	}
	if want := map[string]string{"A": "x", "D": ""}; !maps.Equal(s.Env, want) {
		t.Errorf("Env = %v, want %v", s.Env, want)
	}
}

func TestRegistered(t *testing.T) {
	t.Parallel()
	s, err := Parse([]byte(generated))
	if err != nil {
		t.Fatal(err)
	}
	guard := Hook{Type: "command", Command: guardCommand, If: "Bash(npm *)"}
	tests := []struct {
		name    string
		event   string
		matcher string
		want    Hook
		ok      bool
	}{
		{"exact", EventPreToolUse, "Bash", guard, true},
		{"other matcher", EventPreToolUse, "*", guard, false},
		{"other event", "PostToolUse", "Bash", guard, false},
		{"condition differs", EventPreToolUse, "Bash", Hook{Type: "command", Command: guardCommand}, false},
		{"self-protect", EventPreToolUse, "*", Hook{Type: "command", Command: "qsdev selfprotect"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := s.Registered(tt.event, tt.matcher, tt.want); got != tt.ok {
				t.Errorf("Registered = %v, want %v", got, tt.ok)
			}
		})
	}
}

// TestRegistered_BlockingFields pins that a registration made async or given
// another timeout is not the generated one, so check reports it.
func TestRegistered_BlockingFields(t *testing.T) {
	t.Parallel()
	want := Hook{Type: HookTypeCommand, Command: guardCommand, Timeout: GuardHookTimeout}
	withKey := func(key, value string) string {
		return `{"hooks": {"PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": ` +
			jsonString(guardCommand) + `, "timeout": 30, ` + jsonString(key) + `: ` + value + `}]}]}}`
	}
	tests := []struct {
		name string
		hook Hook
		doc  string // overrides hook when set
		ok   bool
	}{
		{name: "as generated", hook: want, ok: true},
		{name: "async", hook: Hook{Command: guardCommand, Timeout: GuardHookTimeout, Async: true}},
		{name: "shorter timeout", hook: Hook{Command: guardCommand, Timeout: 1}},
		{name: "timeout removed", hook: Hook{Command: guardCommand}},
		{name: "status message", doc: withKey("statusMessage", `"Checking"`), ok: true},
		{name: "args empty", doc: withKey("args", `[]`)},
		{name: "args set", doc: withKey("args", `["--help"]`)},
		{name: "shell powershell", doc: withKey("shell", `"powershell"`)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc := tt.doc
			if doc == "" {
				doc = hooksDoc(t, "Bash", tt.hook)
			}
			s, err := Parse([]byte(doc))
			if err != nil {
				t.Fatal(err)
			}
			if got := s.Registered(EventPreToolUse, "Bash", want); got != tt.ok {
				t.Errorf("Registered = %v, want %v", got, tt.ok)
			}
		})
	}
}

// hooksDoc returns a settings document with one PreToolUse matcher entry
// whose single hook is hook (type "command" when hook.Type is empty; async
// and timeout written only when set).
func hooksDoc(t *testing.T, matcher string, hook Hook) string {
	t.Helper()
	if hook.Type == "" {
		hook.Type = HookTypeCommand
	}
	h := map[string]any{"type": hook.Type, "command": hook.Command}
	if hook.If != "" {
		h["if"] = hook.If
	}
	if hook.Async {
		h["async"] = true
	}
	if hook.Timeout != 0 {
		h["timeout"] = hook.Timeout
	}
	doc, err := json.Marshal(map[string]any{"hooks": map[string]any{
		EventPreToolUse: []any{map[string]any{"matcher": matcher, "hooks": []any{h}}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	return string(doc)
}

// jsonString returns s as a JSON string literal.
func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func TestRunsScript(t *testing.T) {
	t.Parallel()
	const (
		app       = "qsdev"
		script    = ".claude/hooks/package-guard.py"
		sandboxed = app + ` sandbox exec --category linter -- ` + guardCommand
	)
	failClosed := FailClosedCommand("package-guard", guardCommand)
	tests := []struct {
		name     string
		matcher  string
		command  string
		hook     Hook   // overrides command when Command is set
		settings string // overrides matcher/command when set
		event    string
		tool     string
		want     bool
	}{
		{name: "fixture narrowed by if", settings: generated, want: false},
		{name: "plain command", matcher: "Bash", command: guardCommand, want: true},
		{name: "plain command, other tool", matcher: "Bash", command: guardCommand, tool: "Edit", want: false},
		{name: "plain command, other event", matcher: "Bash", command: guardCommand, event: "PostToolUse", want: false},
		{name: "no hooks", settings: `{}`, want: false},
		{name: "fail-closed wrapped", matcher: "Bash", command: failClosed, want: true},
		{name: "sandbox prefixed", matcher: "Bash", command: sandboxed, want: true},
		{name: "sandbox prefixed, fail-closed", matcher: "Bash", command: FailClosedCommand("package-guard", sandboxed), want: true},
		{name: "sandbox without category", matcher: "Bash", command: app + ` sandbox exec -- ` + guardCommand, want: true},
		{name: "unquoted project dir", matcher: "Bash", command: `$CLAUDE_PROJECT_DIR/` + script, want: true},
		{name: "python3 interpreter", matcher: "Bash", command: "python3 " + guardCommand, want: true},
		{name: "python3 interpreter with plain args", matcher: "Bash", command: "python3 " + guardCommand + " --strict -c x", want: true},
		// A bare relative path is not found once the session's directory
		// changes, and "python" may be Python 2 or absent.
		{name: "relative path", matcher: "Bash", command: script, want: false},
		{name: "relative path, fail-closed wrapped", matcher: "Bash", command: FailClosedCommand("package-guard", script), want: false},
		{name: "python3 relative path", matcher: "Bash", command: "python3 " + script, want: false},
		{name: "python interpreter", matcher: "Bash", command: "python " + guardCommand, want: false},
		{name: "matcher empty", matcher: "", command: guardCommand, want: true},
		{name: "matcher star", matcher: "*", command: guardCommand, want: true},
		{name: "matcher Bash", matcher: "Bash", command: guardCommand, want: true},
		{name: "matcher alternation", matcher: "Bash|PowerShell|Monitor", command: guardCommand, want: true},
		{name: "matcher regexp", matcher: "Ba.*", command: guardCommand, want: true},
		{name: "matcher comma list", matcher: "Bash, Edit", command: guardCommand, want: true},
		{name: "matcher padded name", matcher: " Bash ", command: guardCommand, want: true},
		{name: "matcher padded list", matcher: "Edit | Bash", command: guardCommand, want: true},
		{name: "matcher hyphenated list", matcher: "Edit,code-reviewer", command: guardCommand, want: false},
		// Regexp matchers are tested unanchored, as JavaScript's RegExp.test.
		{name: "matcher regexp unanchored", matcher: "^Ba", command: guardCommand, want: true},
		{name: "matcher regexp substring", matcher: "as.", command: guardCommand, want: true},
		{name: "matcher regexp anchored other", matcher: "^Edit$", command: guardCommand, want: false},
		// Syntax RE2 accepts but JavaScript rejects or reads differently.
		{name: "matcher go flag group", matcher: "(?i)bash", command: guardCommand, want: false},
		{name: "matcher posix class", matcher: "[[:alpha:]]+", command: guardCommand, want: false},
		{name: "matcher unicode class", matcher: `\pL+`, command: guardCommand, want: false},
		{name: "matcher go end-of-text", matcher: `Bash\z`, command: guardCommand, want: false},
		{name: "matcher go begin-of-text", matcher: `\ABash`, command: guardCommand, want: false},
		{name: "matcher go hex brace", matcher: `\x{42}ash`, command: guardCommand, want: false},
		// Bracket forms RE2 and JavaScript read differently: in RE2 a "]"
		// right after "[" or "[^" is a literal and a POSIX class may appear
		// anywhere in a bracket; JavaScript reads "[]" as matching nothing,
		// "[^]" as any character and "[:upper:]" as a set of characters.
		{name: "matcher leading close bracket", matcher: "[]]?Bash", command: guardCommand, want: false},
		{name: "matcher negated leading close bracket", matcher: "[^]]?Bash", command: guardCommand, want: false},
		{name: "matcher inner posix class", matcher: "x|[_[:upper:]]ash", command: guardCommand, want: false},
		{name: "matcher trailing posix class", matcher: "[a[:alpha:]]+", command: guardCommand, want: false},
		// Brackets, escapes and braces are never credited, even in forms
		// both engines read alike: an allowlist fails safe.
		{name: "matcher plain bracket", matcher: "[B]ash", command: guardCommand, want: false},
		{name: "matcher escaped", matcher: `\w+`, command: guardCommand, want: false},
		{name: "matcher repetition braces", matcher: "Bas{1}h", command: guardCommand, want: false},
		{name: "matcher NeverMatchesAnything", matcher: "NeverMatchesAnything", command: guardCommand, want: false},
		{name: "matcher Edit", matcher: "Edit", command: guardCommand, want: false},
		{name: "matcher partial only", matcher: "Bas", command: guardCommand, want: false},
		{name: "matcher partial list", matcher: "Bas|Edit", command: guardCommand, want: false},
		{name: "matcher superstring", matcher: "BashX|Edit", command: guardCommand, want: false},
		{name: "matcher invalid regexp", matcher: "Bash(", command: guardCommand, want: false},
		{name: "echo decoy", matcher: "Bash", command: "echo " + script, want: false},
		{name: "after another command", matcher: "Bash", command: "true; " + script, want: false},
		{name: "after another script", matcher: "Bash", command: "sh .claude/hooks/a.sh && python3 " + script, want: false},
		{name: "script path as a prefix only", matcher: "Bash", command: "python3 " + script + ".bak", want: false},
		{name: "script in a sub-directory", matcher: "Bash", command: "python3 .claude/hooks/old/" + script, want: false},
		{name: "other project dir", matcher: "Bash", command: `"$HOME"/` + script, want: false},
		// The sandbox prefix runs only the app's own binary.
		{name: "echo sandbox decoy", matcher: "Bash", command: "echo sandbox exec -- " + guardCommand, want: false},
		{name: "true sandbox decoy", matcher: "Bash", command: "true sandbox exec -- " + script, want: false},
		{name: "other binary named like a path to app", matcher: "Bash", command: "/tmp/x/qsdevil sandbox exec -- " + script, want: false},
		// A tail that discards the guard's decision or exit status.
		{name: "stdout discarded", matcher: "Bash", command: guardCommand + " >/dev/null", want: false},
		{name: "stdout discarded, fail-closed", matcher: "Bash", command: FailClosedCommand("package-guard", guardCommand+" >/dev/null"), want: false},
		{name: "or true", matcher: "Bash", command: guardCommand + " >/dev/null 2>&1 || true", want: false},
		{name: "plain or true", matcher: "Bash", command: guardCommand + " || true", want: false},
		{name: "background then exit 0", matcher: "Bash", command: script + " & exit 0", want: false},
		{name: "piped", matcher: "Bash", command: guardCommand + " | cat >/dev/null", want: false},
		{name: "python args then redirect", matcher: "Bash", command: "python3 " + script + " -c x >/dev/null", want: false},
		{name: "newline then command", matcher: "Bash", command: guardCommand + "\nexit 0", want: false},
		{name: "command substitution arg", matcher: "Bash", command: guardCommand + " $(true)", want: false},
		{name: "wrapper with altered body", matcher: "Bash", command: guardCommand + ` || { rc=$?; exit 0; }`, want: false},
		// Hook fields that decide whether the command runs at all.
		{name: "narrowed by if", matcher: "Bash", hook: Hook{Command: guardCommand, If: "Bash(nonexistent-xyz:*)"}, want: false},
		{name: "prompt type", matcher: "Bash", hook: Hook{Type: "prompt", Command: guardCommand}, want: false},
		// Hook fields that decide whether its decision can block.
		{name: "async true", matcher: "Bash", hook: Hook{Command: guardCommand, Async: true}, want: false},
		{name: "async false", settings: `{"hooks": {"PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": ` + jsonString(guardCommand) + `, "async": false}]}]}}`, want: true},
		{name: "asyncRewake", settings: `{"hooks": {"PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": ` + jsonString(guardCommand) + `, "asyncRewake": true}]}]}}`, want: false},
		{name: "timeout 1", matcher: "Bash", hook: Hook{Command: guardCommand, Timeout: 1}, want: false},
		{name: "timeout 30", matcher: "Bash", hook: Hook{Command: guardCommand, Timeout: GuardHookTimeout}, want: true},
		{name: "timeout 600", matcher: "Bash", hook: Hook{Command: guardCommand, Timeout: 600}, want: true},
		{name: "timeout as string", settings: `{"hooks": {"PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": ` + jsonString(guardCommand) + `, "timeout": "30"}]}]}}`, want: false},
		// Keys that change whether or how the command runs: "args" spawns
		// the command as an executable path with no shell, "shell" picks
		// another shell, "once" runs it once per session. A key the
		// generator does not emit is never credited.
		{name: "args empty", settings: `{"hooks": {"PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": ` + jsonString(failClosed) + `, "args": []}]}]}}`, want: false},
		{name: "args set", settings: `{"hooks": {"PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": ` + jsonString(failClosed) + `, "args": ["x"]}]}]}}`, want: false},
		{name: "shell powershell", settings: `{"hooks": {"PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": ` + jsonString(failClosed) + `, "shell": "powershell"}]}]}}`, want: false},
		{name: "once", settings: `{"hooks": {"PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": ` + jsonString(failClosed) + `, "once": true}]}]}}`, want: false},
		{name: "status message", settings: `{"hooks": {"PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": ` + jsonString(failClosed) + `, "statusMessage": "Checking", "timeout": 30}]}]}}`, want: true},
		{name: "timeout zero", settings: `{"hooks": {"PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": ` + jsonString(guardCommand) + `, "timeout": 0}]}]}}`, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc := tt.settings
			if doc == "" {
				hook := tt.hook
				if hook.Command == "" {
					hook.Command = tt.command
				}
				doc = hooksDoc(t, tt.matcher, hook)
			}
			event, tool := tt.event, tt.tool
			if event == "" {
				event = EventPreToolUse
			}
			if tool == "" {
				tool = "Bash"
			}
			s, err := Parse([]byte(doc))
			if err != nil {
				t.Fatal(err)
			}
			if got := s.RunsScript(event, tool, script, app, GuardHookTimeout); got != tt.want {
				t.Errorf("RunsScript = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRunsScript_NoApp(t *testing.T) {
	t.Parallel()
	s, err := Parse([]byte(hooksDoc(t, "Bash", Hook{Command: " sandbox exec -- " + guardCommand})))
	if err != nil {
		t.Fatal(err)
	}
	if s.RunsScript(EventPreToolUse, "Bash", ".claude/hooks/package-guard.py", "", GuardHookTimeout) {
		t.Error("RunsScript credited a sandbox prefix with no app binary")
	}
}

func TestScripts(t *testing.T) {
	t.Parallel()
	s, err := Parse([]byte(`{"hooks": {
  "PreToolUse": [
    {"matcher": "*", "hooks": [{"type": "command", "command": "qsdev selfprotect"}]},
    {"matcher": "Bash", "hooks": [
      {"type": "command", "command": "\"${CLAUDE_PROJECT_DIR}\"/.claude/hooks/package-guard.py || { exit 2; }"},
      {"type": "command", "command": "sh .claude/hooks/a.sh && python3 .claude/hooks/package-guard.py"}]},
    {"matcher": "Edit", "hooks": [{"type": "command", "command": "qsdev sandbox exec -- .claude/hooks/sub/b.py"}]}
  ],
  "PostToolUse": [{"matcher": "*", "hooks": [{"type": "command", "command": ".claude/hooks/audit-log.sh"}]}]
}}`))
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		event string
		want  []string
	}{
		{EventPreToolUse, []string{".claude/hooks/package-guard.py", ".claude/hooks/a.sh", ".claude/hooks/sub/b.py"}},
		{"PostToolUse", []string{".claude/hooks/audit-log.sh"}},
		{"Stop", nil},
	}
	for _, tt := range tests {
		t.Run(tt.event, func(t *testing.T) {
			t.Parallel()
			if got := s.Scripts(tt.event); !slices.Equal(got, tt.want) {
				t.Errorf("Scripts(%q) = %q, want %q", tt.event, got, tt.want)
			}
		})
	}
}

func TestIsFailClosed(t *testing.T) {
	t.Parallel()
	tests := []struct {
		cmd  string
		want bool
	}{
		{FailClosedCommand("package-guard", `"${CLAUDE_PROJECT_DIR}"/.claude/hooks/package-guard.py`), true},
		{FailClosedCommand("self-protection", "qsdev selfprotect"), true},
		{"qsdev selfprotect", false},
		{FailClosedCommand("x", "a") + "; true", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := IsFailClosed(tt.cmd); got != tt.want {
			t.Errorf("IsFailClosed(%q) = %v, want %v", tt.cmd, got, tt.want)
		}
	}
}

// TestHoldsProjectSettings covers the paths a deletion must never reach: the
// project settings file, which registers the self-protection hook, and the
// directories holding it, in every spelling that opens it on some host.
func TestHoldsProjectSettings(t *testing.T) {
	t.Parallel()
	tests := []struct {
		rel  string
		want bool
	}{
		{ProjectRelPath, true},
		{"./.claude/settings.json", true},
		{".claude//settings.json", true},
		{".claude/x/../settings.json", true},
		{`.claude\settings.json`, true},
		{".Claude/SETTINGS.json", true},
		{".claude/settings.json.", true},
		{".claude /settings.json", true},
		{".claude/settings.json::$DATA", true},
		{".claude", true},
		{".claude/", true},
		{".", true},
		{"./", true},
		{".claude/settings.local.json", false},
		{".claude/hooks/package-guard.py", false},
		{".claude/hooks", false},
		{".claude/settings.json.bak", false},
		{"sub/.claude/settings.json", false},
		{".claudex", false},
		{"cliff.toml", false},
	}
	for _, tt := range tests {
		t.Run(tt.rel, func(t *testing.T) {
			t.Parallel()
			if got := HoldsProjectSettings(tt.rel); got != tt.want {
				t.Errorf("HoldsProjectSettings(%q) = %v, want %v", tt.rel, got, tt.want)
			}
		})
	}
}
