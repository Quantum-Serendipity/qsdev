package devenv_test

import (
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/devenv"
	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/validation"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// newTestRegistry creates a registry and registers the given mock modules.
func newTestRegistry(t *testing.T, mocks ...*ecosystem.MockModule) *ecosystem.Registry {
	t.Helper()
	reg := ecosystem.NewRegistry()
	for _, m := range mocks {
		if err := reg.Register(m); err != nil {
			t.Fatalf("registering mock %q: %v", m.NameVal, err)
		}
	}
	return reg
}

func goMock() *ecosystem.MockModule {
	return &ecosystem.MockModule{
		NameVal:        "go",
		DisplayNameVal: "Go",
		TierVal:        1,
		DevenvNixFragmentVal: `  languages.go = {
    enable = true;
    package = pkgs.go;
  };

  env.GOFLAGS = "-mod=readonly";
  env.GONOSUMCHECK = "";
  env.GONOSUMDB = "";`,
		PreCommitHooksVal: []ecosystem.HookConfig{
			{ID: "gofmt", Name: "gofmt", Description: "Format Go source code", Entry: "gofmt -l -w", Language: "system", Types: []string{"go"}, Stages: []string{"pre-commit"}, PassFilenames: true, BuiltIn: true},
			{ID: "govet", Name: "govet", Description: "Run go vet", Entry: "go vet ./...", Language: "system", Types: []string{"go"}, Stages: []string{"pre-commit"}, BuiltIn: true},
			{ID: "staticcheck", Name: "staticcheck", Description: "Run staticcheck", Entry: "staticcheck ./...", Language: "system", Types: []string{"go"}, Stages: []string{"pre-commit"}, BuiltIn: false, NixPackage: "go-tools"},
		},
	}
}

func pythonMock() *ecosystem.MockModule {
	return &ecosystem.MockModule{
		NameVal:        "python",
		DisplayNameVal: "Python",
		TierVal:        1,
		DevenvNixFragmentVal: `  languages.python = {
    enable = true;
    version = "3.12";
    venv.enable = true;
  };`,
		PreCommitHooksVal: []ecosystem.HookConfig{
			{ID: "ruff", Name: "ruff", Description: "Run ruff linter", Entry: "ruff check --fix", Language: "python", Types: []string{"python"}, Stages: []string{"pre-commit"}, PassFilenames: true, BuiltIn: true},
			{ID: "mypy", Name: "mypy", Description: "Run mypy type checker", Entry: "mypy", Language: "python", Types: []string{"python"}, Stages: []string{"pre-commit"}, PassFilenames: true, BuiltIn: true},
		},
	}
}

func TestGenerateDevenvNix_SingleLanguage(t *testing.T) {
	reg := newTestRegistry(t, goMock())
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "go"},
		},
	}

	got, err := devenv.GenerateDevenvNix(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(got.Content)

	// Verify Go language block is present.
	requireNixAttr(t, nixAttrs(t, got.Content), "languages.go.enable")
	requireContains(t, content, `enable = true`)

	// Verify security defaults.
	requireContains(t, content, `DEVENV_SECURITY_HARDENED`)
	requireContains(t, content, `dotenv.enable = false`)
	requireContains(t, content, `ripsecrets`)
	requireContains(t, content, `unsetEnvVars`)

	// Verify file metadata.
	if got.Path != "devenv.nix" {
		t.Errorf("Path = %q, want %q", got.Path, "devenv.nix")
	}
	if got.Mode != 0o644 {
		t.Errorf("Mode = %#o, want %#o", got.Mode, 0o644)
	}
	if got.Strategy != types.ManualMerge {
		t.Errorf("Strategy = %v, want ManualMerge", got.Strategy)
	}
}

