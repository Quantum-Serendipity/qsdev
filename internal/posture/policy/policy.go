package policy

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

// File represents a .qsdev-policy.yaml configuration.
type File struct {
	Conformance Conformance `yaml:"conformance"`
}

// Conformance holds conformance policy settings.
type Conformance struct {
	Custom *Custom `yaml:"custom,omitempty"`
}

// Custom defines a named set of custom conformance requirements.
type Custom struct {
	Name         string        `yaml:"name"`
	Requirements []Requirement `yaml:"requirements"`
}

// Requirement is a single named check expression.
type Requirement struct {
	Name  string `yaml:"name"`
	Check string `yaml:"check"`
}

// LoadFile loads a .qsdev-policy.yaml from the given path.
// Returns (nil, nil) if the file does not exist.
func LoadFile(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var pf File
	if err := yaml.Unmarshal(data, &pf); err != nil {
		return nil, err
	}
	return &pf, nil
}
