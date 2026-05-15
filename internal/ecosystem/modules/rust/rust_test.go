package rust_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/internal/ecosystem/modules/rust"
)

// newModule returns a fresh Module for testing.
func newModule() *rust.Module {
	return &rust.Module{}
}

// --- Interface compliance ---

func TestInterfaceCompliance(t *testing.T) {
	var _ ecosystem.EcosystemModule = (*rust.Module)(nil)
}

// --- Basic metadata ---

func TestName(t *testing.T) {
	m := newModule()
	if got := m.Name(); got != "rust" {
		t.Errorf("Name() = %q, want %q", got, "rust")
	}
}

func TestDisplayName(t *testing.T) {
	m := newModule()
	if got := m.DisplayName(); got != "Rust" {
		t.Errorf("DisplayName() = %q, want %q", got, "Rust")
	}
}

func TestTier(t *testing.T) {
	m := newModule()
	if got := m.Tier(); got != 1 {
		t.Errorf("Tier() = %d, want %d", got, 1)
	}
}

// --- Detection tests ---

func TestDetect_CargoToml(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[package]\nname = \"myapp\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	r := m.Detect(dir)

	if !r.Detected {
		t.Fatal("expected Detected = true")
	}
	if r.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("Confidence = %v, want Certain", r.Confidence)
	}
	if !containsStr(r.Evidence, "Cargo.toml") {
		t.Errorf("Evidence = %v, want to contain %q", r.Evidence, "Cargo.toml")
	}
	if ch := r.SuggestedConfig.Extras["channel"]; ch != "stable" {
		t.Errorf("channel = %q, want %q", ch, "stable")
	}
}

func TestDetect_WithCargoLock(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[package]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Cargo.lock"), []byte("[[package]]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	r := m.Detect(dir)

	if !r.Detected {
		t.Fatal("expected Detected = true")
	}
	if r.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("Confidence = %v, want Certain", r.Confidence)
	}
	if !containsStr(r.Evidence, "Cargo.toml") {
		t.Errorf("Evidence should contain Cargo.toml")
	}
	if !containsStr(r.Evidence, "Cargo.lock") {
		t.Errorf("Evidence should contain Cargo.lock")
	}
}

