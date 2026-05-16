package php_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/internal/ecosystem/modules/php"
)

// Compile-time interface compliance check.
var _ ecosystem.EcosystemModule = (*php.Module)(nil)

func TestName(t *testing.T) {
	m := &php.Module{}
	if got := m.Name(); got != "php" {
		t.Errorf("Name() = %q, want %q", got, "php")
	}
}

func TestDisplayName(t *testing.T) {
	m := &php.Module{}
	got := m.DisplayName()
	if !strings.Contains(got, "PHP") {
		t.Errorf("DisplayName() = %q, want it to contain %q", got, "PHP")
	}
}

func TestTier(t *testing.T) {
	m := &php.Module{}
	if got := m.Tier(); got != 2 {
		t.Errorf("Tier() = %d, want %d", got, 2)
	}
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

	if result.SuggestedConfig.Version != "8.2" {
		t.Errorf("Version = %q, want %q", result.SuggestedConfig.Version, "8.2")
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
		{"empty version uses latest", "", "pkgs.php83"},
		{"8.3 maps correctly", "8.3", "pkgs.php83"},
		{"8.2 maps correctly", "8.2", "pkgs.php82"},
		{"8.1 maps correctly", "8.1", "pkgs.php81"},
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

func TestDenyRules(t *testing.T) {
	m := &php.Module{}
	rules := m.DenyRules(ecosystem.ModuleConfig{})

	if len(rules) != 0 {
		t.Fatalf("DenyRules() returned %d rules, want 0 (installs handled by ask rules)", len(rules))
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

func TestSecurityConfigs(t *testing.T) {
	m := &php.Module{}
	configs := m.SecurityConfigs(ecosystem.ModuleConfig{})

	if len(configs) != 1 {
		t.Fatalf("SecurityConfigs() returned %d files, want 1", len(configs))
	}
	if configs[0].Path != ".qsdev/composer-security.json" {
		t.Errorf("SecurityConfigs()[0].Path = %q, want %q", configs[0].Path, ".qsdev/composer-security.json")
	}
}

func TestSecurityConfigs_RegistryProxy(t *testing.T) {
	m := &php.Module{}
	proxy := "https://packagist.corp.example.com"
	configs := m.SecurityConfigs(ecosystem.ModuleConfig{
		RegistryProxy: proxy,
	})

	if len(configs) != 1 {
		t.Fatalf("SecurityConfigs() returned %d files, want 1", len(configs))
	}

	content := string(configs[0].Content)
	// Proxy repository must be present.
	if !strings.Contains(content, proxy) {
		t.Errorf("composer-security.json missing proxy URL\ncontent:\n%s", content)
	}
	if !strings.Contains(content, "composer") {
		t.Errorf("composer-security.json missing repository type\ncontent:\n%s", content)
	}
	if !strings.Contains(content, `"repositories"`) {
		t.Errorf("composer-security.json missing repositories key\ncontent:\n%s", content)
	}
	// Existing security settings must be preserved.
	if !strings.Contains(content, `"secure-http"`) {
		t.Errorf("composer-security.json missing secure-http when proxy is set\ncontent:\n%s", content)
	}
	if !strings.Contains(content, `"preferred-install"`) {
		t.Errorf("composer-security.json missing preferred-install when proxy is set\ncontent:\n%s", content)
	}
}

func TestSecurityConfigs_NoRegistryProxy(t *testing.T) {
	m := &php.Module{}
	configs := m.SecurityConfigs(ecosystem.ModuleConfig{})

	content := string(configs[0].Content)
	if strings.Contains(content, `"repositories"`) {
		t.Errorf("composer-security.json should not contain repositories when proxy is empty\ncontent:\n%s", content)
	}
}

func TestSecurityConfigs_RegistryProxyPreservesExisting(t *testing.T) {
	m := &php.Module{}
	proxy := "https://packagist.corp.example.com"
	configs := m.SecurityConfigs(ecosystem.ModuleConfig{
		RegistryProxy: proxy,
	})

	content := string(configs[0].Content)
	for _, s := range []string{`"secure-http"`, `"lock"`, `"audit"`, `"allow-plugins"`, `"preferred-install"`} {
		if !strings.Contains(content, s) {
			t.Errorf("composer-security.json missing %s when proxy is set\ncontent:\n%s", s, content)
		}
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
