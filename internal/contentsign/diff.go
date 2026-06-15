package contentsign

import (
	"encoding/json"
	"fmt"
	"os"
)

// devDocsIndex is a minimal view of a DevDocs index.json. Only the entry name,
// path, and type are relevant to a structural diff; any other top-level fields
// (e.g. "types") are ignored.
type devDocsIndex struct {
	Entries []devDocsEntry `json:"entries"`
}

// devDocsEntry is one entry in a DevDocs index, keyed for diffing on its Path
// (the stable anchor; names and types may change between versions).
type devDocsEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"`
}

// ContentDiffDevDocs computes the structural difference between two DevDocs
// index.json files (old vs new). Entries are matched on their Path, the stable
// anchor across versions: an entry present only in new is counted as added, one
// present only in old as removed, and one present in both whose Name or Type
// changed as modified. SizeChangeBytes is the byte-length delta of the two
// index files (new minus old).
//
// Both paths are required; a read or parse failure for either is returned as a
// wrapped error.
//
// It is a building block for the hosted-mirror content-update workflow (it
// reports what changed between two corpus versions before re-signing); that
// workflow is wired in a later phase, so this ships tested but without an
// in-repo caller today.
func ContentDiffDevDocs(oldIndexPath, newIndexPath string) (*ContentDiff, error) {
	oldBytes, oldIdx, err := readDevDocsIndex(oldIndexPath)
	if err != nil {
		return nil, err
	}
	newBytes, newIdx, err := readDevDocsIndex(newIndexPath)
	if err != nil {
		return nil, err
	}

	oldByPath := indexByPath(oldIdx)
	newByPath := indexByPath(newIdx)

	diff := &ContentDiff{
		SizeChangeBytes: int64(len(newBytes)) - int64(len(oldBytes)),
	}

	for path, newEntry := range newByPath {
		oldEntry, ok := oldByPath[path]
		switch {
		case !ok:
			diff.AddedEntries++
		case oldEntry.Name != newEntry.Name || oldEntry.Type != newEntry.Type:
			diff.ModifiedEntries++
		}
	}

	for path := range oldByPath {
		if _, ok := newByPath[path]; !ok {
			diff.RemovedEntries++
		}
	}

	return diff, nil
}

// readDevDocsIndex reads and parses a DevDocs index.json, returning its raw
// bytes (for size accounting) alongside the parsed index.
func readDevDocsIndex(path string) ([]byte, devDocsIndex, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // path is operator-supplied.
	if err != nil {
		return nil, devDocsIndex{}, fmt.Errorf("reading index %q: %w", path, err)
	}
	var idx devDocsIndex
	if err := json.Unmarshal(raw, &idx); err != nil {
		return nil, devDocsIndex{}, fmt.Errorf("parsing index %q: %w", path, err)
	}
	return raw, idx, nil
}

// indexByPath builds a path-keyed map of an index's entries. When duplicate
// paths occur, the last entry wins.
func indexByPath(idx devDocsIndex) map[string]devDocsEntry {
	byPath := make(map[string]devDocsEntry, len(idx.Entries))
	for _, e := range idx.Entries {
		byPath[e.Path] = e
	}
	return byPath
}
