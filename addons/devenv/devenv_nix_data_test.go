package devenv

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestBuildEcosystemsList_Empty(t *testing.T) {
	answers := types.WizardAnswers{}
	got := buildEcosystemsList(answers)
	if got != "none" {
		t.Errorf("buildEcosystemsList({}) = %q, want %q", got, "none")
	}
}

func TestBuildEcosystemsList_Single(t *testing.T) {
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "go"},
		},
	}
	got := buildEcosystemsList(answers)
	if got != "go" {
		t.Errorf("buildEcosystemsList({go}) = %q, want %q", got, "go")
	}
}

func TestBuildEcosystemsList_Multiple_Sorted(t *testing.T) {
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "python"},
			{Name: "go"},
			{Name: "rust"},
		},
	}
	got := buildEcosystemsList(answers)
	if got != "go,python,rust" {
		t.Errorf("buildEcosystemsList({python, go, rust}) = %q, want %q", got, "go,python,rust")
	}
}

func TestCountEnabledTools_Empty(t *testing.T) {
	answers := types.WizardAnswers{}
	got := countEnabledTools(answers)
	if got != 0 {
		t.Errorf("countEnabledTools({}) = %d, want 0", got)
	}
}

func TestCountEnabledTools_NilMap(t *testing.T) {
	answers := types.WizardAnswers{EnabledTools: nil}
	got := countEnabledTools(answers)
	if got != 0 {
		t.Errorf("countEnabledTools(nil map) = %d, want 0", got)
	}
}

func TestCountEnabledTools_Mixed(t *testing.T) {
	answers := types.WizardAnswers{
		EnabledTools: map[string]bool{
			"tool-a": true,
			"tool-b": false,
			"tool-c": true,
			"tool-d": true,
		},
	}
	got := countEnabledTools(answers)
	if got != 3 {
		t.Errorf("countEnabledTools(3 true, 1 false) = %d, want 3", got)
	}
}

