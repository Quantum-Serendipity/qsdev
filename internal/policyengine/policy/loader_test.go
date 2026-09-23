package policy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadPolicyFile_Valid(t *testing.T) {
	t.Parallel()

	sp, err := LoadPolicyFile("testdata/valid-policy.yaml")
	if err != nil {
		t.Fatalf("LoadPolicyFile returned unexpected error: %v", err)
	}

	if got := len(sp.Rules); got != 9 {
		t.Fatalf("expected 9 rules, got %d", got)
	}

	first := sp.Rules[0]
	if first.ID != "SP-001" {
		t.Errorf("first rule ID: got %q, want %q", first.ID, "SP-001")
	}
	if first.Severity != Critical {
		t.Errorf("first rule severity: got %v, want %v", first.Severity, Critical)
	}
	if first.BypassTier != EnforceAlways {
		t.Errorf("first rule bypass_tier: got %v, want %v", first.BypassTier, EnforceAlways)
	}
	if first.Action.Type != Block {
		t.Errorf("first rule action type: got %q, want %q", first.Action.Type, Block)
	}
	if first.Conditions.Type != All {
		t.Errorf("first rule condition type: got %q, want %q", first.Conditions.Type, All)
	}
}

func TestLoadPolicyFile_Malformed(t *testing.T) {
	t.Parallel()

	_, err := LoadPolicyFile("testdata/malformed-policy.yaml")
	if err == nil {
		t.Fatal("expected error for malformed policy, got nil")
	}
	if !strings.Contains(err.Error(), "rule id is required") {
		t.Errorf("expected error to contain %q, got: %v", "rule id is required", err)
	}
}

func TestLoadPolicyFile_InvalidAPIVersion(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "bad-api-version.yaml")

	content := `apiVersion: v2
kind: SecurityPolicy
metadata:
  name: bad-version
rules:
  - id: RULE-1
    category: test
    name: test rule
    severity: low
    bypass_tier: command
    conditions:
      type: tool_match
      tool_name: Bash
    action:
      type: warn
      message: "test"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	_, err := LoadPolicyFile(path)
	if err == nil {
		t.Fatal("expected error for invalid apiVersion, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported apiVersion") {
		t.Errorf("expected error to contain %q, got: %v", "unsupported apiVersion", err)
	}
}

func TestLoadPolicyFile_DuplicateRuleIDs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "duplicate-ids.yaml")

	content := `apiVersion: qsdev/v1
kind: SecurityPolicy
metadata:
  name: dup-test
rules:
  - id: DUP-001
    category: test
    name: first rule
    severity: high
    bypass_tier: session
    conditions:
      type: tool_match
      tool_name: Bash
    action:
      type: block
      message: "first"
  - id: DUP-001
    category: test
    name: second rule
    severity: low
    bypass_tier: command
    conditions:
      type: tool_match
      tool_name: Edit
    action:
      type: warn
      message: "second"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	_, err := LoadPolicyFile(path)
	if err == nil {
		t.Fatal("expected error for duplicate rule IDs, got nil")
	}
	if !strings.Contains(err.Error(), "duplicate rule id") {
		t.Errorf("expected error to contain %q, got: %v", "duplicate rule id", err)
	}
}

func TestLoadPolicyFiles_SecurityFloor_WeakenBypassTier(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	basePath := filepath.Join(dir, "base.yaml")
	baseContent := `apiVersion: qsdev/v1
kind: SecurityPolicy
metadata:
  name: base
rules:
  - id: FLOOR-001
    category: test
    name: enforce always rule
    severity: critical
    bypass_tier: enforce_always
    conditions:
      type: tool_match
      tool_name: Bash
    action:
      type: block
      message: "blocked"
`
	if err := os.WriteFile(basePath, []byte(baseContent), 0o644); err != nil {
		t.Fatalf("writing base file: %v", err)
	}

	overlayPath := filepath.Join(dir, "overlay.yaml")
	overlayContent := `apiVersion: qsdev/v1
kind: SecurityPolicy
metadata:
  name: overlay
rules:
  - id: FLOOR-001
    category: test
    name: enforce always rule
    severity: critical
    bypass_tier: session
    conditions:
      type: tool_match
      tool_name: Bash
    action:
      type: block
      message: "blocked"
`
	if err := os.WriteFile(overlayPath, []byte(overlayContent), 0o644); err != nil {
		t.Fatalf("writing overlay file: %v", err)
	}

	_, err := LoadPolicyFiles(basePath, overlayPath)
	if err == nil {
		t.Fatal("expected security floor violation error, got nil")
	}
	if !strings.Contains(err.Error(), "security floor violation") {
		t.Errorf("expected error to contain %q, got: %v", "security floor violation", err)
	}
	if !strings.Contains(err.Error(), "bypass_tier") {
		t.Errorf("expected error to mention bypass_tier, got: %v", err)
	}
}

