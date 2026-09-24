package state

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
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

// LoadProjectStates loads every addon state file (see StateFilePaths) under
// projectRoot. Missing files are skipped. A file that cannot be read or parsed
// does not stop the others from loading; the joined errors are returned
// alongside the states that did load.
func LoadProjectStates(projectRoot string) ([]types.GeneratedState, error) {
	var states []types.GeneratedState
	var errs []error
	for _, rel := range StateFilePaths() {
		st, err := LoadStateFromFile(filepath.Join(projectRoot, rel))
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if len(st.Files) > 0 {
			states = append(states, st)
		}
	}
	return states, errors.Join(errs...)
}

// SaveStateToFile marshals state to YAML and writes it atomically to path.
// It creates parent directories as needed. Project state files must be
// written with SaveProjectState, which confines the write to the project.
func SaveStateToFile(path string, state types.GeneratedState) error {
	data, err := marshalState(state)
	if err != nil {
		return err
	}
	// WriteFileAtomic creates parent directories, fsyncs the temp file before
	// renaming, and retries transient Windows rename failures.
	if err := fileutil.WriteFileAtomic(path, data, fileutil.ModeReadWrite); err != nil {
		return fmt.Errorf("writing state file %s: %w", path, err)
	}
	return nil
}

// SaveProjectState marshals state to YAML and writes it atomically to rel, a
// path relative to projectRoot (such as InitStateFile()). A symlinked state
// directory or file that resolves outside projectRoot is refused with an
// error wrapping fileutil.ErrOutsideRoot, so a repository cannot redirect the
// state write to a file elsewhere on the machine.
func SaveProjectState(projectRoot, rel string, state types.GeneratedState) error {
	data, err := marshalState(state)
	if err != nil {
		return err
	}
	if err := fileutil.WriteFileAtomicInRoot(projectRoot, filepath.FromSlash(rel), data, fileutil.ModeReadWrite); err != nil {
		return fmt.Errorf("writing state file %s: %w", rel, err)
	}
	return nil
}

func marshalState(state types.GeneratedState) ([]byte, error) {
	data, err := yaml.Marshal(&state)
	if err != nil {
		return nil, fmt.Errorf("marshaling state: %w", err)
	}
	return data, nil
}
