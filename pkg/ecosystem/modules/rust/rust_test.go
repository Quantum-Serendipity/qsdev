package rust_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/rust"
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

func TestModuleIdentity(t *testing.T) {
	ecosystem.AssertModuleIdentity(t, newModule(), "rust", "Rust", 1)
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
	if !slices.Contains(r.Evidence, "Cargo.toml") {
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
	if !slices.Contains(r.Evidence, "Cargo.toml") {
		t.Errorf("Evidence should contain Cargo.toml")
	}
	if !slices.Contains(r.Evidence, "Cargo.lock") {
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

// TestDevenvNixFragment_PinnedToolchain guards devenv's channel enum:
// languages.rust.channel only accepts stable/beta/nightly, so a pinned release
// or dated nightly must be split into channel + version.
func TestDevenvNixFragment_PinnedToolchain(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		config      ecosystem.ModuleConfig
		wantChannel string
		wantVersion string // empty: no version line
	}{
		{name: "stable", config: ecosystem.ModuleConfig{Extras: map[string]string{"channel": "stable"}}, wantChannel: "stable"},
		{name: "beta", config: ecosystem.ModuleConfig{Extras: map[string]string{"channel": "beta"}}, wantChannel: "beta"},
		{name: "toolchain file release", config: ecosystem.ModuleConfig{Extras: map[string]string{"channel": "1.80.0"}}, wantChannel: "stable", wantVersion: "1.80.0"},
		{name: "toolchain file minor", config: ecosystem.ModuleConfig{Extras: map[string]string{"channel": "1.80"}}, wantChannel: "stable", wantVersion: "1.80.0"},
		{name: "dated nightly", config: ecosystem.ModuleConfig{Extras: map[string]string{"channel": "nightly-2024-05-01"}}, wantChannel: "nightly", wantVersion: "2024-05-01"},
		{name: "flag version", config: ecosystem.ModuleConfig{Version: "1.79.0"}, wantChannel: "stable", wantVersion: "1.79.0"},
		{name: "flag channel", config: ecosystem.ModuleConfig{Version: "nightly"}, wantChannel: "nightly"},
		{name: "flag overrides stable extra", config: ecosystem.ModuleConfig{Version: "beta", Extras: map[string]string{"channel": "stable"}}, wantChannel: "beta"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			frag, err := newModule().DevenvNixFragment(tt.config)
			if err != nil {
				t.Fatalf("DevenvNixFragment() error: %v", err)
			}
			if want := `channel = "` + tt.wantChannel + `";`; !strings.Contains(frag, want) {
				t.Errorf("fragment missing %s:\n%s", want, frag)
			}
			hasVersion := strings.Contains(frag, "version = ")
			if tt.wantVersion == "" && hasVersion {
				t.Errorf("fragment should not pin a version:\n%s", frag)
			}
			if want := `version = "` + tt.wantVersion + `";`; tt.wantVersion != "" && !strings.Contains(frag, want) {
				t.Errorf("fragment missing %s:\n%s", want, frag)
			}
		})
	}
}

func TestDevenvNixFragment_RejectsInvalidToolchain(t *testing.T) {
	t.Parallel()
	for _, spec := range []string{"1.80.0-x86_64-unknown-linux-gnu", "my-custom", `stable"; x = "`, "nightly-2024"} {
		t.Run(spec, func(t *testing.T) {
			t.Parallel()
			cfg := ecosystem.ModuleConfig{Extras: map[string]string{"channel": spec}}
			if frag, err := newModule().DevenvNixFragment(cfg); err == nil {
				t.Errorf("DevenvNixFragment(channel=%q) error = nil, want error\ngot:\n%s", spec, frag)
			}
		})
	}
}

func TestDetect_UnsupportedToolchainFallsBackToStable(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	for name, content := range map[string]string{
		"Cargo.toml":          "[package]\nname = \"x\"\n",
		"rust-toolchain.toml": "[toolchain]\nchannel = \"1.80.0-x86_64-unknown-linux-gnu\"\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r := newModule().Detect(dir)
	if ch := r.SuggestedConfig.Extras["channel"]; ch != "stable" {
		t.Errorf("channel = %q, want %q", ch, "stable")
	}
	if _, err := newModule().DevenvNixFragment(r.SuggestedConfig); err != nil {
		t.Errorf("detected config does not render: %v", err)
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
	if !strings.Contains(content, `registry = "sparse+`+proxy+`/"`) {
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
}

// --- WizardFields tests ---

func TestWizardFields(t *testing.T) {
	m := newModule()
	fields := m.WizardFields()

	if len(fields) != 1 {
		t.Fatalf("WizardFields() returned %d fields, want 1", len(fields))
	}

	f := fields[0]
	if f.Key != "channel" {
		t.Errorf("Key = %q, want %q", f.Key, "channel")
	}
	if f.Type != ecosystem.FieldTypeSelect {
		t.Errorf("Type = %v, want FieldTypeSelect", f.Type)
	}
	if f.Default != "stable" {
		t.Errorf("Default = %q, want %q", f.Default, "stable")
	}
	if len(f.Options) != 3 {
		t.Fatalf("Options count = %d, want 3", len(f.Options))
	}

	values := make(map[string]bool)
	for _, o := range f.Options {
		values[o.Value] = true
	}
	if !values["stable"] {
		t.Error("missing option value stable")
	}
	for _, v := range []string{"beta", "nightly"} {
		if !values[v] {
			t.Errorf("missing option value %s", v)
		}
	}
}

// TestDevenvNixFragment_KeepsFullToolchain guards against overriding
// languages.rust.components: devenv builds a rust-overlay toolchain from that
// list alone, so any explicit list without rustc and cargo leaves the shell
// with no cargo. The fragment must leave devenv's full default in place.
func TestDevenvNixFragment_KeepsFullToolchain(t *testing.T) {
	t.Parallel()
	configs := map[string]ecosystem.ModuleConfig{
		"default":          {},
		"nightly":          {Extras: map[string]string{"channel": "nightly"}},
		"pinned release":   {Version: "1.80.1"},
		"toolchain file":   {Extras: map[string]string{rust.ExtraToolchainFile: "rust-toolchain.toml"}},
		"dated nightly":    {Extras: map[string]string{"channel": "nightly-2024-05-01"}},
		"legacy toolchain": {Extras: map[string]string{rust.ExtraToolchainFile: "rust-toolchain"}},
	}
	for name, config := range configs {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			frag, err := newModule().DevenvNixFragment(config)
			if err != nil {
				t.Fatalf("DevenvNixFragment() error: %v", err)
			}
			if strings.Contains(frag, "components") {
				t.Errorf("fragment overrides devenv's default components (drops rustc/cargo):\n%s", frag)
			}
		})
	}
}

// TestDetect_ToolchainFile covers both rustup toolchain file forms: TOML is
// accepted in either file name (quoted either way), and only non-TOML
// single-line content is read as a legacy bare channel. A usable file is
// handed to devenv through the toolchain_file extra.
func TestDetect_ToolchainFile(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		file        string
		content     string
		wantChannel string
		wantFile    string
	}{
		{"toml double-quoted", "rust-toolchain.toml", "[toolchain]\nchannel = \"nightly\"\n", "nightly", "rust-toolchain.toml"},
		{"toml single-quoted", "rust-toolchain.toml", "[toolchain]\nchannel = 'nightly'\n", "nightly", "rust-toolchain.toml"},
		{"toml pinned release", "rust-toolchain.toml", "[toolchain]\nchannel = \"1.80.1\"\ncomponents = [\"rust-src\"]\n", "1.80.1", "rust-toolchain.toml"},
		{"legacy file with toml content", "rust-toolchain", "[toolchain]\nchannel = \"beta\"\n", "beta", "rust-toolchain"},
		{"legacy bare channel", "rust-toolchain", "nightly-2024-05-01\n", "nightly-2024-05-01", "rust-toolchain"},
		{"toml without channel", "rust-toolchain.toml", "[toolchain]\ncomponents = [\"clippy\"]\n", "stable", ""},
		{"unsupported toolchain", "rust-toolchain", "stable-x86_64-unknown-linux-gnu\n", "stable", ""},
		{"malformed multi-line", "rust-toolchain", "nightly\nstable\n", "stable", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[package]\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, tt.file), []byte(tt.content), 0o644); err != nil {
				t.Fatal(err)
			}
			r := newModule().Detect(dir)
			if got := r.SuggestedConfig.Extras["channel"]; got != tt.wantChannel {
				t.Errorf("channel = %q, want %q", got, tt.wantChannel)
			}
			if got := r.SuggestedConfig.Extras[rust.ExtraToolchainFile]; got != tt.wantFile {
				t.Errorf("toolchain_file = %q, want %q", got, tt.wantFile)
			}
		})
	}
}

