package php_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/php"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
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
	if match[1] != ".qsdev-composer" {
		t.Errorf("COMPOSER_HOME = %q, want %q", match[1], ".qsdev-composer")
	}

	cfgFile := composerConfigFile(t, m.SecurityConfigs(ecosystem.ModuleConfig{}))
	if want := match[1] + "/config.json"; cfgFile.Path != want {
		t.Errorf("config.json Path = %q, want %q (COMPOSER_HOME/config.json)", cfgFile.Path, want)
	}
	if strings.Contains(string(cfgFile.Content), "merge into") {
		t.Errorf("config.json is applied directly and must not ask to be merged by hand:\n%s", cfgFile.Content)
	}
}

// composerConfigFile returns the generated config.json from SecurityConfigs.
func composerConfigFile(t *testing.T, files []types.GeneratedFile) types.GeneratedFile {
	t.Helper()
	for _, f := range files {
		if strings.HasSuffix(f.Path, "/config.json") {
			return f
		}
	}
	t.Fatalf("SecurityConfigs() = %v, want a config.json", files)
	return types.GeneratedFile{}
}

// TestSecurityConfigs_TrackedPath verifies the Composer home is the tracked
// .<app>-composer directory: config.json is committed, and a module-owned
// nested .gitignore ignores everything else in it (cache, vendor, the
// auth.json link) without depending on the root .gitignore.
func TestSecurityConfigs_TrackedPath(t *testing.T) {
	t.Parallel()
	files := (&php.Module{}).SecurityConfigs(ecosystem.ModuleConfig{})
	wantPaths := []string{".qsdev-composer/config.json", ".qsdev-composer/.gitignore"}
	gotPaths := make([]string, len(files))
	for i, f := range files {
		gotPaths[i] = f.Path
		if f.Strategy != types.Overwrite {
			t.Errorf("%s Strategy = %v, want Overwrite", f.Path, f.Strategy)
		}
	}
	if !reflect.DeepEqual(gotPaths, wantPaths) {
		t.Fatalf("SecurityConfigs() paths = %v, want %v", gotPaths, wantPaths)
	}
	gotLines := strings.Split(strings.TrimRight(string(files[1].Content), "\n"), "\n")
	if want := []string{"*", "!.gitignore", "!config.json"}; !reflect.DeepEqual(gotLines, want) {
		t.Errorf(".gitignore lines = %q, want %q", gotLines, want)
	}
}

// TestSecurityConfigs_WhiteLabel verifies both paths follow the branding
// AppName. It changes the process-wide branding, so it does not run in
// parallel (top-level parallel tests start only after it finishes).
func TestSecurityConfigs_WhiteLabel(t *testing.T) {
	orig := branding.Get().AppName
	branding.Set(branding.Config{AppName: "acme"})
	t.Cleanup(func() { branding.Set(branding.Config{AppName: orig}) })

	files := (&php.Module{}).SecurityConfigs(ecosystem.ModuleConfig{})
	got := make([]string, len(files))
	for i, f := range files {
		got[i] = f.Path
	}
	if want := []string{".acme-composer/config.json", ".acme-composer/.gitignore"}; !reflect.DeepEqual(got, want) {
		t.Errorf("SecurityConfigs() paths = %v, want %v", got, want)
	}
	frag, err := (&php.Module{}).DevenvNixFragment(ecosystem.ModuleConfig{})
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}
	if m := composerHomeRe.FindStringSubmatch(frag); m == nil || m[1] != ".acme-composer" {
		t.Errorf("COMPOSER_HOME match = %v, want .acme-composer\n%s", m, frag)
	}
	if got, want := (&php.Module{}).ReadDenyRules(ecosystem.ModuleConfig{}), []string{"./.acme-composer/auth.json"}; !reflect.DeepEqual(got, want) {
		t.Errorf("ReadDenyRules() = %v, want %v", got, want)
	}
}

// baseComposerConfigHead and baseComposerConfigTail frame the config.json
// the module generated before the move to the tracked Composer home; the
// move must not change its bytes.
const baseComposerConfigHead = `{
  "_comment": "Security-hardened Composer global configuration generated by qsdev and loaded through COMPOSER_HOME. Settings in composer.json take precedence. Regenerated on update: do not edit.",
  "_requires": "Composer ` + jsonEscapedGT + `= 2.9 for audit.block-insecure. Composer 2.9+ blocks known-vulnerable packages by default.",
`

// jsonEscapedGT is ">" as encoding/json writes it (HTML-safe escaping).
const jsonEscapedGT = "\\" + "u003e"

const baseComposerConfigTail = `  "config": {
    "secure-http": true,
    "lock": true,
    "audit": {
      "abandoned": "fail"
    },
    "allow-plugins": {},
    "preferred-install": "dist"
  }
}
`

// TestSecurityConfigs_ContentUnchanged pins config.json to the bytes the
// base generated, with and without a registry proxy.
func TestSecurityConfigs_ContentUnchanged(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		proxy string
		want  string
	}{
		{"no proxy", "", baseComposerConfigHead + baseComposerConfigTail},
		{"proxy", "https://nexus.corp.internal/repository/composer-proxy/", baseComposerConfigHead + `  "repositories": [
    {
      "type": "composer",
      "url": "https://nexus.corp.internal/repository/composer-proxy/"
    },
    {
      "packagist.org": false
    }
  ],
` + baseComposerConfigTail},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := composerConfigFile(t, (&php.Module{}).SecurityConfigs(ecosystem.ModuleConfig{RegistryProxy: tt.proxy}))
			if string(got.Content) != tt.want {
				t.Errorf("config.json =\n%s\nwant\n%s", got.Content, tt.want)
			}
		})
	}
}

