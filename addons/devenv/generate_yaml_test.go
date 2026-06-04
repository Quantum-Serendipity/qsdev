package devenv_test

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/addons/devenv"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
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
func mustUnmarshal(t *testing.T, gf *types.GeneratedFile) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
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

	// Verify hardened defaults.
	if m["impure"] != false {
		t.Errorf("impure should be false, got %v", m["impure"])
	}
	if m["allow_unfree"] != true {
		t.Errorf("allow_unfree should be true, got %v", m["allow_unfree"])
	}
	if m["allow_broken"] != false {
		t.Errorf("allow_broken should be false, got %v", m["allow_broken"])
	}
	if m["require_version"] != ">=2.1" {
		t.Errorf("require_version should be >=2.1, got %v", m["require_version"])
	}

	// Verify nixpkgs input.
	inputs, ok := m["inputs"].(map[string]interface{})
	if !ok {
		t.Fatal("inputs should be a map")
	}
	nixpkgs, ok := inputs["nixpkgs"].(map[string]interface{})
	if !ok {
		t.Fatal("inputs.nixpkgs should be a map")
	}
	if nixpkgs["url"] != "github:NixOS/nixpkgs/nixpkgs-unstable" {
		t.Errorf("nixpkgs url wrong: %v", nixpkgs["url"])
	}

	// Verify clean section.
	clean, ok := m["clean"].(map[string]interface{})
	if !ok {
		t.Fatal("clean should be a map")
	}
	if clean["enabled"] != true {
		t.Errorf("clean.enabled should be true, got %v", clean["enabled"])
	}
	keep, ok := clean["keep"].([]interface{})
	if !ok {
		t.Fatal("clean.keep should be a list")
	}
	if len(keep) == 0 {
		t.Error("clean.keep should have entries")
	}

	// Verify permitted_*_packages are present as empty lists.
	unfree, ok := m["permitted_unfree_packages"].([]interface{})
	if !ok {
		t.Fatal("permitted_unfree_packages should be a list")
	}
	if len(unfree) != 0 {
		t.Errorf("permitted_unfree_packages should be empty, got %v", unfree)
	}
	insecure, ok := m["permitted_insecure_packages"].([]interface{})
	if !ok {
		t.Fatal("permitted_insecure_packages should be a list")
	}
	if len(insecure) != 0 {
		t.Errorf("permitted_insecure_packages should be empty, got %v", insecure)
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

	inputs := m["inputs"].(map[string]interface{})
	gitHooks, ok := inputs["git-hooks"].(map[string]interface{})
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
	subInputs, ok := gitHooks["inputs"].(map[string]interface{})
	if !ok {
		t.Fatal("git-hooks should have nested inputs for sub-input follows")
	}
	nixpkgsSub, ok := subInputs["nixpkgs"].(map[string]interface{})
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
	inputs := m["inputs"].(map[string]interface{})
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

	inputs := m["inputs"].(map[string]interface{})
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

	inputs := m["inputs"].(map[string]interface{})
	pythonInput, ok := inputs["nixpkgs-python"].(map[string]interface{})
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
	subInputs, ok := pythonInput["inputs"].(map[string]interface{})
	if !ok {
		t.Fatal("nixpkgs-python should have nested inputs for sub-input follows")
	}
	nixpkgsSub, ok := subInputs["nixpkgs"].(map[string]interface{})
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

	inputs := m["inputs"].(map[string]interface{})
	nixpkgs := inputs["nixpkgs"].(map[string]interface{})
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
	var first map[string]interface{}
	if err := yaml.Unmarshal(gf.Content, &first); err != nil {
		t.Fatalf("first unmarshal failed: %v", err)
	}
	remarshaled, err := yaml.Marshal(first)
	if err != nil {
		t.Fatalf("re-marshal failed: %v", err)
	}
	var second map[string]interface{}
	if err := yaml.Unmarshal(remarshaled, &second); err != nil {
		t.Fatalf("second unmarshal failed: %v", err)
	}

	// Verify key fields survive round-trip.
	if second["impure"] != first["impure"] {
		t.Errorf("impure changed: %v → %v", first["impure"], second["impure"])
	}
	if second["allow_unfree"] != first["allow_unfree"] {
		t.Errorf("allow_unfree changed: %v → %v", first["allow_unfree"], second["allow_unfree"])
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
	if m["allow_unfree"] != true {
		t.Errorf("allow_unfree should be true with empty answers, got %v", m["allow_unfree"])
	}
	if m["allow_broken"] != false {
		t.Errorf("allow_broken should be false with empty answers, got %v", m["allow_broken"])
	}

	clean := m["clean"].(map[string]interface{})
	if clean["enabled"] != true {
		t.Errorf("clean.enabled should be true with empty answers")
	}

	// permitted_*_packages must be present as empty lists.
	if _, ok := m["permitted_unfree_packages"]; !ok {
		t.Error("permitted_unfree_packages must be present in output")
	}
	if _, ok := m["permitted_insecure_packages"]; !ok {
		t.Error("permitted_insecure_packages must be present in output")
	}

	// require_version must be present.
	if m["require_version"] != ">=2.1" {
		t.Errorf("require_version should be >=2.1, got %v", m["require_version"])
	}

	// nixpkgs input must be present.
	inputs := m["inputs"].(map[string]interface{})
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
	inputs := m["inputs"].(map[string]interface{})
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
	inputs := m["inputs"].(map[string]interface{})
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

	// These false values are security-critical and must appear explicitly.
	if !strings.Contains(content, "impure: false") {
		t.Error("'impure: false' must appear explicitly in YAML output")
	}
	if !strings.Contains(content, "allow_unfree: true") {
		t.Error("'allow_unfree: true' must appear explicitly in YAML output")
	}
	if !strings.Contains(content, "allow_broken: false") {
		t.Error("'allow_broken: false' must appear explicitly in YAML output")
	}
}

func TestPermittedPackagesEmptyArrayInOutput(t *testing.T) {
	reg := ecosystem.NewRegistry()
	answers := types.WizardAnswers{}
	gf := mustGenerate(t, answers, reg)
	content := string(gf.Content)

	// Empty lists should be serialized as [] not omitted.
	if !strings.Contains(content, "permitted_unfree_packages: []") {
		t.Errorf("permitted_unfree_packages should be serialized as []\nContent:\n%s", content)
	}
	if !strings.Contains(content, "permitted_insecure_packages: []") {
		t.Errorf("permitted_insecure_packages should be serialized as []\nContent:\n%s", content)
	}
}

func TestCleanKeepContainsExpectedVars(t *testing.T) {
	reg := ecosystem.NewRegistry()
	answers := types.WizardAnswers{}
	gf := mustGenerate(t, answers, reg)
	content := string(gf.Content)

	expectedVars := []string{"TERM", "HOME", "SSH_AUTH_SOCK", "NIX_SSL_CERT_FILE"}
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

	inputs := m["inputs"].(map[string]interface{})
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
