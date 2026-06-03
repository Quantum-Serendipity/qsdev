package types_test

import (
	"sort"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestComposeMode_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		mode     types.ComposeMode
		expected string
	}{
		{types.ComposeReplace, "replace"},
		{types.ComposeAppend, "append"},
		{types.ComposeSection, "section"},
		{types.ComposeMergeJSON, "merge-json"},
		{types.ComposeMergeYAML, "merge-yaml"},
		{types.ComposeMode(99), "unknown"},
		{types.ComposeMode(-1), "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			t.Parallel()
			if got := tt.mode.String(); got != tt.expected {
				t.Errorf("ComposeMode(%d).String() = %q, want %q", int(tt.mode), got, tt.expected)
			}
		})
	}
}

func TestComposeMode_MarshalText(t *testing.T) {
	t.Parallel()

	validModes := []types.ComposeMode{
		types.ComposeReplace,
		types.ComposeAppend,
		types.ComposeSection,
		types.ComposeMergeJSON,
		types.ComposeMergeYAML,
	}
	for _, cm := range validModes {
		t.Run(cm.String(), func(t *testing.T) {
			t.Parallel()
			text, err := cm.MarshalText()
			if err != nil {
				t.Fatalf("ComposeMode(%d).MarshalText() error: %v", int(cm), err)
			}
			if string(text) != cm.String() {
				t.Errorf("MarshalText() = %q, want %q", text, cm.String())
			}
		})
	}

	t.Run("unknown returns error", func(t *testing.T) {
		t.Parallel()
		_, err := types.ComposeMode(99).MarshalText()
		if err == nil {
			t.Error("expected error for unknown ComposeMode marshal, got nil")
		}
	})
}

func TestComposeMode_UnmarshalText(t *testing.T) {
	t.Parallel()

	validNames := []string{"replace", "append", "section", "merge-json", "merge-yaml"}
	expectedModes := []types.ComposeMode{
		types.ComposeReplace,
		types.ComposeAppend,
		types.ComposeSection,
		types.ComposeMergeJSON,
		types.ComposeMergeYAML,
	}

	for i, name := range validNames {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var got types.ComposeMode
			if err := got.UnmarshalText([]byte(name)); err != nil {
				t.Fatalf("UnmarshalText(%q) error: %v", name, err)
			}
			if got != expectedModes[i] {
				t.Errorf("UnmarshalText(%q) = %d, want %d", name, int(got), int(expectedModes[i]))
			}
		})
	}

	t.Run("invalid returns error", func(t *testing.T) {
		t.Parallel()
		var cm types.ComposeMode
		if err := cm.UnmarshalText([]byte("bogus")); err == nil {
			t.Error("expected error for unknown string, got nil")
		}
	})
}

func TestComposeMode_RoundTrip(t *testing.T) {
	t.Parallel()

	allModes := []types.ComposeMode{
		types.ComposeReplace,
		types.ComposeAppend,
		types.ComposeSection,
		types.ComposeMergeJSON,
		types.ComposeMergeYAML,
	}
	for _, cm := range allModes {
		t.Run(cm.String(), func(t *testing.T) {
			t.Parallel()
			text, err := cm.MarshalText()
			if err != nil {
				t.Fatalf("MarshalText() error: %v", err)
			}
			var got types.ComposeMode
			if err := got.UnmarshalText(text); err != nil {
				t.Fatalf("UnmarshalText(%q) error: %v", text, err)
			}
			if got != cm {
				t.Errorf("round-trip: got %v, want %v", got, cm)
			}
		})
	}
}