func TestGenerateDevenvNix_MultiLanguage(t *testing.T) {
	reg := newTestRegistry(t, goMock(), pythonMock())
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "go"},
			{Name: "python", Version: "3.12"},
		},
	}

	got, err := devenv.GenerateDevenvNix(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(got.Content)

	// Both language fragments appear.
	requireContains(t, content, "# Go")
	requireContains(t, content, "# Python")
	attrs := nixAttrs(t, got.Content)
	requireNixAttr(t, attrs, "languages.go.enable")
	requireNixAttr(t, attrs, "languages.python.enable")
}

func TestGenerateDevenvNix_WithServices(t *testing.T) {
	reg := newTestRegistry(t, goMock())
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "go"},
		},
		Services: []types.ServiceChoice{
			{
				Name:    "postgres",
				Version: "16",
				Settings: map[string]string{
					"initial_db": "myapp",
				},
			},
			{
				Name: "redis",
				Settings: map[string]string{
					"port": "6380",
				},
			},
		},
	}

	got, err := devenv.GenerateDevenvNix(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(got.Content)

	attrs := nixAttrs(t, got.Content)
	requireNixAttr(t, attrs, "services.postgres.enable")
	requireContains(t, content, "enable = true")
	requireContains(t, content, "postgresql_16")
	requireContains(t, content, `"myapp"`)

	requireNixAttr(t, attrs, "services.redis.enable")
	requireContains(t, content, "port = 6380")
}

func TestGenerateDevenvNix_SecurityDefaultsAlwaysPresent(t *testing.T) {
	reg := newTestRegistry(t) // No modules, empty answers.
	answers := types.WizardAnswers{}

	got, err := devenv.GenerateDevenvNix(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(got.Content)

	// Security-hardened env var.
	requireContains(t, content, `DEVENV_SECURITY_HARDENED`)
	requireContains(t, content, `"true"`)

	// Unset env vars block.
	requireContains(t, content, `unsetEnvVars`)
	requireContains(t, content, `AWS_ACCESS_KEY_ID`)
	requireContains(t, content, `GITHUB_TOKEN`)
	requireContains(t, content, `VAULT_TOKEN`)

	// Dotenv disabled.
	requireContains(t, content, `dotenv.enable = false`)

	// Git hooks with security hooks (baseline).
	requireContains(t, content, `git-hooks.hooks`)
	requireContains(t, content, `ripsecrets.enable = true`)
	requireContains(t, content, `check-added-large-files.enable = true`)
	requireContains(t, content, `no-commit-to-branch.enable = true`)
	requireContains(t, content, `check-merge-conflicts.enable = true`)
	requireContains(t, content, `shellcheck.enable = true`)
	requireContains(t, content, `statix.enable = true`)

	// Prek hook runner comment.
	requireContains(t, content, `prek`)

	// Specialized hooks (always present).
	requireContains(t, content, `lock-file-audit`)
	requireContains(t, content, `nix-secrets-check`)

	// enterShell and enterTest.
	requireContains(t, content, `enterShell`)
	requireContains(t, content, `enterTest`)
	requireContains(t, content, `Security-Hardened Development Environment`)
	requireContains(t, content, `Security Validation`)
}

func TestGenerateDevenvNix_HookComposition(t *testing.T) {
	reg := newTestRegistry(t, goMock(), pythonMock())
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "go"},
			{Name: "python"},
		},
	}

	got, err := devenv.GenerateDevenvNix(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(got.Content)

	// Security hooks (always present).
	requireContains(t, content, "ripsecrets.enable = true")
	requireContains(t, content, "shellcheck.enable = true")

	// Built-in hooks from Go module.
	requireContains(t, content, "gofmt.enable = true")
	requireContains(t, content, "govet.enable = true")

	// Built-in hooks from Python module.
	requireContains(t, content, "ruff.enable = true")
	requireContains(t, content, "mypy.enable = true")

	// Custom hook from Go module (staticcheck has NixPackage: go-tools).
	requireContains(t, content, "staticcheck")
	requireContains(t, content, `${pkgs.go-tools}/bin/staticcheck`)
}

