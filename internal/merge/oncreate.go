package merge

// MergeOnCreate merges newly generated content (ours) over existing on-disk
// content (theirs) for ThreeWayMerge files that have no recorded base — the
// create / first-generation path. It dispatches by well-known relative path so
// the pipeline can protect user-owned top-level keys (e.g. settings.json "env")
// without threading strategy details through the writer.
//
// Unknown paths fall back to ours (a plain overwrite). Both MergeSettings and
// MergeMcpJson tolerate a nil base and preserve unknown top-level keys, but
// return an error on empty theirs — the pipeline treats that as "nothing to
// preserve" and falls through to a full overwrite.
func MergeOnCreate(relPath string, theirs, ours []byte) ([]byte, error) {
	switch relPath {
	case ".claude/settings.json":
		return MergeSettings(nil, theirs, ours)
	case ".mcp.json":
		return MergeMcpJson(nil, theirs, ours)
	default:
		return ours, nil
	}
}
