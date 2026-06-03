package types

import (
	"fmt"
	"os"
	"time"
)

// ComposeMode determines how multiple fragments targeting the same file
// are combined before writing.
type ComposeMode int

const (
	ComposeReplace   ComposeMode = iota // Highest-priority fragment wins.
	ComposeAppend                       // Concatenate in priority order with separator.
	ComposeSection                      // Delegate to section-marker merge.
	ComposeMergeJSON                    // Key-level JSON object merge.
	ComposeMergeYAML                    // Key-level YAML map merge.
)

var composeModeNames = [...]string{
	ComposeReplace:   "replace",
	ComposeAppend:    "append",
	ComposeSection:   "section",
	ComposeMergeJSON: "merge-json",
	ComposeMergeYAML: "merge-yaml",
}

var composeModeFromString = func() map[string]ComposeMode {
	m := make(map[string]ComposeMode, len(composeModeNames))
	for i, name := range composeModeNames {
		m[name] = ComposeMode(i)
	}
	return m
}()

func (c ComposeMode) String() string {
	if int(c) >= 0 && int(c) < len(composeModeNames) {
		return composeModeNames[c]
	}
	return "unknown"
}

func (c ComposeMode) MarshalText() ([]byte, error) {
	s := c.String()
	if s == "unknown" {
		return nil, fmt.Errorf("cannot marshal unknown ComposeMode value %d", int(c))
	}
	return []byte(s), nil
}

func (c *ComposeMode) UnmarshalText(text []byte) error {
	s := string(text)
	if v, ok := composeModeFromString[s]; ok {
		*c = v
		return nil
	}
	return fmt.Errorf("unknown compose mode: %q", s)
}

// FragmentEntry represents a single contribution from an addon or module
// to a target file.
type FragmentEntry struct {
	Source      string             // Addon or module name (e.g., "devenv", "claudecode").
	Target      string             // Relative file path from project root.
	Content     []byte             // Raw content of this fragment.
	Priority    int                // Higher priority wins in ComposeReplace; sort tiebreaker otherwise.
	ComposeMode ComposeMode        // How to combine with other fragments targeting the same file.
	Tag         string             // Section identifier for ComposeSection.
	Strategy    MergeStrategy      // On-disk merge strategy for the resolved file.
	Mode        os.FileMode        // File permission mode (0 = default 0o644).
	Owner       string             // Tool owner for teardown tracking.
	Provenance  FragmentProvenance // Metadata for the provenance ledger.
}

// SortKey returns a composite sort key ensuring deterministic fragment ordering.
// Priority is inverted so higher-priority fragments sort first.
func (f FragmentEntry) SortKey() string {
	return fmt.Sprintf("%s|%s|%05d|%s", f.Source, f.Target, 99999-f.Priority, f.Tag)
}

// FragmentProvenance records metadata about how and when a fragment was produced.
type FragmentProvenance struct {
	Module    string    // Fully qualified module name.
	Timestamp time.Time // When the fragment was produced.
	Reason    string    // Human-readable reason.
}

// FragmentLedgerEntry is the persisted form of a fragment's provenance,
// stored in GeneratedState.Fragments.
type FragmentLedgerEntry struct {
	Source      string      `yaml:"source"       json:"source"`
	Tag         string      `yaml:"tag"          json:"tag"`
	Priority    int         `yaml:"priority"     json:"priority"`
	ComposeMode ComposeMode `yaml:"compose_mode" json:"compose_mode"`
	ContentHash string      `yaml:"content_hash" json:"content_hash"`
	Timestamp   time.Time   `yaml:"timestamp"    json:"timestamp"`
	Reason      string      `yaml:"reason"       json:"reason"`
}

// FragmentProducer is the interface for anything that contributes fragments
// to the accumulation engine.
type FragmentProducer interface {
	Produce(answers WizardAnswers) ([]FragmentEntry, error)
}
