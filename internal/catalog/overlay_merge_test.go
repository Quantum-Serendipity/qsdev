package catalog

import (
	"strings"
	"testing"
)

func TestLoadWithOrgOverride_CommentOnlyKeepsDocsCorpus(t *testing.T) {
	t.Parallel()

	base, err := LoadEmbeddedOnly()
	if err != nil {
		t.Fatalf("LoadEmbeddedOnly: %v", err)
	}
	f := writeUnifiedFile(t, "# every line commented out\n# tools:\n#   gitleaks: {}\n")

	cat, err := Load(WithOrgConfigFile(f))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got, want := len(cat.DevDocsSlugs()), len(base.DevDocsSlugs()); got != want || want == 0 {
		t.Errorf("DevDocsSlugs() len = %d, want %d (embedded)", got, want)
	}
	if cat.DocsCorpus().DevDocsBaseURL != base.DocsCorpus().DevDocsBaseURL {
		t.Errorf("DevDocsBaseURL = %q, want embedded %q", cat.DocsCorpus().DevDocsBaseURL, base.DocsCorpus().DevDocsBaseURL)
	}
}

func TestLoadWithOrgOverride_DocsCorpusMergesPerLanguage(t *testing.T) {
	t.Parallel()

	f := writeUnifiedFile(t, `
docs_corpus:
  devdocs_slugs:
    go: ["go", "gorm"]
    elixir: ["elixir"]
`)
	cat, err := Load(WithOrgConfigFile(f))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	slugs := cat.DevDocsSlugs()
	if got := strings.Join(slugs["go"], ","); got != "go,gorm" {
		t.Errorf("go slugs = %q, want overlay value", got)
	}
	if len(slugs["elixir"]) != 1 {
		t.Errorf("elixir slugs = %v, want overlay entry added", slugs["elixir"])
	}
	if len(slugs["python"]) == 0 {
		t.Error("python slugs dropped; base entries must survive a partial docs_corpus override")
	}
}

func TestLoadWithOrgOverride_PartialEntryKeepsOtherFields(t *testing.T) {
	t.Parallel()

	base, err := LoadEmbeddedOnly()
	if err != nil {
		t.Fatalf("LoadEmbeddedOnly: %v", err)
	}
	baseTool, _ := base.Tool("gitleaks")
	baseStandard, _ := base.TierDef("standard")
	baseStrict, _ := base.ComplianceLevel("strict")
	if !baseStrict.ScriptBlocking {
		t.Fatal("test precondition: embedded strict compliance must enable script blocking")
	}

	f := writeUnifiedFile(t, `
tools:
  gitleaks:
    description: my desc
tiers:
  standard:
    description: Org standard
compliance:
  strict:
    script_blocking: false
`)
	cat, err := Load(WithOrgConfigFile(f))
	if err != nil {
		t.Fatalf("Load with partial overrides: %v", err)
	}

	tool, _ := cat.Tool("gitleaks")
	if tool.Description != "my desc" {
		t.Errorf("gitleaks description = %q, want overlay value", tool.Description)
	}
	if tool.NixPackage != baseTool.NixPackage || tool.Category != baseTool.Category || len(tool.OwnedFiles) != len(baseTool.OwnedFiles) {
		t.Errorf("gitleaks lost base fields: got %+v, base %+v", tool, baseTool)
	}

	standard, _ := cat.TierDef("standard")
	if standard.Description != "Org standard" || standard.Order != baseStandard.Order ||
		standard.DefaultPermissionPreset != baseStandard.DefaultPermissionPreset {
		t.Errorf("standard tier = %+v, want overlay description on base fields %+v", standard, baseStandard)
	}

	strict, _ := cat.ComplianceLevel("strict")
	if strict.ScriptBlocking {
		t.Error("explicit script_blocking: false in the overlay must be honoured")
	}
	if strict.AgeGatingThresholdHours != baseStrict.AgeGatingThresholdHours || len(strict.RequiredPreCommitHooks) == 0 {
		t.Errorf("strict compliance lost base fields: %+v", strict)
	}
}

