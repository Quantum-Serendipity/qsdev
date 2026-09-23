package types

import "github.com/Quantum-Serendipity/qsdev/internal/enumtext"

// MergeStrategy defines how a generated file should be handled
// when it already exists on disk.
type MergeStrategy int

const (
	Overwrite MergeStrategy = iota
	Append
	Merge
	Skip
	SectionMarker
	ThreeWayMerge
	LibraryManaged
	ManualMerge
)

var mergeStrategyNames = [...]string{
	Overwrite:      "overwrite",
	Append:         "append",
	Merge:          "merge",
	Skip:           "skip",
	SectionMarker:  "section-marker",
	ThreeWayMerge:  "three-way-merge",
	LibraryManaged: "library-managed",
	ManualMerge:    "manual-merge",
}

var mergeStrategyText = enumtext.New[MergeStrategy]("MergeStrategy", "merge strategy", "unknown", mergeStrategyNames[:])

// IsHumanEdited reports whether files written with this strategy are expected
// to be edited by people, so a divergence from the generated content is
// normal rather than drift. Overwrite, Append, Skip and LibraryManaged files
// are machine-owned. It is the single classifier every consumer should use,
// so that one file is never healthy in one report and drifted in another.
func (m MergeStrategy) IsHumanEdited() bool {
	switch m {
	case SectionMarker, ThreeWayMerge, ManualMerge, Merge:
		return true
	default:
		return false
	}
}

func (m MergeStrategy) String() string { return mergeStrategyText.String(m) }

func (m MergeStrategy) MarshalText() ([]byte, error) { return mergeStrategyText.MarshalText(m) }

func (m *MergeStrategy) UnmarshalText(text []byte) error {
	return mergeStrategyText.UnmarshalText(text, m)
}
