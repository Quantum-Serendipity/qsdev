package merge

import (
	"encoding/json"
	"slices"
	"testing"
)

// settingsDoc is a loosely-typed view of merged settings output, so tests can
// assert on fields the merge package does not model.
type settingsDoc struct {
	Sandbox map[string]any `json:"sandbox"`
	Hooks   map[string][]struct {
		Matcher string           `json:"matcher"`
		Hooks   []map[string]any `json:"hooks"`
	} `json:"hooks"`
}

func decodeSettingsDoc(t *testing.T, data []byte) settingsDoc {
	t.Helper()
	var doc settingsDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("merged settings is not valid JSON: %v\n%s", err, data)
	}
	return doc
}

// hookCommands returns the command (or prompt) of each hook under the given
// event and matcher.
func hookCommands(t *testing.T, doc settingsDoc, event, matcher string) []string {
	t.Helper()
	for _, m := range doc.Hooks[event] {
		if m.Matcher != matcher {
			continue
		}
		var out []string
		for _, h := range m.Hooks {
			if c, ok := h["command"].(string); ok {
				out = append(out, c)
			} else if p, ok := h["prompt"].(string); ok {
				out = append(out, p)
			}
		}
		return out
	}
	return nil
}

// TestMergeSettings_PreservesUserHooksAndSandboxKeys covers user hook entries
// under generated matchers, hook types with unmodeled fields, and unmodeled
// sandbox keys, on both the update (recorded base) and create (nil base) paths.
func TestMergeSettings_PreservesUserHooksAndSandboxKeys(t *testing.T) {
	t.Parallel()

	base := []byte(`{
  "permissions": {"allow": [], "deny": []},
  "sandbox":{"filesystem":{"denyWrite":["/etc"]}},
  "hooks": {
    "PreToolUse": [
      {"matcher": "Bash", "hooks": [{"type": "command", "command": "guard.sh", "timeout": 5}]}
    ]
  }
}`)
	theirs := []byte(`{
  "permissions": {"allow": [], "deny": []},
  "sandbox":{"enabled": true,"excludedCommands": ["docker"],"filesystem":{"denyWrite":["/etc"]}},
  "hooks": {
    "PreToolUse": [
      {"matcher": "Bash", "hooks": [
        {"type": "command", "command": "guard.sh", "timeout": 5},
        {"type": "command", "command": "my-audit.sh"}
      ]},
      {"matcher": "Write", "hooks": [{"type": "prompt", "prompt": "Is this safe?", "timeout": 30}]}
    ]
  }
}`)
	ours := []byte(`{
  "permissions": {"allow": [], "deny": []},
  "sandbox":{"filesystem":{"denyWrite":["/etc", "/usr"]}},
  "hooks": {
    "PreToolUse": [
      {"matcher": "Bash", "hooks": [{"type": "command", "command": "guard.sh", "timeout": 10}]}
    ]
  }
}`)

	tests := []struct {
		name string
		base []byte
	}{
		{name: "update path", base: base},
		{name: "create path (nil base)", base: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := MergeSettings(tt.base, theirs, ours)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			doc := decodeSettingsDoc(t, got)

			bash := hookCommands(t, doc, "PreToolUse", "Bash")
			if !slices.Equal(bash, []string{"guard.sh", "my-audit.sh"}) {
				t.Errorf("Bash hooks = %v, want [guard.sh my-audit.sh]", bash)
			}
			// The generated hook must carry ours' updated options, once.
			for _, m := range doc.Hooks["PreToolUse"] {
				if m.Matcher == "Bash" && m.Hooks[0]["timeout"] != float64(10) {
					t.Errorf("generated Bash hook timeout = %v, want 10", m.Hooks[0]["timeout"])
				}
			}

			var prompt map[string]any
			for _, m := range doc.Hooks["PreToolUse"] {
				if m.Matcher == "Write" && len(m.Hooks) == 1 {
					prompt = m.Hooks[0]
				}
			}
			if prompt == nil {
				t.Fatalf("user prompt hook under Write matcher lost: %s", got)
			}
			if prompt["prompt"] != "Is this safe?" {
				t.Errorf("prompt hook lost its prompt field: %v", prompt)
			}
			if _, ok := prompt["command"]; ok {
				t.Errorf("prompt hook gained a command field: %v", prompt)
			}

			if doc.Sandbox["enabled"] != true {
				t.Errorf("sandbox.enabled lost: sandbox = %v", doc.Sandbox)
			}
			if excluded, _ := doc.Sandbox["excludedCommands"].([]any); len(excluded) != 1 {
				t.Errorf("unmodeled sandbox.excludedCommands lost: sandbox = %v", doc.Sandbox)
			}
			fsObj, _ := doc.Sandbox["filesystem"].(map[string]any)
			denyWrite, _ := fsObj["denyWrite"].([]any)
			if len(denyWrite) != 2 {
				t.Errorf("sandbox.filesystem.denyWrite = %v, want [/etc /usr]", fsObj["denyWrite"])
			}
		})
	}
}

