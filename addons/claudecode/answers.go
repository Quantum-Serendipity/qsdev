package claudecode

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"

	"github.com/Quantum-Serendipity/qsdev/internal/answers"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// primaryAnswersFile returns the base name of the primary (devinit) answers
// file, the single source of truth for wizard answers across all addons.
func primaryAnswersFile() string {
	return "." + branding.Get().AppName + "-init-answers.yaml"
}

// answersPath returns the full path to the answers file that claude
// subcommands read and write: the project's primary answers file.
func answersPath(projectRoot string) string {
	return answers.FilePath(projectRoot, branding.Get().StateDir, primaryAnswersFile())
}

// legacyAnswersPath returns the path of the per-addon answers copy that older
// releases kept in .claude/. It went stale whenever `qsdev enable/disable` or
// another addon updated the primary file, so it is never read; saveAnswers
// deletes it.
func legacyAnswersPath(projectRoot string) string {
	return answers.FilePath(projectRoot, AddonDir, "."+branding.Get().AppName+"-claude-answers.yaml")
}

// saveAnswers persists the wizard answers to the primary answers file and
// removes the obsolete per-addon copy.
func saveAnswers(projectRoot string, a types.WizardAnswers) error {
	if err := answers.SavePrimary(projectRoot, a); err != nil {
		return fmt.Errorf("saving primary answers: %w", err)
	}
	if err := os.Remove(legacyAnswersPath(projectRoot)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		// The legacy copy is never read, so failing to delete it is harmless.
		slog.Warn("removing legacy claude answers file", "error", err)
	}
	return nil
}

// SaveAnswers is an exported wrapper around saveAnswers, allowing other
// packages (e.g. devinit) to persist answers for the Claude Code addon.
func SaveAnswers(projectRoot string, a types.WizardAnswers) error {
	return saveAnswers(projectRoot, a)
}

// loadAnswers reads the saved wizard answers from the primary answers file. It
// returns an error telling the user to run init first if the file does not
// exist.
func loadAnswers(projectRoot string) (types.WizardAnswers, error) {
	return answers.LoadFromDir(projectRoot, branding.Get().StateDir, primaryAnswersFile(), "claude init")
}

// overlayInitAnswers merges the answers `claude init` builds from its flags
// onto the project's saved answers, so re-running claude init in a
// qsdev-managed project updates only the Claude Code settings it owns and keeps
// the languages, services, tier and enabled tools recorded by `qsdev init` and
// `qsdev enable`. Without saved answers, the flag answers are returned as-is.
//
// The permission preset, the safety-block hook and the confirmation are always
// taken from the flags; skills and MCP servers only when the flags name some.
func overlayInitAnswers(projectRoot string, flags types.WizardAnswers) (types.WizardAnswers, error) {
	if _, err := os.Stat(answersPath(projectRoot)); errors.Is(err, fs.ErrNotExist) {
		return flags, nil
	}
	merged, err := loadAnswers(projectRoot)
	if err != nil {
		return types.WizardAnswers{}, err
	}

	merged.ProjectRoot = flags.ProjectRoot
	if merged.ProjectName == "" {
		merged.ProjectName = flags.ProjectName
	}
	merged.ClaudeCode = flags.ClaudeCode
	merged.PermissionLevel = flags.PermissionLevel
	merged.Confirmed = flags.Confirmed
	merged.Hooks.SafetyBlock = flags.Hooks.SafetyBlock
	if len(flags.Skills) > 0 {
		merged.Skills = flags.Skills
	}
	if len(flags.MCPServers) > 0 {
		merged.MCPServers = flags.MCPServers
	}
	return merged, nil
}
