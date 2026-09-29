package types

import "github.com/Quantum-Serendipity/qsdev/internal/enumtext"

// ModificationStatus indicates the state of a previously generated file
// relative to its last known generated content.
type ModificationStatus int

const (
	Unmodified ModificationStatus = iota
	Modified
	Deleted
	New
	Unknown
)

var modificationStatusNames = [...]string{
	Unmodified: "unmodified",
	Modified:   "modified",
	Deleted:    "deleted",
	New:        "new",
	Unknown:    "unknown",
}

var modificationStatusText = enumtext.New[ModificationStatus]("ModificationStatus", "modification status", "invalid", modificationStatusNames[:])

func (s ModificationStatus) String() string { return modificationStatusText.String(s) }

func (s ModificationStatus) MarshalText() ([]byte, error) {
	return modificationStatusText.MarshalText(s)
}

func (s *ModificationStatus) UnmarshalText(text []byte) error {
	return modificationStatusText.UnmarshalText(text, s)
}
