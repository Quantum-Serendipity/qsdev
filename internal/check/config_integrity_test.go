package check

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestCheckConfigIntegrity_ValidConfig(t *testing.T) {
	ctx := CheckContext{
		QsdevConfig: &types.QsdevConfig{
			Version: 1,
			Languages: []types.LanguageConfig{
				{Name: "go", Version: "1.24"},
			},
			Services: []types.ServiceConfig{
				{Name: "postgres"},
			},
			Profile: "go-web",
		},
		ProfileNames: []string{"go-web", "ts-fullstack"},
		ToolNames:    []string{"safety-block", "pre-commit"},
	}

	results := CheckConfigIntegrity(ctx)

	// Should have a passing config_exists check.
	var configExists *CheckResult
	for i := range results {
		if results[i].Name == "config_exists" {
			configExists = &results[i]
			break
		}
	}

	if configExists == nil {
		t.Fatal("expected config_exists result")
		return
	}
	if configExists.Status != StatusPass {
		t.Errorf("config_exists.Status = %s, want %s", configExists.Status, StatusPass)
	}

	// No validation failures expected.
	for _, r := range results {
		if r.Status == StatusFail {
			t.Errorf("unexpected failure: %s: %s", r.Name, r.Message)
		}
	}
}

func TestCheckConfigIntegrity_InvalidLanguage(t *testing.T) {
	ctx := CheckContext{
		QsdevConfig: &types.QsdevConfig{
			Version: 1,
			Languages: []types.LanguageConfig{
				{Name: "cobol"},
			},
		},
	}

	results := CheckConfigIntegrity(ctx)

	hasFail := false
	for _, r := range results {
		if r.Status == StatusFail && r.Name == "config_validation" {
			hasFail = true
			break
		}
	}

	if !hasFail {
		t.Error("expected a config_validation failure for unknown language")
	}
}

func TestCheckConfigIntegrity_NoConfig(t *testing.T) {
	ctx := CheckContext{}

	results := CheckConfigIntegrity(ctx)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != StatusFail {
		t.Errorf("Status = %s, want %s", results[0].Status, StatusFail)
	}
	if results[0].Severity != SeverityCritical {
		t.Errorf("Severity = %s, want %s", results[0].Severity, SeverityCritical)
	}
}

func TestCheckConfigIntegrity_InvalidProfile(t *testing.T) {
	ctx := CheckContext{
		QsdevConfig: &types.QsdevConfig{
			Version: 1,
			Profile: "unknown-profile",
		},
		ProfileNames: []string{"go-web", "ts-fullstack"},
	}

	results := CheckConfigIntegrity(ctx)

	hasFail := false
	for _, r := range results {
		if r.Status == StatusFail {
			hasFail = true
			break
		}
	}

	if !hasFail {
		t.Error("expected a validation failure for unknown profile")
	}
}

// Regression: init wrote the infra profile under `profile`, which check
// validated against project-type profiles, so `qsdev init --infra-profile
// enterprise` followed by `qsdev check` failed. Each key now has its own
// registry.
func TestCheckConfigIntegrity_ProfileKeys(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		cfg      types.QsdevConfig
		wantFail string // substring of the failing message, "" for none
		wantPass []string
	}{
		{"infra profile init writes", types.QsdevConfig{InfraProfile: "enterprise"}, "", []string{"infra_profile_valid"}},
		{"both profiles", types.QsdevConfig{Profile: "go-web", InfraProfile: "startup-github"}, "",
			[]string{"profile_valid", "infra_profile_valid"}},
		{"unknown infra profile", types.QsdevConfig{InfraProfile: "go-web"}, "infra_profile", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := tt.cfg
			cfg.Version = types.ConfigVersionCurrent
			results := CheckConfigIntegrity(CheckContext{QsdevConfig: &cfg, ProfileNames: []string{"go-web", "ts-fullstack"}})
			passed := map[string]bool{}
			var fails []string
			for _, r := range results {
				switch r.Status {
				case StatusPass:
					passed[r.Name] = true
				case StatusFail:
					fails = append(fails, r.Message)
				}
			}
			if tt.wantFail == "" && len(fails) != 0 {
				t.Errorf("unexpected failures: %v", fails)
			}
			if tt.wantFail != "" && (len(fails) != 1 || !strings.Contains(fails[0], tt.wantFail)) {
				t.Errorf("failures = %v, want one mentioning %q", fails, tt.wantFail)
			}
			for _, name := range tt.wantPass {
				if !passed[name] {
					t.Errorf("missing %s pass result", name)
				}
			}
		})
	}
}