func TestLoadPolicyFiles_SecurityFloor_DisableEnforceAlways(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	basePath := filepath.Join(dir, "base.yaml")
	baseContent := `apiVersion: qsdev/v1
kind: SecurityPolicy
metadata:
  name: base
rules:
  - id: FLOOR-002
    category: test
    name: enforce always rule
    severity: critical
    bypass_tier: enforce_always
    conditions:
      type: tool_match
      tool_name: Bash
    action:
      type: block
      message: "blocked"
`
	if err := os.WriteFile(basePath, []byte(baseContent), 0o644); err != nil {
		t.Fatalf("writing base file: %v", err)
	}

	overlayPath := filepath.Join(dir, "overlay.yaml")
	overlayContent := `apiVersion: qsdev/v1
kind: SecurityPolicy
metadata:
  name: overlay
rules:
  - id: FLOOR-002
    category: test
    name: enforce always rule
    severity: critical
    bypass_tier: enforce_always
    enabled: false
    conditions:
      type: tool_match
      tool_name: Bash
    action:
      type: block
      message: "blocked"
`
	if err := os.WriteFile(overlayPath, []byte(overlayContent), 0o644); err != nil {
		t.Fatalf("writing overlay file: %v", err)
	}

	_, err := LoadPolicyFiles(basePath, overlayPath)
	if err == nil {
		t.Fatal("expected security floor violation error for disabling enforce_always, got nil")
	}
	if !strings.Contains(err.Error(), "security floor violation") {
		t.Errorf("expected error to contain %q, got: %v", "security floor violation", err)
	}
	if !strings.Contains(err.Error(), "cannot be disabled") {
		t.Errorf("expected error to mention disabling, got: %v", err)
	}
}

func TestLoadPolicyFiles_SecurityFloor_LowerSeverity(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	basePath := filepath.Join(dir, "base.yaml")
	baseContent := `apiVersion: qsdev/v1
kind: SecurityPolicy
metadata:
  name: base
rules:
  - id: FLOOR-003
    category: test
    name: critical rule
    severity: critical
    bypass_tier: session
    conditions:
      type: tool_match
      tool_name: Bash
    action:
      type: block
      message: "blocked"
`
	if err := os.WriteFile(basePath, []byte(baseContent), 0o644); err != nil {
		t.Fatalf("writing base file: %v", err)
	}

	overlayPath := filepath.Join(dir, "overlay.yaml")
	overlayContent := `apiVersion: qsdev/v1
kind: SecurityPolicy
metadata:
  name: overlay
rules:
  - id: FLOOR-003
    category: test
    name: critical rule
    severity: low
    bypass_tier: session
    conditions:
      type: tool_match
      tool_name: Bash
    action:
      type: block
      message: "blocked"
`
	if err := os.WriteFile(overlayPath, []byte(overlayContent), 0o644); err != nil {
		t.Fatalf("writing overlay file: %v", err)
	}

	_, err := LoadPolicyFiles(basePath, overlayPath)
	if err == nil {
		t.Fatal("expected security floor violation error for lowering severity, got nil")
	}
	if !strings.Contains(err.Error(), "security floor violation") {
		t.Errorf("expected error to contain %q, got: %v", "security floor violation", err)
	}
	if !strings.Contains(err.Error(), "severity") {
		t.Errorf("expected error to mention severity, got: %v", err)
	}
}

