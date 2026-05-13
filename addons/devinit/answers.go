package devinit

import (
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/answers"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

// answersPath returns the full path to the devinit answers persistence file.
func answersPath(projectRoot string) string {
	return answers.FilePath(projectRoot, answersDir, answersFileName)
}

// saveAnswers persists the wizard answers to .devinit/.gdev-init-answers.yaml.
func saveAnswers(projectRoot string, a types.WizardAnswers) error {
	return answers.SaveToDir(projectRoot, answersDir, answersFileName, a)
}

// loadAnswers reads and unmarshals saved wizard answers from
// .devinit/.gdev-init-answers.yaml. It returns an error if the file does not exist.
func loadAnswers(projectRoot string) (types.WizardAnswers, error) {
	return answers.LoadFromDir(projectRoot, answersDir, answersFileName, "init")
}
