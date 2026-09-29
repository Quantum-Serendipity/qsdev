package javascript_test

import (
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/javascript"
)

// TestDetect_PackageManagerPin verifies W064: a package.json package-manager
// pin (Corepack "packageManager", then devEngines.packageManager) decides the
// package manager before lockfiles, so a pnpm project without a lockfile is
// not hardened as npm.
func TestDetect_PackageManagerPin(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		packageJSON string
		lockfile    string
		want        string
	}{
		{"packageManager without lockfile", `{"packageManager":"pnpm@10.17.0"}`, "", "pnpm"},
		{"packageManager with integrity suffix", `{"packageManager":"yarn@4.9.2+sha512.abc"}`, "", "yarn"},
		{"packageManager beats lockfile", `{"packageManager":"bun@1.3.0"}`, "package-lock.json", "bun"},
		{"devEngines object", `{"devEngines":{"packageManager":{"name":"pnpm","version":"^10"}}}`, "", "pnpm"},
		{"devEngines array", `{"devEngines":{"packageManager":[{"name":"yarn","version":"4.x"}]}}`, "", "yarn"},
		{"unsupported pin falls back to lockfile", `{"packageManager":"deno@2.0.0"}`, "pnpm-lock.yaml", "pnpm"},
		{"no pin, no lockfile", `{}`, "", "npm"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			writeTestFile(t, dir, "package.json", tt.packageJSON)
			if tt.lockfile != "" {
				writeTestFile(t, dir, tt.lockfile, "")
			}
			if got := (&javascript.Module{}).Detect(dir).SuggestedConfig.PackageManager; got != tt.want {
				t.Errorf("PackageManager = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestDetect_NodeVersionWarning verifies W055/W142: a Node.js version that
// nixpkgs no longer packages is reported at detection, not silently swapped.
func TestDetect_NodeVersionWarning(t *testing.T) {
	t.Parallel()
	tests := []struct {
		nvmrc    string
		wantWarn bool
	}{
		{"20.11.1", true},
		{"lts/iron", true},
		{"22", false},
		{"lts/*", false},
	}
	for _, tt := range tests {
		t.Run(tt.nvmrc, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			writeTestFile(t, dir, "package.json", `{}`)
			writeTestFile(t, dir, ".nvmrc", tt.nvmrc+"\n")
			res := (&javascript.Module{}).Detect(dir)
			got := slices.ContainsFunc(res.Evidence, func(e string) bool {
				return strings.HasPrefix(e, "WARNING:") && strings.Contains(e, "Node.js")
			})
			if got != tt.wantWarn {
				t.Errorf("warning = %v, want %v; evidence: %v", got, tt.wantWarn, res.Evidence)
			}
		})
	}
}

// TestDetect_JSTools verifies W063: Detect records whether the project uses
// ESLint / Prettier and whether the hook should run the project's own copy.
func TestDetect_JSTools(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		packageJSON  string
		files        []string
		wantESLint   string
		wantPrettier string
	}{
		{"neither", `{}`, nil, "", ""},
		{"dev dependencies", `{"devDependencies":{"eslint":"^9","prettier":"^3"}}`, nil, "node_modules", "node_modules"},
		{"config files only", `{}`, []string{"eslint.config.mjs", ".prettierrc.json"}, "nix", "nix"},
		{"inline package.json config", `{"eslintConfig":{},"prettier":{"semi":false}}`, nil, "nix", "nix"},
		{"legacy eslintrc with dependency", `{"dependencies":{"eslint":"8"}}`, []string{".eslintrc.yml"}, "node_modules", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			writeTestFile(t, dir, "package.json", tt.packageJSON)
			for _, f := range tt.files {
				writeTestFile(t, dir, f, "")
			}
			extras := (&javascript.Module{}).Detect(dir).SuggestedConfig.Extras
			if got := extras[javascript.ExtraESLint]; got != tt.wantESLint {
				t.Errorf("eslint = %q, want %q", got, tt.wantESLint)
			}
			if got := extras[javascript.ExtraPrettier]; got != tt.wantPrettier {
				t.Errorf("prettier = %q, want %q", got, tt.wantPrettier)
			}
		})
	}
}

// TestPreCommitHooks_ProjectAware verifies W063/W141: eslint and prettier are
// enabled only when the project uses them; prettier is limited to JS/TS and
// stylesheets (never qsdev's generated YAML/JSON/Markdown); eslint covers
// TypeScript; and a project dependency is run from node_modules.
func TestPreCommitHooks_ProjectAware(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		extras       map[string]string
		wantIDs      []string
		wantBinPaths map[string]string
	}{
		{"unused tools", nil, nil, nil},
		{
			"project dependencies",
			map[string]string{javascript.ExtraPrettier: "node_modules", javascript.ExtraESLint: "node_modules"},
			[]string{"prettier", "eslint"},
			map[string]string{"prettier": "./node_modules/.bin/prettier", "eslint": "./node_modules/.bin/eslint"},
		},
		{"config without dependency", map[string]string{javascript.ExtraESLint: "nix"}, []string{"eslint"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			hooks := (&javascript.Module{}).PreCommitHooks(ecosystem.ModuleConfig{Extras: tt.extras})
			var ids []string
			for _, h := range hooks {
				ids = append(ids, h.ID)
				if got := h.Settings["binPath"]; got != tt.wantBinPaths[h.ID] {
					t.Errorf("%s binPath = %q, want %q", h.ID, got, tt.wantBinPaths[h.ID])
				}
				switch h.ID {
				case "prettier":
					checkPrettierTypes(t, h.TypesOr)
				case "eslint":
					checkESLintExtensions(t, h.Settings["extensions"])
				}
			}
			if !slices.Equal(ids, tt.wantIDs) {
				t.Errorf("hook IDs = %v, want %v", ids, tt.wantIDs)
			}
		})
	}
}

