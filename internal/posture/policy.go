package posture

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

// PolicyFile represents a .qsdev-policy.yaml configuration.
type PolicyFile struct {
	Conformance PolicyConformance `yaml:"conformance"`
}

// PolicyConformance holds conformance policy settings.
type PolicyConformance struct {
	Custom *CustomPolicy `yaml:"custom,omitempty"`
}

// CustomPolicy defines a named set of custom conformance requirements.
type CustomPolicy struct {
	Name         string              `yaml:"name"`
	Requirements []PolicyRequirement `yaml:"requirements"`
}

// PolicyRequirement is a single named check expression.
type PolicyRequirement struct {
	Name  string `yaml:"name"`
	Check string `yaml:"check"`
}

// LoadPolicyFile loads a .qsdev-policy.yaml from the given path.
// Returns (nil, nil) if the file does not exist.
func LoadPolicyFile(path string) (*PolicyFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var pf PolicyFile
	if err := yaml.Unmarshal(data, &pf); err != nil {
		return nil, err
	}
	return &pf, nil
}