func TestGenerateDevenvNix_UnknownLanguageReturnsError(t *testing.T) {
	reg := newTestRegistry(t) // Empty registry.
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "cobol"},
		},
	}

	_, err := devenv.GenerateDevenvNix(answers, reg)
	if err == nil {
		t.Fatal("expected error for unknown language, got nil")
	}
	if !strings.Contains(err.Error(), "cobol") {
		t.Errorf("error message should mention the unknown language: %v", err)
	}
}

func TestGenerateDevenvNix_UnknownServiceReturnsError(t *testing.T) {
	reg := newTestRegistry(t)
	answers := types.WizardAnswers{
		Services: []types.ServiceChoice{
			{Name: "cassandra"},
		},
	}

	_, err := devenv.GenerateDevenvNix(answers, reg)
	if err == nil {
		t.Fatal("expected error for unknown service, got nil")
	}
	if !strings.Contains(err.Error(), "cassandra") {
		t.Errorf("error message should mention the unknown service: %v", err)
	}
}

func TestGenerateDevenvNix_ExtraPackagesAndEnvVars(t *testing.T) {
	reg := newTestRegistry(t)
	answers := types.WizardAnswers{
		ExtraPackages: []string{"ripgrep", "fd"},
		EnvVars: map[string]string{
			"EDITOR": "vim",
		},
	}

	got, err := devenv.GenerateDevenvNix(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(got.Content)

	// Base packages + extras.
	requireContains(t, content, "pkgs.git")
	requireContains(t, content, "pkgs.ripgrep")
	requireContains(t, content, "pkgs.fd")

	// Custom env var (inside env = { ... } block).
	requireContains(t, content, `EDITOR = "vim"`)
	requireContains(t, content, "env = {")
}

func TestGenerateDevenvNix_EnterShellEscaping(t *testing.T) {
	reg := newTestRegistry(t)
	answers := types.WizardAnswers{}

	got, err := devenv.GenerateDevenvNix(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(got.Content)

	// Shell variable references like ${VAR} must be escaped to ''${VAR}
	// for Nix multiline strings.
	requireContains(t, content, `''${AWS_SECRET_ACCESS_KEY:-}`)
	requireContains(t, content, `''${DEVENV_SECURITY_HARDENED:-}`)
}

// TestGenerateDevenvNix_NixInstantiateParse checks that the devenv.nix
// generated for each ecosystem module, with every service and catalog tool
// enabled, is syntactically valid Nix. devenv.nix is a function of
// { pkgs, lib, config, ... }, so raw expressions that reference pkgs are in
// scope for `nix-instantiate --parse`, which also rejects references to
// undefined variables.
func TestGenerateDevenvNix_NixInstantiateParse(t *testing.T) {
	nixInstantiate, err := exec.LookPath("nix-instantiate")
	if err != nil {
		t.Skip("nix-instantiate not available, skipping syntax validation")
	}

	cat, err := catalog.Default()
	if err != nil {
		t.Fatalf("loading catalog: %v", err)
	}
	enabledTools := make(map[string]bool)
	for name := range cat.Tools() {
		enabledTools[name] = true
	}
	var services []types.ServiceChoice
	for _, name := range validation.Services() {
		services = append(services, types.ServiceChoice{Name: name})
	}

	reg := ecosystem.DefaultRegistry()
	for _, lang := range reg.Names() {
		t.Run(lang, func(t *testing.T) {
			t.Parallel()
			answers := types.WizardAnswers{
				ProjectName:   "parse-check",
				Direnv:        true,
				Languages:     []types.LanguageChoice{{Name: lang}},
				Services:      services,
				ExtraPackages: []string{"jq", "python3Packages.requests"},
				EnvVars:       map[string]string{"EDITOR": "vim"},
				Overlays:      []string{"./nix/overlay.nix"},
				EnabledTools:  enabledTools,
			}
			got, err := devenv.GenerateDevenvNix(answers, reg)
			if err != nil {
				t.Fatalf("GenerateDevenvNix: %v", err)
			}

			path := filepath.Join(t.TempDir(), "devenv.nix")
			if err := os.WriteFile(path, got.Content, 0o644); err != nil {
				t.Fatalf("writing devenv.nix: %v", err)
			}
			out, err := exec.Command(nixInstantiate, "--parse", path).CombinedOutput()
			if err != nil {
				t.Fatalf("nix-instantiate --parse rejected generated devenv.nix: %v\n%s", err, out)
			}
		})
	}
}

func TestGenerateDevenvNix_HookDeduplication(t *testing.T) {
	// Two modules returning the same hook ID should not produce duplicates.
	mod1 := &ecosystem.MockModule{
		NameVal:              "lang1",
		DisplayNameVal:       "Lang1",
		TierVal:              1,
		DevenvNixFragmentVal: "  # lang1 fragment",
		PreCommitHooksVal: []ecosystem.HookConfig{
			{ID: "shared-lint", Name: "shared-lint", Description: "Shared linter", Entry: "lint", Language: "system", BuiltIn: true},
		},
	}
	mod2 := &ecosystem.MockModule{
		NameVal:              "lang2",
		DisplayNameVal:       "Lang2",
		TierVal:              1,
		DevenvNixFragmentVal: "  # lang2 fragment",
		PreCommitHooksVal: []ecosystem.HookConfig{
			{ID: "shared-lint", Name: "shared-lint", Description: "Shared linter", Entry: "lint", Language: "system", BuiltIn: true},
		},
	}

	reg := newTestRegistry(t, mod1, mod2)
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "lang1"},
			{Name: "lang2"},
		},
	}

	got, err := devenv.GenerateDevenvNix(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(got.Content)

	// Count occurrences of the hook. It should appear exactly once.
	count := strings.Count(content, "shared-lint.enable = true")
	if count != 1 {
		t.Errorf("shared-lint.enable = true appeared %d times, want exactly 1", count)
	}
}

