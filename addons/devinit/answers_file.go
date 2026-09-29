package devinit

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// maxAnswersFileSize bounds how much of an answers file (or stdin) is read.
// Real answers files are a few KiB.
const maxAnswersFileSize = 1 << 20

// LoadAnswersFile reads WizardAnswers from a YAML file. Use "-" for stdin.
func LoadAnswersFile(path string) (types.WizardAnswers, error) {
	data, source, err := readAnswersFile(path)
	if err != nil {
		return types.WizardAnswers{}, err
	}
	return decodeAnswers(data, source)
}

// LoadAnswersFromReader reads and parses WizardAnswers from an io.Reader.
func LoadAnswersFromReader(r io.Reader, source string) (types.WizardAnswers, error) {
	data, err := readAnswersData(r, source)
	if err != nil {
		return types.WizardAnswers{}, err
	}
	return decodeAnswers(data, source)
}

// OverlayAnswersFile applies the answers file at path (or "-" for stdin) over
// base: every top-level key the file sets replaces base's value, and every
// key it omits keeps base's. Join mode uses it so an answers file adjusts the
// committed .qsdev.yaml configuration instead of discarding it.
func OverlayAnswersFile(base types.WizardAnswers, path string) (types.WizardAnswers, error) {
	data, source, err := readAnswersFile(path)
	if err != nil {
		return types.WizardAnswers{}, err
	}
	// Decode strictly first so typos are reported against the file itself.
	if _, err := decodeAnswers(data, source); err != nil {
		return types.WizardAnswers{}, err
	}
	var overlay map[string]any
	if err := yaml.Unmarshal(data, &overlay); err != nil {
		return types.WizardAnswers{}, fmt.Errorf("parsing answers from %s: %w", source, err)
	}

	baseData, err := yaml.Marshal(&base)
	if err != nil {
		return types.WizardAnswers{}, fmt.Errorf("marshaling base answers: %w", err)
	}
	var merged map[string]any
	if err := yaml.Unmarshal(baseData, &merged); err != nil {
		return types.WizardAnswers{}, fmt.Errorf("re-reading base answers: %w", err)
	}
	if merged == nil {
		merged = make(map[string]any, len(overlay))
	}
	maps.Copy(merged, overlay)

	mergedData, err := yaml.Marshal(merged)
	if err != nil {
		return types.WizardAnswers{}, fmt.Errorf("marshaling merged answers: %w", err)
	}
	return decodeAnswers(mergedData, source)
}

// readAnswersFile reads the raw answers document from path, or stdin for "-",
// and returns it with a human-readable source name for error messages.
func readAnswersFile(path string) ([]byte, string, error) {
	if path == "-" {
		data, err := readAnswersData(os.Stdin, "stdin")
		return data, "stdin", err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, path, fmt.Errorf("reading answers file %q: %w", path, err)
	}
	defer f.Close()
	data, err := readAnswersData(f, path)
	return data, path, err
}

// readAnswersData reads at most maxAnswersFileSize bytes and rejects empty or
// oversized input.
func readAnswersData(r io.Reader, source string) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxAnswersFileSize+1))
	if err != nil {
		return nil, fmt.Errorf("reading answers from %s: %w", source, err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("answers from %s is empty", source)
	}
	if len(data) > maxAnswersFileSize {
		return nil, fmt.Errorf("answers from %s exceed the %d-byte limit", source, maxAnswersFileSize)
	}
	return data, nil
}

// decodeAnswers parses an answers document strictly: an unknown or misspelled
// key is an error rather than silently dropped, so a typo in a security
// setting (e.g. "safty_block:") cannot quietly discard it. This matches the
// strict parsing of .qsdev.yaml.
func decodeAnswers(data []byte, source string) (types.WizardAnswers, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var answers types.WizardAnswers
	if err := dec.Decode(&answers); err != nil {
		if errors.Is(err, io.EOF) {
			return types.WizardAnswers{}, fmt.Errorf("answers from %s is empty", source)
		}
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
	missing := missingAnswerFields(answers)
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("answers file is missing required fields:\n  - %s",
		strings.Join(missing, "\n  - "))
}