func TestLoadPolicyFiles_MergeOverride(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	basePath := filepath.Join(dir, "base.yaml")
	baseContent := `apiVersion: qsdev/v1
kind: SecurityPolicy
metadata:
  name: base
rules:
  - id: MERGE-001
    category: test
    name: original name
    severity: high
    bypass_tier: session
    conditions:
      type: tool_match
      tool_name: Bash
    action:
      type: warn
      message: "original message"
  - id: BASE-ONLY
    category: test
    name: base only rule
    severity: low
    bypass_tier: command
    conditions:
      type: tool_match
      tool_name: Edit
    action:
      type: audit
      message: "base only"
`
	if err := os.WriteFile(basePath, []byte(baseContent), 0o644); err != nil {
		t.Fatalf("writing base file: %v", err)
	}

	overlayPath := filepath.Join(dir, "overlay.yaml")
	overlayContent := `apiVersion: qsdev/v1
kind: SecurityPolicy
metadata:
  name: overlay
rules:
  - id: MERGE-001
    category: test
    name: overridden name
    severity: high
    bypass_tier: session
    conditions:
      type: tool_match
      tool_name: Bash
    action:
      type: block
      message: "overridden message"
  - id: OVERLAY-ONLY
    category: test
    name: overlay only rule
    severity: medium
    bypass_tier: session
    conditions:
      type: tool_match
      tool_name: Read
    action:
      type: warn
      message: "overlay only"
`
	if err := os.WriteFile(overlayPath, []byte(overlayContent), 0o644); err != nil {
		t.Fatalf("writing overlay file: %v", err)
	}

	sp, err := LoadPolicyFiles(basePath, overlayPath)
	if err != nil {
		t.Fatalf("LoadPolicyFiles returned unexpected error: %v", err)
	}

	if got := len(sp.Rules); got != 3 {
		t.Fatalf("expected 3 rules after merge, got %d", got)
	}

	ruleIndex := make(map[string]*PolicyRule, len(sp.Rules))
	for i := range sp.Rules {
		ruleIndex[sp.Rules[i].ID] = &sp.Rules[i]
	}

	merged, ok := ruleIndex["MERGE-001"]
	if !ok {
		t.Fatal("MERGE-001 not found after merge")
	}
	if merged.Name != "overridden name" {
		t.Errorf("MERGE-001 name: got %q, want %q", merged.Name, "overridden name")
	}
	if merged.Action.Type != Block {
		t.Errorf("MERGE-001 action type: got %q, want %q", merged.Action.Type, Block)
	}
	if merged.Action.Message != "overridden message" {
		t.Errorf("MERGE-001 action message: got %q, want %q", merged.Action.Message, "overridden message")
	}

	if _, ok := ruleIndex["BASE-ONLY"]; !ok {
		t.Error("BASE-ONLY rule missing after merge")
	}
	if _, ok := ruleIndex["OVERLAY-ONLY"]; !ok {
		t.Error("OVERLAY-ONLY rule missing after merge")
	}
}

