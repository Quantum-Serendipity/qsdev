package ruby_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/ruby"
)

// newModule returns a fresh Module for testing.
func newModule() *ruby.Module {
	return &ruby.Module{}
}

// --- Interface compliance ---

func TestInterfaceCompliance(t *testing.T) {
	var _ ecosystem.EcosystemModule = (*ruby.Module)(nil)
}

// --- Basic metadata ---

func TestName(t *testing.T) {
	m := newModule()
	if got := m.Name(); got != "ruby" {
		t.Errorf("Name() = %q, want %q", got, "ruby")
	}
}

func TestDisplayName(t *testing.T) {
	m := newModule()
	got := m.DisplayName()
	if !strings.Contains(got, "Ruby") {
		t.Errorf("DisplayName() = %q, want it to contain %q", got, "Ruby")
	}
}

func TestTier(t *testing.T) {
	m := newModule()
	if got := m.Tier(); got != 2 {
		t.Errorf("Tier() = %d, want %d", got, 2)
	}
}

// --- Detection tests ---

func TestDetect_Gemfile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Gemfile"), []byte("source 'https://rubygems.org'\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	r := m.Detect(dir)

	if !r.Detected {
		t.Fatal("expected Detected = true")
	}
	if r.Confidence < ecosystem.ConfidenceProbable {
		t.Errorf("Confidence = %v, want >= Probable", r.Confidence)
	}
	if len(r.Evidence) == 0 {
		t.Error("expected non-empty Evidence")
	}
}

func TestDetect_GemfileWithLock(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Gemfile"), []byte("source 'https://rubygems.org'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Gemfile.lock"), []byte("GEM\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	r := m.Detect(dir)

	if !r.Detected {
		t.Fatal("expected Detected = true")
	}
	if r.Confidence < ecosystem.ConfidenceProbable {
		t.Errorf("Confidence = %v, want >= Probable", r.Confidence)
	}
	foundGemfile := false
	foundLock := false
	for _, e := range r.Evidence {
		if strings.Contains(e, "Gemfile") && !strings.Contains(e, "lock") {
			foundGemfile = true
		}
		if strings.Contains(e, "Gemfile.lock") {
			foundLock = true
		}
	}
	if !foundGemfile {
		t.Error("Evidence should mention Gemfile")
	}
	if !foundLock {
		t.Error("Evidence should mention Gemfile.lock")
	}
}

func TestDetect_WithRubyVersion(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Gemfile"), []byte("source 'https://rubygems.org'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".ruby-version"), []byte("3.3.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	r := m.Detect(dir)

	if !r.Detected {
		t.Fatal("expected Detected = true")
	}
	if r.SuggestedConfig.Version != "3.3.0" {
		t.Errorf("SuggestedConfig.Version = %q, want %q", r.SuggestedConfig.Version, "3.3.0")
	}
}

func TestDetect_NotPresent(t *testing.T) {
	dir := t.TempDir()

	m := newModule()
	r := m.Detect(dir)

	if r.Detected {
		t.Fatal("expected Detected = false for empty directory")
	}
	if r.Confidence != ecosystem.ConfidenceAbsent {
		t.Errorf("Confidence = %v, want Absent", r.Confidence)
	}
	if len(r.Evidence) != 0 {
		t.Errorf("Evidence = %v, want empty", r.Evidence)
	}
}

// --- DevenvNixFragment tests ---

func TestDevenvNixFragment_NonEmpty(t *testing.T) {
	m := newModule()
	config := ecosystem.ModuleConfig{}

	frag, err := m.DevenvNixFragment(config)
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}
	if frag == "" {
		t.Error("DevenvNixFragment() returned empty string")
	}
	if !strings.Contains(frag, "languages.ruby") {
		t.Errorf("fragment missing languages.ruby:\n%s", frag)
	}
	if !strings.Contains(frag, "enable = true") {
		t.Errorf("fragment missing enable = true:\n%s", frag)
	}
	if !strings.Contains(frag, "bundler") {
		t.Errorf("fragment missing bundler:\n%s", frag)
	}
}

// --- SecurityConfigs tests ---

func TestSecurityConfigs(t *testing.T) {
	m := newModule()
	files := m.SecurityConfigs(ecosystem.ModuleConfig{})

	if len(files) != 2 {
		t.Fatalf("SecurityConfigs() returned %d files, want 2", len(files))
	}

	paths := make(map[string]string)
	for _, f := range files {
		paths[f.Path] = string(f.Content)
	}

	if _, ok := paths[".bundle/config"]; !ok {
		t.Error("missing .bundle/config")
	}
	if _, ok := paths[".gemrc"]; !ok {
		t.Error("missing .gemrc")
	}
}

// --- PreCommitHooks tests ---

func TestPreCommitHooks(t *testing.T) {
	m := newModule()
	hooks := m.PreCommitHooks(ecosystem.ModuleConfig{})

	if len(hooks) != 1 {
		t.Fatalf("PreCommitHooks() returned %d hooks, want 1", len(hooks))
	}
	if hooks[0].ID != "rubocop" {
		t.Errorf("hook ID = %q, want %q", hooks[0].ID, "rubocop")
	}
}

// --- DenyRules tests ---

func TestDenyRules(t *testing.T) {
	m := newModule()
	rules := m.DenyRules(ecosystem.ModuleConfig{})

	if len(rules) != 0 {
		t.Fatalf("DenyRules() returned %d rules, want 0 (installs handled by ask rules)", len(rules))
	}
}

// --- CICommands tests ---

func TestCICommands(t *testing.T) {
	m := newModule()
	cmds := m.CICommands(ecosystem.ModuleConfig{})

	if len(cmds) != 2 {
		t.Fatalf("CICommands() returned %d commands, want 2", len(cmds))
	}
}

// --- PackageManagers tests ---

func TestPackageManagers(t *testing.T) {
	m := newModule()
	pms := m.PackageManagers()

	if len(pms) != 1 {
		t.Fatalf("PackageManagers() returned %d entries, want 1", len(pms))
	}
	if pms[0].Name != "bundler" {
		t.Errorf("Name = %q, want %q", pms[0].Name, "bundler")
	}
	if pms[0].LockFile != "Gemfile.lock" {
		t.Errorf("LockFile = %q, want %q", pms[0].LockFile, "Gemfile.lock")
	}
}
