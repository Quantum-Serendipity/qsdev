package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// LocalConfig represents the .qsdev.local.yaml file, which contains
// per-developer overrides. It omits project-level fields (Version,
// QsdevVersion, Profile, InfraProfile, Client, Infrastructure) that only
// belong in the shared .qsdev.yaml.
type LocalConfig struct {
	Languages     []types.LanguageConfig `yaml:"languages,omitempty"`
	Services      []types.ServiceConfig  `yaml:"services,omitempty"`
	Security      types.SecurityConfig   `yaml:"security,omitempty"`
	Tools         types.ToolsConfig      `yaml:"tools,omitempty"`
	ClaudeCode    types.ClaudeCodeConfig `yaml:"claude_code,omitempty"`
	ExtraPackages []string               `yaml:"extra_packages,omitempty"`
}

// ParseLocalConfig reads and parses a .qsdev.local.yaml file.
// Returns (nil, nil) if the file does not exist — this is not an error
// since the local config file is optional. Only parse failures produce errors.
func ParseLocalConfig(path string) (*LocalConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading local config %s: %w", path, err)
	}

	// Strict (known-field) decode: the local file is the highest-precedence
	// layer, so a misspelled key must surface as an error rather than silently
	// dropping the developer's intended override (e.g. a local tool deny).
	var local LocalConfig
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&local); err != nil && !errors.Is(err, io.EOF) {
		// io.EOF means the file is empty or comments-only (the generated
		// template), which is a valid empty override.
		return nil, fmt.Errorf("parsing local config %s: %w", path, err)
	}

	return &local, nil
}
