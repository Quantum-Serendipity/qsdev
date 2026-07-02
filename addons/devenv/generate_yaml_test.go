package devenv_test

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/addons/devenv"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/python"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// helper to generate with defaults and fail on error.
func mustGenerate(t *testing.T, answers types.WizardAnswers, registry *ecosystem.Registry) *types.GeneratedFile {
	t.Helper()
	gf, err := devenv.GenerateDevenvYaml(answers, registry)
	if err != nil {
		t.Fatalf("GenerateDevenvYaml returned error: %v", err)
	}
	return gf
}

// helper to unmarshal the generated YAML (skipping the comment header).
func mustUnmarshal(t *testing.T, gf *types.GeneratedFile) map[string]any {
	t.Helper()
	var m map[string]any
	if err := yaml.Unmarshal(gf.Content, &m); err != nil {
		t.Fatalf("YAML unmarshal failed: %v\nContent:\n%s", err, string(gf.Content))
	}
	return m
}

func goLanguage() types.LanguageChoice {
	return types.LanguageChoice{
		Name:           "go",
		Version:        "1.24",
		PackageManager: "gomod",
	}
}

func pythonLanguage() types.LanguageChoice {
	return types.LanguageChoice{
		Name:           "python",
		Version:        "3.12",
		PackageManager: "pip",
	}
}

func TestBasicGoProject(t *testing.T) {
	reg := ecosystem.NewRegistry()
	_ = reg.Register(&ecosystem.MockModule{
		NameVal:        "go",
		DisplayNameVal: "Go",
		TierVal:        1,
	})

	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{goLanguage()},
	}
	gf := mustGenerate(t, answers, reg)
	m := mustUnmarshal(t, gf)

	// Verify hardened defaults. impure stays top-level; the unfree/insecure
	// keys are camelCase nested under nixpkgs: per the devenv 2.x schema.
	if m["impure"] != false {
		t.Errorf("impure should be false, got %v", m["impure"])
	}
	nixpkgs, ok := m["nixpkgs"].(map[string]any)
	if !ok {
		t.Fatal("nixpkgs should be a map")
	}
	if nixpkgs["allowUnfree"] != true {
		t.Errorf("nixpkgs.allowUnfree should be true, got %v", nixpkgs["allowUnfree"])
	}
	if nixpkgs["allowBroken"] != false {
		t.Errorf("nixpkgs.allowBroken should be false, got %v", nixpkgs["allowBroken"])
	}
	if m["require_version"] != ">=2.1" {
		t.Errorf("require_version should be >=2.1, got %v", m["require_version"])
	}

	// Verify nixpkgs input (distinct from the nixpkgs: config block above).
	inputs, ok := m["inputs"].(map[string]any)
	if !ok {
		t.Fatal("inputs should be a map")
	}
	nixpkgsInput, ok := inputs["nixpkgs"].(map[string]any)
	if !ok {
		t.Fatal("inputs.nixpkgs should be a map")
	}
	if nixpkgsInput["url"] != "github:NixOS/nixpkgs/nixpkgs-unstable" {
		t.Errorf("nixpkgs url wrong: %v", nixpkgsInput["url"])
	}

	// Verify clean section.
	clean, ok := m["clean"].(map[string]any)
	if !ok {
		t.Fatal("clean should be a map")
	}
	if clean["enabled"] != true {
		t.Errorf("clean.enabled should be true, got %v", clean["enabled"])
	}
	keep, ok := clean["keep"].([]any)
	if !ok {
		t.Fatal("clean.keep should be a list")
	}
	if len(keep) == 0 {
		t.Error("clean.keep should have entries")
	}

	// Verify nixpkgs.permitted*Packages are present as empty lists.
	unfree, ok := nixpkgs["permittedUnfreePackages"].([]any)
	if !ok {
		t.Fatal("nixpkgs.permittedUnfreePackages should be a list")
	}
	if len(unfree) != 0 {
		t.Errorf("nixpkgs.permittedUnfreePackages should be empty, got %v", unfree)
	}
	insecure, ok := nixpkgs["permittedInsecurePackages"].([]any)
	if !ok {
		t.Fatal("nixpkgs.permittedInsecurePackages should be a list")
	}
	if len(insecure) != 0 {
		t.Errorf("nixpkgs.permittedInsecurePackages should be empty, got %v", insecure)
	}
}

