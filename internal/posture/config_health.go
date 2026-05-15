package posture

import "github.com/Quantum-Serendipity/qsdev/pkg/types"

// FileCategory returns "machine-owned" or "human-edited" based on MergeStrategy.
func FileCategory(strategy types.MergeStrategy) string {
	switch strategy {
	case types.Overwrite, types.LibraryManaged, types.Skip, types.Append:
		return "machine-owned"
	case types.SectionMarker, types.ThreeWayMerge, types.ManualMerge, types.Merge:
		return "human-edited"
	default:
		return "machine-owned"
	}
}

// ComputeConfigScore scores config file health (0-100).
//
// Scoring per file:
//   - current:  100%
//   - modified + human-edited:  100%  (expected for human-edited files)
//   - modified + machine-owned: 50%   (drift from generated state)
//   - outdated: 50%
//   - missing:  0%
//   - corrupt:  0%
func ComputeConfigScore(files []ConfigFileInfo) float64 {
	if len(files) == 0 {
		return 100.0
	}

	var total float64
	for _, f := range files {
		switch f.State {
		case "current":
			total += 100.0
		case "modified":
			if f.Category == "human-edited" {
				total += 100.0
			} else {
				total += 50.0
			}
		case "outdated":
			total += 50.0
		case "missing", "corrupt":
			// 0 points
		}
	}

	return total / float64(len(files))
}
