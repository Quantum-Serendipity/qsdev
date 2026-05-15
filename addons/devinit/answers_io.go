package devinit

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// loadAnswersOrEmpty reads saved wizard answers from
// .devinit/.qsdev-init-answers.yaml. Unlike loadAnswers (used by init/update),
// it returns a zero-value WizardAnswers instead of an error when the file does
// not exist — lifecycle commands need to work on projects that haven't run
// `qsdev init` yet.
func loadAnswersOrEmpty(projectRoot string) (types.WizardAnswers, error) {
	path := filepath.Join(projectRoot, answersDir, answersFileName)

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return types.WizardAnswers{}, nil
		}
		return types.WizardAnswers{}, err
	}

	var a types.WizardAnswers
	if err := yaml.Unmarshal(data, &a); err != nil {
		return types.WizardAnswers{}, err
	}
	return a, nil
}
