package devinit

import (
	"log/slog"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/addons/devenv"
	"github.com/Quantum-Serendipity/qsdev/internal/profile"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/generate"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

type accumulatorResult struct {
	allFiles        []types.GeneratedFile
	fragments       []types.FragmentEntry
	devenvGenerated bool
	claudeGenerated bool
}

func runAccumulator(answers types.WizardAnswers, opts struct {
	ClaudeOnly bool
	DevenvOnly bool
}) (accumulatorResult, error) {
	registry := ecosystem.DefaultRegistry()
	acc := generate.NewFragmentAccumulator()

	if !opts.ClaudeOnly {
		devenvGen := devenv.NewDevenvGenerator(registry, devenv.WithProfileRegistry(profile.DefaultProfileRegistry()))
		if err := acc.RegisterProducer("devenv", generate.NewGeneratorAdapter("devenv", devenvGen)); err != nil {
			return accumulatorResult{}, err
		}
	}

	if !opts.DevenvOnly && answers.ClaudeCode {
		ccGen := claudecode.NewClaudeCodeGenerator(registry, claudecode.CurrentConfig())
		if err := acc.RegisterProducer("claudecode", generate.NewGeneratorAdapter("claudecode", ccGen)); err != nil {
			return accumulatorResult{}, err
		}
	}

	if err := acc.CollectAll(answers); err != nil {
		return accumulatorResult{}, err
	}

	allFiles, err := acc.Resolve()
	if err != nil {
		return accumulatorResult{}, err
	}

	var devenvCount, claudeCount int
	for _, f := range acc.FragmentSet() {
		switch f.Source {
		case "devenv":
			devenvCount++
		case "claudecode":
			claudeCount++
		}
	}

	slog.Info("files generated via fragment accumulator",
		"devenv", devenvCount,
		"claudecode", claudeCount,
		"total_files", len(allFiles))

	return accumulatorResult{
		allFiles:        allFiles,
		fragments:       acc.FragmentSet(),
		devenvGenerated: devenvCount > 0,
		claudeGenerated: claudeCount > 0,
	}, nil
}