func TestFragmentEntry_SortKey(t *testing.T) {
	t.Parallel()

	entries := []types.FragmentEntry{
		{Source: "devenv", Target: ".envrc", Priority: 10, Tag: "base"},
		{Source: "devenv", Target: ".envrc", Priority: 50, Tag: "extra"},
		{Source: "claudecode", Target: ".envrc", Priority: 10, Tag: "hooks"},
		{Source: "devenv", Target: ".envrc", Priority: 10, Tag: "alpha"},
	}

	keys := make([]string, len(entries))
	for i, e := range entries {
		keys[i] = e.SortKey()
	}

	sorted := make([]string, len(keys))
	copy(sorted, keys)
	sort.Strings(sorted)

	t.Run("higher priority sorts first within same source and target", func(t *testing.T) {
		t.Parallel()
		// priority 50 should sort before priority 10 (inverted in key)
		key50 := entries[1].SortKey() // devenv|.envrc|49949|extra
		key10 := entries[0].SortKey() // devenv|.envrc|99989|base
		if key50 >= key10 {
			t.Errorf("priority 50 key %q should sort before priority 10 key %q", key50, key10)
		}
	})

	t.Run("different sources sort alphabetically", func(t *testing.T) {
		t.Parallel()
		keyC := entries[2].SortKey() // claudecode|...
		keyD := entries[0].SortKey() // devenv|...
		if keyC >= keyD {
			t.Errorf("claudecode key %q should sort before devenv key %q", keyC, keyD)
		}
	})

	t.Run("tag is the last tiebreaker", func(t *testing.T) {
		t.Parallel()
		// Same source, target, priority — only tag differs.
		keyAlpha := entries[3].SortKey() // devenv|.envrc|99989|alpha
		keyBase := entries[0].SortKey()  // devenv|.envrc|99989|base
		if keyAlpha >= keyBase {
			t.Errorf("tag alpha key %q should sort before tag base key %q", keyAlpha, keyBase)
		}
	})

	t.Run("all keys produce deterministic sort", func(t *testing.T) {
		t.Parallel()
		if !sort.StringsAreSorted(sorted) {
			t.Errorf("sorted keys should be in order: %v", sorted)
		}
	})
}

func TestFragmentLedgerEntry_YAMLRoundTrip(t *testing.T) {
	t.Parallel()

	ts := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	original := types.FragmentLedgerEntry{
		Source:      "claudecode",
		Tag:         "hooks",
		Priority:    42,
		ComposeMode: types.ComposeSection,
		ContentHash: "sha256:abc123",
		Timestamp:   ts,
		Reason:      "generated by claudecode addon",
	}

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("yaml.Marshal: %v", err)
	}

	var got types.FragmentLedgerEntry
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("yaml.Unmarshal: %v", err)
	}

	if got.Source != original.Source {
		t.Errorf("Source: got %q, want %q", got.Source, original.Source)
	}
	if got.Tag != original.Tag {
		t.Errorf("Tag: got %q, want %q", got.Tag, original.Tag)
	}
	if got.Priority != original.Priority {
		t.Errorf("Priority: got %d, want %d", got.Priority, original.Priority)
	}
	if got.ComposeMode != original.ComposeMode {
		t.Errorf("ComposeMode: got %v, want %v", got.ComposeMode, original.ComposeMode)
	}
	if got.ContentHash != original.ContentHash {
		t.Errorf("ContentHash: got %q, want %q", got.ContentHash, original.ContentHash)
	}
	if !got.Timestamp.Equal(original.Timestamp) {
		t.Errorf("Timestamp: got %v, want %v", got.Timestamp, original.Timestamp)
	}
	if got.Reason != original.Reason {
		t.Errorf("Reason: got %q, want %q", got.Reason, original.Reason)
	}
}

func TestFragmentEntry_SortKey_OverflowPriority(t *testing.T) {
	t.Parallel()

	f := types.FragmentEntry{
		Source:   "test",
		Target:   "file.txt",
		Priority: types.PriorityCeiling + 1,
	}
	key := f.SortKey()
	// Priority > PriorityCeiling produces a negative inverted value in the sort
	// key, causing such fragments to sort before all valid-priority fragments.
	if !strings.Contains(key, "-") {
		t.Fatalf("expected negative value in sort key for overflow priority, got %q", key)
	}
}
