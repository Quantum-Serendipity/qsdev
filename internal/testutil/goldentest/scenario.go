package goldentest

import (
	"path/filepath"
	"sort"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// ScenarioRunner calls generators with specific WizardAnswers and compares all
// output files against golden files stored on disk.
type ScenarioRunner struct {
	generators map[string]types.Generator
	answers    types.WizardAnswers
}

func NewScenarioRunner() *ScenarioRunner {
	return &ScenarioRunner{generators: make(map[string]types.Generator)}
}

func (r *ScenarioRunner) AddGenerator(name string, gen types.Generator) {
	r.generators[name] = gen
}

func (r *ScenarioRunner) SetAnswers(a types.WizardAnswers) {
	r.answers = a
}

// Run generates files from all generators and compares each against golden files
// in goldenDir. Files are matched by their GeneratedFile.Path.
func (r *ScenarioRunner) Run(t *testing.T, goldenDir string) {
	t.Helper()

	var allFiles []types.GeneratedFile

	// Sort generator names for deterministic ordering.
	names := make([]string, 0, len(r.generators))
	for name := range r.generators {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		gen := r.generators[name]
		files, err := gen.Generate(r.answers)
		if err != nil {
			t.Fatalf("generator %q: %v", name, err)
		}
		allFiles = append(allFiles, files...)
	}

	if len(allFiles) == 0 {
		t.Fatal("no files generated")
	}

	for _, f := range allFiles {
		goldenPath := filepath.Join(goldenDir, f.Path)
		Assert(t, goldenPath, f.Content)
	}
}