func TestDetect_ToolchainToml_Stable(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[package]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	toolchainContent := `[toolchain]
channel = "stable"
components = ["rustfmt", "clippy"]
`
	if err := os.WriteFile(filepath.Join(dir, "rust-toolchain.toml"), []byte(toolchainContent), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	r := m.Detect(dir)

	if !r.Detected {
		t.Fatal("expected Detected = true")
	}
	if ch := r.SuggestedConfig.Extras["channel"]; ch != "stable" {
		t.Errorf("channel = %q, want %q", ch, "stable")
	}
}

func TestDetect_ToolchainToml_Nightly(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[package]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	toolchainContent := `[toolchain]
channel = "nightly"
`
	if err := os.WriteFile(filepath.Join(dir, "rust-toolchain.toml"), []byte(toolchainContent), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	r := m.Detect(dir)

	if !r.Detected {
		t.Fatal("expected Detected = true")
	}
	if ch := r.SuggestedConfig.Extras["channel"]; ch != "nightly" {
		t.Errorf("channel = %q, want %q", ch, "nightly")
	}
}

func TestDetect_LegacyToolchainFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[package]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "rust-toolchain"), []byte("nightly\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	r := m.Detect(dir)

	if !r.Detected {
		t.Fatal("expected Detected = true")
	}
	if ch := r.SuggestedConfig.Extras["channel"]; ch != "nightly" {
		t.Errorf("channel = %q, want %q", ch, "nightly")
	}
}

func TestDetect_ToolchainTomlPrecedence(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[package]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// TOML file says nightly.
	tomlContent := `[toolchain]
channel = "nightly"
`
	if err := os.WriteFile(filepath.Join(dir, "rust-toolchain.toml"), []byte(tomlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Legacy file says stable.
	if err := os.WriteFile(filepath.Join(dir, "rust-toolchain"), []byte("stable\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	r := m.Detect(dir)

	if !r.Detected {
		t.Fatal("expected Detected = true")
	}
	// rust-toolchain.toml should take precedence.
	if ch := r.SuggestedConfig.Extras["channel"]; ch != "nightly" {
		t.Errorf("channel = %q, want %q (TOML should take precedence)", ch, "nightly")
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

func TestDevenvNixFragment_Stable(t *testing.T) {
	m := newModule()
	config := ecosystem.ModuleConfig{
		Extras: map[string]string{"channel": "stable"},
	}

	frag, err := m.DevenvNixFragment(config)
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}

	if !strings.Contains(frag, `channel = "stable"`) {
		t.Errorf("fragment missing channel = stable:\n%s", frag)
	}
	if !strings.Contains(frag, "languages.rust") {
		t.Errorf("fragment missing languages.rust:\n%s", frag)
	}
	if !strings.Contains(frag, "enable = true") {
		t.Errorf("fragment missing enable = true:\n%s", frag)
	}
	if !strings.Contains(frag, `"rustfmt"`) {
		t.Errorf("fragment missing rustfmt component:\n%s", frag)
	}
	if !strings.Contains(frag, `"clippy"`) {
		t.Errorf("fragment missing clippy component:\n%s", frag)
	}
}

func TestDevenvNixFragment_Nightly(t *testing.T) {
	m := newModule()
	config := ecosystem.ModuleConfig{
		Extras: map[string]string{"channel": "nightly"},
	}

	frag, err := m.DevenvNixFragment(config)
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}

	if !strings.Contains(frag, `channel = "nightly"`) {
		t.Errorf("fragment missing channel = nightly:\n%s", frag)
	}
}

func TestDevenvNixFragment_Default(t *testing.T) {
	m := newModule()

	// No extras at all.
	config := ecosystem.ModuleConfig{}

	frag, err := m.DevenvNixFragment(config)
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}

	if !strings.Contains(frag, `channel = "stable"`) {
		t.Errorf("fragment should default to stable channel:\n%s", frag)
	}
}

// --- SecurityConfigs tests ---

func TestSecurityConfigs_Base(t *testing.T) {
	m := newModule()
	config := ecosystem.ModuleConfig{}

	files := m.SecurityConfigs(config)
	if len(files) != 1 {
		t.Fatalf("SecurityConfigs() returned %d files, want 1", len(files))
	}

	f := files[0]
	if f.Path != ".cargo/config.toml" {
		t.Errorf("Path = %q, want %q", f.Path, ".cargo/config.toml")
	}
	content := string(f.Content)
	if !strings.Contains(content, "git-fetch-with-cli = true") {
		t.Errorf("content missing git-fetch-with-cli:\n%s", content)
	}
	if strings.Contains(content, "sccache") {
		t.Errorf("content should not contain sccache without build_cache config:\n%s", content)
	}
	if strings.Contains(content, "rustc-wrapper") {
		t.Errorf("content should not contain rustc-wrapper without build_cache config:\n%s", content)
	}
	if !f.SkipValidation {
		t.Error("SkipValidation should be true")
	}
}

func TestSecurityConfigs_WithSccache(t *testing.T) {
	m := newModule()
	config := ecosystem.ModuleConfig{
		Extras: map[string]string{"build_cache": "sccache"},
	}

	files := m.SecurityConfigs(config)
	if len(files) != 1 {
		t.Fatalf("SecurityConfigs() returned %d files, want 1", len(files))
	}

	content := string(files[0].Content)
	if !strings.Contains(content, "git-fetch-with-cli = true") {
		t.Errorf("content missing git-fetch-with-cli:\n%s", content)
	}
	if !strings.Contains(content, `rustc-wrapper = "sccache"`) {
		t.Errorf("content missing rustc-wrapper = sccache:\n%s", content)
	}
}

func TestSecurityConfigs_RegistryProxy(t *testing.T) {
	m := newModule()
	proxy := "https://crates.corp.example.com"
	config := ecosystem.ModuleConfig{
		RegistryProxy: proxy,
	}

	files := m.SecurityConfigs(config)
	if len(files) != 1 {
		t.Fatalf("SecurityConfigs() returned %d files, want 1", len(files))
	}

	content := string(files[0].Content)
	if !strings.Contains(content, "[source.crates-io]") {
		t.Errorf("content missing [source.crates-io]:\n%s", content)
	}
	if !strings.Contains(content, `replace-with = "corporate-proxy"`) {
		t.Errorf("content missing replace-with:\n%s", content)
	}
	if !strings.Contains(content, "[source.corporate-proxy]") {
		t.Errorf("content missing [source.corporate-proxy]:\n%s", content)
	}
	if !strings.Contains(content, `registry = "`+proxy+`"`) {
		t.Errorf("content missing registry URL:\n%s", content)
	}
	// Existing security settings must be preserved.
	if !strings.Contains(content, "git-fetch-with-cli = true") {
		t.Errorf("content missing git-fetch-with-cli when proxy is set:\n%s", content)
	}
}

func TestSecurityConfigs_NoRegistryProxy(t *testing.T) {
	m := newModule()
	config := ecosystem.ModuleConfig{}

	files := m.SecurityConfigs(config)
	content := string(files[0].Content)
	if strings.Contains(content, "[source.crates-io]") {
		t.Errorf("content should not contain [source.crates-io] when proxy is empty:\n%s", content)
	}
	if strings.Contains(content, "corporate-proxy") {
		t.Errorf("content should not contain corporate-proxy when proxy is empty:\n%s", content)
	}
}

func TestSecurityConfigs_RegistryProxyWithSccache(t *testing.T) {
	m := newModule()
	proxy := "https://crates.corp.example.com"
	config := ecosystem.ModuleConfig{
		RegistryProxy: proxy,
		Extras:        map[string]string{"build_cache": "sccache"},
	}

	files := m.SecurityConfigs(config)
	content := string(files[0].Content)
	// Both proxy and sccache settings must be present.
	if !strings.Contains(content, "[source.corporate-proxy]") {
		t.Errorf("content missing proxy config:\n%s", content)
	}
	if !strings.Contains(content, `rustc-wrapper = "sccache"`) {
		t.Errorf("content missing sccache config:\n%s", content)
	}
	if !strings.Contains(content, "git-fetch-with-cli = true") {
		t.Errorf("content missing git-fetch-with-cli:\n%s", content)
	}
}

// --- PreCommitHooks tests ---

func TestPreCommitHooks(t *testing.T) {
	m := newModule()
	hooks := m.PreCommitHooks(ecosystem.ModuleConfig{})

	if len(hooks) != 2 {
		t.Fatalf("PreCommitHooks() returned %d hooks, want 2", len(hooks))
	}

	ids := make(map[string]bool)
	for _, h := range hooks {
		ids[h.ID] = true
		if !h.BuiltIn {
			t.Errorf("hook %q: BuiltIn = false, want true", h.ID)
		}
	}

	if !ids["rustfmt"] {
		t.Error("missing hook with ID rustfmt")
	}
	if !ids["clippy"] {
		t.Error("missing hook with ID clippy")
	}

	// Verify entry commands.
	for _, h := range hooks {
		switch h.ID {
		case "rustfmt":
			if h.Entry != "cargo fmt -- --check" {
				t.Errorf("rustfmt entry = %q, want %q", h.Entry, "cargo fmt -- --check")
			}
		case "clippy":
			if h.Entry != "cargo clippy -- -D warnings" {
				t.Errorf("clippy entry = %q, want %q", h.Entry, "cargo clippy -- -D warnings")
			}
		}
	}
}

// --- DenyRules tests ---

func TestDenyRules(t *testing.T) {
	m := newModule()
	rules := m.DenyRules(ecosystem.ModuleConfig{})

	if len(rules) != 2 {
		t.Fatalf("DenyRules() returned %d rules, want 2", len(rules))
	}

	expected := map[string]bool{
		"Bash(cargo add *)":     true,
		"Bash(cargo install *)": true,
	}
	for _, r := range rules {
		if !expected[r] {
			t.Errorf("unexpected deny rule: %q", r)
		}
	}
}

// --- CICommands tests ---

func TestCICommands(t *testing.T) {
	m := newModule()
	cmds := m.CICommands(ecosystem.ModuleConfig{})

	if len(cmds) != 2 {
		t.Fatalf("CICommands() returned %d commands, want 2", len(cmds))
	}

	foundInstall := false
	foundScan := false
	for _, c := range cmds {
		switch c.Phase {
		case ecosystem.CIPhaseInstall:
			foundInstall = true
			if c.Command != "cargo build --locked" {
				t.Errorf("install command = %q, want %q", c.Command, "cargo build --locked")
			}
		case ecosystem.CIPhaseScan:
			foundScan = true
			if c.Command != "cargo audit" {
				t.Errorf("scan command = %q, want %q", c.Command, "cargo audit")
			}
		default:
			t.Errorf("unexpected phase %v for command %q", c.Phase, c.Name)
		}
	}

	if !foundInstall {
		t.Error("missing CI command with Install phase")
	}
	if !foundScan {
		t.Error("missing CI command with Scan phase")
	}
}

// --- PackageManagers tests ---

func TestPackageManagers(t *testing.T) {
	m := newModule()
	pms := m.PackageManagers()

	if len(pms) != 1 {
		t.Fatalf("PackageManagers() returned %d entries, want 1", len(pms))
	}

	pm := pms[0]
	if pm.Name != "cargo" {
		t.Errorf("Name = %q, want %q", pm.Name, "cargo")
	}
	if pm.LockFile != "Cargo.lock" {
		t.Errorf("LockFile = %q, want %q", pm.LockFile, "Cargo.lock")
	}
	if pm.FrozenInstallCommand != "cargo build --locked" {
		t.Errorf("FrozenInstallCommand = %q, want %q", pm.FrozenInstallCommand, "cargo build --locked")
	}
	if pm.AuditCommand != "cargo audit" {
		t.Errorf("AuditCommand = %q, want %q", pm.AuditCommand, "cargo audit")
	}
	if pm.AgeGatingSupport {
		t.Error("AgeGatingSupport should be false for cargo")
	}
}

// --- WizardFields tests ---

func TestWizardFields(t *testing.T) {
	m := newModule()
	fields := m.WizardFields()

	if len(fields) != 1 {
		t.Fatalf("WizardFields() returned %d fields, want 1", len(fields))
	}

	f := fields[0]
	if f.Key != "rust_channel" {
		t.Errorf("Key = %q, want %q", f.Key, "rust_channel")
	}
	if f.Type != ecosystem.FieldTypeSelect {
		t.Errorf("Type = %v, want FieldTypeSelect", f.Type)
	}
	if f.Default != "stable" {
		t.Errorf("Default = %q, want %q", f.Default, "stable")
	}
	if len(f.Options) != 2 {
		t.Fatalf("Options count = %d, want 2", len(f.Options))
	}

	values := make(map[string]bool)
	for _, o := range f.Options {
		values[o.Value] = true
	}
	if !values["stable"] {
		t.Error("missing option value stable")
	}
	if !values["nightly"] {
		t.Error("missing option value nightly")
	}
}

// --- helpers ---

func containsStr(ss []string, target string) bool {
	for _, s := range ss {
		if s == target {
			return true
		}
	}
	return false
}
