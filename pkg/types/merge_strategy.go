package types

import "fmt"

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

var mergeStrategyFromString = func() map[string]MergeStrategy {
	m := make(map[string]MergeStrategy, len(mergeStrategyNames))
	for i, name := range mergeStrategyNames {
		m[name] = MergeStrategy(i)
	}
	return m
}()

func (m MergeStrategy) String() string {
	if int(m) >= 0 && int(m) < len(mergeStrategyNames) {
		return mergeStrategyNames[m]
	}
	return "unknown"
}

func (m MergeStrategy) MarshalText() ([]byte, error) {
	s := m.String()
	if s == "unknown" {
		return nil, fmt.Errorf("cannot marshal unknown MergeStrategy value %d", int(m))
	}
	return []byte(s), nil
}

func (m *MergeStrategy) UnmarshalText(text []byte) error {
	s := string(text)
	if v, ok := mergeStrategyFromString[s]; ok {
		*m = v
		return nil
	}
	return fmt.Errorf("unknown merge strategy: %q", s)
}
