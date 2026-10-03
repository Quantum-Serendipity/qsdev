package claudesettings

import (
	"slices"
	"strings"
	"testing"
)

// TestParse_Unloadable pins which hook entries make Claude Code refuse the
// whole settings file (a PreToolUse or PermissionRequest entry it cannot
// load, or guard hooks where it expects something else) and which it only
// drops.
func TestParse_Unloadable(t *testing.T) {
	t.Parallel()
	const guard = `{"matcher": "Bash", "hooks": [{"type": "command", "command": "g", "timeout": 30}]}`
	withHooks := func(hooks string) string { return `{"permissions": {"deny": ["Read(.env)"]}, "hooks": ` + hooks + `}` }
	preToolUse := func(entry string) string { return withHooks(`{"PreToolUse": [` + guard + `, ` + entry + `]}`) }
	tests := []struct {
		name string
		doc  string
		want string // substring of Unloadable; "" means loadable
	}{
		{"generated", generated, ""},
		{"no hooks", `{"permissions": {}}`, ""},
		{"empty events", withHooks(`{}`), ""},
		{"hooks null", withHooks(`null`), ""},
		{"hooks a string", withHooks(`"x"`), ""},
		{"prompt hook", preToolUse(`{"hooks": [{"type": "prompt", "prompt": "ok?", "model": "m", "timeout": 5}]}`), ""},
		{"agent hook", preToolUse(`{"hooks": [{"type": "agent", "prompt": "verify"}]}`), ""},
		{"http hook", preToolUse(`{"hooks": [{"type": "http", "url": "https://example.test/h", "headers": {"A": "b"}}]}`), ""},
		{"mcp_tool hook", preToolUse(`{"hooks": [{"type": "mcp_tool", "server": "s", "tool": "t", "input": {"x": 1}}]}`), ""},
		{"unknown key accepted", preToolUse(`{"hooks": [{"type": "command", "command": "x", "future": 1}]}`), ""},
		{"cloud read leniently", preToolUse(`{"hooks": [{"type": "command", "command": "x", "cloud": 5}]}`), ""},
		{"matcher omitted", preToolUse(`{"hooks": []}`), ""},
		{"guard event null", withHooks(`{"PreToolUse": null}`), ""},
		{"bad entry of another event is dropped", withHooks(`{"PostToolUse": [5, {"hooks": [{"type": "bogus"}]}]}`), ""},
		{"another event not an array is dropped", withHooks(`{"Stop": {"hooks": 1}}`), ""},
		{"unknown event is dropped", withHooks(`{"Bogus": [{"hooks": [{"type": "bogus"}]}]}`), ""},

		{"hook if number", preToolUse(`{"hooks": [{"type": "command", "command": "true", "if": 1}]}`), "hooks.PreToolUse.1: hooks.0: command hook has an invalid \"if\""},
		{"hook timeout string", preToolUse(`{"hooks": [{"type": "command", "command": "true", "timeout": "x"}]}`), `invalid "timeout"`},
		{"hook timeout zero", preToolUse(`{"hooks": [{"type": "command", "command": "true", "timeout": 0}]}`), `invalid "timeout"`},
		{"hook once string", preToolUse(`{"hooks": [{"type": "command", "command": "true", "once": "yes"}]}`), `invalid "once"`},
		{"hook shell unknown", preToolUse(`{"hooks": [{"type": "command", "command": "true", "shell": "zsh"}]}`), `invalid "shell"`},
		{"hook args not strings", preToolUse(`{"hooks": [{"type": "command", "command": "true", "args": [1]}]}`), `invalid "args"`},
		{"hook rewake message empty", preToolUse(`{"hooks": [{"type": "command", "command": "true", "rewakeMessage": ""}]}`), `invalid "rewakeMessage"`},
		{"hook command missing", preToolUse(`{"hooks": [{"type": "command"}]}`), `has no "command"`},
		{"prompt hook without prompt", preToolUse(`{"hooks": [{"type": "prompt"}]}`), `has no "prompt"`},
		{"http hook relative url", preToolUse(`{"hooks": [{"type": "http", "url": "/h"}]}`), `invalid "url"`},
		{"hook not an object", preToolUse(`{"hooks": [5]}`), "not an object"},
		{"hook type unknown", preToolUse(`{"hooks": [{"type": "bogus"}]}`), `unknown hook type "bogus"`},
		{"hook type missing", preToolUse(`{"hooks": [{"command": "x"}]}`), `no string "type"`},
		{"hooks not an array", preToolUse(`{"hooks": {"type": "command", "command": "x"}}`), `"hooks" is not an array`},
		{"hooks missing", preToolUse(`{"matcher": "Write"}`), `"hooks" is not an array`},
		{"matcher not a string", preToolUse(`{"matcher": 5, "hooks": []}`), `"matcher" is not a string`},
		{"matcher null", preToolUse(`{"matcher": null, "hooks": []}`), `"matcher" is not a string`},
		{"entry not an object", preToolUse(`"Write"`), "matcher entry is not an object"},
		{"permission request entry", withHooks(`{"PermissionRequest": [{"hooks": [{"type": "bogus"}]}]}`), "hooks.PermissionRequest.0"},
		{"guard event not an array", withHooks(`{"PreToolUse": {"matcher": "Bash"}}`), "hooks.PreToolUse is not an array"},
		{"permission request a string", withHooks(`{"PermissionRequest": "x"}`), "hooks.PermissionRequest is not an array"},
		{"guard hooks nested in another event", withHooks(`{"Stop": [{"hooks": [], "PreToolUse": [{"hooks": []}]}]}`), "where a matcher was expected"},
		{"guard hooks under an unknown event", withHooks(`{"pretooluse": [{"PreToolUse": [1]}]}`), "hooks.pretooluse is not a hook event"},
		{"single matcher under an unknown event", withHooks(`{"Pretooluse": {"matcher": "Bash", "hooks": [{"type": "command", "command": "g"}]}}`), "hooks.Pretooluse is not a hook event"},
		{"hooks is a single matcher", withHooks(guard), `"hooks" is a single matcher`},
		{"hooks is an array of matchers", withHooks(`[` + guard + `]`), `"hooks" is an array of matchers`},

		// Guard hooks declared outside "hooks" refuse the file too.
		{"top-level guard event", `{"PreToolUse": true, "hooks": {}}`, "at the top level"},
		{"top-level guard matchers", `{"PreToolUse": [` + guard + `]}`, "at the top level"},
		{"top-level permission request", `{"PermissionRequest": [1]}`, "at the top level"},
		{"guard event under permissions", `{"permissions": {"deny": [], "PreToolUse": [1]}}`, "permissions holds"},
		{"matcher under another key", `{"x": {"hooks": [1]}}`, "x holds"},
		{"guard event three levels down", `{"x": {"a": {"b": {"PreToolUse": [1]}}}}`, "x holds"},
		{"guard event under an event-named key", `{"Stop": {"PreToolUse": [1]}}`, "Stop holds"},
		{"matcher under an event-named key does not count", `{"Stop": {"hooks": [1]}}`, ""},
		{"guard event four levels down", `{"x": {"a": {"b": {"c": {"PreToolUse": [1]}}}}}`, "x holds"},
		{"guard event five levels down is not searched", `{"x": {"a": {"b": {"c": {"d": {"PreToolUse": [1]}}}}}}`, ""},
		{"empty guard event at top level", `{"PreToolUse": [], "PermissionRequest": null}`, ""},
		{"exempt env", `{"env": {"PreToolUse": "1"}}`, ""},
		{"exempt mcpServers", `{"mcpServers": {"s": {"PreToolUse": [1], "hooks": [1]}}}`, ""},
		{"exempt key nested elsewhere", `{"x": {"env": {"PreToolUse": [1]}}}`, ""},
		{"isolation is dropped first", `{"isolation": {"PreToolUse": [1]}}`, ""},
		{"additionalMarketplaces folds into an exempt key", `{"additionalMarketplaces": {"m": {"PreToolUse": [1]}}}`, ""},
		{"non-string permission rules are dropped first", `{"permissions": {"deny": [{"PreToolUse": [1]}], "allow": [{"hooks": [1]}]}}`, ""},
		{"permission rules not a list are refused", `{"permissions": {"ask": {"PreToolUse": [1]}}}`, "permissions."},

		// A posture key of a type Claude Code's schema rejects refuses the
		// file rather than reading as the policy it seems to set.
		{"disableAllHooks string", `{"disableAllHooks": "false"}`, `"disableAllHooks" is not a boolean`},
		{"disableAllHooks null", `{"disableAllHooks": null}`, `"disableAllHooks" is not a boolean`},
		{"env an array", `{"env": ["A=b"]}`, `"env" is not an object`},
		{"permissions an array", `{"permissions": []}`, `"permissions" is not an object`},
		{"deny not an array", `{"permissions": {"deny": "Bash(curl *)"}}`, "permissions.deny is not an array"},
		{"allow not an array", `{"permissions": {"allow": {}}}`, "permissions.allow is not an array"},
		{"defaultMode not a string", `{"permissions": {"defaultMode": 1}}`, "permissions.defaultMode is not a string"},
		{"disableBypass other value", `{"permissions": {"disableBypassPermissionsMode": "enable"}}`, `permissions.disableBypassPermissionsMode is not "disable"`},
		{"disableBypass boolean", `{"permissions": {"disableBypassPermissionsMode": true}}`, `is not "disable"`},
		{"posture keys well typed", `{"disableAllHooks": false, "env": {"A": "b"}, "permissions": {"allow": [], "deny": [5], "ask": [], "defaultMode": "manual", "disableBypassPermissionsMode": "disable"}}`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s, err := Parse([]byte(tt.doc))
			if err != nil {
				t.Fatal(err)
			}
			switch {
			case tt.want == "" && s.Unloadable != "":
				t.Errorf("Unloadable = %q, want the file loadable", s.Unloadable)
			case tt.want != "" && !strings.Contains(s.Unloadable, tt.want):
				t.Errorf("Unloadable = %q, want it to contain %q", s.Unloadable, tt.want)
			}
		})
	}
}

