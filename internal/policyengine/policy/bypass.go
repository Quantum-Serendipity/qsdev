package policy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

type FileSessionStateReader struct {
	path string
}

type sessionState struct {
	SessionBypassOverrides []string `json:"sessionBypassOverrides"`
}

func NewFileSessionStateReader(path string) *FileSessionStateReader {
	return &FileSessionStateReader{path: path}
}

func (r *FileSessionStateReader) SessionOverrides() []string {
	data, err := os.ReadFile(r.path)
	if err != nil {
		return nil
	}

	var state sessionState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil
	}

	return state.SessionBypassOverrides
}

type StaticSessionStateReader struct {
	Overrides []string
}

func (r *StaticSessionStateReader) SessionOverrides() []string {
	return r.Overrides
}

func SaveSessionOverrides(path string, overrides []string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, fileutil.ModeDirDefault); err != nil {
		return fmt.Errorf("creating session state directory: %w", err)
	}

	state := sessionState{
		SessionBypassOverrides: overrides,
	}

	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshaling session state: %w", err)
	}

	if err := os.WriteFile(path, data, fileutil.ModeReadWrite); err != nil {
		return fmt.Errorf("writing session state: %w", err)
	}

	return nil
}

func ClearSessionOverrides(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("clearing session state: %w", err)
	}
	return nil
}