func TestMultiLanguageGoAndPython(t *testing.T) {
	reg := ecosystem.NewRegistry()
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

	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{goLanguage(), pythonLanguage()},
	}
	gf := mustGenerate(t, answers, reg)
	m := mustUnmarshal(t, gf)

	// Should still produce valid YAML with hardened defaults.
	if m["impure"] != false {
		t.Errorf("impure should be false")
	}
	if m["require_version"] != ">=2.1" {
		t.Errorf("require_version wrong: %v", m["require_version"])
	}
}

func TestWithGitHooksInputPresent(t *testing.T) {
	reg := ecosystem.NewRegistry()
	_ = reg.Register(&ecosystem.MockModule{
		NameVal:        "go",
		DisplayNameVal: "Go",
		TierVal:        1,
	})

	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{goLanguage()},
		GitHooks:  []string{"ripsecrets", "check-added-large-files"},
	}
	gf := mustGenerate(t, answers, reg)
	m := mustUnmarshal(t, gf)

	inputs := m["inputs"].(map[string]any)
	gitHooks, ok := inputs["git-hooks"].(map[string]any)
	if !ok {
		t.Fatal("git-hooks input should be present when GitHooks are set")
	}
	if gitHooks["url"] != "github:cachix/git-hooks.nix" {
		t.Errorf("git-hooks url wrong: %v", gitHooks["url"])
	}

	// git-hooks must NOT have top-level follows (that aliases the entire input).
	if _, hasFollows := gitHooks["follows"]; hasFollows {
		t.Error("git-hooks must not have top-level follows (aliases entire input to nixpkgs)")
	}

	// git-hooks should have nested inputs.nixpkgs.follows = "nixpkgs".
	subInputs, ok := gitHooks["inputs"].(map[string]any)
	if !ok {
		t.Fatal("git-hooks should have nested inputs for sub-input follows")
	}
	nixpkgsSub, ok := subInputs["nixpkgs"].(map[string]any)
	if !ok {
		t.Fatal("git-hooks.inputs should have nixpkgs sub-input")
	}
	if nixpkgsSub["follows"] != "nixpkgs" {
		t.Errorf("git-hooks.inputs.nixpkgs.follows should be 'nixpkgs', got %v", nixpkgsSub["follows"])
	}
}

func TestWithoutExplicitHooksStillHasGitHooksInput(t *testing.T) {
	reg := ecosystem.NewRegistry()
	_ = reg.Register(&ecosystem.MockModule{
		NameVal:        "go",
		DisplayNameVal: "Go",
		TierVal:        1,
		// No PreCommitHooksVal set → returns nil.
	})

	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{goLanguage()},
		// No GitHooks.
	}
	gf := mustGenerate(t, answers, reg)
	m := mustUnmarshal(t, gf)

	// git-hooks input should always be present because security hooks are mandatory.
	inputs := m["inputs"].(map[string]any)
	if _, ok := inputs["git-hooks"]; !ok {
		t.Error("git-hooks input should always be present (security hooks are mandatory)")
	}
}

func TestGitHooksInferredFromModulePreCommitHooks(t *testing.T) {
	reg := ecosystem.NewRegistry()
	_ = reg.Register(&ecosystem.MockModule{
		NameVal:        "go",
		DisplayNameVal: "Go",
		TierVal:        1,
		PreCommitHooksVal: []ecosystem.HookConfig{
			{ID: "golangci-lint", Name: "golangci-lint", BuiltIn: true},
		},
	})

	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{goLanguage()},
		// No explicit GitHooks — should be inferred from module.
	}
	gf := mustGenerate(t, answers, reg)
	m := mustUnmarshal(t, gf)

	inputs := m["inputs"].(map[string]any)
	if _, ok := inputs["git-hooks"]; !ok {
		t.Error("git-hooks input should be present when module has PreCommitHooks")
	}
}

