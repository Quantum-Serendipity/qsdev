package devenv

import (
	"fastcat.org/go/gdev-secure-devenv-bootstrap/internal/answers"
	"fastcat.org/go/gdev-secure-devenv-bootstrap/pkg/types"
)

const answersFileName = ".gdev-answers.yaml"

// answersPath returns the full path to the answers persistence file.
func answersPath(projectRoot string) string {
	return answers.FilePath(projectRoot, ".devenv", answersFileName)
}

// saveAnswers persists the wizard answers to .devenv/.gdev-answers.yaml.
func saveAnswers(projectRoot string, a types.WizardAnswers) error {
	return answers.SaveToDir(projectRoot, ".devenv", answersFileName, a)
}

// SaveAnswers is an exported wrapper around saveAnswers, allowing other
// packages (e.g. devinit) to persist answers to the devenv answers file.
func SaveAnswers(projectRoot string, a types.WizardAnswers) error {
	return saveAnswers(projectRoot, a)
}

// loadAnswers reads and unmarshals saved wizard answers from
// .devenv/.gdev-answers.yaml. It returns an error if the file does not exist.
func loadAnswers(projectRoot string) (types.WizardAnswers, error) {
	return answers.LoadFromDir(projectRoot, ".devenv", answersFileName, "devenv")
}
