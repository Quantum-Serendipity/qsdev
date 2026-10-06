package catalog

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
)

// embeddedSecurityHooks lists the always-on hooks the embedded catalog ships;
// every layer may only add to them.
var embeddedSecurityHooks = []string{
	"ripsecrets", "check-added-large-files", "no-commit-to-branch", "shellcheck", "statix",
}

// An org overlay's unset_vars adds to the stripped credentials; it never
// replaces the built-in list.
func TestMergeCatalogs_UnsetVarsUnion(t *testing.T) {
	t.Parallel()

	base, err := LoadEmbeddedOnly()
	if err != nil {
		t.Fatalf("LoadEmbeddedOnly: %v", err)
	}
	overlay, err := parseUnifiedBytes([]byte("unset_vars: [MY_TEAM_TOKEN, GITHUB_TOKEN]\n"))
	if err != nil {
		t.Fatalf("parse overlay: %v", err)
	}

	got := MergeCatalogs(base, overlay).UnsetVars()
	want := append(slices.Clone(base.UnsetVars()), "MY_TEAM_TOKEN")
	if len(base.UnsetVars()) != 38 {
		t.Fatalf("embedded unset_vars has %d entries, want 38", len(base.UnsetVars()))
	}
	if !slices.Equal(got, want) {
		t.Errorf("UnsetVars() = %v (len %d), want embedded list plus MY_TEAM_TOKEN (len %d)", got, len(got), len(want))
	}
}

// An org overlay's security_hooks adds to the always-on hooks; it never
// replaces the built-in list.
func TestMergeCatalogs_SecurityHooksUnion(t *testing.T) {
	t.Parallel()

	base, err := LoadEmbeddedOnly()
	if err != nil {
		t.Fatalf("LoadEmbeddedOnly: %v", err)
	}
	for _, add := range []string{"check-merge-conflicts", "my-hook"} {
		t.Run(add, func(t *testing.T) {
			t.Parallel()
			overlay := &Catalog{}
			overlay.security.Hooks.Default = []string{add}

			got := MergeCatalogs(base, overlay).SecurityHooks()
			for _, h := range append(slices.Clone(embeddedSecurityHooks), add) {
				if !slices.Contains(got, h) {
					t.Errorf("SecurityHooks() = %v, missing %q", got, h)
				}
			}
			if !slices.Equal(got[:len(base.SecurityHooks())], base.SecurityHooks()) {
				t.Errorf("SecurityHooks() = %v, want the embedded hooks first, in order", got)
			}
		})
	}
}

// An org file may not keep a variable the catalog strips.
func TestLoad_KeepVarsOverlapUnsetVarsRejected(t *testing.T) {
	t.Parallel()

	f := writeUnifiedFile(t, "keep_vars: [PATH, GITHUB_TOKEN]\n")
	_, err := Load(WithOrgConfigFile(f))
	if !errors.Is(err, ErrOverlayLoosens) {
		t.Fatalf("Load() error = %v, want ErrOverlayLoosens", err)
	}
	for _, want := range []string{"GITHUB_TOKEN", f} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Load() error = %q, want it to name %q", err, want)
		}
	}
	if strings.Contains(err.Error(), `"PATH"`) {
		t.Errorf("Load() error = %q, PATH is not stripped and must not be reported", err)
	}
}

// XS-WS2 B6: a hostile org overlay that empties security_hooks and keeps an
// AWS credential fails to load.
func TestLoad_HostileOrgOverlay(t *testing.T) {
	t.Parallel()

	f := writeUnifiedFile(t, "security_hooks: []\nkeep_vars: [AWS_SECRET_ACCESS_KEY]\n")
	_, err := Load(WithOrgConfigFile(f))
	if !errors.Is(err, ErrOverlayLoosens) {
		t.Fatalf("Load() error = %v, want ErrOverlayLoosens", err)
	}
	if !strings.Contains(err.Error(), "AWS_SECRET_ACCESS_KEY") {
		t.Errorf("Load() error = %q, want it to name AWS_SECRET_ACCESS_KEY", err)
	}
}

