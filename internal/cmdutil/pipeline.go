package cmdutil

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/Quantum-Serendipity/qsdev/pkg/generate"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// PipelineConfig configures the write-state-save pipeline.
type PipelineConfig struct {
	ProjectRoot string
	StatePath   string // relative path to state file
	DryRun      bool
	Out         io.Writer
}

// PipelineResult holds the outcome of a pipeline execution.
type PipelineResult struct {
	WriteResult generate.WriteResult
	StateFile   string
}

// RunPipeline executes the standard write→state→save sequence.
// For dry-run mode, it previews files and returns early.
func RunPipeline(cfg PipelineConfig, files []types.GeneratedFile) (PipelineResult, error) {
	if cfg.DryRun {
		preview := generate.PreviewFiles(files, nil, cfg.ProjectRoot)
		_, _ = fmt.Fprint(cfg.Out, preview)
		return PipelineResult{}, nil
	}

	result, err := generate.WriteFiles(files, generate.PipelineOptions{
		ProjectRoot: cfg.ProjectRoot,
	})
	if err != nil {
		return PipelineResult{}, fmt.Errorf("writing files: %w", err)
	}

	genState := state.RecordFiles(files)
	stateFile := filepath.Join(cfg.ProjectRoot, cfg.StatePath)
	if err := state.SaveStateToFile(stateFile, genState); err != nil {
		return PipelineResult{}, fmt.Errorf("saving state: %w", err)
	}

	return PipelineResult{
		WriteResult: result,
		StateFile:   stateFile,
	}, nil
}
