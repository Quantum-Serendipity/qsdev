package devenv

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/Quantum-Serendipity/qsdev/internal/answers"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// answersFile returns the answers file name, using the branding app name.
func answersFile() string {
	return path.Base(answers.DevenvCopyFile())
}

// answersPath returns the full path to the legacy per-addon answers file.
func answersPath(projectRoot string) string {
	return filepath.Join(projectRoot, filepath.FromSlash(answers.DevenvCopyFile()))
}

// saveAnswers persists the wizard answers to the primary answers file and
// mirrors them to the legacy .devenv/.qsdev-answers.yaml copy. The primary
// file is written first and its failure is an error, because loadAnswers
// reads it in preference to the mirror.
func saveAnswers(projectRoot string, a types.WizardAnswers) error {
	if err := answers.SavePrimary(projectRoot, a); err != nil {
		return fmt.Errorf("saving primary answers: %w", err)
	}
	if err := answers.SaveToDir(projectRoot, AddonDir, answersFile(), a); err != nil {
		return fmt.Errorf("saving devenv answers: %w", err)
	}
	return nil
}

// SaveAnswers is an exported wrapper around saveAnswers, allowing other
// packages (e.g. devinit) to persist answers to the devenv answers file.
func SaveAnswers(projectRoot string, a types.WizardAnswers) error {
	return saveAnswers(projectRoot, a)
}

// loadAnswers reads the saved wizard answers. The primary answers file is
// authoritative because qsdev enable/disable and init --update write only that
// file; reading the per-addon mirror instead would regenerate from stale
// answers and then overwrite the primary copy, silently undoing those changes.
// The legacy .devenv copy is used only when no primary file exists yet (a
// project initialized by an older release). It returns an error if neither
// file exists.
func loadAnswers(projectRoot string) (types.WizardAnswers, error) {
	_, err := os.Stat(answers.PrimaryPath(projectRoot))
	switch {
	case err == nil:
		return answers.LoadPrimary(projectRoot)
	case !errors.Is(err, os.ErrNotExist):
		return types.WizardAnswers{}, fmt.Errorf("checking primary answers: %w", err)
	}
	return answers.LoadFromDir(projectRoot, AddonDir, answersFile(), "devenv init")
}
