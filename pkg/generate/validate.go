package generate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
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
// Dotfiles such as .envrc need no special case: filepath.Ext(".envrc") is
// ".envrc".
func (r *ValidatorRegistry) Validate(path string, content []byte) ValidationResult {
	v, ok := r.validators[filepath.Ext(path)]
	if !ok {
		return ValidationResult{Path: path, Valid: true, Skipped: true}
	}

	result := v.Validate(content)
	result.Path = path
	return result
}

// ErrInvalidContent marks generated content rejected by its syntax validator.
var ErrInvalidContent = errors.New("invalid generated content")

// defaultValidators is the registry shared by ValidateContent; validators
// cache their tool lookups, so it is built once.
var defaultValidators = sync.OnceValue(NewValidatorRegistry)

// ValidateContent checks content destined for path with the syntax validator
// for its file type (Nix, JSON, YAML, shell). It returns an error when the
// content is invalid; a file type without a validator, or whose validator tool
// is unavailable, passes.
func ValidateContent(path string, content []byte) error {
	vr := defaultValidators().Validate(path, content)
	if !vr.Valid && !vr.Skipped {
		return fmt.Errorf("%w: validation failed for %s: %w", ErrInvalidContent, path, vr.Error)
	}
	return nil
}

// WriteGeneratedFile validates f's content as WriteFiles does (unless
// f.SkipValidation) and writes it atomically to projectRoot/f.Path with
// f.Mode (default fileutil.ModeReadWrite). It is the write path for generated
// content outside WriteFiles (update, enable/disable, repair, auto-fix, the
// claude subcommands), so content WriteFiles rejects is never written by
// another command. Like WriteFiles it never writes outside projectRoot: a
// path, symlinked parent directory or symlinked file that resolves outside
// the root is refused with an error wrapping fileutil.ErrOutsideRoot.
func WriteGeneratedFile(projectRoot string, f types.GeneratedFile) error {
	if !f.SkipValidation {
		if err := ValidateContent(f.Path, f.Content); err != nil {
			return err
		}
	}
	mode := f.Mode
	if mode == 0 {
		mode = fileutil.ModeReadWrite
	}
	if err := fileutil.WriteFileAtomicInRoot(projectRoot, filepath.FromSlash(f.Path), f.Content, mode); err != nil {
		return fmt.Errorf("writing %s: %w", f.Path, err)
	}
	return nil
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

	cmd := exec.CommandContext(ctx, v.nixPath, "--parse", "-")
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

	cmd := exec.CommandContext(ctx, v.bashPath, "-n")
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
