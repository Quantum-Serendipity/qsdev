package devinit

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/addons/devenv"
	"github.com/Quantum-Serendipity/qsdev/internal/profile"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
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

// Persisted values of WizardAnswers.MergeMode recording which generators a
// project opted into at init time.
const (
	mergeModeClaudeOnly = "claude-only"
	mergeModeDevenvOnly = "devenv-only"
)

// generationScope restricts which addon generators runAccumulator registers:
// ClaudeOnly skips devenv, DevenvOnly skips Claude Code. Tool files (see
// toolFilesGenerator) belong to neither addon and are produced in every scope.
type generationScope struct {
	ClaudeOnly bool
	DevenvOnly bool
}

// scopeFromAnswers derives the generation scope from persisted answers so that
// every regeneration path (init, join, update, repair) honours the scope the
// project was initialized with.
func scopeFromAnswers(answers types.WizardAnswers) generationScope {
	return generationScope{
		ClaudeOnly: answers.MergeMode == mergeModeClaudeOnly,
		DevenvOnly: answers.MergeMode == mergeModeDevenvOnly,
	}
}

// applyScopeFlags records --claude-only / --devenv-only in answers so the
// scope survives into later update and repair runs.
func applyScopeFlags(opts InitOptions, answers *types.WizardAnswers) {
	switch {
	case opts.ClaudeOnly:
		answers.MergeMode = mergeModeClaudeOnly
	case opts.DevenvOnly:
		answers.MergeMode = mergeModeDevenvOnly
	}
}

func runAccumulator(answers types.WizardAnswers, scope generationScope) (accumulatorResult, error) {
	registry := ecosystem.DefaultRegistry()
	acc := generate.NewFragmentAccumulator()

	if !scope.ClaudeOnly {
		devenvGen := devenv.NewDevenvGenerator(registry, devenv.WithProfileRegistry(profile.DefaultProfileRegistry()))
		if err := acc.RegisterProducer("devenv", generate.NewGeneratorAdapter("devenv", devenvGen)); err != nil {
			return accumulatorResult{}, err
		}
	}

	if !scope.DevenvOnly && answers.ClaudeCode {
		ccGen := claudecode.NewClaudeCodeGenerator(registry, claudecode.CurrentConfig())
		if err := acc.RegisterProducer("claudecode", generate.NewGeneratorAdapter("claudecode", ccGen)); err != nil {
			return accumulatorResult{}, err
		}
	}

	tools := toolFilesGenerator{registry: toolreg.DefaultRegistry()}
	if err := acc.RegisterProducer(toolFilesSource, generate.NewGeneratorAdapter(toolFilesSource, tools)); err != nil {
		return accumulatorResult{}, err
	}

	if err := acc.CollectAll(answers); err != nil {
		return accumulatorResult{}, err
	}

	allFiles, err := acc.Resolve()
	if err != nil {
		return accumulatorResult{}, err
	}

	var devenvCount, claudeCount, toolCount int
	for _, f := range acc.FragmentSet() {
		switch f.Source {
		case "devenv":
			devenvCount++
		case "claudecode":
			claudeCount++
		case toolFilesSource:
			toolCount++
		}
	}

	slog.Info("files generated via fragment accumulator",
		"devenv", devenvCount,
		"claudecode", claudeCount,
		toolFilesSource, toolCount,
		"total_files", len(allFiles))

	return accumulatorResult{
		allFiles:        allFiles,
		fragments:       acc.FragmentSet(),
		devenvGenerated: devenvCount > 0,
		claudeGenerated: claudeCount > 0,
	}, nil
}

// warnEcosystemSetup writes a warning for each file the configured ecosystems
// need but the project at projectRoot lacks (see ecosystem.SetupWarner), such
// as the pyproject.toml a Poetry project needs. Only the devenv configuration
// depends on those files, so a Claude-only scope skips the check.
func warnEcosystemSetup(w io.Writer, projectRoot string, answers types.WizardAnswers, scope generationScope) {
	if scope.ClaudeOnly {
		return
	}
	for _, msg := range ecosystem.DefaultRegistry().SetupWarnings(projectRoot, answers) {
		_, _ = fmt.Fprintln(w, "Warning: "+msg)
	}
}