func TestEcosystemModuleInputsMerged(t *testing.T) {
	reg := ecosystem.NewRegistry()
	_ = reg.Register(&ecosystem.MockModule{
		NameVal:        "python",
		DisplayNameVal: "Python",
		TierVal:        1,
		DevenvYamlInputsVal: []ecosystem.DevenvInput{
			{
				URL:     "github:cachix/nixpkgs-python",
				Follows: "nixpkgs",
			},
		},
	})

	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{pythonLanguage()},
	}
	gf := mustGenerate(t, answers, reg)
	m := mustUnmarshal(t, gf)

	inputs := m["inputs"].(map[string]any)
	pythonInput, ok := inputs["nixpkgs-python"].(map[string]any)
	if !ok {
		t.Fatal("nixpkgs-python input should be present from ecosystem module")
	}
	if pythonInput["url"] != "github:cachix/nixpkgs-python" {
		t.Errorf("nixpkgs-python url wrong: %v", pythonInput["url"])
	}
	// Must NOT have top-level follows (that aliases the entire input).
	if _, hasFollows := pythonInput["follows"]; hasFollows {
		t.Error("nixpkgs-python must not have top-level follows")
	}
	// Should have nested inputs.nixpkgs.follows = "nixpkgs".
	subInputs, ok := pythonInput["inputs"].(map[string]any)
	if !ok {
		t.Fatal("nixpkgs-python should have nested inputs for sub-input follows")
	}
	nixpkgsSub, ok := subInputs["nixpkgs"].(map[string]any)
	if !ok {
		t.Fatal("nixpkgs-python.inputs should have nixpkgs sub-input")
	}
	if nixpkgsSub["follows"] != "nixpkgs" {
		t.Errorf("nixpkgs-python.inputs.nixpkgs.follows should be 'nixpkgs', got %v", nixpkgsSub["follows"])
	}
}

func TestEcosystemInputDoesNotOverrideNixpkgs(t *testing.T) {
	reg := ecosystem.NewRegistry()
	_ = reg.Register(&ecosystem.MockModule{
		NameVal:        "go",
		DisplayNameVal: "Go",
		TierVal:        1,
		DevenvYamlInputsVal: []ecosystem.DevenvInput{
			{
				URL: "github:NixOS/nixpkgs/some-other-branch",
			},
		},
	})

	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{goLanguage()},
	}
	gf := mustGenerate(t, answers, reg)
	m := mustUnmarshal(t, gf)

	inputs := m["inputs"].(map[string]any)
	nixpkgs := inputs["nixpkgs"].(map[string]any)
	// The hardened nixpkgs URL should not be overridden by ecosystem input.
	if nixpkgs["url"] != "github:NixOS/nixpkgs/nixpkgs-unstable" {
		t.Errorf("nixpkgs URL was overridden by ecosystem input: %v", nixpkgs["url"])
	}
}