func checkPrettierTypes(t *testing.T, typesOr []string) {
	t.Helper()
	for _, generated := range []string{"yaml", "json", "markdown", "text"} {
		if slices.Contains(typesOr, generated) {
			t.Errorf("prettier types_or %v includes %q, which covers qsdev-generated files", typesOr, generated)
		}
	}
	if !slices.Contains(typesOr, "ts") {
		t.Errorf("prettier types_or %v misses TypeScript", typesOr)
	}
}

func checkESLintExtensions(t *testing.T, extensions string) {
	t.Helper()
	re := regexp.MustCompile(extensions)
	for _, f := range []string{"a.js", "a.mjs", "a.cjs", "a.jsx", "a.ts", "a.tsx", "a.mts"} {
		if !re.MatchString(f) {
			t.Errorf("eslint extensions %q does not match %s", extensions, f)
		}
	}
	if re.MatchString("a.json") {
		t.Errorf("eslint extensions %q matches a.json", extensions)
	}
}

// bunInstallTable decodes the [install] table of a generated bunfig.toml.
func bunInstallTable(t *testing.T, content []byte) map[string]any {
	t.Helper()
	var bunfig struct {
		Install map[string]any `toml:"install"`
	}
	if _, err := toml.Decode(string(content), &bunfig); err != nil {
		t.Fatalf("bunfig.toml invalid: %v\n%s", err, content)
	}
	return bunfig.Install
}

// TestSecurityConfigs_RegistryProxyKeys verifies W056: each package manager
// gets the registry proxy under the key it actually reads (pnpm: `registry`,
// not the Yarn-only `npmRegistryServer`; Bun: `[install] registry`).
func TestSecurityConfigs_RegistryProxyKeys(t *testing.T) {
	t.Parallel()
	const proxy = `https://nexus.corp.example/repository/npm-proxy/?x="y"`
	m := &javascript.Module{}

	pnpm := m.SecurityConfigs(ecosystem.ModuleConfig{PackageManager: "pnpm", RegistryProxy: proxy})[0]
	var ws map[string]any
	if err := yaml.Unmarshal(pnpm.Content, &ws); err != nil {
		t.Fatalf("pnpm-workspace.yaml invalid: %v\n%s", err, pnpm.Content)
	}
	if ws["registry"] != proxy {
		t.Errorf("pnpm registry = %#v, want %q\n%s", ws["registry"], proxy, pnpm.Content)
	}
	if _, ok := ws["npmRegistryServer"]; ok {
		t.Errorf("pnpm-workspace.yaml uses the Yarn-only npmRegistryServer key\n%s", pnpm.Content)
	}

	bun := m.SecurityConfigs(ecosystem.ModuleConfig{PackageManager: "bun", RegistryProxy: proxy})[0]
	if got := bunInstallTable(t, bun.Content)["registry"]; got != proxy {
		t.Errorf("bun install.registry = %#v, want %q\n%s", got, proxy, bun.Content)
	}
}

