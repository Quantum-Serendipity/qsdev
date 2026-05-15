package state

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
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

	slog.Debug("state loaded", "path", path, "files", len(state.Files))
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

	// Atomic write: create a temp file with a random suffix in the same
	// directory (guarantees same filesystem for rename), write, then rename.
	tmp, err := os.CreateTemp(dir, ".qsdev-state-*")
	if err != nil {
		return fmt.Errorf("creating temp state file in %s: %w", dir, err)
	}
	tmpPath := tmp.Name()
	defer func() {
		// Clean up temp file on any failure path.
		_ = os.Remove(tmpPath)
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("writing temp state file %s: %w", tmpPath, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp state file %s: %w", tmpPath, err)
	}
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		return fmt.Errorf("setting permissions on temp state file %s: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("renaming temp state file to %s: %w", path, err)
	}

	// Rename succeeded — prevent deferred Remove from deleting the final file.
	tmpPath = ""
	return nil
}
