package cigeneration

import (
	"fmt"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

// Generator produces CI workflow files for a specific platform.
type Generator interface {
	Generate(cfg GenerateConfig, steps map[CIJobID][]CIStep) ([]types.GeneratedFile, error)
}

// NewGenerator returns a platform-specific Generator, or nil for PlatformNone.
func NewGenerator(platform CIPlatform) Generator {
	switch platform {
	case PlatformGitHubActions:
		return &GitHubActionsGenerator{}
	case PlatformGitLabCI:
		return &GitLabCIGenerator{}
	default:
		return nil
	}
}

// GenerateWorkflow is the top-level entry point: it collects steps from the
// registry, prunes empty jobs, and delegates to the platform generator.
func GenerateWorkflow(cfg GenerateConfig, registry *StepRegistry) ([]types.GeneratedFile, error) {
	if cfg.Platform == PlatformNone {
		return nil, nil
	}

	gen := NewGenerator(cfg.Platform)
	if gen == nil {
		return nil, fmt.Errorf("unsupported CI platform: %q", cfg.Platform)
	}

	steps := registry.CollectSteps(cfg)

	// Prune jobs that only have infrastructure steps (harden-runner, checkout)
	// and no actual tool steps.
	pruned := make(map[CIJobID][]CIStep)
	for jobID, jobSteps := range steps {
		hasToolStep := false
		for _, s := range jobSteps {
			if s.ToolName != "harden-runner" && s.ToolName != "checkout" {
				hasToolStep = true
				break
			}
		}
		if hasToolStep {
			pruned[jobID] = jobSteps
		}
	}

	if len(pruned) == 0 {
		return nil, nil
	}

	return gen.Generate(cfg, pruned)
}
