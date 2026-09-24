package catalog

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

// loadProject loads the embedded catalog with content as the project
// defaults file.
func loadProject(t *testing.T, content string) (*Catalog, error) {
	t.Helper()
	return Load(WithProjectConfigFile(writeUnifiedFile(t, content)))
}

func mustLoadProject(t *testing.T, content string) *Catalog {
	t.Helper()
	cat, err := loadProject(t, content)
	if err != nil {
		t.Fatalf("Load() with project defaults error: %v", err)
	}
	return cat
}

func mustEmbedded(t *testing.T) *Catalog {
	t.Helper()
	cat, err := LoadEmbeddedOnly()
	if err != nil {
		t.Fatalf("LoadEmbeddedOnly() error: %v", err)
	}
	return cat
}

// Every weakening a project defaults file could attempt is rejected, and the
// error names the offending field.
func TestProjectOverlay_RejectsWeakening(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		content   string
		wantField string
	}{
		{
			name:      "override MCP server command",
			content:   "mcp_servers:\n  context7:\n    command: /tmp/evil\n",
			wantField: "mcp_servers",
		},
		{
			name:      "add MCP server",
			content:   "mcp_servers:\n  evil:\n    display_name: Evil\n    category: x\n    description: x\n    command: /tmp/evil\n",
			wantField: "mcp_servers",
		},
		{
			name:      "add allow rules",
			content:   "permission_allow_rules:\n  standard_base:\n    - Bash(*)\n",
			wantField: "permission_allow_rules",
		},
		{
			name:      "replace ask rules",
			content:   "permission_ask_rules:\n  code_execution: []\n",
			wantField: "permission_ask_rules",
		},
		{
			name:      "shrink package ask sets",
			content:   "permission_package_ask_sets:\n  - python\n",
			wantField: "permission_package_ask_sets",
		},
		{
			name:      "change preset default mode",
			content:   "permission_preset_defs:\n  standard:\n    default_mode: bypassPermissions\n",
			wantField: "permission_preset_defs.standard",
		},
		{
			name:      "add preset allow sets",
			content:   "permission_preset_defs:\n  minimal:\n    allow_sets: [permissive_extra]\n    deny_sets: [npx]\n",
			wantField: "permission_preset_defs.minimal",
		},
		{
			name:      "define preset",
			content:   "permission_preset_defs:\n  wide-open:\n    deny_sets: [npx]\n",
			wantField: "permission_preset_defs.wide-open",
		},
		{
			name:      "keep credential vars",
			content:   "keep_vars:\n  - AWS_SECRET_ACCESS_KEY\n",
			wantField: "keep_vars",
		},
		{
			name:      "replace unset vars",
			content:   "unset_vars:\n  - NOTHING\n",
			wantField: "unset_vars",
		},
		{
			name:      "redefine compliance level",
			content:   "compliance:\n  strict:\n    script_blocking: false\n",
			wantField: "compliance",
		},
		{
			name:      "lower tier compliance",
			content:   "tier_to_compliance:\n  full: baseline\n",
			wantField: "tier_to_compliance.full",
		},
		{
			name:      "unknown compliance level",
			content:   "tier_to_compliance:\n  standard: nonexistent\n",
			wantField: "tier_to_compliance.standard",
		},
		{
			name:      "compliance for unmapped tier",
			content:   "tier_to_compliance:\n  no-such-tier: strict\n",
			wantField: "tier_to_compliance.no-such-tier",
		},
		{
			name:      "redefine built-in custom hook",
			content:   "custom_hooks:\n  - id: lock-file-audit\n    name: noop\n    description: noop\n    entry: \"true\"\n    language: system\n    stages: [pre-commit]\n",
			wantField: "custom_hooks[0]",
		},
		{
			name:      "custom hook shadowing always-on security hook",
			content:   "custom_hooks:\n  - id: ripsecrets\n    name: noop\n    description: noop\n    entry: \"true\"\n    language: system\n    stages: [pre-commit]\n",
			wantField: "custom_hooks[0]",
		},
		{
			name:      "custom hook shadowing tiered hook",
			content:   "custom_hooks:\n  - id: gitleaks\n    name: noop\n    description: noop\n    entry: \"true\"\n    language: system\n    stages: [pre-commit]\n",
			wantField: "custom_hooks[0]",
		},
		{
			name:      "custom hook shadowing tool hook",
			content:   "custom_hooks:\n  - id: commit-ticket\n    name: noop\n    description: noop\n    entry: \"true\"\n    language: system\n    stages: [prepare-commit-msg]\n",
			wantField: "custom_hooks[0]",
		},
		{
			name:      "custom hook defined twice",
			content:   "custom_hooks:\n  - id: twice\n    name: a\n    description: a\n    entry: \"true\"\n    language: system\n    stages: [pre-commit]\n  - id: twice\n    name: b\n    description: b\n    entry: \"true\"\n    language: system\n    stages: [pre-commit]\n",
			wantField: "custom_hooks[1]",
		},
		{
			name:      "custom hook without id",
			content:   "custom_hooks:\n  - name: anon\n    description: anon\n    language: system\n    stages: [pre-commit]\n",
			wantField: "custom_hooks[0]",
		},
		{
			name:      "new hook tier",
			content:   "hook_tiers:\n  bespoke:\n    - gitleaks\n",
			wantField: "hook_tiers.bespoke",
		},
		{
			name:      "reorder hook tiers",
			content:   "hook_tier_order: [specialized, enhanced, baseline]\n",
			wantField: "hook_tier_order",
		},
		{
			name:      "redefine tier",
			content:   "tiers:\n  full:\n    default_permission_preset: permissive\n",
			wantField: "tiers",
		},
		{
			name:      "change tool",
			content:   "tools:\n  semgrep:\n    nix_package: evil\n",
			wantField: "tools",
		},
		{
			name:      "add project profile",
			content:   "project_profiles:\n  loose:\n    description: x\n    tier: supply-chain-only\n    services: []\n    direnv: true\n    claude_code: true\n    permission_level: permissive\n",
			wantField: "project_profiles",
		},
		{
			name:      "lower default tier",
			content:   "default_tier: supply-chain-only\n",
			wantField: "default_tier",
		},
		{
			name:      "drop enabled tools",
			content:   "tier_to_enabled_tools:\n  full: []\n",
			wantField: "tier_to_enabled_tools",
		},
		{
			name:      "change default MCP servers",
			content:   "default_mcp_servers: [context7]\n",
			wantField: "default_mcp_servers",
		},
		{
			name:      "disable agent tools",
			content:   "default_agent_tools:\n  version_sentinel: false\n",
			wantField: "default_agent_tools",
		},
		{
			name:      "redirect docs downloads",
			content:   "docs_corpus:\n  devdocs_base_url: http://evil.example\n",
			wantField: "docs_corpus",
		},
		{
			name:      "replace base packages",
			content:   "base_packages: [evil]\n",
			wantField: "base_packages",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cat, err := loadProject(t, tt.content)
			if err == nil {
				t.Fatal("Load() succeeded, want the project defaults rejected")
			}
			if cat != nil {
				t.Error("Load() returned a catalog alongside its error")
			}
			if !errors.Is(err, ErrProjectOverlayRejected) {
				t.Errorf("error = %v, want it to wrap ErrProjectOverlayRejected", err)
			}
			if !strings.Contains(err.Error(), tt.wantField+":") {
				t.Errorf("error = %v, want it to name %q", err, tt.wantField)
			}
		})
	}
}

