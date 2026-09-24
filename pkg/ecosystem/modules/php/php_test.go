package php_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/php"
)

// Compile-time interface compliance check.
var _ ecosystem.EcosystemModule = (*php.Module)(nil)

func TestModuleIdentity(t *testing.T) {
	ecosystem.AssertModuleIdentity(t, &php.Module{}, "php", "PHP", 2)
}

func TestDetect_ComposerJsonPresent(t *testing.T) {
	dir := t.TempDir()
	composerJSON := `{"require": {"php": ">=8.2"}}`
	if err := os.WriteFile(filepath.Join(dir, "composer.json"), []byte(composerJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	m := &php.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true when composer.json is present")
	}
	if result.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("Confidence = %v, want ConfidenceCertain", result.Confidence)
	}
	if len(result.Evidence) < 1 {
		t.Fatal("expected at least one evidence entry")
	}
	found := false
	for _, e := range result.Evidence {
		if strings.Contains(e, "composer.json") {
			found = true
		}
	}
	if !found {
		t.Error("evidence should mention composer.json")
	}
}

func TestDetect_ComposerJsonVersionExtracted(t *testing.T) {
	dir := t.TempDir()
	composerJSON := `{"require": {"php": ">=8.2"}}`
	if err := os.WriteFile(filepath.Join(dir, "composer.json"), []byte(composerJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	m := &php.Module{}
	result := m.Detect(dir)

	// ">=8.2" resolves to the newest supported series, not its lower bound.
	if result.SuggestedConfig.Version != "8.5" {
		t.Errorf("Version = %q, want %q", result.SuggestedConfig.Version, "8.5")
	}
}

func TestDetect_ComposerConstraintResolution(t *testing.T) {
	t.Parallel()

	tests := []struct {
		constraint string
		want       string
	}{
		{"^8.1", "8.5"},
		{">=8.1", "8.5"},
		{">= 8.1, <8.4", "8.3"},
		{"~8.2.0", "8.2"},
		{"8.3.*", "8.3"},
		{"^7.4 || ^8.0", "8.5"},
		{"^7.4|^8.2", "8.5"},
		{"8.4.1", "8.4"},
		{">=8.2 <8.3", "8.2"},
		{"^8.2@dev", "8.5"},
		{">8.3.5 <8.3.7", "8.3"},
		// Only end-of-life series satisfy these: no version is suggested
		// rather than one that fails devenv.nix evaluation.
		{"~8.1.0", ""},
		{"^7.4", ""},
		{"8.1", ""},
		// Unparseable constraints suggest nothing instead of guessing.
		{"not-a-version", ""},
	}
	for _, tt := range tests {
		t.Run(tt.constraint, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			composerJSON := `{"require": {"php": "` + tt.constraint + `"}}`
			if err := os.WriteFile(filepath.Join(dir, "composer.json"), []byte(composerJSON), 0o644); err != nil {
				t.Fatal(err)
			}
			got := (&php.Module{}).Detect(dir).SuggestedConfig.Version
			if got != tt.want {
				t.Errorf("Detect(require.php=%q).Version = %q, want %q", tt.constraint, got, tt.want)
			}
		})
	}
}

func TestDetect_ComposerLockOnly(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "composer.lock"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := &php.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true when composer.lock is present")
	}
	// composer.lock alone gives Probable, not Certain.
	if result.Confidence != ecosystem.ConfidenceProbable {
		t.Errorf("Confidence = %v, want ConfidenceProbable", result.Confidence)
	}
}

func TestDetect_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	m := &php.Module{}
	result := m.Detect(dir)

	if result.Detected {
		t.Error("expected Detected=false when no PHP indicators present")
	}
	if result.Confidence != ecosystem.ConfidenceAbsent {
		t.Errorf("Confidence = %v, want ConfidenceAbsent", result.Confidence)
	}
}

