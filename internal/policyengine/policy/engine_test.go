package policy

import (
	"os"
	"path/filepath"
	"testing"
)

// TestFilePathDenyRulesNormalizesType is the red→green guard for the deny-rule
// Type mismatch (F-CAP-21.6-1): the compiler emits denied_path / path_glob but
// the trust confused-deputy check only acts on Type=="path". FilePathDenyRules
// must normalize both compiled types onto "path" so real compiled rules match.
func TestFilePathDenyRulesNormalizesType(t *testing.T) {
	t.Parallel()

	policyYAML := `apiVersion: qsdev/v1
kind: SecurityPolicy
metadata:
  name: normalize-test
  version: "1.0.0"
rules:
  - id: DENY-PATH
    category: config-guard
    name: Denied path check
    severity: high
    bypass_tier: session
    conditions:
      type: denied_path_check
      pattern: "/etc/secret/*"
    action:
      type: block
      message: "denied"
  - id: DENY-GLOB
    category: config-guard
    name: Path glob check
    severity: high
    bypass_tier: session
    conditions:
      type: path_glob
      pattern: "**/.ssh/*"
    action:
      type: block
      message: "denied"
`

	dir := t.TempDir()
	policyFile := filepath.Join(dir, "policy.yaml")
	if err := os.WriteFile(policyFile, []byte(policyYAML), 0o644); err != nil {
		t.Fatalf("writing policy file: %v", err)
	}

	// The raw compiler output must use the condition-specific types, proving the
	// normalization is load-bearing (the trust layer would skip these verbatim).
	sp, err := LoadPolicyFiles(policyFile)
	if err != nil {
		t.Fatalf("loading policy: %v", err)
	}
	compiled, err := Compile(sp)
	if err != nil {
		t.Fatalf("compiling policy: %v", err)
	}
	rawTypes := make(map[string]bool)
	for _, r := range compiled.DenyRules {
		rawTypes[r.Type] = true
	}
	if !rawTypes["denied_path"] || !rawTypes["path_glob"] {
		t.Fatalf("expected compiler to emit denied_path and path_glob, got %v", rawTypes)
	}

	engine, err := NewPolicyEngine([]string{policyFile}, nil, EngineOptions{})
	if err != nil {
		t.Fatalf("creating engine: %v", err)
	}

	rules := engine.FilePathDenyRules()
	if len(rules) != 2 {
		t.Fatalf("expected 2 deny rules, got %d", len(rules))
	}
	for _, r := range rules {
		if r.Type != "path" {
			t.Errorf("deny rule %q not normalized: got type %q, want \"path\"", r.Pattern, r.Type)
		}
	}
}

// denyPolicyYAML returns a one-rule policy denying pattern via denied_path_check.
func denyPolicyYAML(pattern string) string {
	return `apiVersion: qsdev/v1
kind: SecurityPolicy
metadata:
  name: reload-test
  version: "1.0.0"
rules:
  - id: DENY-PATH
    category: config-guard
    name: Denied path check
    severity: high
    bypass_tier: session
    conditions:
      type: denied_path_check
      pattern: "` + pattern + `"
    action:
      type: block
      message: "denied"
`
}

// TestReload_DenyRuleChangesLatestWins is the regression guard for the
// deny-rule change channel: when a consumer has not yet drained an earlier
// update, a newer reload must replace it (not be dropped), and the published
// rules must carry the normalized "path" type the trust layer acts on.
func TestReload_DenyRuleChangesLatestWins(t *testing.T) {
	t.Parallel()

	policyFile := filepath.Join(t.TempDir(), "policy.yaml")
	write := func(pattern string) {
		t.Helper()
		if err := os.WriteFile(policyFile, []byte(denyPolicyYAML(pattern)), 0o644); err != nil {
			t.Fatalf("writing policy file: %v", err)
		}
	}

	write("/first/*")
	engine, err := NewPolicyEngine([]string{policyFile}, nil, EngineOptions{})
	if err != nil {
		t.Fatalf("creating engine: %v", err)
	}

	for _, pattern := range []string{"/second/*", "/third/*"} {
		write(pattern)
		if err := engine.Reload(); err != nil {
			t.Fatalf("Reload(%s): %v", pattern, err)
		}
	}

	select {
	case rules := <-engine.DenyRuleChanges():
		if len(rules) != 1 || rules[0].Pattern != "/third/*" || rules[0].Type != "path" {
			t.Fatalf("DenyRuleChanges delivered %+v, want the latest normalized rule {/third/* path}", rules)
		}
	default:
		t.Fatal("DenyRuleChanges delivered nothing after reloads that changed the deny rules")
	}

	select {
	case rules := <-engine.DenyRuleChanges():
		t.Fatalf("DenyRuleChanges delivered a second, stale update: %+v", rules)
	default:
	}

	// Reloading an unchanged policy publishes nothing.
	if err := engine.Reload(); err != nil {
		t.Fatalf("Reload(unchanged): %v", err)
	}
	select {
	case rules := <-engine.DenyRuleChanges():
		t.Fatalf("unchanged reload published %+v", rules)
	default:
	}
}

// TestCompile_DenyRulesOnlyFromBlockingRules guards the deny-rule projection
// the confused-deputy check enforces as hard blocks: only a pattern that blocks
// the first-party call may block MCP access, and a pattern under `not` names an
// allowed set rather than a denied one.
func TestCompile_DenyRulesOnlyFromBlockingRules(t *testing.T) {
	t.Parallel()

	pathRule := func(id string, action Action, monitor bool, cond Condition) PolicyRule {
		return PolicyRule{
			ID: id, Category: "test", Name: id, Severity: High, BypassTier: Session,
			MonitorMode: monitor, Conditions: cond, Action: action,
		}
	}
	glob := func(p string) Condition { return Condition{Type: PathGlob, Pattern: p} }

	tests := []struct {
		name string
		rule PolicyRule
		want bool
	}{
		{"block rule projects", pathRule("B", Action{Type: Block}, false, glob("**/.ssh/*")), true},
		{"fail-closed prompt projects", pathRule("P", Action{Type: Prompt, DefaultOnTimeout: "block"}, false, glob("**/.ssh/*")), true},
		{"allow-default prompt does not", pathRule("PA", Action{Type: Prompt, DefaultOnTimeout: "allow"}, false, glob("**/.ssh/*")), false},
		{"warn rule does not", pathRule("W", Action{Type: Warn}, false, glob("**/.npmrc")), false},
		{"audit rule does not", pathRule("A", Action{Type: Audit}, false, glob("**/.npmrc")), false},
		{"monitor-mode block does not", pathRule("M", Action{Type: Block}, true, glob("**/.ssh/*")), false},
		{"pattern under not does not", pathRule("N", Action{Type: Block}, false, Condition{Type: All, Conditions: []Condition{
			{Type: ToolMatch, ToolName: "Edit"},
			{Type: Not, Condition: &Condition{Type: PathGlob, Pattern: "src/**"}},
		}}), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			set := compileTestPolicy(t, makePolicy(tt.rule))
			if got := len(set.DenyRules) > 0; got != tt.want {
				t.Errorf("DenyRules = %+v, want projected=%v", set.DenyRules, tt.want)
			}
		})
	}
}