func TestYAMLRoundTrip(t *testing.T) {
	reg := ecosystem.NewRegistry()
	_ = reg.Register(&ecosystem.MockModule{
		NameVal:        "go",
		DisplayNameVal: "Go",
		TierVal:        1,
		PreCommitHooksVal: []ecosystem.HookConfig{
			{ID: "golangci-lint", Name: "golangci-lint"},
		},
	})

	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{goLanguage()},
		GitHooks:  []string{"ripsecrets"},
	}
	gf := mustGenerate(t, answers, reg)

	// Unmarshal and re-marshal to verify round-trip stability.
	var first map[string]any
	if err := yaml.Unmarshal(gf.Content, &first); err != nil {
		t.Fatalf("first unmarshal failed: %v", err)
	}
	remarshaled, err := yaml.Marshal(first)
	if err != nil {
		t.Fatalf("re-marshal failed: %v", err)
	}
	var second map[string]any
	if err := yaml.Unmarshal(remarshaled, &second); err != nil {
		t.Fatalf("second unmarshal failed: %v", err)
	}

	// Verify key fields survive round-trip.
	if second["impure"] != first["impure"] {
		t.Errorf("impure changed: %v → %v", first["impure"], second["impure"])
	}
	firstNixpkgs, _ := first["nixpkgs"].(map[string]any)
	secondNixpkgs, _ := second["nixpkgs"].(map[string]any)
	if firstNixpkgs == nil || secondNixpkgs == nil {
		t.Fatalf("nixpkgs block missing after round-trip: first=%v second=%v", first["nixpkgs"], second["nixpkgs"])
	}
	if secondNixpkgs["allowUnfree"] != firstNixpkgs["allowUnfree"] {
		t.Errorf("nixpkgs.allowUnfree changed: %v → %v", firstNixpkgs["allowUnfree"], secondNixpkgs["allowUnfree"])
	}
	if second["require_version"] != first["require_version"] {
		t.Errorf("require_version changed: %v → %v", first["require_version"], second["require_version"])
	}
}

func TestSecurityDefaultsWithEmptyAnswers(t *testing.T) {
	reg := ecosystem.NewRegistry()
	answers := types.WizardAnswers{}
	gf := mustGenerate(t, answers, reg)
	m := mustUnmarshal(t, gf)

	// All security-critical fields must be present.
	if m["impure"] != false {
		t.Errorf("impure should be false with empty answers, got %v", m["impure"])
	}
	nixpkgs, ok := m["nixpkgs"].(map[string]any)
	if !ok {
		t.Fatal("nixpkgs block should be present with empty answers")
	}
	if nixpkgs["allowUnfree"] != true {
		t.Errorf("nixpkgs.allowUnfree should be true with empty answers, got %v", nixpkgs["allowUnfree"])
	}
	if nixpkgs["allowBroken"] != false {
		t.Errorf("nixpkgs.allowBroken should be false with empty answers, got %v", nixpkgs["allowBroken"])
	}

	clean := m["clean"].(map[string]any)
	if clean["enabled"] != true {
		t.Errorf("clean.enabled should be true with empty answers")
	}

	// nixpkgs.permitted*Packages must be present as empty lists.
	if _, ok := nixpkgs["permittedUnfreePackages"]; !ok {
		t.Error("nixpkgs.permittedUnfreePackages must be present in output")
	}
	if _, ok := nixpkgs["permittedInsecurePackages"]; !ok {
		t.Error("nixpkgs.permittedInsecurePackages must be present in output")
	}

	// require_version must be present.
	if m["require_version"] != ">=2.1" {
		t.Errorf("require_version should be >=2.1, got %v", m["require_version"])
	}

	// nixpkgs input must be present.
	inputs := m["inputs"].(map[string]any)
	if _, ok := inputs["nixpkgs"]; !ok {
		t.Error("nixpkgs input must be present")
	}
}

func TestFileMetadata(t *testing.T) {
	reg := ecosystem.NewRegistry()
	answers := types.WizardAnswers{}
	gf := mustGenerate(t, answers, reg)

	if gf.Path != "devenv.yaml" {
		t.Errorf("path should be devenv.yaml, got %q", gf.Path)
	}
	if gf.Mode != 0o644 {
		t.Errorf("mode should be 0o644, got %04o", gf.Mode)
	}
	if gf.Strategy != types.Overwrite {
		t.Errorf("strategy should be Overwrite, got %v", gf.Strategy)
	}
	if len(gf.Content) == 0 {
		t.Error("content should not be empty")
	}
}