func TestGenerateDevenvNix_ModuleHookMatchingSecurityHookRenderedOnce(t *testing.T) {
	t.Parallel()
	mod := &ecosystem.MockModule{
		NameVal:              "shelly",
		DisplayNameVal:       "Shelly",
		TierVal:              1,
		DevenvNixFragmentVal: "  # shelly fragment",
		PreCommitHooksVal: []ecosystem.HookConfig{
			{ID: "shellcheck", Name: "shellcheck", Entry: "shellcheck", Language: "system", BuiltIn: true},
			{ID: "statix", Name: "statix", Entry: "statix check", Language: "system", NixPackage: "statix"},
		},
	}
	reg := newTestRegistry(t, mod)
	answers := types.WizardAnswers{Languages: []types.LanguageChoice{{Name: "shelly"}}}

	got, err := devenv.GenerateDevenvNix(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content := string(got.Content)
	for _, id := range []string{"shellcheck", "statix"} {
		if n := strings.Count(content, "    "+id+".enable = true;") + strings.Count(content, "    "+id+" = {"); n != 1 {
			t.Errorf("hook %q defined %d times in devenv.nix, want exactly 1", id, n)
		}
	}
}

func TestBuildDevenvNixData_LanguageFragmentErrorPropagates(t *testing.T) {
	mod := &ecosystem.MockModule{
		NameVal:              "broken",
		DisplayNameVal:       "Broken",
		TierVal:              1,
		DevenvNixFragmentErr: errBrokenModule,
	}

	reg := newTestRegistry(t, mod)
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "broken"},
		},
	}

	_, err := devenv.GenerateDevenvNix(answers, reg)
	if err == nil {
		t.Fatal("expected error from broken module, got nil")
	}
	if !strings.Contains(err.Error(), "broken") {
		t.Errorf("error should mention module name: %v", err)
	}
}

var errBrokenModule = &brokenError{}

type brokenError struct{}

func (e *brokenError) Error() string { return "module is broken" }