// TestMergeSettings_SandboxAllowRespectsBase verifies allow-type sandbox
// arrays use the three-way algorithm: an entry the generator dropped is
// removed, a user-added entry is kept, and deny arrays are never shrunk.
func TestMergeSettings_SandboxAllowRespectsBase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		base          string
		theirs        string
		ours          string
		wantNetAllow  []string
		wantWriteDeny []string
		wantSandbox   bool
	}{
		{
			name:          "generator dropped allow entry",
			base:          `{"permissions":{"allow":[],"deny":[]},"sandbox":{"filesystem":{"denyWrite":["/etc"]},"network":{"allowedDomains":["old.example.com"]}}}`,
			theirs:        `{"permissions":{"allow":[],"deny":[]},"sandbox":{"filesystem":{"denyWrite":["/etc"]},"network":{"allowedDomains":["old.example.com"]}}}`,
			ours:          `{"permissions":{"allow":[],"deny":[]},"sandbox":{"filesystem":{"denyWrite":["/etc"]}}}`,
			wantNetAllow:  nil,
			wantWriteDeny: []string{"/etc"},
			wantSandbox:   true,
		},
		{
			name:          "user-added allow entry kept",
			base:          `{"permissions":{"allow":[],"deny":[]},"sandbox":{"network":{"allowedDomains":["old.example.com"]}}}`,
			theirs:        `{"permissions":{"allow":[],"deny":[]},"sandbox":{"network":{"allowedDomains":["old.example.com","mine.example.com"]}}}`,
			ours:          `{"permissions":{"allow":[],"deny":[]},"sandbox":{"network":{"allowedDomains":["new.example.com"]}}}`,
			wantNetAllow:  []string{"new.example.com", "mine.example.com"},
			wantWriteDeny: nil,
			wantSandbox:   true,
		},
		{
			name:          "generator removed sandbox entirely",
			base:          `{"permissions":{"allow":[],"deny":[]},"sandbox":{"network":{"allowedDomains":["old.example.com"]}}}`,
			theirs:        `{"permissions":{"allow":[],"deny":[]},"sandbox":{"network":{"allowedDomains":["old.example.com"]}}}`,
			ours:          `{"permissions":{"allow":[],"deny":[]}}`,
			wantNetAllow:  nil,
			wantWriteDeny: nil,
			wantSandbox:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := MergeSettings([]byte(tt.base), []byte(tt.theirs), []byte(tt.ours))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			var parsed settingsJSON
			if err := json.Unmarshal(got, &parsed); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}
			if (parsed.Sandbox != nil) != tt.wantSandbox {
				t.Fatalf("sandbox present = %v, want %v: %s", parsed.Sandbox != nil, tt.wantSandbox, got)
			}
			if parsed.Sandbox == nil {
				return
			}
			assertStringSlice(t, "network.allowedDomains", derefOr(parsed.Sandbox.Network).AllowedDomains, tt.wantNetAllow)
			assertStringSlice(t, "filesystem.denyWrite", derefOr(parsed.Sandbox.Filesystem).DenyWrite, tt.wantWriteDeny)
		})
	}
}
