package claudesettings

import (
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

func TestRunsScript(t *testing.T) {
	t.Parallel()
	const script = ".claude/hooks/package-guard.py"
	tests := []struct {
		name     string
		settings string
		event    string
		want     bool
	}{
		{"registered", generated, EventPreToolUse, true},
		{"other event", generated, "PostToolUse", false},
		{"no hooks", `{}`, EventPreToolUse, false},
		{"script path as a prefix only", `{"hooks": {"PreToolUse": [{"matcher": "Bash", "hooks": [
			{"type": "command", "command": "python3 .claude/hooks/package-guard.py.bak"}]}]}}`, EventPreToolUse, false},
		{"script in a sub-directory", `{"hooks": {"PreToolUse": [{"matcher": "Bash", "hooks": [
			{"type": "command", "command": "python3 .claude/hooks/old/.claude/hooks/package-guard.py"}]}]}}`, EventPreToolUse, false},
		{"one of several scripts", `{"hooks": {"PreToolUse": [{"matcher": "Bash", "hooks": [
			{"type": "command", "command": "sh .claude/hooks/a.sh && python3 .claude/hooks/package-guard.py"}]}]}}`, EventPreToolUse, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s, err := Parse([]byte(tt.settings))
			if err != nil {
				t.Fatal(err)
			}
			if got := s.RunsScript(tt.event, script); got != tt.want {
				t.Errorf("RunsScript = %v, want %v", got, tt.want)
			}
		})
	}
}
