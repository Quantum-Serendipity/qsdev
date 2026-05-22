package posture

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/Quantum-Serendipity/qsdev/internal/answers"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// MergedState holds the unified view of all three addon state files.
type MergedState struct {
	// Files is the merged map of all tracked files across all state files.
	Files map[string]types.FileState

	// QsdevVersion is the most recent qsdev version found across state files.
	QsdevVersion string

	// EnabledTools is the merged map of tool enablement flags.
	EnabledTools map[string]bool

	// Sources tracks which state files were successfully loaded.
	Sources []string

	// Errors records any issues encountered during loading (e.g., corrupt YAML).
	// Loading continues past errors to provide the best available view.
	Errors []StateLoadError
}

// StateLoadError records a failure to load a specific state file.
type StateLoadError struct {
	Path string
	Err  error
}

func (e StateLoadError) Error() string {
	return fmt.Sprintf("loading state from %s: %s", e.Path, e.Err)
}

// HasAnyState returns true if at least one state file was successfully loaded
// and contained data (non-empty Files map or non-empty QsdevVersion).
func (m *MergedState) HasAnyState() bool {
	return len(m.Sources) > 0
}

// LoadAllStates reads and merges the three addon state files from the given
// project root directory. It uses state.LoadStateFromFile which returns a
// zero-value state with nil error when a file does not exist. Corrupt YAML
// files are recorded as errors but do not prevent other files from loading.
//
// Files maps are merged with later entries overriding earlier ones when keys
// collide. The QsdevVersion is taken from the most recently loaded state that
// has one set. EnabledTools are merged with later entries winning on conflict.
func LoadAllStates(projectRoot string) *MergedState {
	merged := &MergedState{
		Files:        make(map[string]types.FileState),
		EnabledTools: make(map[string]bool),
	}

	for _, relPath := range state.StateFilePaths() {
		absPath := filepath.Join(projectRoot, relPath)
		st, err := state.LoadStateFromFile(absPath)
		if err != nil {
			merged.Errors = append(merged.Errors, StateLoadError{
				Path: relPath,
				Err:  err,
			})
			continue
		}

		// A zero-value state with empty Files means the file didn't exist.
		// Only count it as a source if it had real content.
		if len(st.Files) == 0 && st.QsdevVersion == "" && len(st.EnabledTools) == 0 && st.LastRun.IsZero() {
			continue
		}

		merged.Sources = append(merged.Sources, relPath)

		for k, v := range st.Files {
			merged.Files[k] = v
		}

		if st.QsdevVersion != "" {
			merged.QsdevVersion = st.QsdevVersion
		}

		for k, v := range st.EnabledTools {
			merged.EnabledTools[k] = v
		}
	}

	// Fallback: if state files were loaded but none contributed EnabledTools,
	// try the primary answers file and run the standard inference logic.
	// This handles projects initialized before EnabledTools was persisted to state.
	if len(merged.EnabledTools) == 0 && len(merged.Sources) > 0 {
		a, err := answers.LoadPrimary(projectRoot)
		if err == nil {
			toolreg.MergeInferredTools(&a, toolreg.DefaultRegistry())
			for k, v := range a.EnabledTools {
				merged.EnabledTools[k] = v
			}
			slog.Debug("enabled tools loaded from answers fallback", "count", len(merged.EnabledTools))
		}
	}

	return merged
}
