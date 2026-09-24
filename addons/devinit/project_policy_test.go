package devinit

import (
	"bytes"
	"context"
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	qsdevconfig "github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// strictAcmeClient is a client that requires the strict level and allows
// only the context7 MCP server.
func strictAcmeClient() *types.ClientConfig {
	return &types.ClientConfig{
		Name:          "acme",
		SecurityLevel: "strict",
		BlockedMCP:    []string{types.MCPWildcard},
		AllowedMCP:    []string{"context7"},
	}
}

// addClientPolicy edits the committed .qsdev.yaml in dir to declare client.
func addClientPolicy(t *testing.T, dir string, client *types.ClientConfig) {
	t.Helper()
	path := filepath.Join(dir, branding.Get().ConfigFile)
	cfg, err := qsdevconfig.ParseQsdevConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Client = client
	if err := qsdevconfig.WriteProjectConfig(path, *cfg); err != nil {
		t.Fatal(err)
	}
}

func writeLocalConfig(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, branding.Get().LocalConfig), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// mcpServersIn returns the sorted server names of .mcp.json content, or nil
// when there is none.
func mcpServersIn(t *testing.T, content string) []string {
	t.Helper()
	if content == "" {
		return nil
	}
	var doc struct {
		MCPServers map[string]json.RawMessage `json:"mcpServers"`
	}
	if err := json.Unmarshal([]byte(content), &doc); err != nil {
		t.Fatalf("parsing .mcp.json: %v\n%s", err, content)
	}
	return slices.Sorted(maps.Keys(doc.MCPServers))
}

// assertStrictAcmePolicy checks answers carry the strictAcmeClient policy.
func assertStrictAcmePolicy(t *testing.T, a types.WizardAnswers) {
	t.Helper()
	if a.ComplianceLevel != "strict" {
		t.Errorf("ComplianceLevel = %q, want the client's strict", a.ComplianceLevel)
	}
	if !a.Hooks.AuditLog || !a.Hooks.PreCommit {
		t.Errorf("strict hooks missing: %+v", a.Hooks)
	}
	if !slices.Equal(a.MCPPolicy.Blocked, []string{types.MCPWildcard}) {
		t.Errorf("MCPPolicy = %+v, want the client's", a.MCPPolicy)
	}
	if got := mcpServersIn(t, generatedContent(t, a)[".mcp.json"]); !slices.Equal(got, []string{"context7"}) {
		t.Errorf(".mcp.json servers = %v, want only the allowed context7", got)
	}
}

// TestJoin_AppliesCommittedClientPolicy is the regression test for join
// ignoring the committed security floor and client policy: a teammate joining
// a project whose .qsdev.yaml declares a strict client that blocks MCP
// servers must get the strict hooks and only the allowed server, and a local
// override below the floor is reported and not applied.
func TestJoin_AppliesCommittedClientPolicy(t *testing.T) {
	dir := newGoProject(t)
	commitConfig(t, dir, createAnswers(t, dir, "--lang", "go", "--tier", "full"))
	addClientPolicy(t, dir, strictAcmeClient())
	writeLocalConfig(t, dir, "security:\n  level: baseline\n  age_gating: false\n")

	cmd, out := newJoinTestCmd()
	joined, err := buildJoinAnswers(cmd, InitOptions{Quiet: true}, dir)
	if err != nil {
		t.Fatalf("buildJoinAnswers: %v", err)
	}
	assertStrictAcmePolicy(t, joined)
	for _, want := range []string{"security.level: baseline raised to strict", "security.age_gating: false raised to true"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("join output lacks warning %q:\n%s", want, out)
		}
	}
}

// TestJoin_NoOrgDefaultsLayer checks join resolves with an empty
// organization layer: a project without a security block is not raised to
// the built-in enhanced defaults.
func TestJoin_NoOrgDefaultsLayer(t *testing.T) {
	dir := newGoProject(t)
	created := createAnswers(t, dir, "--lang", "go", "--tier", "supply-chain-only")
	cfg := qsdevconfig.AnswersToConfig(created, "test")
	cfg.Security = types.SecurityConfig{}
	if err := qsdevconfig.WriteProjectConfig(filepath.Join(dir, branding.Get().ConfigFile), cfg); err != nil {
		t.Fatal(err)
	}
	cmd, _ := newJoinTestCmd()
	joined, err := buildJoinAnswers(cmd, InitOptions{Quiet: true}, dir)
	if err != nil {
		t.Fatalf("buildJoinAnswers: %v", err)
	}
	if joined.ComplianceLevel != "" {
		t.Errorf("ComplianceLevel = %q, want unset (no org defaults layer)", joined.ComplianceLevel)
	}
}

