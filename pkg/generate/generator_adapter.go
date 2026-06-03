package generate

import (
	"time"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// GeneratorAdapter wraps a legacy Generator into a FragmentProducer.
type GeneratorAdapter struct {
	name      string
	generator types.Generator
}

func NewGeneratorAdapter(name string, gen types.Generator) *GeneratorAdapter {
	return &GeneratorAdapter{name: name, generator: gen}
}

func (a *GeneratorAdapter) Produce(answers types.WizardAnswers) ([]types.FragmentEntry, error) {
	files, err := a.generator.Generate(answers)
	if err != nil {
		return nil, err
	}

	entries := make([]types.FragmentEntry, len(files))
	for i, f := range files {
		mode := f.Mode
		if mode == 0 {
			mode = 0o644
		}
		entries[i] = types.FragmentEntry{
			Source:      a.name,
			Target:      f.Path,
			Content:     f.Content,
			Priority:    1000,
			ComposeMode: types.ComposeReplace,
			Strategy:    f.Strategy,
			Mode:        mode,
			Owner:       f.Owner,
			Provenance: types.FragmentProvenance{
				Module:    a.name,
				Timestamp: time.Now().UTC(),
				Reason:    "generator output",
			},
		}
	}
	return entries, nil
}
