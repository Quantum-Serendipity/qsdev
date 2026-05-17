package claudecode

import (
	"github.com/Quantum-Serendipity/qsdev/internal/answers"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// answersFile returns the answers file name, using the branding app name.
func answersFile() string {
	return "." + branding.Get().AppName + "-claude-answers.yaml"
}

// answersPath returns the full path to the answers persistence file.
func answersPath(projectRoot string) string {
	return answers.FilePath(projectRoot, ".claude", answersFile())
}

// saveAnswers persists the wizard answers to .claude/.qsdev-claude-answers.yaml
// and syncs to the primary (devinit) answers file for cross-addon consistency.
func saveAnswers(projectRoot string, a types.WizardAnswers) error {
	if err := answers.SaveToDir(projectRoot, ".claude", answersFile(), a); err != nil {
		return err
	}
	_ = answers.SavePrimary(projectRoot, a)
	return nil
}

// SaveAnswers is an exported wrapper around saveAnswers, allowing other
// packages (e.g. devinit) to persist answers to the Claude Code answers file.
func SaveAnswers(projectRoot string, a types.WizardAnswers) error {
	return saveAnswers(projectRoot, a)
}

// loadAnswers reads and unmarshals saved wizard answers from
// .claude/.qsdev-claude-answers.yaml. It returns an error if the file does not exist.
func loadAnswers(projectRoot string) (types.WizardAnswers, error) {
	return answers.LoadFromDir(projectRoot, ".claude", answersFile(), "claude")
}
