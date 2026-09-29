package ruby_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/ruby"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
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

func TestModuleIdentity(t *testing.T) {
	ecosystem.AssertModuleIdentity(t, newModule(), "ruby", "Ruby", 2)
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

	if len(files) != 1 {
		t.Fatalf("SecurityConfigs() returned %d files, want 1", len(files))
	}
	if files[0].Path != ".gemrc" {
		t.Errorf("Path = %q, want .gemrc", files[0].Path)
	}
	if files[0].Strategy != types.Skip {
		t.Errorf(".gemrc Strategy = %v, want Skip", files[0].Strategy)
	}
}

// TestSecurityConfigs_NeverOwnsBundleConfig guards against replacing the
// user's (usually gitignored, unrecoverable) .bundle/config; the Bundler
// hardening is delivered through devenv env vars instead.
func TestSecurityConfigs_NeverOwnsBundleConfig(t *testing.T) {
	t.Parallel()
	for _, f := range newModule().SecurityConfigs(ecosystem.ModuleConfig{}) {
		if f.Path == ".bundle/config" {
			t.Fatal("SecurityConfigs must not generate .bundle/config")
		}
	}

	frag, err := newModule().DevenvNixFragment(ecosystem.ModuleConfig{})
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}
	for _, want := range []string{`env.BUNDLE_FROZEN = "true";`, `env.BUNDLE_DISABLE_EXEC_LOAD = "true";`} {
		if !strings.Contains(frag, want) {
			t.Errorf("fragment missing %q:\n%s", want, frag)
		}
	}
}

// --- Ruby version tests ---

func TestDevenvNixFragment_Version(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		version     string
		wantLine    string
		wantInput   bool
		wantErr     bool
		wantNoLines []string
	}{
		{name: "unset", version: "", wantNoLines: []string{"version ="}},
		{name: "plain", version: "3.1.4", wantLine: `version = "3.1.4";`, wantInput: true},
		{name: "ruby- prefix", version: "ruby-3.3.0", wantLine: `version = "3.3.0";`, wantInput: true},
		{name: "preview", version: "3.4.0-preview1", wantLine: `version = "3.4.0-preview1";`, wantInput: true},
		{name: "nix injection rejected", version: "3.3${builtins.abort \"x\"}", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := ecosystem.ModuleConfig{Version: tt.version}
			frag, err := newModule().DevenvNixFragment(cfg)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for version %q, got fragment:\n%s", tt.version, frag)
				}
				if inputs := newModule().DevenvYamlInputs(cfg); len(inputs) != 0 {
					t.Errorf("DevenvYamlInputs = %v for invalid version, want none", inputs)
				}
				return
			}
			if err != nil {
				t.Fatalf("DevenvNixFragment() error: %v", err)
			}
			if tt.wantLine != "" && !strings.Contains(frag, tt.wantLine) {
				t.Errorf("fragment missing %q:\n%s", tt.wantLine, frag)
			}
			for _, no := range tt.wantNoLines {
				if strings.Contains(frag, no) {
					t.Errorf("fragment unexpectedly contains %q:\n%s", no, frag)
				}
			}

			inputs := newModule().DevenvYamlInputs(cfg)
			if got := len(inputs) == 1 && inputs[0].URL == "github:bobvanderlinden/nixpkgs-ruby"; got != tt.wantInput {
				t.Errorf("DevenvYamlInputs = %v, want nixpkgs-ruby input: %v", inputs, tt.wantInput)
			}
		})
	}
}

func TestDetect_RubyVersionFile(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"first line only", "3.2.2\n# comment\n", "3.2.2"},
		{"ruby- prefix stripped", "ruby-3.3.0\n", "3.3.0"},
		{"non-MRI ignored", "jruby-9.4.5.0\n", ""},
		{"garbage ignored", "${evil}\n", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "Gemfile"), []byte("source 'https://rubygems.org'\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, ".ruby-version"), []byte(tt.content), 0o644); err != nil {
				t.Fatal(err)
			}
			if got := newModule().Detect(dir).SuggestedConfig.Version; got != tt.want {
				t.Errorf("SuggestedConfig.Version = %q, want %q", got, tt.want)
			}
		})
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