func TestNilRegistryHandledGracefully(t *testing.T) {
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{goLanguage()},
	}
	gf, err := devenv.GenerateDevenvYaml(answers, nil)
	if err != nil {
		t.Fatalf("nil registry should not cause error: %v", err)
	}
	m := mustUnmarshal(t, gf)

	// Should still have all hardened defaults.
	if m["impure"] != false {
		t.Errorf("impure should be false with nil registry")
	}
	if m["require_version"] != ">=2.1" {
		t.Errorf("require_version should be >=2.1 with nil registry")
	}

	// git-hooks should be present even with nil registry (security hooks are mandatory).
	inputs := m["inputs"].(map[string]any)
	if _, ok := inputs["git-hooks"]; !ok {
		t.Error("git-hooks should be present even with nil registry (security hooks are mandatory)")
	}
}

func TestNilRegistryWithExplicitGitHooks(t *testing.T) {
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{goLanguage()},
		GitHooks:  []string{"ripsecrets"},
	}
	gf, err := devenv.GenerateDevenvYaml(answers, nil)
	if err != nil {
		t.Fatalf("nil registry should not cause error: %v", err)
	}
	m := mustUnmarshal(t, gf)

	// git-hooks SHOULD be present because explicit hooks were requested.
	inputs := m["inputs"].(map[string]any)
	if _, ok := inputs["git-hooks"]; !ok {
		t.Error("git-hooks should be present when explicit GitHooks are set, even with nil registry")
	}
}

func TestHeaderCommentPresent(t *testing.T) {
	reg := ecosystem.NewRegistry()
	answers := types.WizardAnswers{}
	gf := mustGenerate(t, answers, reg)
	content := string(gf.Content)

	if !strings.HasPrefix(content, "# "+branding.GeneratedBy()+" init") {
		t.Error("YAML should start with header comment")
	}
	if !strings.Contains(content, "devenv.sh/reference/yaml-options") {
		t.Error("YAML header should contain reference URL")
	}
}

func TestBoolFieldsExplicitInOutput(t *testing.T) {
	reg := ecosystem.NewRegistry()
	answers := types.WizardAnswers{}
	gf := mustGenerate(t, answers, reg)
	content := string(gf.Content)

	// These false values are security-critical and must appear explicitly,
	// in the devenv 2.x camelCase form (allowUnfree/allowBroken under nixpkgs:).
	if !strings.Contains(content, "impure: false") {
		t.Error("'impure: false' must appear explicitly in YAML output")
	}
	if !strings.Contains(content, "allowUnfree: true") {
		t.Error("'allowUnfree: true' must appear explicitly in YAML output")
	}
	if !strings.Contains(content, "allowBroken: false") {
		t.Error("'allowBroken: false' must appear explicitly in YAML output")
	}
}

func TestPermittedPackagesEmptyArrayInOutput(t *testing.T) {
	reg := ecosystem.NewRegistry()
	answers := types.WizardAnswers{}
	gf := mustGenerate(t, answers, reg)
	content := string(gf.Content)

	// Empty lists should be serialized as [] not omitted (camelCase, nested).
	if !strings.Contains(content, "permittedUnfreePackages: []") {
		t.Errorf("permittedUnfreePackages should be serialized as []\nContent:\n%s", content)
	}
	if !strings.Contains(content, "permittedInsecurePackages: []") {
		t.Errorf("permittedInsecurePackages should be serialized as []\nContent:\n%s", content)
	}
}

