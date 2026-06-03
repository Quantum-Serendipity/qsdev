package generate

import (
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/internal/cigeneration"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// CIFragmentProducer wraps the existing CI generation system as a
// FragmentProducer, converting GenerateWorkflow output into ComposeReplace
// fragments for the accumulation pipeline.
type CIFragmentProducer struct {
	Registry *cigeneration.StepRegistry
	Config   cigeneration.GenerateConfig
}

func (p *CIFragmentProducer) Produce(_ types.WizardAnswers) ([]types.FragmentEntry, error) {
	if p.Config.Platform == cigeneration.PlatformNone {
		return nil, nil
	}

	files, err := cigeneration.GenerateWorkflow(p.Config, p.Registry)
	if err != nil {
		return nil, fmt.Errorf("generating CI workflow: %w", err)
	}
	if len(files) == 0 {
		return nil, nil
	}

	fragments := make([]types.FragmentEntry, 0, len(files))
	for _, file := range files {
		fragments = append(fragments, types.FragmentEntry{
			Source:      "ci-generation",
			Target:      file.Path,
			Content:     file.Content,
			Priority:    types.PriorityCIWorkflow,
			ComposeMode: types.ComposeReplace,
			Strategy:    file.Strategy,
			Mode:        file.Mode,
			Owner:       file.Owner,
		})
	}

	return fragments, nil
}
