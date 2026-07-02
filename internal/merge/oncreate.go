package merge

import "github.com/Quantum-Serendipity/qsdev/pkg/types"

// MergeOnCreate merges newly generated content (ours) over existing on-disk
// content (theirs) for ThreeWayMerge files that have no recorded base — the
// create / first-generation path. It is a nil-base wrapper around Dispatch, so
// the create path protects user-owned top-level keys (e.g. settings.json "env")
// using the same routing and unknown-path policy as the update paths.
//
// Both MergeSettings and MergeMcpJson tolerate a nil base and preserve unknown
// keys, but return an error on empty theirs; the pipeline calls MergeOnCreate
// only for non-empty existing files and surfaces any error rather than
// silently overwriting.
func MergeOnCreate(relPath string, theirs, ours []byte) ([]byte, error) {
	return Dispatch(relPath, types.ThreeWayMerge, nil, theirs, ours)
}
