package state

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

// FileStatus describes the current on-disk state of a previously generated
// file compared to the stored hash in GeneratedState.
type FileStatus struct {
	Path        string
	Status      types.ModificationStatus
	Error       error
	StoredHash  string
	CurrentHash string
}

// RecordFiles creates a GeneratedState from a slice of GeneratedFile,
// computing content hashes for each file and recording its merge strategy
// and file mode.
func RecordFiles(files []types.GeneratedFile) types.GeneratedState {
	state := types.GeneratedState{
		LastRun: time.Now().UTC(),
		Files:   make(map[string]types.FileState, len(files)),
	}
	for _, f := range files {
		fs := types.FileState{
			Hash:     ComputeHash(f.Content),
			Strategy: f.Strategy,
			Mode:     f.Mode,
			Owner:    f.Owner,
		}
		if f.Strategy == types.ThreeWayMerge {
			fs.BaseContent = f.Content
		}
		state.Files[f.Path] = fs
	}
	return state
}

// CheckModified compares each file in stored against its current on-disk
// state under projectRoot and returns a map of path to FileStatus.
func CheckModified(stored types.GeneratedState, projectRoot string) map[string]FileStatus {
	if len(stored.Files) == 0 {
		return map[string]FileStatus{}
	}

	results := make(map[string]FileStatus, len(stored.Files))
	for relPath, fs := range stored.Files {
		absPath := filepath.Join(projectRoot, relPath)
		status := FileStatus{
			Path:       relPath,
			StoredHash: fs.Hash,
		}

		info, err := os.Stat(absPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				status.Status = types.Deleted
			} else {
				status.Status = types.Unknown
				status.Error = err
			}
			results[relPath] = status
			continue
		}

		hash, err := ComputeFileHash(absPath)
		if err != nil {
			status.Status = types.Unknown
			status.Error = err
			results[relPath] = status
			continue
		}
		status.CurrentHash = hash

		hashMatch := hash == fs.Hash
		modeMatch := runtime.GOOS == "windows" || info.Mode().Perm() == fs.Mode.Perm()

		switch {
		case hashMatch && modeMatch:
			status.Status = types.Unmodified
		default:
			status.Status = types.Modified
		}

		results[relPath] = status
	}

	return results
}
