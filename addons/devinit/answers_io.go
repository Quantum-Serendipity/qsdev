package devinit

import (
	"github.com/Quantum-Serendipity/qsdev/internal/answers"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// loadAnswersOrEmpty reads the saved primary wizard answers. Unlike
// loadAnswers (used by init/update), it returns a zero-value WizardAnswers
// instead of an error when the file does not exist — lifecycle commands need
// to work on projects that haven't run `qsdev init` yet. A present but
// unreadable or corrupt file is still an error, wrapped with its path.
//
// It delegates to answers.LoadPrimary so the answers path and parsing have a
// single implementation.
func loadAnswersOrEmpty(projectRoot string) (types.WizardAnswers, error) {
	return answers.LoadPrimary(projectRoot)
}
