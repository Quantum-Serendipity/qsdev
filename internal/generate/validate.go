package generate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Validator checks the syntactic validity of file content.
type Validator interface {
	Validate(content []byte) ValidationResult
}

// ValidatorRegistry dispatches validation to the appropriate Validator
// based on file extension.
type ValidatorRegistry struct {
	validators map[string]Validator
}

// NewValidatorRegistry returns a registry populated with built-in validators.
func NewValidatorRegistry() *ValidatorRegistry {
	return &ValidatorRegistry{
		validators: map[string]Validator{
			".nix":   &NixValidator{},
			".yaml":  &YAMLValidator{},
			".yml":   &YAMLValidator{},
			".json":  &JSONValidator{},
			".sh":    &ShellValidator{},
			".envrc": &ShellValidator{},
		},
	}
}

// Validate checks the content of the file at the given path using the
// appropriate validator for the file extension. Unknown extensions are skipped.
func (r *ValidatorRegistry) Validate(path string, content []byte) ValidationResult {
	ext := filepath.Ext(path)
	// .envrc has no normal extension; match on base name.
	if ext == "" && filepath.Base(path) == ".envrc" {
		ext = ".envrc"
	}

	v, ok := r.validators[ext]
	if !ok {
		return ValidationResult{Path: path, Valid: true, Skipped: true}
	}

	result := v.Validate(content)
	result.Path = path
	return result
}

// NewNixValidator returns a new NixValidator.
func NewNixValidator() *NixValidator {
	return &NixValidator{}
}

// NixValidator checks Nix expression syntax via nix-instantiate --parse.
type NixValidator struct {
	once     sync.Once
	nixPath  string
	nixFound bool
}

func (v *NixValidator) lookupNix() {
	v.once.Do(func() {
		p, err := exec.LookPath("nix-instantiate")
		if err == nil {
			v.nixPath = p
			v.nixFound = true
		}
	})
}

func (v *NixValidator) Validate(content []byte) ValidationResult {
	v.lookupNix()
	if !v.nixFound {
		return ValidationResult{
			Valid:   true,
			Skipped: true,
			Warning: "nix-instantiate not found on PATH; skipping Nix validation",
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, v.nixPath, "--parse", "/dev/stdin")
	cmd.Stdin = bytes.NewReader(content)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return ValidationResult{
			Valid: false,
			Error: fmt.Errorf("nix parse error: %s: %w", strings.TrimSpace(string(output)), err),
		}
	}
	return ValidationResult{Valid: true}
}

// YAMLValidator checks YAML syntax by attempting to unmarshal the content.
type YAMLValidator struct{}

func (v *YAMLValidator) Validate(content []byte) ValidationResult {
	var out any
	if err := yaml.Unmarshal(content, &out); err != nil {
		return ValidationResult{
			Valid: false,
			Error: fmt.Errorf("YAML parse error: %w", err),
		}
	}
	return ValidationResult{Valid: true}
}

// JSONValidator checks JSON syntax by attempting to unmarshal the content.
type JSONValidator struct{}

func (v *JSONValidator) Validate(content []byte) ValidationResult {
	var out any
	if err := json.Unmarshal(content, &out); err != nil {
		return ValidationResult{
			Valid: false,
			Error: fmt.Errorf("JSON parse error: %w", err),
		}
	}
	return ValidationResult{Valid: true}
}

// NewShellValidator returns a new ShellValidator.
func NewShellValidator() *ShellValidator {
	return &ShellValidator{}
}

// ShellValidator checks shell script syntax via bash -n.
type ShellValidator struct {
	once      sync.Once
	bashPath  string
	bashFound bool
}

func (v *ShellValidator) lookupBash() {
	v.once.Do(func() {
		p, err := exec.LookPath("bash")
		if err == nil {
			v.bashPath = p
			v.bashFound = true
		}
	})
}

func (v *ShellValidator) Validate(content []byte) ValidationResult {
	v.lookupBash()
	if !v.bashFound {
		return ValidationResult{
			Valid:   true,
			Skipped: true,
			Warning: "bash not found on PATH; skipping shell validation",
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, v.bashPath, "-n", "/dev/stdin")
	cmd.Stdin = bytes.NewReader(content)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return ValidationResult{
			Valid: false,
			Error: fmt.Errorf("shell syntax error: %s: %w", strings.TrimSpace(string(output)), err),
		}
	}
	return ValidationResult{Valid: true}
}