func TestCheckConfigIntegrity_InvalidService(t *testing.T) {
	ctx := CheckContext{
		QsdevConfig: &types.QsdevConfig{
			Version: 1,
			Services: []types.ServiceConfig{
				{Name: "oracle"},
			},
		},
	}

	results := CheckConfigIntegrity(ctx)

	hasFail := false
	for _, r := range results {
		if r.Status == StatusFail {
			hasFail = true
			break
		}
	}

	if !hasFail {
		t.Error("expected a validation failure for unknown service")
	}
}

// TestCheckConfigIntegrity_ParseErrorIsNotNotFound verifies that a config that
// exists but fails to parse is reported with its real error rather than as
// "not found", and that dependent checks explain why they are skipped.
func TestCheckConfigIntegrity_ParseErrorIsNotNotFound(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		content     string // "" means the file does not exist
		wantName    string
		wantMessage string
		wantSkip    string
	}{
		{name: "missing", wantName: "config_exists", wantMessage: "not found", wantSkip: "found; skipping"},
		{name: "newer schema", content: fmt.Sprintf("version: %d\n", types.ConfigVersionMax+1), wantName: "config_parse", wantMessage: "newer than this binary", wantSkip: "could not be parsed"},
		{name: "unknown key", content: "version: 1\nsecurity:\n  script_blockng: true\n", wantName: "config_parse", wantMessage: "script_blockng", wantSkip: "could not be parsed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), ".qsdev.yaml")
			if tt.content != "" {
				if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			cfg, err := config.ParseQsdevConfig(path)
			ctx := CheckContext{QsdevConfig: cfg, ConfigErr: err}

			results := CheckConfigIntegrity(ctx)
			if len(results) != 1 {
				t.Fatalf("expected 1 result, got %d", len(results))
			}
			r := results[0]
			if r.Name != tt.wantName || r.Status != StatusFail || r.Severity != SeverityCritical {
				t.Errorf("got %s %s/%s, want %s fail/critical", r.Name, r.Status, r.Severity, tt.wantName)
			}
			if !strings.Contains(r.Message, tt.wantMessage) {
				t.Errorf("Message = %q, want it to contain %q", r.Message, tt.wantMessage)
			}

			for _, dep := range [][]CheckResult{CheckRequiredTools(ctx), CheckSecurityHardening(ctx), CheckBinaryCompatibility(ctx)} {
				if len(dep) != 1 || dep[0].Status != StatusSkip || !strings.Contains(dep[0].Message, tt.wantSkip) {
					t.Errorf("dependent check = %+v, want skip mentioning %q", dep, tt.wantSkip)
				}
			}
		})
	}
}

// TestCheckConfigIntegrity_MCPDisabledTools is the F228 regression: an MCP
// tool disabled through mcp.disabled_tools passes `qsdev check`, while a
// misspelled one, or an MCP tool placed in the catalog's tools.disabled, fails
// it.
func TestCheckConfigIntegrity_MCPDisabledTools(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		cfg      types.QsdevConfig
		wantFail string // substring of the failing message, "" for none
	}{
		{"mcp tool disabled", types.QsdevConfig{MCP: types.MCPConfig{DisabledTools: []string{"qsdev_nix_run"}}}, ""},
		{"misspelled mcp tool", types.QsdevConfig{MCP: types.MCPConfig{DisabledTools: []string{"qsdev_nixrun"}}},
			"mcp.disabled_tools"},
		{"mcp tool in tools.disabled", types.QsdevConfig{Tools: types.ToolsConfig{Disabled: []string{"qsdev_nix_run"}}},
			"tools.disabled"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := tt.cfg
			cfg.Version = types.ConfigVersionCurrent
			results := CheckConfigIntegrity(CheckContext{
				QsdevConfig:  &cfg,
				ToolNames:    []string{"gitleaks", "safety-block"},
				MCPToolNames: []string{"qsdev_nix_run", "qsdev_status"},
			})
			var fails []string
			for _, r := range results {
				if r.Status == StatusFail {
					fails = append(fails, r.Message)
				}
			}
			if tt.wantFail == "" && len(fails) != 0 {
				t.Errorf("unexpected failures: %v", fails)
			}
			if tt.wantFail != "" && (len(fails) != 1 || !strings.Contains(fails[0], tt.wantFail)) {
				t.Errorf("failures = %v, want one mentioning %q", fails, tt.wantFail)
			}
		})
	}
}