func TestBuildDevenvNixData_GdevEnvVars(t *testing.T) {
	reg := ecosystem.NewRegistry()

	answers := types.WizardAnswers{
		ProjectName:     "myproject",
		ComplianceLevel: "enhanced",
		EnabledTools: map[string]bool{
			"tool-a": true,
			"tool-b": true,
		},
		Languages: []types.LanguageChoice{
			{Name: "go"},
			{Name: "python"},
		},
	}

	// Register mock modules so BuildDevenvNixData doesn't error.
	_ = reg.Register(&ecosystem.MockModule{
		NameVal:        "go",
		DisplayNameVal: "Go",
		TierVal:        1,
	})
	_ = reg.Register(&ecosystem.MockModule{
		NameVal:        "python",
		DisplayNameVal: "Python",
		TierVal:        1,
	})

	data, err := BuildDevenvNixData(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify qsdev env vars are present.
	checks := map[string]string{
		"QSDEV_PROJECT_NAME":     "myproject",
		"QSDEV_SECURITY_PROFILE": "enhanced",
		"QSDEV_ECOSYSTEMS":       "go,python",
		"QSDEV_TOOL_COUNT":       "2",
	}
	for key, want := range checks {
		got, ok := data.EnvVars[key]
		if !ok {
			t.Errorf("EnvVars missing key %q", key)
			continue
		}
		if got != want {
			t.Errorf("EnvVars[%q] = %q, want %q", key, got, want)
		}
	}

	// QSDEV_VERSION should be present and non-empty.
	if v, ok := data.EnvVars["QSDEV_VERSION"]; !ok || v == "" {
		t.Error("EnvVars missing or empty QSDEV_VERSION")
	}
}

func TestBuildDevenvNixData_GdevEnvVars_Defaults(t *testing.T) {
	reg := ecosystem.NewRegistry()

	// Empty answers should produce safe defaults.
	answers := types.WizardAnswers{}

	data, err := BuildDevenvNixData(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if v := data.EnvVars["QSDEV_PROJECT_NAME"]; v != "unknown" {
		t.Errorf("default QSDEV_PROJECT_NAME = %q, want %q", v, "unknown")
	}
	if v := data.EnvVars["QSDEV_SECURITY_PROFILE"]; v != "standard" {
		t.Errorf("default QSDEV_SECURITY_PROFILE = %q, want %q", v, "standard")
	}
	if v := data.EnvVars["QSDEV_ECOSYSTEMS"]; v != "none" {
		t.Errorf("default QSDEV_ECOSYSTEMS = %q, want %q", v, "none")
	}
	if v := data.EnvVars["QSDEV_TOOL_COUNT"]; v != "0" {
		t.Errorf("default QSDEV_TOOL_COUNT = %q, want %q", v, "0")
	}
}

func TestBuildDevenvNixData_HookPackagesBareName(t *testing.T) {
	t.Parallel()
	reg := ecosystem.NewRegistry()
	_ = reg.Register(&ecosystem.MockModule{
		NameVal:        "go",
		DisplayNameVal: "Go",
		TierVal:        1,
		PreCommitHooksVal: []ecosystem.HookConfig{
			{
				ID:         "staticcheck",
				Name:       "staticcheck",
				Entry:      "staticcheck ./...",
				NixPackage: "go-tools",
				Language:   "system",
				Types:      []string{"go"},
				Stages:     []string{"pre-commit"},
			},
		},
	})

	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{{Name: "go"}},
	}

	data, err := BuildDevenvNixData(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, pkg := range data.Packages {
		if pkg == "pkgs.go-tools" {
			t.Error("data.Packages contains \"pkgs.go-tools\"; want bare name \"go-tools\"")
		}
	}

	found := false
	for _, pkg := range data.Packages {
		if pkg == "go-tools" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("data.Packages does not contain \"go-tools\"; got %v", data.Packages)
	}
}

func TestBuildDevenvNixData_NoUvInBasePackages(t *testing.T) {
	t.Parallel()
	reg := ecosystem.NewRegistry()
	answers := types.WizardAnswers{}

	data, err := BuildDevenvNixData(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, pkg := range data.Packages {
		if pkg == "uv" {
			t.Error("data.Packages contains \"uv\"; Python tool should not be in base packages")
		}
	}
}

func TestBuildDevenvNixData_ModulePackagesCollected(t *testing.T) {
	t.Parallel()
	reg := ecosystem.NewRegistry()
	_ = reg.Register(&ecosystem.MockModule{
		NameVal:           "go",
		DisplayNameVal:    "Go",
		TierVal:           1,
		DevenvPackagesVal: []string{"gopls", "delve"},
	})

	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{{Name: "go"}},
	}

	data, err := BuildDevenvNixData(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pkgSet := make(map[string]bool, len(data.Packages))
	for _, p := range data.Packages {
		pkgSet[p] = true
	}
	for _, want := range []string{"gopls", "delve"} {
		if !pkgSet[want] {
			t.Errorf("data.Packages missing %q from module PackageProvider; got %v", want, data.Packages)
		}
	}
}

// exprModule is a mock ecosystem module that also implements
// ecosystem.PackageExprProvider.
type exprModule struct {
	ecosystem.MockModule
	exprs []string
}

func (m *exprModule) DevenvPackageExprs(_ ecosystem.ModuleConfig) []string { return m.exprs }

func TestBuildDevenvNixData_ModulePackageExprsCollected(t *testing.T) {
	t.Parallel()
	const expr = "(pkgs.google-cloud-sdk.withExtraComponents [ pkgs.google-cloud-sdk.components.gke-gcloud-auth-plugin ])"
	reg := ecosystem.NewRegistry()
	_ = reg.Register(&exprModule{
		MockModule: ecosystem.MockModule{NameVal: "gcp", DisplayNameVal: "Google Cloud CLI", TierVal: 2},
		exprs:      []string{expr},
	})

	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{{Name: "gcp"}},
	}

	data, err := BuildDevenvNixData(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !slices.Contains(data.PackageExprs, expr) {
		t.Errorf("data.PackageExprs missing module PackageExprProvider expression; got %v", data.PackageExprs)
	}
}

func TestBuildEnterShellScript_ContainsGdevVars(t *testing.T) {
	script := buildEnterShellScript(catalog.MustDefault().UnsetVars())

	for _, want := range []string{"QSDEV_PROJECT_NAME", "QSDEV_SECURITY_PROFILE", "QSDEV_TOOL_COUNT"} {
		if !strings.Contains(script, want) {
			t.Errorf("enterShell script does not contain %q", want)
		}
	}
}

func TestBuildDevenvNixData_CatalogBasePackages(t *testing.T) {
	t.Parallel()
	reg := ecosystem.NewRegistry()
	data, err := BuildDevenvNixData(types.WizardAnswers{}, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	catPkgs := catalog.MustDefault().BasePackages()
	for _, want := range catPkgs {
		if !slices.Contains(data.Packages, want) {
			t.Errorf("data.Packages missing catalog base package %q", want)
		}
	}
}

func TestBuildDevenvNixData_CatalogSecurityHooks(t *testing.T) {
	t.Parallel()
	reg := ecosystem.NewRegistry()
	data, err := BuildDevenvNixData(types.WizardAnswers{}, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	catHooks := catalog.MustDefault().SecurityHooks()
	for _, want := range catHooks {
		if !slices.Contains(data.SecurityHooks, want) {
			t.Errorf("data.SecurityHooks missing catalog hook %q", want)
		}
	}
}

func TestBuildDevenvNixData_CatalogCustomHooks(t *testing.T) {
	t.Parallel()
	reg := ecosystem.NewRegistry()
	data, err := BuildDevenvNixData(types.WizardAnswers{}, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	catCustom := catalog.MustDefault().CustomHooks()
	catIDs := make(map[string]bool, len(catCustom))
	for _, h := range catCustom {
		catIDs[h.ID] = true
	}

	for _, h := range data.CustomHooks {
		if catIDs[h.ID] {
			delete(catIDs, h.ID)
		}
	}
	for id := range catIDs {
		t.Errorf("catalog custom hook %q not found in data.CustomHooks", id)
	}
}

func TestBuildDevenvNixData_FillDefaultsThenBuild(t *testing.T) {
	t.Parallel()
	reg := ecosystem.NewRegistry()

	answers := types.WizardAnswers{
		ClaudeCode:  true,
		Tier:        "full",
		ProjectName: "integration-test",
	}
	answers.FillDefaults(types.DetectedProject{}, catalog.MustDefault())

	data, err := BuildDevenvNixData(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cat := catalog.MustDefault()

	wantCompliance := cat.TierCompliance("full")
	if got := data.EnvVars["QSDEV_SECURITY_PROFILE"]; got != wantCompliance {
		t.Errorf("QSDEV_SECURITY_PROFILE = %q, want %q (from catalog TierCompliance)", got, wantCompliance)
	}

	wantTools := cat.TierEnabledTools("full")
	wantCount := strconv.Itoa(len(wantTools))
	if got := data.EnvVars["QSDEV_TOOL_COUNT"]; got != wantCount {
		t.Errorf("QSDEV_TOOL_COUNT = %q, want %q (from catalog TierEnabledTools)", got, wantCount)
	}

	catPkgs := cat.BasePackages()
	for _, want := range catPkgs {
		if !slices.Contains(data.Packages, want) {
			t.Errorf("data.Packages missing catalog base package %q after FillDefaults", want)
		}
	}
}

// mockHookProject returns answers selecting a single mock module that
// declares hook, and the registry holding it.
func mockHookProject(t *testing.T, hook ecosystem.HookConfig) (types.WizardAnswers, *ecosystem.Registry) {
	t.Helper()
	reg := ecosystem.NewRegistry()
	if err := reg.Register(&ecosystem.MockModule{
		NameVal:           "mock",
		DisplayNameVal:    "Mock",
		TierVal:           1,
		PreCommitHooksVal: []ecosystem.HookConfig{hook},
	}); err != nil {
		t.Fatalf("registering mock module: %v", err)
	}
	return types.WizardAnswers{Languages: []types.LanguageChoice{{Name: "mock"}}}, reg
}

// buildWithHook runs BuildDevenvNixData for mockHookProject(hook).
func buildWithHook(t *testing.T, hook ecosystem.HookConfig) (*DevenvNixTemplateData, error) {
	t.Helper()
	return BuildDevenvNixData(mockHookProject(t, hook))
}

// TestBuildDevenvNixData_LanguagePackageHook verifies a LanguagePackage hook
// runs the devenv language's pinned package: its entry and package point at
// config.languages.<name>.package, the package line overrides git-hooks.nix's
// built-in default without colliding with the language module, and nothing
// is added to packages.
func TestBuildDevenvNixData_LanguagePackageHook(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		hook        ecosystem.HookConfig
		wantEntry   string
		wantPkgExpr string
		wantPkgs    []string
	}{
		{
			name:        "language package",
			hook:        ecosystem.HookConfig{ID: "zig-fmt", Name: "zig-fmt", Entry: "zig fmt --check", Language: "system", LanguagePackage: "zig"},
			wantEntry:   `"${config.languages.zig.package}/bin/zig fmt --check"`,
			wantPkgExpr: "config.languages.zig.package",
		},
		{
			name:        "nix package",
			hook:        ecosystem.HookConfig{ID: "cppcheck", Name: "cppcheck", Entry: "cppcheck --quiet", Language: "system", NixPackage: "cppcheck"},
			wantEntry:   `"${pkgs.cppcheck}/bin/cppcheck --quiet"`,
			wantPkgExpr: "pkgs.cppcheck",
			wantPkgs:    []string{"cppcheck"},
		},
		{
			name: "no package",
			hook: ecosystem.HookConfig{ID: "bare", Name: "bare", Entry: "bare --check", Language: "system"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := buildWithHook(t, tt.hook)
			if err != nil {
				t.Fatalf("BuildDevenvNixData: %v", err)
			}
			idx := slices.IndexFunc(data.CustomHooks, func(h CustomHookData) bool { return h.ID == tt.hook.ID })
			if idx < 0 {
				t.Fatalf("custom hook %q not rendered", tt.hook.ID)
			}
			got := data.CustomHooks[idx]
			if tt.wantEntry != "" && (got.Entry != tt.wantEntry || !got.RawEntry) {
				t.Errorf("entry = %q (raw %v), want raw %q", got.Entry, got.RawEntry, tt.wantEntry)
			}
			if got.PackageExpr != tt.wantPkgExpr {
				t.Errorf("PackageExpr = %q, want %q", got.PackageExpr, tt.wantPkgExpr)
			}
			for _, p := range []string{"zig", "cppcheck"} {
				if want := slices.Contains(tt.wantPkgs, p); slices.Contains(data.Packages, p) != want {
					t.Errorf("packages %v: contains %q = %v, want %v", data.Packages, p, !want, want)
				}
			}

			rendered, err := renderDevenvNix(mockHookProject(t, tt.hook))
			if err != nil {
				t.Fatalf("rendering: %v", err)
			}
			out := string(rendered)
			wantLine := "      package = lib.mkOverride 999 " + tt.wantPkgExpr + ";\n"
			if has := strings.Contains(out, wantLine); has != (tt.wantPkgExpr != "") {
				t.Errorf("rendered package line %q present = %v, want %v\n%s", wantLine, has, tt.wantPkgExpr != "", out)
			}
		})
	}
}

// TestBuildDevenvNixData_CIToolsAtEveryTier verifies the programs module CI
// steps run are in devenv.nix at every hook tier, the supply-chain-only
// baseline included, where the language hooks that also carry them (cppcheck,
// tflint) are left out; and that a program both a hook and its module
// provision is listed once.
func TestBuildDevenvNixData_CIToolsAtEveryTier(t *testing.T) {
	t.Parallel()
	langs := []types.LanguageChoice{
		{Name: "cpp", PackageManager: "conan"},
		{Name: "terraform"},
		{Name: "go"},
		{Name: "perl"},
		{Name: "lua", PackageManager: "luarocks"},
	}
	ciTools := []string{"cppcheck", "conan", "tflint", "tfsec", "govulncheck", "perlPackages.Carton", "perlPackages.CPANAudit", "luaPackages.luarocks"}
	for _, tier := range []string{"", "baseline", "enhanced", "strict"} {
		name := tier
		if name == "" {
			name = "unset"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			answers := types.WizardAnswers{ProjectName: "ci-tools", Languages: langs, HookTier: tier, ComplianceLevel: tier}
			out, err := renderDevenvNix(answers, ecosystem.DefaultRegistry())
			if err != nil {
				t.Fatalf("renderDevenvNix: %v", err)
			}
			nix := string(out)
			for _, tool := range ciTools {
				if n := strings.Count(nix, " pkgs."+tool+" "); n != 1 {
					t.Errorf("devenv.nix package list holds pkgs.%s %d times, want once", tool, n)
				}
			}
			data, err := BuildDevenvNixData(answers, ecosystem.DefaultRegistry())
			if err != nil {
				t.Fatalf("BuildDevenvNixData: %v", err)
			}
			seen := make(map[string]bool, len(data.Packages))
			for _, p := range data.Packages {
				if seen[p] {
					t.Errorf("data.Packages lists %q more than once: %q", p, data.Packages)
				}
				seen[p] = true
			}
		})
	}
}

// TestBuildDevenvNixData_HookPackageFieldsExclusive verifies a hook may not
// name both a nixpkgs attribute and a devenv language as its package.
func TestBuildDevenvNixData_HookPackageFieldsExclusive(t *testing.T) {
	t.Parallel()
	_, err := buildWithHook(t, ecosystem.HookConfig{
		ID: "zig-fmt", Entry: "zig fmt --check", Language: "system", NixPackage: "zig", LanguagePackage: "zig",
	})
	if err == nil || !strings.Contains(err.Error(), "zig-fmt") || !strings.Contains(err.Error(), "both") {
		t.Fatalf("err = %v, want an error naming zig-fmt that sets both package fields", err)
	}
}

// TestBuildDevenvNixData_InvalidLanguagePackage verifies LanguagePackage is
// rendered into Nix only when it is a single plain identifier.
func TestBuildDevenvNixData_InvalidLanguagePackage(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"zig.package", "1zig", "zig fmt", "zig;", "${x}", "zig\"", "pkgs.zig"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := buildWithHook(t, ecosystem.HookConfig{
				ID: "fmt", Entry: "fmt --check", Language: "system", LanguagePackage: name,
			})
			if err == nil || !strings.Contains(err.Error(), "language package") {
				t.Fatalf("LanguagePackage %q: err = %v, want an invalid language package error", name, err)
			}
		})
	}
}

// TestBuildDevenvNixData_BuiltInHookRejectsPackage verifies a built-in hook
// that names a package is an error rather than silently running git-hooks.nix's
// own package instead.
func TestBuildDevenvNixData_BuiltInHookRejectsPackage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		hook ecosystem.HookConfig
	}{
		{"nix package", ecosystem.HookConfig{ID: "mix-format", BuiltIn: true, NixPackage: "elixir"}},
		{"language package", ecosystem.HookConfig{ID: "mix-format", BuiltIn: true, LanguagePackage: "elixir"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := buildWithHook(t, tt.hook)
			if err == nil || !strings.Contains(err.Error(), "built-in hook") {
				t.Fatalf("err = %v, want a built-in hook package error", err)
			}
		})
	}
	if _, err := buildWithHook(t, ecosystem.HookConfig{ID: "mix-format", BuiltIn: true}); err != nil {
		t.Fatalf("plain built-in hook: %v", err)
	}
}
