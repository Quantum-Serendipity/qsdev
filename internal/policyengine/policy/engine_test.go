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