func TestMergeCatalogs_PartialEntryDoesNotMutateBase(t *testing.T) {
	t.Parallel()

	base, err := LoadEmbeddedOnly()
	if err != nil {
		t.Fatalf("LoadEmbeddedOnly: %v", err)
	}
	before, _ := base.TierDef("full")
	if before.Security == nil || before.Security.Level == "strict" {
		t.Fatalf("test precondition: full tier security = %+v", before.Security)
	}
	wantLevel := before.Security.Level

	overlay, err := parseUnifiedBytes([]byte("tiers:\n  full:\n    security:\n      level: strict\n"))
	if err != nil {
		t.Fatalf("parseUnifiedBytes: %v", err)
	}
	merged := MergeCatalogs(base, overlay)

	full, _ := merged.TierDef("full")
	if full.Security == nil || full.Security.Level != "strict" {
		t.Errorf("merged full security = %+v, want level strict", full.Security)
	}
	if full.ClaudeCode == nil || len(full.ClaudeCode.MCPServers) == 0 {
		t.Errorf("merged full tier lost claude_code: %+v", full.ClaudeCode)
	}
	after, _ := base.TierDef("full")
	if after.Security.Level != wantLevel {
		t.Errorf("base full tier mutated: security level %q, want %q", after.Security.Level, wantLevel)
	}
}

