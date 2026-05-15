package devenv_test

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/devenv"
	"github.com/Quantum-Serendipity/qsdev/internal/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestGenerate_ImplementsGeneratorInterface is a compile-time check.
var _ types.Generator = (*devenv.DevenvGenerator)(nil)

func TestGenerate_AllFileTypes(t *testing.T) {
	reg := ecosystem.NewRegistry()
	_ = reg.Register(&ecosystem.MockModule{
		NameVal:        "go",
		DisplayNameVal: "Go",
		TierVal:        1,
	})
	_ = reg.Register(&ecosystem.MockModule{
		NameVal:        "javascript",
		DisplayNameVal: "JavaScript/TypeScript",
		TierVal:        1,
		SecurityConfigsVal: []types.GeneratedFile{
			{
				Path:    ".npmrc",
				Content: []byte("audit=true\n"),
				Mode:    0o644,
			},
		},
	})

	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "go", Version: "1.24"},
			{Name: "javascript", Version: "22"},
		},
		Direnv: true,
	}

	gen := devenv.NewDevenvGenerator(reg)
	files, err := gen.Generate(answers)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	paths := make(map[string]bool)
	for _, f := range files {
		paths[f.Path] = true
	}

	expected := []string{"devenv.yaml", "devenv.nix", ".envrc", ".npmrc"}
	for _, p := range expected {
		if !paths[p] {
			t.Errorf("expected file %q in output, got paths: %v", p, pathKeys(files))
		}
	}
}

func TestGenerate_NoDirenv(t *testing.T) {
	reg := ecosystem.NewRegistry()
	_ = reg.Register(&ecosystem.MockModule{
		NameVal:        "go",
		DisplayNameVal: "Go",
		TierVal:        1,
	})

	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "go", Version: "1.24"},
		},
		Direnv: false,
	}

	gen := devenv.NewDevenvGenerator(reg)
	files, err := gen.Generate(answers)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	for _, f := range files {
		if f.Path == ".envrc" {
			t.Error(".envrc should not be in output when Direnv=false")
		}
	}
}

func TestGenerate_NoLanguages(t *testing.T) {
	reg := ecosystem.NewRegistry()

	answers := types.WizardAnswers{
		Direnv: false,
	}

	gen := devenv.NewDevenvGenerator(reg)
	files, err := gen.Generate(answers)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	paths := make(map[string]bool)
	for _, f := range files {
		paths[f.Path] = true
	}

	if !paths["devenv.yaml"] {
		t.Error("expected devenv.yaml in minimal output")
	}
	if !paths["devenv.nix"] {
		t.Error("expected devenv.nix in minimal output")
	}

	// Should only have these two files.
	if len(files) != 2 {
		t.Errorf("expected 2 files for minimal generation, got %d: %v", len(files), pathKeys(files))
	}
}

func TestGenerate_SecurityConfigsCollected(t *testing.T) {
	reg := ecosystem.NewRegistry()
	_ = reg.Register(&ecosystem.MockModule{
		NameVal:        "python",
		DisplayNameVal: "Python",
		TierVal:        1,
		SecurityConfigsVal: []types.GeneratedFile{
			{Path: "pip.conf", Content: []byte("[global]\nrequire-hashes = true\n"), Mode: 0o644},
			{Path: ".python-version", Content: []byte("3.12\n"), Mode: 0o644},
		},
	})

	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "python", Version: "3.12"},
		},
		Direnv: true,
	}

	gen := devenv.NewDevenvGenerator(reg)
	files, err := gen.Generate(answers)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	paths := make(map[string]bool)
	for _, f := range files {
		paths[f.Path] = true
	}

	if !paths["pip.conf"] {
		t.Error("expected pip.conf from Python SecurityConfigs")
	}
	if !paths[".python-version"] {
		t.Error("expected .python-version from Python SecurityConfigs")
	}
}

func TestGenerate_ErrorOnUnknownLanguage(t *testing.T) {
	reg := ecosystem.NewRegistry()
	// Registry has no modules registered.

	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "cobol"},
		},
	}

	gen := devenv.NewDevenvGenerator(reg)
	_, err := gen.Generate(answers)
	if err == nil {
		t.Fatal("expected error for unknown language, got nil")
	}

	// The devenv.nix generator (BuildDevenvNixData) also errors on unknown modules.
	// Either that or our SecurityConfigs lookup should produce an error.
}

// pathKeys extracts file paths from a slice of GeneratedFile for diagnostics.
func pathKeys(files []types.GeneratedFile) []string {
	paths := make([]string, len(files))
	for i, f := range files {
		paths[i] = f.Path
	}
	return paths
}
