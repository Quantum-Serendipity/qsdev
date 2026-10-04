package state

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/shebang"
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
			// The merge base is what the generator produced ("ours"), which
			// differs from Content when a merge preserved user additions.
			fs.BaseContent = f.Content
			if f.BaseContent != nil {
				fs.BaseContent = f.BaseContent
			}
		}
		state.Files[f.Path] = fs
	}
	return state
}

// IsRecordedOutput reports whether content is what one of states recorded
// for relPath, exactly or as a CRLF checkout of it (see MatchesHash), i.e. a
// file on disk holding it is unmodified qsdev output that may be regenerated
// in place rather than a file the user owns.
func IsRecordedOutput(states []types.GeneratedState, relPath string, content []byte) bool {
	for _, st := range states {
		if recorded, ok := st.Files[relPath]; ok && MatchesHash(content, recorded.Hash) {
			return true
		}
	}
	return false
}

// MatchesHash reports whether data is the content hash names, exactly or as
// a checkout of it whose lines end in CRLF (Git's core.autocrlf, the Git for
// Windows default, converts LF to CRLF in the working tree while the
// repository keeps LF). Recorded hashes are over the generated LF content.
func MatchesHash(data []byte, hash string) bool {
	if ComputeHash(data) == hash {
		return true
	}
	return bytes.Contains(data, []byte("\r\n")) && ComputeHash(crlfToLF(data)) == hash
}

// HoldsOutput reports whether existing, a file's content on disk, holds the
// generated content: exactly, or as a CRLF checkout of it (EqualText) that
// still works on goos. A script whose interpreter line then ends in CR cannot
// be started by the Linux or macOS kernel, so that spelling does not hold it
// there and the script must be rewritten.
func HoldsOutput(existing, content []byte, goos string) bool {
	if bytes.Equal(existing, content) {
		return true
	}
	return EqualText(existing, content) && !shebang.Parse(existing).CRLFFails(goos)
}

// EqualText reports whether a and b are the same text once CRLF line endings
// are read as LF: one is the other as a CRLF checkout. A lone CR is content.
func EqualText(a, b []byte) bool {
	return bytes.Equal(a, b) || bytes.Equal(crlfToLF(a), crlfToLF(b))
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
	results := make(map[string]FileStatus, len(stored.Files))
	for relPath, fs := range stored.Files {
		results[relPath] = CheckFile(projectRoot, relPath, fs)
	}
	return results
}

// CheckFile compares the generated file at relPath under projectRoot with its
// recorded state: it is unmodified only when both the content hash (of the
// file or of its CRLF checkout read as LF, see MatchesHash) and, where
// recorded and meaningful, the permission bits match. An empty file is
// content like any other.
func CheckFile(projectRoot, relPath string, fs types.FileState) FileStatus {
	return checkOnDisk(projectRoot, relPath, fs.Hash, fs.Mode, func(data []byte) bool {
		return MatchesHash(data, fs.Hash)
	})
}

// CheckContent compares the file at relPath under projectRoot with want, the
// content the generator writes for it, rather than with a recorded hash, so a
// committed state or manifest re-hashed over other content cannot vouch for
// it. A checkout whose lines end in CRLF (Git's core.autocrlf on Windows) of
// want is unmodified; any other difference, a lone CR included, is not. mode
// is compared as CheckFile compares the recorded mode.
func CheckContent(projectRoot, relPath string, want []byte, mode os.FileMode) FileStatus {
	return checkOnDisk(projectRoot, relPath, ComputeHash(want), mode, func(data []byte) bool {
		return EqualText(data, want)
	})
}

// crlfToLF replaces every CRLF line ending in data with LF.
func crlfToLF(data []byte) []byte {
	return bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
}

// checkOnDisk reads the file at relPath under projectRoot and reports it
// unmodified when matches accepts its content and, where recorded and
// meaningful, its permission bits equal mode. storedHash is reported as the
// expected hash.
func checkOnDisk(projectRoot, relPath, storedHash string, mode os.FileMode, matches func([]byte) bool) FileStatus {
	absPath := filepath.Join(projectRoot, relPath)
	status := FileStatus{
		Path:       relPath,
		StoredHash: storedHash,
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			status.Status = types.Deleted
		} else {
			status.Status = types.Unknown
			status.Error = err
		}
		return status
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		status.Status = types.Unknown
		status.Error = fmt.Errorf("computing file hash for %s: %w", absPath, err)
		return status
	}
	status.CurrentHash = ComputeHash(data)

	// A zero stored mode means the mode was never recorded (legacy state or a
	// generator that relied on the pipeline default), so only the content can
	// be compared.
	modeMatch := runtime.GOOS == "windows" || mode == 0 || info.Mode().Perm() == mode.Perm()
	if matches(data) && modeMatch {
		status.Status = types.Unmodified
	} else {
		status.Status = types.Modified
	}
	return status
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