// writePolicyFile writes a qsdev/v1 SecurityPolicy with the given rules YAML
// (and optional settings YAML) and returns its path.
func writePolicyFile(t *testing.T, dir, name, settings, rules string) string {
	t.Helper()

	content := "apiVersion: qsdev/v1\nkind: SecurityPolicy\nmetadata:\n  name: " + name + "\n"
	if settings != "" {
		content += "settings:\n" + settings
	}
	content += "rules:\n" + rules

	path := filepath.Join(dir, name+".yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
	return path
}

const floorBaseRule = `  - id: FLOOR-100
    category: test
    name: enforce always rule
    severity: high
    bypass_tier: enforce_always
    conditions:
      type: command_match
      pattern: curl
    action:
      type: block
      message: "blocked"
`

// TestLoadPolicyFiles_SecurityFloor_EnforceAlwaysNeutralization pins every
// overlay vector that keeps bypass_tier: enforce_always but still stops the
// rule from blocking (F178): monitor mode, a weaker action, a changed action
// verdict field, or replaced conditions. Each must be rejected, and the
// harmless rewording overrides must still merge.
func TestLoadPolicyFiles_SecurityFloor_EnforceAlwaysNeutralization(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		overlay string
		wantErr string
	}{
		{
			name: "monitor mode",
			overlay: `  - id: FLOOR-100
    category: test
    name: enforce always rule
    severity: high
    bypass_tier: enforce_always
    monitor_mode: true
    conditions:
      type: command_match
      pattern: curl
    action:
      type: block
      message: "blocked"
`,
			wantErr: "monitor_mode",
		},
		{
			name: "action downgraded to audit",
			overlay: `  - id: FLOOR-100
    category: test
    name: enforce always rule
    severity: high
    bypass_tier: enforce_always
    conditions:
      type: command_match
      pattern: curl
    action:
      type: audit
      message: "blocked"
`,
			wantErr: "action",
		},
		{
			name: "prompt with allow default",
			overlay: `  - id: FLOOR-100
    category: test
    name: enforce always rule
    severity: high
    bypass_tier: enforce_always
    conditions:
      type: command_match
      pattern: curl
    action:
      type: prompt
      default_on_timeout: allow
      message: "blocked"
`,
			wantErr: "action",
		},
		{
			name: "conditions replaced",
			overlay: `  - id: FLOOR-100
    category: test
    name: enforce always rule
    severity: high
    bypass_tier: enforce_always
    conditions:
      type: command_match
      pattern: zzzznever
    action:
      type: block
      message: "blocked"
`,
			wantErr: "conditions",
		},
		{
			name: "reworded message and raised severity allowed",
			overlay: `  - id: FLOOR-100
    category: test
    name: renamed rule
    severity: critical
    bypass_tier: enforce_always
    conditions:
      type: command_match
      pattern: curl
    action:
      type: block
      message: "blocked by org policy"
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			basePath := writePolicyFile(t, dir, "base", "", floorBaseRule)
			overlayPath := writePolicyFile(t, dir, "overlay", "", tt.overlay)

			sp, err := LoadPolicyFiles(basePath, overlayPath)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("LoadPolicyFiles: unexpected error: %v", err)
				}
				if got := sp.Rules[0].Action.Message; got != "blocked by org policy" {
					t.Errorf("merged message = %q, want overlay message", got)
				}
				if got := sp.Rules[0].Severity; got != Critical {
					t.Errorf("merged severity = %v, want %v", got, Critical)
				}
				return
			}
			if err == nil {
				t.Fatal("expected security floor violation, got nil")
			}
			if !strings.Contains(err.Error(), "security floor violation") || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %v, want security floor violation mentioning %q", err, tt.wantErr)
			}
		})
	}
}

// TestLoadPolicyFile_UnknownField pins strict decoding (F587): a misspelled
// key must be rejected rather than silently leaving the field at its default.
func TestLoadPolicyFile_UnknownField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		rules string
		field string
	}{
		{
			name: "misspelled bypass_tier",
			rules: `  - id: TYPO-001
    category: test
    name: typo rule
    severity: high
    bypas_tier: session
    conditions:
      type: tool_match
      tool_name: Bash
    action:
      type: block
`,
			field: "bypas_tier",
		},
		{
			name: "misspelled monitor_mode",
			rules: `  - id: TYPO-002
    category: test
    name: typo rule
    severity: high
    bypass_tier: session
    monitor_mod: true
    conditions:
      type: tool_match
      tool_name: Bash
    action:
      type: block
`,
			field: "monitor_mod",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			path := writePolicyFile(t, t.TempDir(), "typo", "", tt.rules)
			_, err := LoadPolicyFile(path)
			if err == nil {
				t.Fatalf("expected error for unknown field %q, got nil", tt.field)
			}
			if !strings.Contains(err.Error(), tt.field) {
				t.Errorf("error = %v, want it to name field %q", err, tt.field)
			}
		})
	}
}

// TestResolveFailMode pins the fail-closed default for a policy set that
// cannot be loaded (F177): only a base file that itself loads cleanly and
// explicitly opts into fail_open may fail open.
func TestResolveFailMode(t *testing.T) {
	t.Parallel()

	const rule = `  - id: FM-001
    category: test
    name: rule
    severity: high
    bypass_tier: session
    conditions:
      type: tool_match
      tool_name: Bash
    action:
      type: block
`

	tests := []struct {
		name  string
		setup func(t *testing.T, dir string) []string
		want  FailMode
	}{
		{
			name:  "no files",
			setup: func(_ *testing.T, _ string) []string { return nil },
			want:  FailClosed,
		},
		{
			name: "base unset",
			setup: func(t *testing.T, dir string) []string {
				return []string{writePolicyFile(t, dir, "base", "", rule)}
			},
			want: FailClosed,
		},
		{
			name: "base fail_open",
			setup: func(t *testing.T, dir string) []string {
				return []string{writePolicyFile(t, dir, "base", "  fail_mode: fail_open\n", rule)}
			},
			want: FailOpen,
		},
		{
			name: "overlay cannot opt into fail_open",
			setup: func(t *testing.T, dir string) []string {
				return []string{
					writePolicyFile(t, dir, "base", "", rule),
					writePolicyFile(t, dir, "overlay", "  fail_mode: fail_open\n", rule),
				}
			},
			want: FailClosed,
		},
		{
			name: "malformed base",
			setup: func(t *testing.T, dir string) []string {
				return []string{writePolicyFile(t, dir, "base", "  fail_mode: fail_open\n", rule+"    bogus: true\n")}
			},
			want: FailClosed,
		},
		{
			name: "missing base",
			setup: func(_ *testing.T, dir string) []string {
				return []string{filepath.Join(dir, "missing.yaml")}
			},
			want: FailClosed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ResolveFailMode(tt.setup(t, t.TempDir())); got != tt.want {
				t.Errorf("ResolveFailMode = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLoadPolicyFile_InvalidFailMode(t *testing.T) {
	t.Parallel()

	path := writePolicyFile(t, t.TempDir(), "base", "  fail_mode: fail_opne\n", `  - id: FM-002
    category: test
    name: rule
    severity: high
    bypass_tier: session
    conditions:
      type: tool_match
      tool_name: Bash
    action:
      type: block
`)
	_, err := LoadPolicyFile(path)
	if err == nil || !strings.Contains(err.Error(), "fail_mode") {
		t.Fatalf("expected invalid fail_mode error, got %v", err)
	}
}