// TestDevenvNixFragment_ToolchainFile verifies a detected toolchain file is
// passed to devenv as languages.rust.toolchainFile without channel or
// version (devenv asserts they cannot be combined), that an explicit Version
// still wins, and that only rustup's file names can reach the Nix path.
func TestDevenvNixFragment_ToolchainFile(t *testing.T) {
	t.Parallel()
	m := newModule()

	frag, err := m.DevenvNixFragment(ecosystem.ModuleConfig{Extras: map[string]string{
		rust.ExtraToolchainFile: "rust-toolchain.toml", "channel": "nightly",
	}})
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}
	if !strings.Contains(frag, "toolchainFile = ./rust-toolchain.toml;") {
		t.Errorf("fragment missing toolchainFile:\n%s", frag)
	}
	if strings.Contains(frag, "channel =") || strings.Contains(frag, "version =") {
		t.Errorf("toolchainFile must not be combined with channel/version:\n%s", frag)
	}

	frag, err = m.DevenvNixFragment(ecosystem.ModuleConfig{Version: "beta", Extras: map[string]string{
		rust.ExtraToolchainFile: "rust-toolchain.toml",
	}})
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}
	if !strings.Contains(frag, `channel = "beta"`) || strings.Contains(frag, "toolchainFile") {
		t.Errorf("explicit Version should override the toolchain file:\n%s", frag)
	}

	for _, bad := range []string{"../rust-toolchain.toml", "rust-toolchain.toml; x = 1", "/etc/passwd"} {
		if _, err := m.DevenvNixFragment(ecosystem.ModuleConfig{Extras: map[string]string{rust.ExtraToolchainFile: bad}}); err == nil {
			t.Errorf("DevenvNixFragment(toolchain_file=%q) should fail", bad)
		}
	}
}