// requireContains asserts that s contains the substring sub.
// nixAttrs returns the flattened attribute paths devenv.nix defines.
func nixAttrs(t *testing.T, content []byte) map[string]string {
	t.Helper()
	attrs, err := devenv.NixModuleAttrs(string(content))
	if err != nil {
		t.Fatalf("NixModuleAttrs: %v\n%s", err, content)
	}
	return attrs
}

// hasNixAttr reports whether path, or an attribute below it, is defined.
func hasNixAttr(attrs map[string]string, path string) bool {
	if _, ok := attrs[path]; ok {
		return true
	}
	for k := range attrs {
		if strings.HasPrefix(k, path+".") {
			return true
		}
	}
	return false
}

func requireNixAttr(t *testing.T, attrs map[string]string, path string) {
	t.Helper()
	if !hasNixAttr(attrs, path) {
		t.Errorf("devenv.nix does not define %s; defined: %v", path, slices.Sorted(maps.Keys(attrs)))
	}
}

func requireContains(t *testing.T, s, sub string) {
	t.Helper()
	if !strings.Contains(s, sub) {
		t.Errorf("output does not contain %q\n\nFull output:\n%s", sub, s)
	}
}

// TestGenerateDevenvNix_BuiltInHookOptions verifies W063/W141: built-in
// hooks carry their file-type limits and settings into devenv.nix instead of
// rendering as a bare `.enable = true` that runs git-hooks.nix's defaults
// (prettier on every text file, eslint on .js only, nixpkgs binaries).
func TestGenerateDevenvNix_BuiltInHookOptions(t *testing.T) {
	t.Parallel()
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{{
			Name:           "javascript",
			PackageManager: "npm",
			Extras:         []string{"prettier=node_modules", "eslint=node_modules"},
		}},
	}
	got, err := devenv.GenerateDevenvNix(answers, ecosystem.DefaultRegistry())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content := string(got.Content)

	requireContains(t, content, `    prettier = {
      enable = true;
      types_or = [ "javascript" "jsx" "ts" "tsx" "css" "scss" "less" ];
      settings.binPath = "./node_modules/.bin/prettier";
    };`)
	requireContains(t, content, `    eslint = {
      enable = true;
      settings = {
        binPath = "./node_modules/.bin/eslint";
        extensions = "\\.(c|m)?[jt]sx?$";
      };
    };`)
	if strings.Contains(content, "prettier.enable = true") {
		t.Errorf("prettier rendered without its type limits:\n%s", content)
	}
}

// TestGenerateDevenvNix_BuiltInHookWithoutProjectToolIsOff verifies that a
// JavaScript project using neither ESLint nor Prettier gets neither hook, so
// commits are not blocked by tools the project never adopted (W063).
func TestGenerateDevenvNix_BuiltInHookWithoutProjectToolIsOff(t *testing.T) {
	t.Parallel()
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{{Name: "javascript", PackageManager: "npm"}},
	}
	got, err := devenv.GenerateDevenvNix(answers, ecosystem.DefaultRegistry())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, id := range []string{"prettier", "eslint"} {
		if strings.Contains(string(got.Content), "    "+id+" = {") || strings.Contains(string(got.Content), id+".enable") {
			t.Errorf("%s hook enabled for a project that does not use it", id)
		}
	}
}

// TestGenerateDevenvNix_BuiltInHookRejectsBadSettingKey guards the unquoted
// `settings.<key>` rendering against keys that are not Nix attribute names.
func TestGenerateDevenvNix_BuiltInHookRejectsBadSettingKey(t *testing.T) {
	t.Parallel()
	mock := goMock()
	mock.PreCommitHooksVal = []ecosystem.HookConfig{{
		ID: "gofmt", BuiltIn: true, Settings: map[string]string{`x"; evil = "`: "v"},
	}}
	answers := types.WizardAnswers{Languages: []types.LanguageChoice{{Name: "go"}}}
	if _, err := devenv.GenerateDevenvNix(answers, newTestRegistry(t, mock)); err == nil {
		t.Fatal("expected an error for an invalid hook setting key")
	}
}