// G-01: the overlay that used to strip every built-in hook and credential
// now loads only when it is purely additive, and then keeps the floor.
func TestLoad_OrgOverlayOnlyAdds(t *testing.T) {
	t.Parallel()

	f := writeUnifiedFile(t, "security_hooks: [check-merge-conflicts]\nunset_vars: [MY_TEAM_TOKEN]\n")
	cat, err := Load(WithOrgConfigFile(f))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	for _, h := range embeddedSecurityHooks {
		if !slices.Contains(cat.SecurityHooks(), h) {
			t.Errorf("SecurityHooks() = %v, missing %q", cat.SecurityHooks(), h)
		}
	}
	if got := len(cat.UnsetVars()); got != 39 {
		t.Errorf("len(UnsetVars()) = %d, want 39", got)
	}
}

// An org file cannot lower what a compliance level demands, whether by
// dropping a required hook, mapping a tier to a lower level or renumbering a
// built-in level; it may raise and add.
func TestLoad_OrgOverlayCannotLowerCompliance(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		content   string
		wantField string // "" means the file loads
	}{
		{
			name: "drop required hook",
			content: `compliance:
  strict:
    required_pre_commit_hooks: [gitleaks, semgrep, license-compliance]
`,
			wantField: "compliance.strict.required_pre_commit_hooks",
		},
		{
			name: "empty required hooks",
			content: `compliance:
  baseline:
    required_pre_commit_hooks: []
`,
			wantField: "compliance.baseline.required_pre_commit_hooks",
		},
		{
			name:      "map full to a lower level",
			content:   "tier_to_compliance:\n  full: baseline\n",
			wantField: "tier_to_compliance.full",
		},
		{
			name: "renumber built-in level",
			content: `compliance:
  strict:
    order: 0
`,
			wantField: "compliance.strict.order",
		},
		{
			name: "map to a new lower-order level",
			content: `compliance:
  lax:
    order: -1
    required_pre_commit_hooks: [ripsecrets]
tier_to_compliance:
  standard: lax
`,
			wantField: "tier_to_compliance.standard",
		},
		{
			// A higher order alone is not a raise: the level must demand
			// at least what the tier's built-in level does.
			name: "map to a new higher-order level with fewer required hooks",
			content: `compliance:
  paranoid:
    order: 9
    age_gating_threshold_hours: 0
    script_blocking: false
    required_pre_commit_hooks: [ripsecrets]
    mcp_server_policy: allow-list
    claude_permission_level: standard
hook_tier_order: [baseline, enhanced, strict, paranoid]
security_levels: [baseline, enhanced, strict, paranoid]
hook_tiers:
  paranoid: []
tier_to_compliance:
  full: paranoid
`,
			wantField: "tier_to_compliance.full",
		},
		{
			name:      "shorten a built-in age gate",
			content:   "compliance:\n  strict:\n    age_gating_threshold_hours: 0\n",
			wantField: "compliance.strict.age_gating_threshold_hours",
		},
		{
			name:      "turn off script blocking",
			content:   "compliance:\n  strict:\n    script_blocking: false\n",
			wantField: "compliance.strict.script_blocking",
		},
		{
			name:      "turn off the Claude audit log",
			content:   "compliance:\n  strict:\n    claude_audit_log: false\n",
			wantField: "compliance.strict.claude_audit_log",
		},
		{
			name:      "turn off license scanning",
			content:   "compliance:\n  strict:\n    license_scanning: false\n",
			wantField: "compliance.strict.license_scanning",
		},
		{
			name:      "loosen the Claude permission preset",
			content:   "compliance:\n  strict:\n    claude_permission_level: standard\n",
			wantField: "compliance.strict.claude_permission_level",
		},
		{
			name: "raise and add",
			content: `compliance:
  enhanced:
    required_pre_commit_hooks: [ripsecrets, gitleaks, semgrep, license-compliance]
  sovereign:
    order: 3
    age_gating_threshold_hours: 720
    script_blocking: true
    required_pre_commit_hooks: [ripsecrets, gitleaks, semgrep, license-compliance]
    mcp_server_policy: explicit-only
    claude_permission_level: minimal
    claude_audit_log: true
    sbom_policy: every-build
    license_scanning: true
tier_to_compliance:
  standard: strict
  full: sovereign
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := writeUnifiedFile(t, tt.content)
			cat, err := Load(WithOrgConfigFile(f))
			if tt.wantField == "" {
				if err != nil {
					t.Fatalf("Load() error = %v, want the raising overlay to load", err)
				}
				if got := cat.TierCompliance("full"); got != "sovereign" {
					t.Errorf("tier_to_compliance.full = %q, want sovereign", got)
				}
				return
			}
			if !errors.Is(err, ErrOverlayLoosens) {
				t.Fatalf("Load() error = %v, want ErrOverlayLoosens", err)
			}
			if !strings.Contains(err.Error(), tt.wantField) {
				t.Errorf("Load() error = %q, want it to name %q", err, tt.wantField)
			}
		})
	}
}

// G-V02: the committed project policy applies after the org file, so an
// org security_hooks list cannot erase the project's additions.
func TestLoad_UserLayerKeepsProjectSecurityHooks(t *testing.T) {
	t.Parallel()

	projFile := writeUnifiedFile(t, `security_hooks: [my-hook]
custom_hooks:
  - id: my-hook
    name: my hook
    entry: "true"
    language: system
    stages: [pre-commit]
`)
	orgFile := writeUnifiedFile(t, "security_hooks: [ripsecrets, check-added-large-files]\n")

	cat, err := Load(WithOrgConfigFile(orgFile), WithProjectConfigFile(projFile))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	for _, h := range append(slices.Clone(embeddedSecurityHooks), "my-hook") {
		if !slices.Contains(cat.SecurityHooks(), h) {
			t.Errorf("SecurityHooks() = %v, missing %q", cat.SecurityHooks(), h)
		}
	}
}

// G-01: an org file cannot drop an always-on hook from devenv.nix by tiering
// it above a security level, whether by moving it to a higher built-in tier
// or to a new tier, or by reordering the built-in tiers.
func TestLoad_OrgOverlayCannotTierAwayAlwaysOnHooks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		content   string
		wantField string
	}{
		{
			name: "move always-on hooks to strict",
			content: `hook_tiers:
  baseline: [ripsecrets, gitleaks]
  strict: [check-added-large-files, no-commit-to-branch, shellcheck, lock-file-audit, nix-secrets-check]
`,
			wantField: "hook_tiers.strict",
		},
		{
			name: "move always-on hooks to a new tier",
			content: `hook_tier_order: [baseline, enhanced, strict, paranoid]
security_levels: [baseline, enhanced, strict, paranoid]
hook_tiers:
  baseline: [ripsecrets, gitleaks]
  paranoid: [check-added-large-files, no-commit-to-branch, shellcheck, lock-file-audit, nix-secrets-check]
`,
			wantField: "hook_tiers.paranoid",
		},
		{
			name: "move an enhanced always-on hook to strict",
			content: `hook_tiers:
  enhanced: [gofmt]
  strict: [statix]
`,
			wantField: "hook_tiers.strict",
		},
		{
			name: "reorder the built-in tiers",
			content: `hook_tier_order: [strict, enhanced, baseline]
security_levels: [strict, enhanced, baseline]
`,
			wantField: "hook_tier_order",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := writeUnifiedFile(t, tt.content)
			_, err := Load(WithOrgConfigFile(f))
			if !errors.Is(err, ErrOverlayLoosens) {
				t.Fatalf("Load() error = %v, want ErrOverlayLoosens", err)
			}
			if !strings.Contains(err.Error(), tt.wantField) {
				t.Errorf("Load() error = %q, want it to name %q", err, tt.wantField)
			}
		})
	}

	// Adding a tier above the built-in ones, and tiering a non-security
	// hook higher, still loads.
	f := writeUnifiedFile(t, `hook_tier_order: [baseline, enhanced, strict, paranoid]
security_levels: [baseline, enhanced, strict, paranoid]
hook_tiers:
  enhanced: []
  paranoid: [gofmt]
`)
	if _, err := Load(WithOrgConfigFile(f)); err != nil && errors.Is(err, ErrOverlayLoosens) {
		t.Errorf("Load() error = %v, want an additive tier layout to pass the floor", err)
	}
}

// The committed project file is judged against the built-in catalog, so an
// org file never makes it fail to load (which would drop the whole org
// file, see loadDefault), and the stricter of the two settings applies.
func TestLoad_ProjectOverlayJudgedAgainstBuiltin(t *testing.T) {
	t.Parallel()

	t.Run("org raises a tier above the project's raise", func(t *testing.T) {
		t.Parallel()
		org := writeUnifiedFile(t, "unset_vars: [MY_TEAM_TOKEN]\ntier_to_compliance:\n  supply-chain-only: strict\n")
		proj := writeUnifiedFile(t, "tier_to_compliance:\n  supply-chain-only: enhanced\n")
		cat, err := Load(WithOrgConfigFile(org), WithProjectConfigFile(proj))
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if got := cat.TierCompliance("supply-chain-only"); got != "strict" {
			t.Errorf("TierCompliance(supply-chain-only) = %q, want the stricter org level strict", got)
		}
		if !slices.Contains(cat.UnsetVars(), "MY_TEAM_TOKEN") {
			t.Error("UnsetVars() lost the org file's MY_TEAM_TOKEN")
		}
	})

	t.Run("project raises a tier above the org's raise", func(t *testing.T) {
		t.Parallel()
		org := writeUnifiedFile(t, "tier_to_compliance:\n  supply-chain-only: enhanced\n")
		proj := writeUnifiedFile(t, "tier_to_compliance:\n  supply-chain-only: strict\n")
		cat, err := Load(WithOrgConfigFile(org), WithProjectConfigFile(proj))
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if got := cat.TierCompliance("supply-chain-only"); got != "strict" {
			t.Errorf("TierCompliance(supply-chain-only) = %q, want the project's strict", got)
		}
	})

	t.Run("org and project define the same custom hook", func(t *testing.T) {
		t.Parallel()
		hook := `custom_hooks:
  - id: team-hook
    name: %s
    entry: "true"
    language: system
    stages: [pre-commit]
`
		org := writeUnifiedFile(t, fmt.Sprintf(hook, "org hook"))
		proj := writeUnifiedFile(t, fmt.Sprintf(hook, "project hook")+"security_hooks: [team-hook]\n")
		cat, err := Load(WithOrgConfigFile(org), WithProjectConfigFile(proj))
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		var names []string
		for _, h := range cat.CustomHooks() {
			if h.ID == "team-hook" {
				names = append(names, h.Name)
			}
		}
		if !slices.Equal(names, []string{"org hook"}) {
			t.Errorf("team-hook definitions = %v, want only the org file's", names)
		}
		if !slices.Contains(cat.SecurityHooks(), "team-hook") {
			t.Errorf("SecurityHooks() = %v, missing the project's team-hook", cat.SecurityHooks())
		}
	})

	t.Run("project still cannot redefine a built-in hook", func(t *testing.T) {
		t.Parallel()
		org := writeUnifiedFile(t, "unset_vars: [MY_TEAM_TOKEN]\n")
		proj := writeUnifiedFile(t, "custom_hooks:\n  - id: ripsecrets\n    entry: \"true\"\n    language: system\n")
		_, err := Load(WithOrgConfigFile(org), WithProjectConfigFile(proj))
		if !errors.Is(err, ErrProjectOverlayRejected) {
			t.Errorf("Load() error = %v, want ErrProjectOverlayRejected", err)
		}
	})
}