func TestDevenvNixFragment(t *testing.T) {
	m := &php.Module{}
	fragment, err := m.DevenvNixFragment(ecosystem.ModuleConfig{})
	if err != nil {
		t.Fatalf("DevenvNixFragment() returned error: %v", err)
	}

	if fragment == "" {
		t.Fatal("DevenvNixFragment() returned empty string")
	}

	requiredStrings := []string{
		"languages.php",
		"enable = true",
	}
	for _, s := range requiredStrings {
		if !strings.Contains(fragment, s) {
			t.Errorf("DevenvNixFragment() missing %q\ngot:\n%s", s, fragment)
		}
	}
}

func TestDevenvNixFragment_VersionMapping(t *testing.T) {
	m := &php.Module{}

	tests := []struct {
		name    string
		version string
		wantPkg string
	}{
		{"empty version uses default", "", "pkgs.php83"},
		{"8.5 maps correctly", "8.5", "pkgs.php85"},
		{"8.4 maps correctly", "8.4", "pkgs.php84"},
		{"8.3 maps correctly", "8.3", "pkgs.php83"},
		{"8.2 maps correctly", "8.2", "pkgs.php82"},
		{"patch version maps to its series", "8.4.2", "pkgs.php84"},
		{"constraint resolves to newest match", "^8.2", "pkgs.php85"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fragment, err := m.DevenvNixFragment(ecosystem.ModuleConfig{Version: tt.version})
			if err != nil {
				t.Fatalf("DevenvNixFragment() returned error: %v", err)
			}
			if !strings.Contains(fragment, tt.wantPkg) {
				t.Errorf("DevenvNixFragment(version=%q) should contain %q\ngot:\n%s", tt.version, tt.wantPkg, fragment)
			}
		})
	}
}

// TestDevenvNixFragment_UnsupportedVersion verifies end-of-life or unknown
// versions fail loudly instead of emitting a throw-alias (php81) or silently
// substituting a different PHP series.
func TestDevenvNixFragment_UnsupportedVersion(t *testing.T) {
	t.Parallel()
	for _, version := range []string{"8.1", "8.0", "7.4", "9.0", "latest"} {
		t.Run(version, func(t *testing.T) {
			t.Parallel()
			frag, err := (&php.Module{}).DevenvNixFragment(ecosystem.ModuleConfig{Version: version})
			if !errors.Is(err, php.ErrUnsupportedPHPVersion) {
				t.Fatalf("DevenvNixFragment(%q) error = %v, want ErrUnsupportedPHPVersion (fragment %q)", version, err, frag)
			}
		})
	}
}

func TestWizardFields_OnlySupportedVersions(t *testing.T) {
	t.Parallel()
	m := &php.Module{}
	fields := m.WizardFields()
	if len(fields) != 1 {
		t.Fatalf("WizardFields() returned %d fields, want 1", len(fields))
	}
	for _, opt := range fields[0].Options {
		if _, err := m.DevenvNixFragment(ecosystem.ModuleConfig{Version: opt.Value}); err != nil {
			t.Errorf("wizard offers PHP %q, which DevenvNixFragment rejects: %v", opt.Value, err)
		}
		if opt.Value == "8.1" {
			t.Error("wizard must not offer end-of-life PHP 8.1")
		}
	}
}

func TestPreCommitHooks(t *testing.T) {
	m := &php.Module{}
	hooks := m.PreCommitHooks(ecosystem.ModuleConfig{})

	if len(hooks) != 2 {
		t.Fatalf("PreCommitHooks() returned %d hooks, want 2", len(hooks))
	}
	if hooks[0].ID != "phpcs" {
		t.Errorf("hooks[0].ID = %q, want %q", hooks[0].ID, "phpcs")
	}
	if hooks[1].ID != "phpstan" {
		t.Errorf("hooks[1].ID = %q, want %q", hooks[1].ID, "phpstan")
	}
}

// composerHomeRe extracts the COMPOSER_HOME the devenv fragment exports,
// relative to the project root.
var composerHomeRe = regexp.MustCompile(`(?m)^  env\.COMPOSER_HOME = "\$\{config\.devenv\.root\}/([^"]+)";$`)

