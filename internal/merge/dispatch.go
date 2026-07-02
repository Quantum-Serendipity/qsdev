package merge

import (
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Dispatch is the single entry point that routes a file to the merge function
// for its strategy, so every write path (init/join create, claude update,
// devinit update) shares one dispatch table and one unknown-path policy.
//
// base is the last-generated content (nil on the create path); theirs is the
// current on-disk content; ours is the newly generated content. Routing uses
// SUFFIX matching on relPath so nested files (e.g. "sub/.mcp.json") resolve the
// same as top-level ones.
//
// Unknown-path policy: for ThreeWayMerge, a path that matches no known handler
// returns an error rather than silently overwriting — callers surface or skip
// it instead of discarding user content. Unknown strategies likewise error.
func Dispatch(relPath string, strategy types.MergeStrategy, base, theirs, ours []byte) ([]byte, error) {
	switch strategy {
	case types.ThreeWayMerge:
		switch {
		case strings.HasSuffix(relPath, ".mcp.json"):
			return MergeMcpJson(base, theirs, ours)
		case strings.HasSuffix(relPath, "settings.json"):
			return MergeSettings(base, theirs, ours)
		default:
			return nil, fmt.Errorf("no three-way merge handler for %q", relPath)
		}
	case types.SectionMarker:
		return SectionMarkers(theirs, ours)
	case types.LibraryManaged:
		return ours, nil
	default:
		return nil, fmt.Errorf("merge strategy %s not implemented for %q", strategy, relPath)
	}
}