// TestRead_UnloadableFileContributesNothing pins that a settings file Claude
// Code refuses to load adds neither hooks nor deny rules nor env to the
// effective view, while the other files still do.
func TestRead_UnloadableFileContributesNothing(t *testing.T) {
	t.Parallel()
	const bad = `{"hooks": {"PreToolUse": [{"hooks": [{"type": "bogus"}]}]}, "permissions": {"deny": ["Read(local)"]}, "env": {"A": "b"}, "disableAllHooks": true}`
	t.Run("project", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		writeSettings(t, root, ProjectRelPath, strings.Replace(generated, `"hooks": {`, `"hooks": {"PermissionRequest": 5, `, 1))
		e, err := Read(root)
		if err != nil {
			t.Fatal(err)
		}
		if e.Project == nil || e.Project.Unloadable == "" {
			t.Fatalf("Project = %+v, want it parsed and marked unloadable", e.Project)
		}
		if len(e.Hooks) != 0 || len(e.Deny) != 0 || len(e.Env) != 0 {
			t.Errorf("merged view = hooks %v, deny %v, env %v, want nothing from the unloadable file", e.Hooks, e.Deny, e.Env)
		}
	})
	t.Run("local", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		writeSettings(t, root, ProjectRelPath, generated)
		writeSettings(t, root, LocalRelPath, bad)
		e, err := Read(root)
		if err != nil {
			t.Fatal(err)
		}
		if e.Local == nil || e.Local.Unloadable == "" {
			t.Fatalf("Local = %+v, want it parsed and marked unloadable", e.Local)
		}
		if e.DisableAllHooks || slices.Contains(e.Deny, "Read(local)") || e.Env["A"] != "" {
			t.Errorf("merged view took settings from the unloadable local file: %+v", e.Settings)
		}
		if !slices.Equal(e.Deny, []string{"Bash(curl *)"}) || len(e.Hooks[EventPreToolUse]) != 2 {
			t.Errorf("merged view = deny %v, hooks %v, want the committed file's", e.Deny, e.Hooks)
		}
	})
}