// TestSecurityConfigs_RegistryProxySparse guards the cargo index protocol:
// without a sparse+ prefix cargo treats an http(s) proxy as a git index and
// every resolution fails, while explicit protocols are left alone.
func TestSecurityConfigs_RegistryProxySparse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		proxy string
		want  string
	}{
		{"https://nexus.example.com/repository/cargo-proxy/", "sparse+https://nexus.example.com/repository/cargo-proxy/"},
		{"https://crates.corp.example.com", "sparse+https://crates.corp.example.com/"},
		{"http://127.0.0.1:8080/index", "sparse+http://127.0.0.1:8080/index/"},
		{"sparse+https://crates.corp.example.com/", "sparse+https://crates.corp.example.com/"},
		{"https://git.corp.example.com/crates-index.git", "https://git.corp.example.com/crates-index.git"},
		{"ssh://git@git.corp.example.com/crates-index", "ssh://git@git.corp.example.com/crates-index"},
	}
	for _, tt := range tests {
		t.Run(tt.proxy, func(t *testing.T) {
			t.Parallel()
			files := newModule().SecurityConfigs(ecosystem.ModuleConfig{RegistryProxy: tt.proxy})
			content := string(files[0].Content)
			if want := `registry = "` + tt.want + `"`; !strings.Contains(content, want) {
				t.Errorf("content missing %s:\n%s", want, content)
			}
		})
	}
}

// TestDevenvPackages_Sccache verifies that selecting sccache as the rustc
// wrapper also provides the sccache binary; otherwise every compile fails
// with "could not execute process `sccache`".
func TestDevenvPackages_Sccache(t *testing.T) {
	t.Parallel()
	m := newModule()
	cfg := ecosystem.ModuleConfig{Extras: map[string]string{"build_cache": "sccache"}}
	wrapper := strings.Contains(string(m.SecurityConfigs(cfg)[0].Content), `rustc-wrapper = "sccache"`)
	if !wrapper {
		t.Fatal("expected sccache rustc-wrapper in .cargo/config.toml")
	}
	if got := m.DevenvPackages(cfg); !slices.Contains(got, "sccache") {
		t.Errorf("DevenvPackages() = %v, want sccache when it is the rustc wrapper", got)
	}
	if got := m.DevenvPackages(ecosystem.ModuleConfig{}); slices.Contains(got, "sccache") {
		t.Errorf("DevenvPackages() = %v, want no sccache without a build cache", got)
	}
}

// TestReadDenyRules verifies the Cargo registry credential files are
// read-denied to the agent.
func TestReadDenyRules(t *testing.T) {
	t.Parallel()
	got := newModule().ReadDenyRules(ecosystem.ModuleConfig{})
	for _, want := range []string{"~/.cargo/credentials.toml", "~/.cargo/credentials"} {
		if !slices.Contains(got, want) {
			t.Errorf("ReadDenyRules() = %v, missing %q", got, want)
		}
	}
}
