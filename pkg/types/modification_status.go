package types

import "fmt"

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

var modificationStatusFromString = func() map[string]ModificationStatus {
	m := make(map[string]ModificationStatus, len(modificationStatusNames))
	for i, name := range modificationStatusNames {
		m[name] = ModificationStatus(i)
	}
	return m
}()

func (s ModificationStatus) String() string {
	if int(s) >= 0 && int(s) < len(modificationStatusNames) {
		return modificationStatusNames[s]
	}
	return "invalid"
}

func (s ModificationStatus) MarshalText() ([]byte, error) {
	str := s.String()
	if str == "invalid" {
		return nil, fmt.Errorf("cannot marshal invalid ModificationStatus value %d", int(s))
	}
	return []byte(str), nil
}

func (s *ModificationStatus) UnmarshalText(text []byte) error {
	str := string(text)
	if v, ok := modificationStatusFromString[str]; ok {
		*s = v
		return nil
	}
	return fmt.Errorf("unknown modification status: %q", str)
}
