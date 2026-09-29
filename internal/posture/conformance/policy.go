package conformance

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
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

// PolicyFileName returns the name of the custom conformance policy file at
// the project root (".qsdev-policy.yaml" under the default branding).
func PolicyFileName() string {
	return "." + branding.Get().AppName + "-policy.yaml"
}

// LoadFile loads a .qsdev-policy.yaml from the given path.
// Returns (nil, nil) if the file does not exist.
//
// Decoding is strict: an unknown key (for example a misspelt "requirement")
// is an error rather than a silently empty policy, and a custom section must
// declare at least one requirement, each with a unique name and a check.
func LoadFile(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading policy file: %w", err)
	}

	var pf File
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&pf); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("parsing policy file %s: %w", path, err)
	}
	if err := pf.validate(); err != nil {
		return nil, fmt.Errorf("invalid policy file %s: %w", path, err)
	}
	return &pf, nil
}

// validate checks the structural rules a policy must meet before any of its
// requirements are evaluated.
func (f *File) validate() error {
	custom := f.Conformance.Custom
	if custom == nil {
		return nil
	}
	if len(custom.Requirements) == 0 {
		return errors.New("conformance.custom declares no requirements")
	}
	seen := make(map[string]bool, len(custom.Requirements))
	for i, req := range custom.Requirements {
		name := strings.TrimSpace(req.Name)
		if name == "" {
			return fmt.Errorf("conformance.custom.requirements[%d] has no name", i)
		}
		if strings.TrimSpace(req.Check) == "" {
			return fmt.Errorf("requirement %q has no check expression", name)
		}
		if seen[name] {
			return fmt.Errorf("duplicate requirement name %q", name)
		}
		seen[name] = true
	}
	return nil
}