// projectPathRe extracts every project-relative path a fragment exports.
var projectPathRe = regexp.MustCompile(`\$\{config\.devenv\.root\}/([^"]*)"`)

// TestDevenvNixFragment_ComposerCacheAndAuth verifies the shell exports the
// tracked Composer home and keeps Composer's cache inside it, where the
// nested .gitignore ignores it and the sandbox leaves it writable.
func TestDevenvNixFragment_ComposerCacheAndAuth(t *testing.T) {
	t.Parallel()
	frag, err := (&php.Module{}).DevenvNixFragment(ecosystem.ModuleConfig{})
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}
	for _, want := range []string{
		`  env.COMPOSER_HOME = "${config.devenv.root}/.qsdev-composer";`,
		`  env.COMPOSER_CACHE_DIR = "${config.devenv.root}/.qsdev-composer/cache";`,
	} {
		if strings.Count(frag, want) != 1 {
			t.Errorf("fragment should contain %q exactly once:\n%s", want, frag)
		}
	}
	// Every project path the fragment exports is inside the tracked home,
	// none under the sandbox-read-only state directory.
	for _, m := range projectPathRe.FindAllStringSubmatch(frag, -1) {
		if m[1] != ".qsdev-composer" && !strings.HasPrefix(m[1], ".qsdev-composer/") {
			t.Errorf("fragment exports %q outside .qsdev-composer:\n%s", m[1], frag)
		}
	}
}

// TestCICommands_NoPlugins verifies no CI Composer command runs plugins and
// validate checks lock freshness without failing on publish-only warnings.
func TestCICommands_NoPlugins(t *testing.T) {
	t.Parallel()
	byName := map[string]string{}
	for _, c := range (&php.Module{}).CICommands(ecosystem.ModuleConfig{}) {
		byName[c.Name] = c.Command
	}
	if got, want := byName["composer-install"], "composer install --no-dev --no-scripts --no-plugins --no-interaction"; got != want {
		t.Errorf("install = %q, want %q", got, want)
	}
	validate := strings.Fields(byName["composer-validate"])
	for _, want := range []string{"--no-check-publish", "--no-check-all", "--no-plugins"} {
		if !slices.Contains(validate, want) {
			t.Errorf("validate %q missing %s", byName["composer-validate"], want)
		}
	}
	for _, bad := range []string{"--strict", "--check-lock"} {
		if slices.Contains(validate, bad) {
			t.Errorf("validate %q must not contain %s", byName["composer-validate"], bad)
		}
	}
	if !slices.Contains(strings.Fields(byName["composer-audit"]), "--no-plugins") {
		t.Errorf("audit %q missing --no-plugins", byName["composer-audit"])
	}
}

// TestReadDenyRules_ComposerAuthLink verifies the in-project link to the
// user's Composer credentials is read-denied, project-relative.
func TestReadDenyRules_ComposerAuthLink(t *testing.T) {
	t.Parallel()
	var m ecosystem.EcosystemModule = &php.Module{}
	rdp, ok := m.(ecosystem.ReadDenyRuleProvider)
	if !ok {
		t.Fatal("php.Module does not implement ReadDenyRuleProvider")
	}
	if got, want := rdp.ReadDenyRules(ecosystem.ModuleConfig{}), []string{"./.qsdev-composer/auth.json"}; !reflect.DeepEqual(got, want) {
		t.Errorf("ReadDenyRules() = %v, want %v", got, want)
	}
}

// TestEnterShellScript_ContainsGuards verifies the auth passthrough snippet
// names both global auth.json locations and never replaces an existing file
// or a dangling link.
func TestEnterShellScript_ContainsGuards(t *testing.T) {
	t.Parallel()
	var m ecosystem.EcosystemModule = &php.Module{}
	esp, ok := m.(ecosystem.EnterShellProvider)
	if !ok {
		t.Fatal("php.Module does not implement EnterShellProvider")
	}
	script := esp.EnterShellScript(ecosystem.ModuleConfig{})
	for _, want := range []string{
		`"${XDG_CONFIG_HOME:-$HOME/.config}/composer/auth.json"`,
		`"$HOME/.composer/auth.json"`,
		`[ ! -e "$COMPOSER_HOME/auth.json" ]`,
		`[ ! -L "$COMPOSER_HOME/auth.json" ]`,
		`mkdir -p "$COMPOSER_HOME"`,
		`ln -s `,
	} {
		if !strings.Contains(script, want) {
			t.Errorf("snippet missing %q:\n%s", want, script)
		}
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
			cfgFile := composerConfigFile(t, (&php.Module{}).SecurityConfigs(ecosystem.ModuleConfig{RegistryProxy: tt.proxy}))
			var got composerGlobalConfig
			if err := json.Unmarshal(cfgFile.Content, &got); err != nil {
				t.Fatalf("config.json is not valid JSON: %v\n%s", err, cfgFile.Content)
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