// TestUpdate_AppliesCommittedClientPolicy checks update regenerates under a
// client policy added to .qsdev.yaml after the project was created.
func TestUpdate_AppliesCommittedClientPolicy(t *testing.T) {
	dir := initLifecycleProject(t)
	addClientPolicy(t, dir, strictAcmeClient())

	var out bytes.Buffer
	answers, err := loadAndRefreshForUpdate(context.Background(), &out, dir)
	if err != nil {
		t.Fatalf("loadAndRefreshForUpdate: %v", err)
	}
	assertStrictAcmePolicy(t, answers)
}

// TestUpdate_UnreadableConfigFailsClosed checks update refuses to regenerate
// when the committed policy cannot be read, rather than dropping it.
func TestUpdate_UnreadableConfigFailsClosed(t *testing.T) {
	dir := initLifecycleProject(t)
	path := filepath.Join(dir, branding.Get().ConfigFile)
	if err := os.WriteFile(path, []byte("version: 2\nclient: [\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if _, err := loadAndRefreshForUpdate(context.Background(), &out, dir); err == nil {
		t.Fatal("expected an error for an unreadable .qsdev.yaml")
	}
}

// TestCreateForce_KeepsCommittedPolicy checks re-creating a project over a
// committed .qsdev.yaml applies and keeps its security floor and client
// policy instead of overwriting them with the flag-derived config.
func TestCreateForce_KeepsCommittedPolicy(t *testing.T) {
	dir := initLifecycleProject(t)
	addClientPolicy(t, dir, strictAcmeClient())

	if out, err := executeInitCmd(t, dir, "--yes", "--force", "--lang", "go", "--tier", "full"); err != nil {
		t.Fatalf("re-init: %v\n%s", err, out)
	}
	cfg, err := qsdevconfig.ParseQsdevConfig(filepath.Join(dir, branding.Get().ConfigFile))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Client == nil || cfg.Client.Name != "acme" || cfg.Security.Level != "strict" {
		t.Errorf("committed policy dropped: client %+v, security %+v", cfg.Client, cfg.Security)
	}
	if got := mcpServersIn(t, readOptionalFile(t, dir, ".mcp.json")); !slices.Equal(got, []string{"context7"}) {
		t.Errorf(".mcp.json servers = %v, want only context7", got)
	}
	assertStrictAcmePolicy(t, loadProjectAnswers(t, dir))
}

func readOptionalFile(t *testing.T, dir, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, rel))
	if os.IsNotExist(err) {
		return ""
	}
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// TestRunJoin_StripsForbiddenServersFromCommittedMcpJson is the end-to-end
// failure scenario: the committed .mcp.json lists servers the committed
// client block forbids; a teammate's join must not keep them.
func TestRunJoin_StripsForbiddenServersFromCommittedMcpJson(t *testing.T) {
	src := initLifecycleProject(t)
	addClientPolicy(t, src, strictAcmeClient())
	if got := mcpServersIn(t, readProjectFile(t, src, ".mcp.json")); !slices.Contains(got, "github") {
		t.Fatalf("fixture .mcp.json lacks github: %v", got)
	}

	dst := cloneCommitted(t, src, branding.Get().ConfigFile, "go.mod", ".mcp.json")
	if out, err := executeInitCmd(t, dst, "--mode", "join", "--yes"); err != nil {
		t.Fatalf("join: %v\n%s", err, out)
	}
	if got := mcpServersIn(t, readProjectFile(t, dst, ".mcp.json")); !slices.Equal(got, []string{"context7"}) {
		t.Errorf("joined .mcp.json servers = %v, want only context7", got)
	}
	settings := readProjectFile(t, dst, ".claude/settings.json")
	if !strings.Contains(settings, "audit-log") {
		t.Errorf("strict client did not enable the audit-log hook:\n%s", settings)
	}
}