// TestSecurityConfigs_LoadedAsComposerHomeConfig verifies the generated
// Composer configuration is written to config.json inside the COMPOSER_HOME
// the devenv fragment exports. Composer reads its global configuration only
// from $COMPOSER_HOME/config.json; any other path is never loaded.
func TestSecurityConfigs_LoadedAsComposerHomeConfig(t *testing.T) {
	t.Parallel()
	m := &php.Module{}
	frag, err := m.DevenvNixFragment(ecosystem.ModuleConfig{})
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}
	match := composerHomeRe.FindStringSubmatch(frag)
	if match == nil {
		t.Fatalf("fragment does not export COMPOSER_HOME under the project root:\n%s", frag)
	}
	if match[1] != ".qsdev/composer" {
		t.Errorf("COMPOSER_HOME = %q, want %q", match[1], ".qsdev/composer")
	}

	configs := m.SecurityConfigs(ecosystem.ModuleConfig{})
	if len(configs) != 1 {
		t.Fatalf("SecurityConfigs() returned %d files, want 1", len(configs))
	}
	if want := match[1] + "/config.json"; configs[0].Path != want {
		t.Errorf("SecurityConfigs()[0].Path = %q, want %q (COMPOSER_HOME/config.json)", configs[0].Path, want)
	}
	if strings.Contains(string(configs[0].Content), "merge into") {
		t.Errorf("config.json is applied directly and must not ask to be merged by hand:\n%s", configs[0].Content)
	}
}

// composerGlobalConfig is the subset of Composer's config.json schema the
// module generates.
type composerGlobalConfig struct {
	Repositories []map[string]any `json:"repositories"`
	Config       map[string]any   `json:"config"`
}

func TestSecurityConfigs_Content(t *testing.T) {
	t.Parallel()
	const proxy = "https://packagist.corp.example.com"
	tests := []struct {
		name      string
		proxy     string
		wantRepos []map[string]any
	}{
		{name: "no proxy keeps packagist.org", proxy: "", wantRepos: nil},
		{
			name:  "proxy replaces packagist.org",
			proxy: proxy,
			wantRepos: []map[string]any{
				{"type": "composer", "url": proxy},
				{"packagist.org": false},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			configs := (&php.Module{}).SecurityConfigs(ecosystem.ModuleConfig{RegistryProxy: tt.proxy})
			if len(configs) != 1 {
				t.Fatalf("SecurityConfigs() returned %d files, want 1", len(configs))
			}
			var got composerGlobalConfig
			if err := json.Unmarshal(configs[0].Content, &got); err != nil {
				t.Fatalf("config.json is not valid JSON: %v\n%s", err, configs[0].Content)
			}
			if !reflect.DeepEqual(got.Repositories, tt.wantRepos) {
				t.Errorf("repositories = %v, want %v", got.Repositories, tt.wantRepos)
			}
			wantConfig := map[string]any{
				"secure-http":       true,
				"lock":              true,
				"audit":             map[string]any{"abandoned": "fail"},
				"allow-plugins":     map[string]any{},
				"preferred-install": "dist",
			}
			if !reflect.DeepEqual(got.Config, wantConfig) {
				t.Errorf("config = %v, want %v", got.Config, wantConfig)
			}
		})
	}
}

func TestCICommands(t *testing.T) {
	m := &php.Module{}
	cmds := m.CICommands(ecosystem.ModuleConfig{})

	if len(cmds) != 3 {
		t.Fatalf("CICommands() returned %d commands, want 3", len(cmds))
	}
}

func TestRegistration(t *testing.T) {
	reg := ecosystem.DefaultRegistry()
	mod, ok := reg.ByName("php")
	if !ok {
		t.Fatal("expected module 'php' to be registered in DefaultRegistry")
	}
	if mod.Name() != "php" {
		t.Errorf("registered module Name() = %q, want %q", mod.Name(), "php")
	}
}