func TestLoadWithOrgOverride_RejectsInvalidFiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name:    "misspelled top-level key",
			content: "permision_deny_rules:\n  x: [a]\n",
			wantErr: "permision_deny_rules",
		},
		{
			name:    "misspelled nested key",
			content: "tools:\n  gitleaks:\n    nix_pakage: foo\n",
			wantErr: "nix_pakage",
		},
		{
			name:    "second YAML document",
			content: "tools: {}\n---\npermission_all_deny_sets: []\n",
			wantErr: "multiple YAML documents",
		},
		{
			name:    "third document after empty one",
			content: "tools: {}\n---\n---\npermission_all_deny_sets: []\n",
			wantErr: "multiple YAML documents",
		},
		{
			name:    "unknown all-deny set",
			content: "permission_all_deny_sets: [supply-chian]\n",
			wantErr: `unknown deny set "supply-chian"`,
		},
		{
			name:    "unknown supply-chain deny set",
			content: "permission_supply_chain_deny_sets: [npx, nixx]\n",
			wantErr: `unknown deny set "nixx"`,
		},
		{
			name:    "unknown package ask set",
			content: "permission_package_ask_sets: [pythn]\n",
			wantErr: `unknown ask set "pythn"`,
		},
		{
			name:    "duplicate preset strictness",
			content: "permission_preset_defs:\n  permissive:\n    strictness: 3\n",
			wantErr: `strictness 3 is also used by preset "minimal"`,
		},
		{
			name:    "negative preset strictness",
			content: "permission_preset_defs:\n  custom:\n    strictness: -1\n",
			wantErr: "strictness must not be negative",
		},
		{
			name:    "unknown preset allow set",
			content: "permission_preset_defs:\n  standard:\n    allow_sets: [nope]\n",
			wantErr: `unknown allow set "nope"`,
		},
		{
			name:    "unknown tier MCP server",
			content: "tiers:\n  full:\n    claude_code:\n      mcp_servers: [ghub]\n",
			wantErr: `unknown MCP server "ghub"`,
		},
		{
			name:    "unknown default MCP server",
			content: "default_mcp_servers: [ctx7]\n",
			wantErr: `unknown MCP server "ctx7"`,
		},
		{
			name:    "unknown compliance permission preset",
			content: "compliance:\n  strict:\n    claude_permission_level: restricted\n",
			wantErr: `unknown permission preset "restricted"`,
		},
		{
			name:    "unknown compliance hook",
			content: "compliance:\n  strict:\n    required_pre_commit_hooks: [ripsecretz]\n",
			wantErr: `unknown hook or tool "ripsecretz"`,
		},
		{
			name:    "duplicate tier order",
			content: "tiers:\n  paranoid:\n    order: 2\n    description: x\n",
			wantErr: "tier orders must be unique",
		},
		{
			name:    "renumbered built-in tier",
			content: "tiers:\n  full:\n    order: 30\n",
			wantErr: "order of built-in tier cannot change",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := writeUnifiedFile(t, tt.content)
			_, err := Load(WithOrgConfigFile(f))
			if err == nil {
				t.Fatalf("Load() = nil error, want error containing %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Load() error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestLoadWithOrgOverride_AddsTierWithNewOrder(t *testing.T) {
	t.Parallel()

	f := writeUnifiedFile(t, "tiers:\n  paranoid:\n    order: 4\n    description: Beyond full\n    default_permission_preset: minimal\n")
	cat, err := Load(WithOrgConfigFile(f))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	order := cat.TierOrder()
	if got := strings.Join(order, ","); got != "supply-chain-only,standard,full,paranoid" {
		t.Errorf("TierOrder() = %s", got)
	}
}

func TestTierOrder_TiesBrokenByName(t *testing.T) {
	t.Parallel()

	c := &Catalog{}
	c.tiers.Tiers = map[string]TierDef{
		"b": {Order: 1}, "a": {Order: 1}, "c": {Order: 0},
	}
	for range 20 {
		if got := strings.Join(c.TierOrder(), ","); got != "c,a,b" {
			t.Fatalf("TierOrder() = %s, want c,a,b", got)
		}
	}
}

func TestParseUnifiedBytes_AcceptsEmptyDocuments(t *testing.T) {
	t.Parallel()

	for _, content := range []string{
		"",
		"# only comments\n",
		"# header\n---\ntools: {}\n",
		"tools: {}\n---\n",
		"tools: {}\n---\n# trailing comment\n",
	} {
		if _, err := parseUnifiedBytes([]byte(content)); err != nil {
			t.Errorf("parseUnifiedBytes(%q) = %v, want no error", content, err)
		}
	}
}

// The tier-profile sections were removed because nothing read them. A
// defaults file from an older template may still set them; they must be
// dropped without making the rest of the file (and its real overrides) fail.
func TestParseUnifiedBytes_IgnoresRemovedProfileSections(t *testing.T) {
	t.Parallel()

	const rest = "default_tier: full\n"
	tests := []struct {
		name    string
		content string
	}{
		{"profiles", "profiles:\n  full:\n    tier: full\n" + rest},
		{"profile_aliases", "profile_aliases:\n  startup-fast: standard\n" + rest},
		{"both, with comments", "# header\nprofiles: {}\nprofile_aliases: {}\n" + rest},
		{"last section, trailing empty document", rest + "profiles:\n  full:\n    tier: full\n---\n"},
		{"flow-style root", "{profiles: {full: {tier: full}}, default_tier: full}\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cat, err := parseUnifiedBytes([]byte(tt.content))
			if err != nil {
				t.Fatalf("parseUnifiedBytes(%q) error = %v, want nil", tt.content, err)
			}
			if got := cat.derivations.DefaultTier; got != "full" {
				t.Errorf("default_tier = %q, want full (the rest of the file must still apply)", got)
			}
		})
	}
}

// Dropping removed sections must not weaken strict parsing of the rest.
func TestParseUnifiedBytes_RemovedSectionsKeepStrictness(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{"misspelled key", "profiles: {}\npermision_deny_rules: {}\n", "permision_deny_rules"},
		{"second document", "profiles: {}\n---\ntools: {}\n", "multiple YAML documents"},
		// The error must point at the misspelled key's line in the file as
		// written, not in a copy with the removed section taken out.
		{"line numbers kept", "profiles:\n  full:\n    tier: full\n\ndefault_tier: full\npermision_deny_rules: {}\n", "line 6"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := parseUnifiedBytes([]byte(tt.content))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("parseUnifiedBytes(%q) error = %v, want it to contain %q", tt.content, err, tt.wantErr)
			}
		})
	}
}
