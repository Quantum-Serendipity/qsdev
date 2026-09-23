package policy

import (
	"encoding/json"
	"fmt"
	"os"

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

// SaveSessionOverrides writes the session bypass overrides atomically, so a
// hook reading the state concurrently sees either the previous or the new
// overrides and never a truncated file.
func SaveSessionOverrides(path string, overrides []string) error {
	state := sessionState{
		SessionBypassOverrides: overrides,
	}

	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshaling session state: %w", err)
	}

	if err := fileutil.WriteFileAtomic(path, data, fileutil.ModeReadWrite); err != nil {
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
