package claudecode

import (
	"github.com/Quantum-Serendipity/qsdev/internal/answers"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

const answersFileName = ".qsdev-claude-answers.yaml"

// answersPath returns the full path to the answers persistence file.
func answersPath(projectRoot string) string {
	return answers.FilePath(projectRoot, ".claude", answersFileName)
}

// saveAnswers persists the wizard answers to .claude/.qsdev-claude-answers.yaml.
func saveAnswers(projectRoot string, a types.WizardAnswers) error {
	return answers.SaveToDir(projectRoot, ".claude", answersFileName, a)
}

// SaveAnswers is an exported wrapper around saveAnswers, allowing other
// packages (e.g. devinit) to persist answers to the Claude Code answers file.
func SaveAnswers(projectRoot string, a types.WizardAnswers) error {
	return saveAnswers(projectRoot, a)
}

// loadAnswers reads and unmarshals saved wizard answers from
// .claude/.qsdev-claude-answers.yaml. It returns an error if the file does not exist.
func loadAnswers(projectRoot string) (types.WizardAnswers, error) {
	return answers.LoadFromDir(projectRoot, ".claude", answersFileName, "claude")
}