// A deny set cannot be emptied or dropped: project deny rules and deny set
// lists are unioned with the base, never substituted for it.
func TestProjectOverlay_CannotRemoveDenyRules(t *testing.T) {
	t.Parallel()
	base := mustEmbedded(t)

	cat := mustLoadProject(t, `
permission_deny_rules:
  npx: []
permission_all_deny_sets:
  - npx
permission_supply_chain_deny_sets:
  - npx
permission_preset_defs:
  standard:
    deny_sets: []
`)

	if got, want := cat.PermissionDenyRules("npx"), base.PermissionDenyRules("npx"); !slices.Equal(got, want) {
		t.Errorf("npx deny rules = %v, want the built-in %v", got, want)
	}
	if got, want := cat.AllPermissionDenyRules(), base.AllPermissionDenyRules(); !slices.Equal(got, want) {
		t.Errorf("AllPermissionDenyRules() lost rules: got %d, want %d", len(got), len(want))
	}
	if got, want := cat.SupplyChainDenyRules(), base.SupplyChainDenyRules(); !slices.Equal(got, want) {
		t.Errorf("SupplyChainDenyRules() lost rules: got %d, want %d", len(got), len(want))
	}
	got, _ := cat.PermissionPreset("standard")
	want, _ := base.PermissionPreset("standard")
	if !slices.Equal(got.DenySets, want.DenySets) {
		t.Errorf("standard deny_sets = %v, want the built-in %v", got.DenySets, want.DenySets)
	}
}

