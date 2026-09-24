// Package answers provides shared YAML persistence for wizard answers
// across all qsdev addons. Each addon delegates to SaveToDir/LoadFromDir
// with its own directory and filename.
package answers

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
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
	if err := fileutil.WriteFileAtomic(path, data, fileutil.ModeReadWrite); err != nil {
		return fmt.Errorf("writing answers file: %w", err)
	}

	return nil
}

// LoadFromDir reads and unmarshals wizard answers from a YAML file.
// initCmd is the subcommand (without the app name) that creates the file,
// e.g. "init", "devenv init" or "claude init"; it is used in the error that
// tells the user how to create missing answers.
func LoadFromDir(projectRoot, dir, filename, initCmd string) (types.WizardAnswers, error) {
	path := filepath.Join(projectRoot, dir, filename)

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return types.WizardAnswers{}, notInitializedError(path, initCmd)
		}
		return types.WizardAnswers{}, fmt.Errorf("reading answers file: %w", err)
	}

	var a types.WizardAnswers
	if err := yaml.Unmarshal(data, &a); err != nil {
		return types.WizardAnswers{}, fmt.Errorf("unmarshaling answers: %w", err)
	}

	return a, nil
}

// notInitializedError reports a missing answers file together with the
// command that creates it.
func notInitializedError(path, initCmd string) error {
	return fmt.Errorf("no saved answers found at %s: run '%s %s' first", path, branding.Get().AppName, initCmd)
}

// LoadAddon loads the answers for an addon that keeps its own copy of the
// answers at dir/filename.
//
// The addon copy only marks the addon as initialized; its content is not
// authoritative. Every save path writes the primary answers file (addon saves
// through SavePrimary, and the top-level lifecycle commands such as enable and
// disable), but the lifecycle commands never refresh the addon copies. Loading
// the addon copy and saving it back through SavePrimary would therefore
// overwrite newer primary state (EnabledTools, EnvVars, MCP servers, ...)
// with a stale snapshot. LoadAddon returns the primary answers whenever the
// primary file exists and falls back to the addon copy only for projects that
// have no primary file.
func LoadAddon(projectRoot, dir, filename, initCmd string) (types.WizardAnswers, error) {
	primary, found, err := loadPrimary(projectRoot)
	if err != nil {
		return types.WizardAnswers{}, err
	}
	if !found {
		return LoadFromDir(projectRoot, dir, filename, initCmd)
	}

	path := filepath.Join(projectRoot, dir, filename)
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return types.WizardAnswers{}, notInitializedError(path, initCmd)
		}
		return types.WizardAnswers{}, fmt.Errorf("checking answers file: %w", err)
	}
	return primary, nil
}

// FilePath returns the full path to the answers file for a given
// project root, subdirectory, and filename.
func FilePath(projectRoot, dir, filename string) string {
	return filepath.Join(projectRoot, dir, filename)
}

// PrimaryDir returns the directory, relative to the project root, that holds
// the primary (devinit) answers file.
func PrimaryDir() string {
	return branding.Get().StateDir
}

// PrimaryFilename returns the base name of the primary answers file.
func PrimaryFilename() string {
	return "." + branding.Get().AppName + "-init-answers.yaml"
}

// DevenvCopyFile returns the project-relative path of the devenv addon's
// mirror of the answers.
func DevenvCopyFile() string {
	return path.Join(".devenv", "."+branding.Get().AppName+"-answers.yaml")
}

// LegacyClaudeCopyFile returns the project-relative path of the per-addon
// answers copy older releases kept in .claude/.
func LegacyClaudeCopyFile() string {
	return path.Join(".claude", "."+branding.Get().AppName+"-claude-answers.yaml")
}

// PrimaryPath returns the full path to the primary answers file.
func PrimaryPath(projectRoot string) string {
	return filepath.Join(projectRoot, PrimaryDir(), PrimaryFilename())
}

// SavePrimary persists answers to the primary (devinit) answers file so that
// per-addon modifications stay in sync with the unified init state. It
// replaces the whole file, so a must have been loaded from the primary file
// (LoadPrimary or LoadAddon) rather than from a possibly stale addon copy.
func SavePrimary(projectRoot string, a types.WizardAnswers) error {
	return SaveToDir(projectRoot, PrimaryDir(), PrimaryFilename(), a)
}

// LoadPrimary reads the primary (devinit) answers file. Returns a zero-value
// WizardAnswers and nil error only when the file does not exist. A corrupt or
// unparseable file returns an error so the corruption is surfaced to callers
// rather than being silently treated as empty state (config data loss).
func LoadPrimary(projectRoot string) (types.WizardAnswers, error) {
	a, _, err := loadPrimary(projectRoot)
	return a, err
}

// RequirePrimary reads the primary (devinit) answers file. Unlike LoadPrimary,
// a missing file is an error telling the user to run init, for callers whose
// result would be meaningless (vacuously empty) without recorded answers.
func RequirePrimary(projectRoot string) (types.WizardAnswers, error) {
	return LoadFromDir(projectRoot, PrimaryDir(), PrimaryFilename(), "init")
}

// loadPrimary reads the primary answers file and reports whether it exists.
func loadPrimary(projectRoot string) (types.WizardAnswers, bool, error) {
	path := PrimaryPath(projectRoot)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return types.WizardAnswers{}, false, nil
		}
		return types.WizardAnswers{}, false, fmt.Errorf("reading primary answers: %w", err)
	}

	var a types.WizardAnswers
	if err := yaml.Unmarshal(data, &a); err != nil {
		return types.WizardAnswers{}, false, fmt.Errorf("parsing primary answers file %s: %w", path, err)
	}
	return a, true, nil
}
