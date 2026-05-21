// Package answers provides shared YAML persistence for wizard answers
// across all qsdev addons. Each addon delegates to SaveToDir/LoadFromDir
// with its own directory and filename.
package answers

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// SaveToDir persists wizard answers to a YAML file atomically.
// dir is the subdirectory relative to projectRoot (e.g., ".devenv", ".claude").
// filename is the base name (e.g., ".qsdev-answers.yaml").
func SaveToDir(projectRoot, dir, filename string, answers types.WizardAnswers) error {
	data, err := yaml.Marshal(&answers)
	if err != nil {
		return fmt.Errorf("marshaling answers: %w", err)
	}

	path := filepath.Join(projectRoot, dir, filename)
	if err := fileutil.WriteFileAtomic(path, data, 0o644); err != nil {
		return fmt.Errorf("writing answers file: %w", err)
	}

	return nil
}

// LoadFromDir reads and unmarshals wizard answers from a YAML file.
// cmdName is used in error messages (e.g., "devenv", "claude", "init").
func LoadFromDir(projectRoot, dir, filename, cmdName string) (types.WizardAnswers, error) {
	path := filepath.Join(projectRoot, dir, filename)

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return types.WizardAnswers{}, fmt.Errorf("no saved answers found at %s: run 'qsdev %s init' first", path, cmdName)
		}
		return types.WizardAnswers{}, fmt.Errorf("reading answers file: %w", err)
	}

	var a types.WizardAnswers
	if err := yaml.Unmarshal(data, &a); err != nil {
		return types.WizardAnswers{}, fmt.Errorf("unmarshaling answers: %w", err)
	}

	return a, nil
}

// FilePath returns the full path to the answers file for a given
// project root, subdirectory, and filename.
func FilePath(projectRoot, dir, filename string) string {
	return filepath.Join(projectRoot, dir, filename)
}

// SavePrimary persists answers to the primary (devinit) answers file so that
// per-addon modifications stay in sync with the unified init state.
func SavePrimary(projectRoot string, a types.WizardAnswers) error {
	return SaveToDir(projectRoot, ".devinit", ".qsdev-init-answers.yaml", a)
}

// LoadPrimary reads the primary (devinit) answers file. Returns a zero-value
// WizardAnswers and nil error if the file does not exist or is corrupt.
func LoadPrimary(projectRoot string) (types.WizardAnswers, error) {
	path := filepath.Join(projectRoot, ".devinit", ".qsdev-init-answers.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return types.WizardAnswers{}, nil
		}
		return types.WizardAnswers{}, fmt.Errorf("reading primary answers: %w", err)
	}

	var a types.WizardAnswers
	if err := yaml.Unmarshal(data, &a); err != nil {
		return types.WizardAnswers{}, nil
	}
	return a, nil
}