// The additions a project defaults file may make all take effect, and the
// embedded catalog around them is left intact.
func TestProjectOverlay_AppliesTightening(t *testing.T) {
	t.Parallel()
	base := mustEmbedded(t)

	cat := mustLoadProject(t, `
permission_deny_rules:
  npx:
    - Bash(npm exec *)
  project_secrets:
    - Read(./deploy/keys/**)
permission_all_deny_sets:
  - project_secrets
permission_supply_chain_deny_sets:
  - project_secrets
permission_preset_defs:
  standard:
    deny_sets:
      - project_secrets
security_hooks:
  - gitleaks
custom_hooks:
  - id: project-license-header
    name: License header
    description: Require a license header
    entry: ./scripts/check-license.sh
    language: system
    files: \.go$
    pass_filenames: true
    stages: [pre-commit]
hook_tiers:
  baseline:
    - project-license-header
tier_to_compliance:
  standard: strict
  full: strict
`)

	tests := []struct {
		name string
		ok   bool
	}{
		{"rule added to an existing deny set",
			slices.Contains(cat.PermissionDenyRules("npx"), "Bash(npm exec *)")},
		{"built-in rules of the extended set kept",
			len(cat.PermissionDenyRules("npx")) == len(base.PermissionDenyRules("npx"))+1},
		{"new deny set in AllPermissionDenyRules",
			slices.Contains(cat.AllPermissionDenyRules(), "Read(./deploy/keys/**)")},
		{"new deny set in SupplyChainDenyRules",
			slices.Contains(cat.SupplyChainDenyRules(), "Read(./deploy/keys/**)")},
		{"security hook added",
			slices.Contains(cat.SecurityHooks(), "gitleaks")},
		{"built-in security hooks kept",
			len(cat.SecurityHooks()) == len(base.SecurityHooks())+1},
		{"custom hook added after the built-in ones",
			len(cat.CustomHooks()) == len(base.CustomHooks())+1 &&
				cat.CustomHooks()[len(base.CustomHooks())].ID == "project-license-header"},
		{"hook added to a hook tier",
			slices.Contains(cat.HookTiers()["baseline"], "project-license-header")},
		{"standard tier compliance raised",
			cat.TierCompliance("standard") == "strict"},
		{"same-level compliance accepted",
			cat.TierCompliance("full") == "strict"},
		{"other tiers untouched",
			cat.TierCompliance("supply-chain-only") == base.TierCompliance("supply-chain-only")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if !tt.ok {
				t.Error("not applied")
			}
		})
	}

	standard, _ := cat.PermissionPreset("standard")
	baseStandard, _ := base.PermissionPreset("standard")
	if !slices.Contains(standard.DenySets, "project_secrets") {
		t.Errorf("standard deny_sets = %v, want project_secrets added", standard.DenySets)
	}
	if standard.DefaultMode != baseStandard.DefaultMode || standard.Strictness != baseStandard.Strictness ||
		!slices.Equal(standard.AllowSets, baseStandard.AllowSets) {
		t.Errorf("standard preset fields other than deny_sets changed: %+v", standard)
	}
}

// Applying a project overlay never mutates the catalog it is applied to.
func TestProjectOverlay_DoesNotMutateBase(t *testing.T) {
	t.Parallel()
	base := mustEmbedded(t)
	before, _ := base.PermissionPreset("standard")
	beforeDeny := slices.Clone(base.PermissionDenyRules("npx"))

	proj, err := parseUnifiedBytes([]byte(`
permission_deny_rules:
  npx: [Bash(extra *)]
permission_preset_defs:
  standard:
    deny_sets: [npx]
`))
	if err != nil {
		t.Fatal(err)
	}
	ov := &projectOverlay{path: "defaults.yaml", cat: proj,
		sections: []string{"permission_deny_rules", sectionPermissionPresetDefs}}
	if _, errs := applyProjectOverlay(base, ov); len(errs) > 0 {
		t.Fatalf("applyProjectOverlay() errors: %v", errs)
	}

	after, _ := base.PermissionPreset("standard")
	if !slices.Equal(before.DenySets, after.DenySets) {
		t.Error("base preset deny_sets mutated")
	}
	if !slices.Equal(beforeDeny, base.PermissionDenyRules("npx")) {
		t.Error("base npx deny rules mutated")
	}
}

// Empty files and sections that carry no value set nothing, so they load.
func TestProjectOverlay_EmptyInputsAccepted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
	}{
		{"empty file", ""},
		{"comments only", "# project defaults\n"},
		{"empty mapping", "{}\n"},
		{"null section", "tools:\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := loadProject(t, tt.content); err != nil {
				t.Errorf("Load() error: %v", err)
			}
		})
	}
}

// A project file with several violations reports all of them at once.
func TestProjectOverlay_ReportsEveryViolation(t *testing.T) {
	t.Parallel()

	_, err := loadProject(t, `
keep_vars: [GITHUB_TOKEN]
mcp_servers:
  context7:
    command: /tmp/evil
tier_to_compliance:
  full: baseline
`)
	if err == nil {
		t.Fatal("Load() succeeded, want rejection")
	}
	for _, field := range []string{"keep_vars:", "mcp_servers:", "tier_to_compliance.full:"} {
		if !strings.Contains(err.Error(), field) {
			t.Errorf("error = %v, want it to name %s", err, field)
		}
	}
}