// TestRealPythonModuleAddsNixpkgsPythonInput is the end-to-end proof (not a
// mock) that the real Python module contributes the nixpkgs-python flake input
// required by languages.python.version. Regression guard for DEFECT-1.
func TestRealPythonModuleAddsNixpkgsPythonInput(t *testing.T) {
	reg := ecosystem.NewRegistry()
	if err := reg.Register(&python.Module{}); err != nil {
		t.Fatalf("registering python module: %v", err)
	}
	answers := types.WizardAnswers{Languages: []types.LanguageChoice{pythonLanguage()}}
	m := mustUnmarshal(t, mustGenerate(t, answers, reg))

	inputs, ok := m["inputs"].(map[string]any)
	if !ok {
		t.Fatal("inputs should be a map")
	}
	np, ok := inputs["nixpkgs-python"].(map[string]any)
	if !ok {
		t.Fatalf("inputs.nixpkgs-python missing; got inputs=%v", inputs)
	}
	if np["url"] != "github:cachix/nixpkgs-python" {
		t.Errorf("nixpkgs-python url = %v, want github:cachix/nixpkgs-python", np["url"])
	}
	// Must follow the root nixpkgs (rendered as inputs.nixpkgs.follows).
	sub, ok := np["inputs"].(map[string]any)
	if !ok {
		t.Fatalf("nixpkgs-python.inputs missing; got %v", np)
	}
	nested, ok := sub["nixpkgs"].(map[string]any)
	if !ok {
		t.Fatalf("nixpkgs-python.inputs.nixpkgs missing; got %v", sub)
	}
	if nested["follows"] != "nixpkgs" {
		t.Errorf("nixpkgs-python follows = %v, want nixpkgs", nested["follows"])
	}
}

func TestCleanKeepContainsExpectedVars(t *testing.T) {
	reg := ecosystem.NewRegistry()
	answers := types.WizardAnswers{}
	gf := mustGenerate(t, answers, reg)
	content := string(gf.Content)

	// PATH must be kept: clearing it strips /usr/bin, ~/.local/bin (claude),
	// and the qsdev binary from the devenv shell (devenv's own rcfile calls
	// mktemp before setting PATH).
	expectedVars := []string{"PATH", "TERM", "HOME", "SSH_AUTH_SOCK", "NIX_SSL_CERT_FILE"}
	for _, v := range expectedVars {
		if !strings.Contains(content, v) {
			t.Errorf("clean.keep should contain %s", v)
		}
	}
}

func TestMultipleEcosystemInputs(t *testing.T) {
	reg := ecosystem.NewRegistry()
	_ = reg.Register(&ecosystem.MockModule{
		NameVal:        "python",
		DisplayNameVal: "Python",
		TierVal:        1,
		DevenvYamlInputsVal: []ecosystem.DevenvInput{
			{
				URL:     "github:cachix/nixpkgs-python",
				Follows: "nixpkgs",
			},
		},
	})
	_ = reg.Register(&ecosystem.MockModule{
		NameVal:        "rust",
		DisplayNameVal: "Rust",
		TierVal:        1,
		DevenvYamlInputsVal: []ecosystem.DevenvInput{
			{
				URL: "github:oxalica/rust-overlay",
			},
		},
	})

	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			pythonLanguage(),
			{Name: "rust", Version: "stable", PackageManager: "cargo"},
		},
	}
	gf := mustGenerate(t, answers, reg)
	m := mustUnmarshal(t, gf)

	inputs := m["inputs"].(map[string]any)
	if _, ok := inputs["nixpkgs-python"]; !ok {
		t.Error("nixpkgs-python input should be present")
	}
	if _, ok := inputs["rust-overlay"]; !ok {
		t.Error("rust-overlay input should be present")
	}
	// nixpkgs should still be present.
	if _, ok := inputs["nixpkgs"]; !ok {
		t.Error("nixpkgs input should still be present")
	}
}

func TestUnknownLanguageInRegistrySkipped(t *testing.T) {
	reg := ecosystem.NewRegistry()
	// Registry has no "go" module registered.
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{goLanguage()},
	}
	gf, err := devenv.GenerateDevenvYaml(answers, reg)
	if err != nil {
		t.Fatalf("unknown language should not cause error: %v", err)
	}
	m := mustUnmarshal(t, gf)

	// Should still produce valid YAML with hardened defaults.
	if m["impure"] != false {
		t.Errorf("impure should be false")
	}
}
