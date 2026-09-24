package types

import (
	"cmp"
	"fmt"
	"os"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/enumtext"
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

const (
	// PriorityCeiling is the maximum priority SortKey can encode. SortKey
	// inverts priorities using this ceiling so higher-priority fragments sort
	// first; CompareFragments has no such limit.
	PriorityCeiling = 99999

	// PriorityGeneratorDefault is the default for fragments produced by legacy
	// Generator adapters (devenv, claudecode).
	PriorityGeneratorDefault = 1000

	// PriorityCIWorkflow is the priority for CI workflow fragments.
	PriorityCIWorkflow = 500
)

var composeModeNames = [...]string{
	ComposeReplace:   "replace",
	ComposeAppend:    "append",
	ComposeSection:   "section",
	ComposeMergeJSON: "merge-json",
	ComposeMergeYAML: "merge-yaml",
}

var composeModeText = enumtext.New[ComposeMode]("ComposeMode", "compose mode", "unknown", composeModeNames[:])

func (c ComposeMode) String() string { return composeModeText.String(c) }

func (c ComposeMode) MarshalText() ([]byte, error) { return composeModeText.MarshalText(c) }

func (c *ComposeMode) UnmarshalText(text []byte) error { return composeModeText.UnmarshalText(text, c) }

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

// CompareFragments orders fragments deterministically for resolution: by
// Target, then Priority descending (so the highest-priority fragment for a
// target comes first), then Source, then Tag. Priorities compare numerically,
// so negative or above-ceiling values order correctly.
func CompareFragments(a, b FragmentEntry) int {
	return cmp.Or(
		cmp.Compare(a.Target, b.Target),
		cmp.Compare(b.Priority, a.Priority),
		cmp.Compare(a.Source, b.Source),
		cmp.Compare(a.Tag, b.Tag),
	)
}

// SortKey returns a composite string key of Target, inverted Priority (higher
// priorities sort first), Source, and Tag, with out-of-range priorities
// clamped to [0, PriorityCeiling]. It is an identifier, not a sort order: the
// "|" separator makes its lexical order diverge from CompareFragments when one
// Target or Source is a prefix of another, so sort with CompareFragments.
func (f FragmentEntry) SortKey() string {
	p := min(max(f.Priority, 0), PriorityCeiling)
	return fmt.Sprintf("%s|%05d|%s|%s", f.Target, PriorityCeiling-p, f.Source, f.Tag)
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
