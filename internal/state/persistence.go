package state

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"fastcat.org/go/gdev-secure-devenv-bootstrap/pkg/types"
	"gopkg.in/yaml.v3"
)

// LoadStateFromFile reads and unmarshals a GeneratedState from the YAML file
// at path. If the file does not exist, it returns a zero-value state with an
// initialized Files map and no error.
func LoadStateFromFile(path string) (types.GeneratedState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return types.GeneratedState{
				Files: make(map[string]types.FileState),
			}, nil
		}
		return types.GeneratedState{}, fmt.Errorf("reading state file %s: %w", path, err)
	}

	var state types.GeneratedState
	if err := yaml.Unmarshal(data, &state); err != nil {
		return types.GeneratedState{}, fmt.Errorf("unmarshaling state file %s: %w", path, err)
	}

	// Ensure Files map is initialized even if the YAML had no files key.
	if state.Files == nil {
		state.Files = make(map[string]types.FileState)
	}

	return state, nil
}

// SaveStateToFile marshals state to YAML and writes it atomically to path.
// It creates parent directories as needed.
func SaveStateToFile(path string, state types.GeneratedState) error {
	data, err := yaml.Marshal(&state)
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating state directory %s: %w", dir, err)
	}

	// Atomic write: write to a temp file in the same directory, then rename.
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("writing temp state file %s: %w", tmpPath, err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		// Clean up temp file on rename failure.
		_ = os.Remove(tmpPath)
		return fmt.Errorf("renaming temp state file to %s: %w", path, err)
	}

	return nil
}
