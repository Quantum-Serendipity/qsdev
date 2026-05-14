package devinit

import (
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

// LoadAnswersFile reads WizardAnswers from a YAML file. Use "-" for stdin.
func LoadAnswersFile(path string) (types.WizardAnswers, error) {
	if path == "-" {
		return LoadAnswersFromReader(os.Stdin, "stdin")
	}
	f, err := os.Open(path)
	if err != nil {
		return types.WizardAnswers{}, fmt.Errorf("reading answers file %q: %w", path, err)
	}
	defer f.Close()
	return LoadAnswersFromReader(f, path)
}

// LoadAnswersFromReader reads and parses WizardAnswers from an io.Reader.
func LoadAnswersFromReader(r io.Reader, source string) (types.WizardAnswers, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return types.WizardAnswers{}, fmt.Errorf("reading answers from %s: %w", source, err)
	}
	if len(data) == 0 {
		return types.WizardAnswers{}, fmt.Errorf("answers from %s is empty", source)
	}
	var answers types.WizardAnswers
	if err := yaml.Unmarshal(data, &answers); err != nil {
		return types.WizardAnswers{}, fmt.Errorf("parsing answers from %s: %w", source, err)
	}
	return answers, nil
}

// MergeFileWithFlags merges file-loaded answers (base) with flag-derived
// overrides. Only explicitly-set flags override file values.
func MergeFileWithFlags(base types.WizardAnswers, overrides types.WizardAnswers, changed map[string]bool) types.WizardAnswers {
	return MergeProfileWithFlags(base, overrides, changed)
}

// ValidateAnswersFileCompleteness checks that an answers file contains
// the minimum required fields for non-interactive execution.
func ValidateAnswersFileCompleteness(answers types.WizardAnswers) error {
	var missing []string
	if len(answers.Languages) == 0 {
		missing = append(missing, "languages (at least one language is required)")
	}
	if answers.ClaudeCode && answers.PermissionLevel == "" {
		missing = append(missing, "permission_level (required when claude_code is true)")
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("answers file is missing required fields:\n  - %s",
		strings.Join(missing, "\n  - "))
}