func TestCheckConfigIntegrity_InvalidRegistryProxyPath(t *testing.T) {
	t.Parallel()
	yaml := "version: 1\ninfrastructure:\n  registry_proxy: https://artifactory.corp.io\n" +
		"  registry_proxy_paths:\n    npm: \"@attacker.io/npm/\"\n"
	cfg, err := config.ParseQsdevConfigBytes([]byte(yaml))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	results := CheckConfigIntegrity(CheckContext{QsdevConfig: cfg})

	for _, r := range results {
		if r.Name == "config_validation" && r.Status == StatusFail &&
			strings.Contains(r.Message, "infrastructure.registry_proxy_paths.npm") {
			return
		}
	}
	t.Errorf("expected a config_validation failure naming infrastructure.registry_proxy_paths.npm, got %+v", results)
}

func TestCatalogLoadFailure(t *testing.T) {
	t.Parallel()

	loadErr := errors.New("project defaults may only add or tighten: keep_vars")
	got := CatalogLoadFailure(loadErr)

	app := branding.Get().AppName
	want := CheckResult{
		Category: CategoryConfigIntegrity,
		Name:     "config_catalog",
		Status:   StatusFail,
		Severity: SeverityCritical,
	}
	if got.Category != want.Category || got.Name != want.Name || got.Status != want.Status || got.Severity != want.Severity {
		t.Errorf("CatalogLoadFailure = %s/%s/%s/%s, want %s/%s/%s/%s",
			got.Category, got.Name, got.Status, got.Severity,
			want.Category, want.Name, want.Status, want.Severity)
	}
	if wantMsg := "loading " + app + " defaults: " + loadErr.Error(); got.Message != wantMsg {
		t.Errorf("Message = %q, want %q", got.Message, wantMsg)
	}
	if !strings.Contains(got.Remediation, app+" defaults validate") {
		t.Errorf("Remediation %q does not name '%s defaults validate'", got.Remediation, app)
	}
}

// TestToolRegistryFailure pins that a registry build error after the catalog
// loaded is reported in the config_catalog slot without the defaults-file
// wording, so its cause is not mislabeled as a rejected defaults file.
func TestToolRegistryFailure(t *testing.T) {
	t.Parallel()

	buildErr := errors.New(`tool "x": unknown ownership "weird"`)
	got := ToolRegistryFailure(buildErr)

	if got.Category != CategoryConfigIntegrity || got.Name != "config_catalog" ||
		got.Status != StatusFail || got.Severity != SeverityCritical {
		t.Errorf("ToolRegistryFailure = %s/%s/%s/%s, want config_integrity/config_catalog/fail/critical",
			got.Category, got.Name, got.Status, got.Severity)
	}
	if want := "building tool registry: " + buildErr.Error(); got.Message != want {
		t.Errorf("Message = %q, want %q", got.Message, want)
	}
	if strings.Contains(got.Message, "defaults:") {
		t.Errorf("Message %q blames the defaults file", got.Message)
	}
	if app := branding.Get().AppName; !strings.Contains(got.Remediation, app+" report bug") {
		t.Errorf("Remediation %q does not name '%s report bug'", got.Remediation, app)
	}
}
