package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
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