// TestLocalGenerationAnswers checks .qsdev.local.yaml reaches only the
// answers generation uses, with its additions applied and its loosening
// overrides dropped, and that invalid local values fail.
func TestLocalGenerationAnswers(t *testing.T) {
	tests := []struct {
		name      string
		committed bool
		local     string
		wantErr   string
		check     func(t *testing.T, base, gen types.WizardAnswers)
	}{
		{
			name:      "adds and tightens",
			committed: true,
			local: "extra_packages: [neovim]\ntools:\n  enabled: [changelog]\n  disabled: [gitleaks]\n" +
				"claude_code:\n  permission_level: minimal\n",
			check: func(t *testing.T, base, gen types.WizardAnswers) {
				t.Helper()
				if !slices.Contains(gen.ExtraPackages, "neovim") || slices.Contains(base.ExtraPackages, "neovim") {
					t.Errorf("ExtraPackages gen %v, base %v; want neovim in gen only", gen.ExtraPackages, base.ExtraPackages)
				}
				if gen.PermissionLevel != "minimal" || base.PermissionLevel == "minimal" {
					t.Errorf("PermissionLevel gen %q, base %q; want minimal in gen only", gen.PermissionLevel, base.PermissionLevel)
				}
				if !gen.EnabledTools["changelog"] || base.EnabledTools["changelog"] {
					t.Errorf("changelog gen %v, base %v; want enabled in gen only", gen.EnabledTools["changelog"], base.EnabledTools["changelog"])
				}
				if !base.EnabledTools["gitleaks"] {
					t.Fatal("fixture does not enable gitleaks, so the tools.disabled case proves nothing")
				}
				if !gen.EnabledTools["gitleaks"] {
					t.Error("a local tools.disabled turned gitleaks off")
				}
			},
		},
		{
			name:      "loosening permission level ignored",
			committed: true,
			local:     "claude_code:\n  permission_level: permissive\n",
			check: func(t *testing.T, base, gen types.WizardAnswers) {
				t.Helper()
				if gen.PermissionLevel != base.PermissionLevel {
					t.Errorf("PermissionLevel = %q, want the committed %q", gen.PermissionLevel, base.PermissionLevel)
				}
			},
		},
		{name: "invalid package fails", committed: true, local: "extra_packages: ['bad;pkg']\n", wantErr: "invalid package name"},
		{
			name:  "no committed config",
			local: "extra_packages: [neovim]\n",
			check: func(t *testing.T, base, gen types.WizardAnswers) {
				t.Helper()
				if len(gen.ExtraPackages) != 0 {
					t.Errorf("ExtraPackages = %v, want none before the project has a .qsdev.yaml", gen.ExtraPackages)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			var base types.WizardAnswers
			if tt.committed {
				dir = initLifecycleProject(t)
				base = loadProjectAnswers(t, dir)
			}
			writeLocalConfig(t, dir, tt.local)
			baseCopy := cloneAnswers(base)

			gen, err := localGenerationAnswers(dir, base)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(base, baseCopy) {
				t.Error("localGenerationAnswers modified the answers it was given")
			}
			tt.check(t, base, gen)
		})
	}
}

// TestUpdateAndJoin_ApplyLocalOverridesWithoutPersisting is the finding's
// failure scenario: extra_packages in .qsdev.local.yaml must reach the
// generated devenv.nix on update and join, but never .qsdev.yaml or the saved
// answers, which the team shares or regenerates from.
func TestUpdateAndJoin_ApplyLocalOverridesWithoutPersisting(t *testing.T) {
	const local = "extra_packages: [neovim]\n"
	assertLocalOnly := func(t *testing.T, dir string) {
		t.Helper()
		if nix := readProjectFile(t, dir, "devenv.nix"); !strings.Contains(nix, "neovim") {
			t.Errorf("devenv.nix lacks the local extra package:\n%s", nix)
		}
		cfg, err := qsdevconfig.ParseQsdevConfig(filepath.Join(dir, branding.Get().ConfigFile))
		if err != nil {
			t.Fatal(err)
		}
		if slices.Contains(cfg.Packages, "neovim") {
			t.Errorf("local package written to %s: %v", branding.Get().ConfigFile, cfg.Packages)
		}
		if a := loadProjectAnswers(t, dir); slices.Contains(a.ExtraPackages, "neovim") {
			t.Errorf("local package saved in the answers: %v", a.ExtraPackages)
		}
	}

	t.Run("update", func(t *testing.T) {
		dir := initLifecycleProject(t)
		writeLocalConfig(t, dir, local)
		if out, err := executeInitCmd(t, dir, "--update"); err != nil {
			t.Fatalf("update: %v\n%s", err, out)
		}
		assertLocalOnly(t, dir)
	})
	t.Run("join", func(t *testing.T) {
		src := initLifecycleProject(t)
		dst := cloneCommitted(t, src, branding.Get().ConfigFile, "go.mod")
		writeLocalConfig(t, dst, local)
		if out, err := executeInitCmd(t, dst, "--mode", "join", "--yes"); err != nil {
			t.Fatalf("join: %v\n%s", err, out)
		}
		assertLocalOnly(t, dst)
	})
}
