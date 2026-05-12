package merge

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMergeSettings_AllUnmodified(t *testing.T) {
	base := []byte(`{
  "permissions": {
    "allow": ["Read(*)"],
    "deny": ["Bash(npm install *)"]
  }
}`)
	theirs := []byte(`{
  "permissions": {
    "allow": ["Read(*)"],
    "deny": ["Bash(npm install *)"]
  }
}`)
	ours := []byte(`{
  "permissions": {
    "allow": ["Read(*)", "Edit(*)"],
    "deny": ["Bash(npm install *)", "Bash(pip install *)"]
  }
}`)

	got, err := MergeSettings(base, theirs, ours)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed settingsJSON
	if err := json.Unmarshal(got, &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	wantAllow := []string{"Read(*)", "Edit(*)"}
	wantDeny := []string{"Bash(npm install *)", "Bash(pip install *)"}
	assertStringSlice(t, "allow", parsed.Permissions.Allow, wantAllow)
	assertStringSlice(t, "deny", parsed.Permissions.Deny, wantDeny)
}

func TestMergeSettings_UserAddedAllowRule(t *testing.T) {
	base := []byte(`{
  "permissions": {
    "allow": ["Read(*)"],
    "deny": ["Bash(npm install *)"]
  }
}`)
	theirs := []byte(`{
  "permissions": {
    "allow": ["Read(*)", "Bash(my-tool *)"],
    "deny": ["Bash(npm install *)"]
  }
}`)
	ours := []byte(`{
  "permissions": {
    "allow": ["Read(*)", "Edit(*)"],
    "deny": ["Bash(npm install *)", "Bash(pip install *)"]
  }
}`)

	got, err := MergeSettings(base, theirs, ours)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed settingsJSON
	if err := json.Unmarshal(got, &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	// ours allow + user-added "Bash(my-tool *)"
	wantAllow := []string{"Read(*)", "Edit(*)", "Bash(my-tool *)"}
	assertStringSlice(t, "allow", parsed.Permissions.Allow, wantAllow)
}

func TestMergeSettings_UserAddedDenyRule(t *testing.T) {
	base := []byte(`{
  "permissions": {
    "allow": ["Read(*)"],
    "deny": ["Bash(npm install *)"]
  }
}`)
	theirs := []byte(`{
  "permissions": {
    "allow": ["Read(*)"],
    "deny": ["Bash(npm install *)", "Bash(curl *)"]
  }
}`)
	ours := []byte(`{
  "permissions": {
    "allow": ["Read(*)"],
    "deny": ["Bash(npm install *)", "Bash(pip install *)"]
  }
}`)

	got, err := MergeSettings(base, theirs, ours)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed settingsJSON
	if err := json.Unmarshal(got, &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	// ours deny + user-added "Bash(curl *)"
	wantDeny := []string{"Bash(npm install *)", "Bash(pip install *)", "Bash(curl *)"}
	assertStringSlice(t, "deny", parsed.Permissions.Deny, wantDeny)
}

func TestMergeSettings_GeneratedRuleAdded(t *testing.T) {
	base := []byte(`{
  "permissions": {
    "allow": ["Read(*)"],
    "deny": []
  }
}`)
	theirs := []byte(`{
  "permissions": {
    "allow": ["Read(*)"],
    "deny": []
  }
}`)
	ours := []byte(`{
  "permissions": {
    "allow": ["Read(*)", "Edit(*)"],
    "deny": ["Bash(npm install *)"]
  }
}`)

	got, err := MergeSettings(base, theirs, ours)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed settingsJSON
	if err := json.Unmarshal(got, &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	wantAllow := []string{"Read(*)", "Edit(*)"}
	wantDeny := []string{"Bash(npm install *)"}
	assertStringSlice(t, "allow", parsed.Permissions.Allow, wantAllow)
	assertStringSlice(t, "deny", parsed.Permissions.Deny, wantDeny)
}

func TestMergeSettings_GeneratedRuleRemoved(t *testing.T) {
	base := []byte(`{
  "permissions": {
    "allow": ["Read(*)", "Bash(make *)"],
    "deny": []
  }
}`)
	theirs := []byte(`{
  "permissions": {
    "allow": ["Read(*)", "Bash(make *)"],
    "deny": []
  }
}`)
	// ours removed "Bash(make *)"
	ours := []byte(`{
  "permissions": {
    "allow": ["Read(*)"],
    "deny": []
  }
}`)

	got, err := MergeSettings(base, theirs, ours)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed settingsJSON
	if err := json.Unmarshal(got, &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	// "Bash(make *)" was in base, not in ours, user didn't add it → removed.
	wantAllow := []string{"Read(*)"}
	assertStringSlice(t, "allow", parsed.Permissions.Allow, wantAllow)
}

func TestMergeSettings_GeneratedRuleRemovedButUserAlsoAdded(t *testing.T) {
	base := []byte(`{
  "permissions": {
    "allow": ["Read(*)"],
    "deny": []
  }
}`)
	// User independently added "Bash(make *)" (it wasn't in base)
	theirs := []byte(`{
  "permissions": {
    "allow": ["Read(*)", "Bash(make *)"],
    "deny": []
  }
}`)
	// ours doesn't have "Bash(make *)" either
	ours := []byte(`{
  "permissions": {
    "allow": ["Read(*)", "Edit(*)"],
    "deny": []
  }
}`)

	got, err := MergeSettings(base, theirs, ours)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed settingsJSON
	if err := json.Unmarshal(got, &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	// "Bash(make *)" was NOT in base → user added → preserved.
	wantAllow := []string{"Read(*)", "Edit(*)", "Bash(make *)"}
	assertStringSlice(t, "allow", parsed.Permissions.Allow, wantAllow)
}

func TestMergeSettings_UserAddedHook(t *testing.T) {
	base := []byte(`{
  "permissions": {
    "allow": ["Read(*)"],
    "deny": []
  },
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [{"type": "command", "command": "check-bash.sh"}]
      }
    ]
  }
}`)
	// User added a new matcher to PreToolUse and a new event.
	theirs := []byte(`{
  "permissions": {
    "allow": ["Read(*)"],
    "deny": []
  },
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [{"type": "command", "command": "check-bash.sh"}]
      },
      {
        "matcher": "Edit",
        "hooks": [{"type": "command", "command": "my-edit-check.sh"}]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "Bash",
        "hooks": [{"type": "command", "command": "log-bash.sh"}]
      }
    ]
  }
}`)
	// ours updates the existing generated hook.
	ours := []byte(`{
  "permissions": {
    "allow": ["Read(*)"],
    "deny": []
  },
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [{"type": "command", "command": "check-bash-v2.sh"}]
      }
    ]
  }
}`)

	got, err := MergeSettings(base, theirs, ours)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed settingsJSON
	if err := json.Unmarshal(got, &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	// PreToolUse should have: updated "Bash" from ours + user-added "Edit"
	preToolUse := parsed.Hooks["PreToolUse"]
	if len(preToolUse) != 2 {
		t.Fatalf("expected 2 PreToolUse matchers, got %d: %+v", len(preToolUse), preToolUse)
	}

	bashMatcher, found := findMatcher(preToolUse, "Bash")
	if !found {
		t.Fatal("expected Bash matcher in PreToolUse")
	}
	if bashMatcher.Hooks[0].Command != "check-bash-v2.sh" {
		t.Errorf("expected updated Bash command, got %s", bashMatcher.Hooks[0].Command)
	}

	editMatcher, found := findMatcher(preToolUse, "Edit")
	if !found {
		t.Fatal("expected user-added Edit matcher in PreToolUse")
	}
	if editMatcher.Hooks[0].Command != "my-edit-check.sh" {
		t.Errorf("expected user Edit command, got %s", editMatcher.Hooks[0].Command)
	}

	// PostToolUse should be preserved (user-added event not in base).
	postToolUse := parsed.Hooks["PostToolUse"]
	if len(postToolUse) != 1 {
		t.Fatalf("expected 1 PostToolUse matcher, got %d", len(postToolUse))
	}
	if postToolUse[0].Matcher != "Bash" {
		t.Errorf("expected Bash matcher in PostToolUse, got %s", postToolUse[0].Matcher)
	}
}

func TestMergeSettings_GeneratedHookUpdated(t *testing.T) {
	base := []byte(`{
  "permissions": {
    "allow": [],
    "deny": []
  },
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [{"type": "command", "command": "old-check.sh", "timeout": 5000}]
      }
    ]
  }
}`)
	theirs := []byte(`{
  "permissions": {
    "allow": [],
    "deny": []
  },
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [{"type": "command", "command": "old-check.sh", "timeout": 5000}]
      }
    ]
  }
}`)
	ours := []byte(`{
  "permissions": {
    "allow": [],
    "deny": []
  },
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [{"type": "command", "command": "new-check.sh", "timeout": 10000}]
      }
    ]
  }
}`)

	got, err := MergeSettings(base, theirs, ours)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed settingsJSON
	if err := json.Unmarshal(got, &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	bashMatcher, found := findMatcher(parsed.Hooks["PreToolUse"], "Bash")
	if !found {
		t.Fatal("expected Bash matcher")
	}
	if bashMatcher.Hooks[0].Command != "new-check.sh" {
		t.Errorf("expected new-check.sh, got %s", bashMatcher.Hooks[0].Command)
	}
	if bashMatcher.Hooks[0].Timeout != 10000 {
		t.Errorf("expected timeout 10000, got %d", bashMatcher.Hooks[0].Timeout)
	}
}

func TestMergeSettings_SandboxUnion(t *testing.T) {
	base := []byte(`{
  "permissions": {
    "allow": [],
    "deny": []
  }
}`)
	theirs := []byte(`{
  "permissions": {
    "allow": [],
    "deny": []
  },
  "sandbox": {
    "writeDeny": ["/etc"],
    "netAllow": ["api.example.com"]
  }
}`)
	ours := []byte(`{
  "permissions": {
    "allow": [],
    "deny": []
  },
  "sandbox": {
    "writeDeny": ["/usr"],
    "readDeny": ["/secrets"],
    "netAllow": ["registry.npmjs.org"]
  }
}`)

	got, err := MergeSettings(base, theirs, ours)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed settingsJSON
	if err := json.Unmarshal(got, &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	if parsed.Sandbox == nil {
		t.Fatal("expected sandbox to be non-nil")
	}
	assertStringSlice(t, "writeDeny", parsed.Sandbox.WriteDeny, []string{"/usr", "/etc"})
	assertStringSlice(t, "readDeny", parsed.Sandbox.ReadDeny, []string{"/secrets"})
	assertStringSlice(t, "netAllow", parsed.Sandbox.NetAllow, []string{"registry.npmjs.org", "api.example.com"})
}

func TestMergeSettings_SandboxOursOnly(t *testing.T) {
	base := []byte(`{
  "permissions": {
    "allow": [],
    "deny": []
  }
}`)
	theirs := []byte(`{
  "permissions": {
    "allow": [],
    "deny": []
  }
}`)
	ours := []byte(`{
  "permissions": {
    "allow": [],
    "deny": []
  },
  "sandbox": {
    "writeDeny": ["/etc"]
  }
}`)

	got, err := MergeSettings(base, theirs, ours)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed settingsJSON
	if err := json.Unmarshal(got, &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	if parsed.Sandbox == nil {
		t.Fatal("expected sandbox from ours")
	}
	assertStringSlice(t, "writeDeny", parsed.Sandbox.WriteDeny, []string{"/etc"})
}

func TestMergeSettings_SandboxTheirsOnly(t *testing.T) {
	base := []byte(`{
  "permissions": {
    "allow": [],
    "deny": []
  }
}`)
	theirs := []byte(`{
  "permissions": {
    "allow": [],
    "deny": []
  },
  "sandbox": {
    "netAllow": ["internal.corp.com"]
  }
}`)
	ours := []byte(`{
  "permissions": {
    "allow": [],
    "deny": []
  }
}`)

	got, err := MergeSettings(base, theirs, ours)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed settingsJSON
	if err := json.Unmarshal(got, &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	if parsed.Sandbox == nil {
		t.Fatal("expected sandbox from theirs")
	}
	assertStringSlice(t, "netAllow", parsed.Sandbox.NetAllow, []string{"internal.corp.com"})
}

func TestMergeSettings_EmptyBase(t *testing.T) {
	// nil base — upgrade case. All theirs content treated as user-added.
	theirs := []byte(`{
  "permissions": {
    "allow": ["Read(*)", "Bash(my-tool *)"],
    "deny": ["Bash(curl *)"]
  }
}`)
	ours := []byte(`{
  "permissions": {
    "allow": ["Read(*)", "Edit(*)"],
    "deny": ["Bash(npm install *)"]
  }
}`)

	got, err := MergeSettings(nil, theirs, ours)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed settingsJSON
	if err := json.Unmarshal(got, &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	// All theirs rules are user-added (since base is empty).
	// ours allow + user-added from theirs.
	wantAllow := []string{"Read(*)", "Edit(*)", "Bash(my-tool *)"}
	wantDeny := []string{"Bash(npm install *)", "Bash(curl *)"}
	assertStringSlice(t, "allow", parsed.Permissions.Allow, wantAllow)
	assertStringSlice(t, "deny", parsed.Permissions.Deny, wantDeny)
}

func TestMergeSettings_InvalidJSON(t *testing.T) {
	base := []byte(`{"permissions":{"allow":[],"deny":[]}}`)
	theirs := []byte(`{invalid json}`)
	ours := []byte(`{"permissions":{"allow":[],"deny":[]}}`)

	_, err := MergeSettings(base, theirs, ours)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "theirs") {
		t.Errorf("expected error to mention 'theirs', got: %v", err)
	}
}

func TestMergeSettings_OutputIsValidJSON(t *testing.T) {
	base := []byte(`{
  "permissions": {
    "defaultMode": "ask",
    "allow": ["Read(*)"],
    "deny": ["Bash(rm *)"]
  },
  "hooks": {
    "PreToolUse": [
      {"matcher": "Bash", "hooks": [{"type": "command", "command": "check.sh"}]}
    ]
  },
  "sandbox": {
    "writeDeny": ["/etc"]
  }
}`)
	theirs := []byte(`{
  "permissions": {
    "defaultMode": "ask",
    "allow": ["Read(*)", "Bash(custom *)"],
    "deny": ["Bash(rm *)"]
  },
  "hooks": {
    "PreToolUse": [
      {"matcher": "Bash", "hooks": [{"type": "command", "command": "check.sh"}]},
      {"matcher": "Edit", "hooks": [{"type": "command", "command": "user-edit.sh"}]}
    ]
  },
  "sandbox": {
    "writeDeny": ["/etc"],
    "netAllow": ["example.com"]
  }
}`)
	ours := []byte(`{
  "permissions": {
    "defaultMode": "deny",
    "allow": ["Read(*)", "Edit(*)"],
    "deny": ["Bash(rm *)", "Bash(npm install *)"]
  },
  "hooks": {
    "PreToolUse": [
      {"matcher": "Bash", "hooks": [{"type": "command", "command": "check-v2.sh"}]}
    ]
  },
  "sandbox": {
    "writeDeny": ["/etc", "/usr"]
  }
}`)

	got, err := MergeSettings(base, theirs, ours)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var raw json.RawMessage
	if err := json.Unmarshal(got, &raw); err != nil {
		t.Fatalf("result is not valid JSON: %v\nresult:\n%s", err, got)
	}
}

func TestMergeSettings_PolicyFieldsFromOurs(t *testing.T) {
	base := []byte(`{
  "permissions": {
    "defaultMode": "ask",
    "disableBypassPermissionsMode": "false",
    "allow": [],
    "deny": []
  }
}`)
	// User changed defaultMode — but policy fields always come from ours.
	theirs := []byte(`{
  "permissions": {
    "defaultMode": "allow",
    "disableBypassPermissionsMode": "false",
    "allow": [],
    "deny": []
  }
}`)
	ours := []byte(`{
  "permissions": {
    "defaultMode": "deny",
    "disableBypassPermissionsMode": "true",
    "allow": [],
    "deny": []
  }
}`)

	got, err := MergeSettings(base, theirs, ours)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed settingsJSON
	if err := json.Unmarshal(got, &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	if parsed.Permissions.DefaultMode != "deny" {
		t.Errorf("expected defaultMode 'deny' from ours, got %q", parsed.Permissions.DefaultMode)
	}
	if parsed.Permissions.DisableBypassPermissionsMode != "true" {
		t.Errorf("expected disableBypassPermissionsMode 'true' from ours, got %q", parsed.Permissions.DisableBypassPermissionsMode)
	}
}

func TestMergeSettings_ExtraTopLevelKeys(t *testing.T) {
	base := []byte(`{
  "permissions": {
    "allow": ["Read(*)"],
    "deny": []
  }
}`)
	theirs := []byte(`{
  "permissions": {
    "allow": ["Read(*)"],
    "deny": []
  },
  "customField": true,
  "anotherCustom": {"nested": "value"}
}`)
	ours := []byte(`{
  "permissions": {
    "allow": ["Read(*)", "Edit(*)"],
    "deny": []
  }
}`)

	got, err := MergeSettings(base, theirs, ours)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the result is valid JSON.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(got, &raw); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	// customField should be preserved.
	if _, ok := raw["customField"]; !ok {
		t.Error("expected customField to be preserved in output")
	}
	if _, ok := raw["anotherCustom"]; !ok {
		t.Error("expected anotherCustom to be preserved in output")
	}

	// Verify permissions were still merged correctly.
	var parsed settingsJSON
	if err := json.Unmarshal(got, &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}
	wantAllow := []string{"Read(*)", "Edit(*)"}
	assertStringSlice(t, "allow", parsed.Permissions.Allow, wantAllow)
}

// assertStringSlice is a helper that compares two string slices.
func assertStringSlice(t *testing.T, name string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s: length mismatch: got %d (%v), want %d (%v)", name, len(got), got, len(want), want)
		return
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("%s[%d]: got %q, want %q (full: got %v, want %v)", name, i, got[i], want[i], got, want)
			return
		}
	}
}