// TestSecurityConfigs_BunIgnoresScripts verifies W059: Bun runs install
// scripts for its built-in default-trusted packages unless ignoreScripts is
// set, so the generated bunfig.toml must set it.
func TestSecurityConfigs_BunIgnoresScripts(t *testing.T) {
	t.Parallel()
	bun := (&javascript.Module{}).SecurityConfigs(ecosystem.ModuleConfig{PackageManager: "bun"})[0]
	if got := bunInstallTable(t, bun.Content)["ignoreScripts"]; got != true {
		t.Errorf("install.ignoreScripts = %#v, want true\n%s", got, bun.Content)
	}
}

// TestSecurityConfigs_PnpmOldPin verifies W147: when package.json pins a pnpm
// that predates the hardening settings, pnpm would download and switch to it
// and ignore them, so the workspace file keeps the devenv pnpm (pmOnFail:
// warn) and detection warns.
func TestSecurityConfigs_PnpmOldPin(t *testing.T) {
	t.Parallel()
	tests := []struct {
		pin      string
		wantKeep bool
	}{
		{"9.12.3", true},
		{"10.15.1", true},
		{"10.16.0", false},
		{"11.1.1", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run("pin_"+tt.pin, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			pkg := `{}`
			if tt.pin != "" {
				pkg = `{"packageManager":"pnpm@` + tt.pin + `"}`
			}
			writeTestFile(t, dir, "package.json", pkg)
			writeTestFile(t, dir, "pnpm-lock.yaml", "")
			res := (&javascript.Module{}).Detect(dir)

			cfg := (&javascript.Module{}).SecurityConfigs(res.SuggestedConfig)[0]
			var ws map[string]any
			if err := yaml.Unmarshal(cfg.Content, &ws); err != nil {
				t.Fatalf("pnpm-workspace.yaml invalid: %v\n%s", err, cfg.Content)
			}
			if got := ws["pmOnFail"] == "warn"; got != tt.wantKeep {
				t.Errorf("pmOnFail: warn = %v, want %v\n%s", got, tt.wantKeep, cfg.Content)
			}
			warned := slices.ContainsFunc(res.Evidence, func(e string) bool { return strings.Contains(e, "pins pnpm") })
			if warned != tt.wantKeep {
				t.Errorf("detection warning = %v, want %v; evidence %v", warned, tt.wantKeep, res.Evidence)
			}
		})
	}
}

// TestSecurityConfigs_NPMAuditLevelComment verifies W067: audit-level only
// sets `npm audit`'s exit code, so the .npmrc must not claim installs fail.
func TestSecurityConfigs_NPMAuditLevelComment(t *testing.T) {
	t.Parallel()
	npmrc := string((&javascript.Module{}).SecurityConfigs(ecosystem.ModuleConfig{PackageManager: "npm"})[0].Content)
	if strings.Contains(npmrc, "# Fail on moderate") {
		t.Errorf(".npmrc claims audit-level fails installs:\n%s", npmrc)
	}
	for _, want := range []string{"Installs are not blocked", "Run `npm audit` in CI", "audit-level=moderate\n"} {
		if !strings.Contains(npmrc, want) {
			t.Errorf(".npmrc lacks %q explaining audit-level scope:\n%s", want, npmrc)
		}
	}
}
