package state

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"time"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
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

// OrphanedFiles returns paths that exist in oldState but are not present in
// the newFiles set. These are files that were previously generated but are
// no longer produced after a configuration change (e.g., removing a language).
func OrphanedFiles(oldState types.GeneratedState, newFiles []types.GeneratedFile) []string {
	newSet := make(map[string]bool, len(newFiles))
	for _, f := range newFiles {
		newSet[f.Path] = true
	}
	var orphans []string
	for path := range oldState.Files {
		if !newSet[path] {
			orphans = append(orphans, path)
		}
	}
	sort.Strings(orphans)
	return orphans
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

// RecordFragments converts a fragment set into ledger entries grouped by target path.
func RecordFragments(fragments []types.FragmentEntry) map[string][]types.FragmentLedgerEntry {
	ledger := make(map[string][]types.FragmentLedgerEntry)
	for _, f := range fragments {
		entry := types.FragmentLedgerEntry{
			Source:      f.Source,
			Tag:         f.Tag,
			Priority:    f.Priority,
			ComposeMode: f.ComposeMode,
			ContentHash: ComputeHash(f.Content),
			Timestamp:   f.Provenance.Timestamp,
			Reason:      f.Provenance.Reason,
		}
		ledger[f.Target] = append(ledger[f.Target], entry)
	}
	for target := range ledger {
		sort.Slice(ledger[target], func(i, j int) bool {
			a, b := ledger[target][i], ledger[target][j]
			if a.Source != b.Source {
				return a.Source < b.Source
			}
			return a.Tag < b.Tag
		})
	}
	return ledger
}

// FragmentsBySource returns all ledger entries contributed by the given source
// across all target files.
func FragmentsBySource(state types.GeneratedState, source string) []types.FragmentLedgerEntry {
	var result []types.FragmentLedgerEntry
	for _, entries := range state.Fragments {
		for _, e := range entries {
			if e.Source == source {
				result = append(result, e)
			}
		}
	}
	return result
}

// FragmentsByTarget returns all ledger entries contributing to the given file path.
func FragmentsByTarget(state types.GeneratedState, target string) []types.FragmentLedgerEntry {
	return state.Fragments[target]
}

// RemoveFragmentsBySource removes all ledger entries from the given source
// and returns the target file paths that were affected.
func RemoveFragmentsBySource(state *types.GeneratedState, source string) []string {
	if state.Fragments == nil {
		return nil
	}
	var affected []string
	for target, entries := range state.Fragments {
		var kept []types.FragmentLedgerEntry
		found := false
		for _, e := range entries {
			if e.Source == source {
				found = true
			} else {
				kept = append(kept, e)
			}
		}
		if found {
			affected = append(affected, target)
			if len(kept) == 0 {
				delete(state.Fragments, target)
			} else {
				state.Fragments[target] = kept
			}
		}
	}
	sort.Strings(affected)
	return affected
}
