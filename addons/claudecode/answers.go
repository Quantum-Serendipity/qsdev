package claudecode

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"fastcat.org/go/gdev-secure-devenv-bootstrap/pkg/types"
)

const answersFileName = ".gdev-claude-answers.yaml"

// answersPath returns the full path to the answers persistence file.
func answersPath(projectRoot string) string {
	return filepath.Join(projectRoot, ".claude", answersFileName)
}

// saveAnswers persists the wizard answers to .claude/.gdev-claude-answers.yaml.
// It creates the .claude/ directory if needed and writes atomically via
// a temp file + rename.
func saveAnswers(projectRoot string, answers types.WizardAnswers) error {
	data, err := yaml.Marshal(&answers)
	if err != nil {
		return fmt.Errorf("marshaling answers: %w", err)
	}

	dir := filepath.Join(projectRoot, ".claude")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating .claude directory: %w", err)
	}

	path := answersPath(projectRoot)
	tmpPath := path + ".tmp"

	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("writing temp answers file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("renaming temp answers file: %w", err)
	}

	return nil
}

// SaveAnswers is an exported wrapper around saveAnswers, allowing other
// packages (e.g. devinit) to persist answers to the Claude Code answers file.
func SaveAnswers(projectRoot string, answers types.WizardAnswers) error {
	return saveAnswers(projectRoot, answers)
}

// loadAnswers reads and unmarshals saved wizard answers from
// .claude/.gdev-claude-answers.yaml. It returns an error if the file does not exist.
func loadAnswers(projectRoot string) (types.WizardAnswers, error) {
	path := answersPath(projectRoot)

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return types.WizardAnswers{}, fmt.Errorf("no saved answers found at %s: run 'gdev claude init' first", path)
		}
		return types.WizardAnswers{}, fmt.Errorf("reading answers file: %w", err)
	}

	var answers types.WizardAnswers
	if err := yaml.Unmarshal(data, &answers); err != nil {
		return types.WizardAnswers{}, fmt.Errorf("unmarshaling answers: %w", err)
	}

	return answers, nil
}
