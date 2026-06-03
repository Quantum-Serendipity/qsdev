package generate

import (
	"encoding/json"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

type mockProducer struct {
	fragments []types.FragmentEntry
	err       error
}

func (m *mockProducer) Produce(types.WizardAnswers) ([]types.FragmentEntry, error) {
	return m.fragments, m.err
}

func TestNewFragmentAccumulator(t *testing.T) {
	t.Parallel()

	a := NewFragmentAccumulator()

	if got := len(a.producers); got != 0 {
		t.Fatalf("expected 0 producers, got %d", got)
	}
	if got := len(a.fragments); got != 0 {
		t.Fatalf("expected 0 fragments, got %d", got)
	}
	if got := a.FragmentSet(); len(got) != 0 {
		t.Fatalf("expected empty fragment set, got %d", len(got))
	}
}

func TestRegisterProducer_Duplicate(t *testing.T) {
	t.Parallel()

	a := NewFragmentAccumulator()
	p := &mockProducer{}

	if err := a.RegisterProducer("devenv", p); err != nil {
		t.Fatalf("first registration failed: %v", err)
	}
	if err := a.RegisterProducer("devenv", p); err == nil {
		t.Fatal("expected error on duplicate registration, got nil")
	}
}

func TestCollectAll_SingleProducer(t *testing.T) {
	t.Parallel()

	a := NewFragmentAccumulator()
	p := &mockProducer{
		fragments: []types.FragmentEntry{
			{Source: "devenv", Target: "a.txt", Content: []byte("a")},
			{Source: "devenv", Target: "b.txt", Content: []byte("b")},
			{Source: "devenv", Target: "c.txt", Content: []byte("c")},
		},
	}
	if err := a.RegisterProducer("devenv", p); err != nil {
		t.Fatal(err)
	}

	if err := a.CollectAll(types.WizardAnswers{}); err != nil {
		t.Fatalf("CollectAll failed: %v", err)
	}

	if got := len(a.FragmentSet()); got != 3 {
		t.Fatalf("expected 3 fragments, got %d", got)
	}
}

func TestCollectAll_MultipleProducers(t *testing.T) {
	t.Parallel()

	a := NewFragmentAccumulator()
	p1 := &mockProducer{
		fragments: []types.FragmentEntry{
			{Source: "devenv", Target: "a.txt", Content: []byte("a")},
		},
	}
	p2 := &mockProducer{
		fragments: []types.FragmentEntry{
			{Source: "claude", Target: "b.txt", Content: []byte("b")},
			{Source: "claude", Target: "c.txt", Content: []byte("c")},
		},
	}

	if err := a.RegisterProducer("devenv", p1); err != nil {
		t.Fatal(err)
	}
	if err := a.RegisterProducer("claude", p2); err != nil {
		t.Fatal(err)
	}

	if err := a.CollectAll(types.WizardAnswers{}); err != nil {
		t.Fatalf("CollectAll failed: %v", err)
	}

	if got := len(a.FragmentSet()); got != 3 {
		t.Fatalf("expected 3 fragments, got %d", got)
	}
}

func TestCollectAll_ErrorAggregation(t *testing.T) {
	t.Parallel()

	a := NewFragmentAccumulator()

	failing := &mockProducer{err: errTest}
	ok := &mockProducer{
		fragments: []types.FragmentEntry{
			{Source: "ok", Target: "a.txt", Content: []byte("a")},
		},
	}

	if err := a.RegisterProducer("failing", failing); err != nil {
		t.Fatal(err)
	}
	if err := a.RegisterProducer("ok", ok); err != nil {
		t.Fatal(err)
	}

	// One producer succeeds, so CollectAll should not return an error.
	if err := a.CollectAll(types.WizardAnswers{}); err != nil {
		t.Fatalf("expected nil error when at least one producer succeeds, got: %v", err)
	}

	if got := len(a.FragmentSet()); got != 1 {
		t.Fatalf("expected 1 fragment from successful producer, got %d", got)
	}
}

func TestCollectAll_AllFail(t *testing.T) {
	t.Parallel()

	a := NewFragmentAccumulator()

	f1 := &mockProducer{err: errTest}
	f2 := &mockProducer{err: errTest}

	if err := a.RegisterProducer("f1", f1); err != nil {
		t.Fatal(err)
	}
	if err := a.RegisterProducer("f2", f2); err != nil {
		t.Fatal(err)
	}

	if err := a.CollectAll(types.WizardAnswers{}); err == nil {
		t.Fatal("expected error when all producers fail, got nil")
	}
}

var errTest = errorString("test error")

type errorString string

func (e errorString) Error() string { return string(e) }

func TestResolve_ComposeReplace(t *testing.T) {
	t.Parallel()

	a := NewFragmentAccumulator()
	a.Add(types.FragmentEntry{
		Source:      "low",
		Target:      "config.json",
		Content:     []byte("low priority"),
		Priority:    100,
		ComposeMode: types.ComposeReplace,
		Mode:        0o644,
	})
	a.Add(types.FragmentEntry{
		Source:      "high",
		Target:      "config.json",
		Content:     []byte("high priority"),
		Priority:    500,
		ComposeMode: types.ComposeReplace,
		Mode:        0o644,
	})

	files, err := a.Resolve()
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if got := string(files[0].Content); got != "high priority" {
		t.Fatalf("expected high priority content, got %q", got)
	}
}

func TestResolve_ComposeAppend(t *testing.T) {
	t.Parallel()

	a := NewFragmentAccumulator()
	// Add fragments with different priorities to test sort-order concatenation.
	for _, entry := range []types.FragmentEntry{
		{Source: "c", Target: "out.txt", Content: []byte("third"), Priority: 100, ComposeMode: types.ComposeAppend},
		{Source: "a", Target: "out.txt", Content: []byte("first"), Priority: 300, ComposeMode: types.ComposeAppend},
		{Source: "b", Target: "out.txt", Content: []byte("second"), Priority: 200, ComposeMode: types.ComposeAppend},
	} {
		a.Add(entry)
	}

	files, err := a.Resolve()
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	got := string(files[0].Content)
	if !strings.Contains(got, "first") || !strings.Contains(got, "second") || !strings.Contains(got, "third") {
		t.Fatalf("expected all three fragments in output, got %q", got)
	}

	// The SortKey sorts by source then inverted priority.
	// "a" with priority 300 sorts first (99999-300=99699),
	// "b" with priority 200 sorts second (99999-200=99799),
	// "c" with priority 100 sorts third (99999-100=99899).
	parts := strings.Split(got, "\n")
	if parts[0] != "first" {
		t.Fatalf("expected 'first' as first part, got %q", parts[0])
	}
}

func TestResolve_ComposeSection(t *testing.T) {
	t.Parallel()

	base := "# Header\nSome content\n<!-- BEGIN GENERATED SECTION — hooks -->\nold hooks\n<!-- END GENERATED SECTION -->\n# Footer\n"
	hooks := "new hook content"

	a := NewFragmentAccumulator()
	a.Add(types.FragmentEntry{
		Source:      "base",
		Target:      "CLAUDE.md",
		Content:     []byte(base),
		Priority:    100,
		ComposeMode: types.ComposeSection,
		Tag:         "",
	})
	a.Add(types.FragmentEntry{
		Source:      "hooks",
		Target:      "CLAUDE.md",
		Content:     []byte(hooks),
		Priority:    200,
		ComposeMode: types.ComposeSection,
		Tag:         "hooks",
	})

	files, err := a.Resolve()
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	got := string(files[0].Content)
	if !strings.Contains(got, "# Header") {
		t.Fatal("expected header preserved")
	}
	if !strings.Contains(got, "# Footer") {
		t.Fatal("expected footer preserved")
	}
	if !strings.Contains(got, "new hook content") {
		t.Fatal("expected new hook content inserted")
	}
	if strings.Contains(got, "old hooks") {
		t.Fatal("expected old hooks replaced")
	}
}

func TestResolve_ComposeSectionNoBase(t *testing.T) {
	t.Parallel()

	a := NewFragmentAccumulator()
	a.Add(types.FragmentEntry{
		Source:      "hooks",
		Target:      "CLAUDE.md",
		Content:     []byte("hook content"),
		Priority:    200,
		ComposeMode: types.ComposeSection,
		Tag:         "hooks",
	})
	a.Add(types.FragmentEntry{
		Source:      "security",
		Target:      "CLAUDE.md",
		Content:     []byte("security content"),
		Priority:    100,
		ComposeMode: types.ComposeSection,
		Tag:         "security",
	})

	files, err := a.Resolve()
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	got := string(files[0].Content)
	if !strings.Contains(got, "<!-- BEGIN GENERATED SECTION") {
		t.Fatal("expected section markers in output")
	}
	if !strings.Contains(got, "hook content") {
		t.Fatal("expected hook content")
	}
	if !strings.Contains(got, "security content") {
		t.Fatal("expected security content")
	}
}

func TestResolve_ComposeMergeJSON(t *testing.T) {
	t.Parallel()

	low := `{"a": 1, "b": {"nested": "low"}, "c": 3}`
	high := `{"a": 99, "b": {"nested": "high", "extra": true}}`

	a := NewFragmentAccumulator()
	a.Add(types.FragmentEntry{
		Source:      "low",
		Target:      "config.json",
		Content:     []byte(low),
		Priority:    100,
		ComposeMode: types.ComposeMergeJSON,
	})
	a.Add(types.FragmentEntry{
		Source:      "high",
		Target:      "config.json",
		Content:     []byte(high),
		Priority:    500,
		ComposeMode: types.ComposeMergeJSON,
	})

	files, err := a.Resolve()
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(files[0].Content, &result); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	// High priority wins for "a".
	if got := result["a"]; got != float64(99) {
		t.Fatalf("expected a=99, got %v", got)
	}

	// "c" from low priority is preserved.
	if got := result["c"]; got != float64(3) {
		t.Fatalf("expected c=3, got %v", got)
	}

	// Nested merge: "nested" from high wins, "extra" from high is added.
	b, ok := result["b"].(map[string]any)
	if !ok {
		t.Fatalf("expected b to be a map, got %T", result["b"])
	}
	if got := b["nested"]; got != "high" {
		t.Fatalf("expected nested=high, got %v", got)
	}
	if got := b["extra"]; got != true {
		t.Fatalf("expected extra=true, got %v", got)
	}
}

func TestResolve_ComposeMergeYAML(t *testing.T) {
	t.Parallel()

	low := "a: 1\nb:\n  nested: low\nc: 3\n"
	high := "a: 99\nb:\n  nested: high\n  extra: true\n"

	a := NewFragmentAccumulator()
	a.Add(types.FragmentEntry{
		Source:      "low",
		Target:      "config.yaml",
		Content:     []byte(low),
		Priority:    100,
		ComposeMode: types.ComposeMergeYAML,
	})
	a.Add(types.FragmentEntry{
		Source:      "high",
		Target:      "config.yaml",
		Content:     []byte(high),
		Priority:    500,
		ComposeMode: types.ComposeMergeYAML,
	})

	files, err := a.Resolve()
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	var result map[string]any
	if err := yaml.Unmarshal(files[0].Content, &result); err != nil {
		t.Fatalf("output is not valid YAML: %v", err)
	}

	if got := result["a"]; got != 99 {
		t.Fatalf("expected a=99, got %v", got)
	}
	if got := result["c"]; got != 3 {
		t.Fatalf("expected c=3, got %v", got)
	}

	b, ok := result["b"].(map[string]any)
	if !ok {
		t.Fatalf("expected b to be a map, got %T", result["b"])
	}
	if got := b["nested"]; got != "high" {
		t.Fatalf("expected nested=high, got %v", got)
	}
	if got := b["extra"]; got != true {
		t.Fatalf("expected extra=true, got %v", got)
	}
}

func TestResolve_DeterministicOrdering(t *testing.T) {
	t.Parallel()

	a := NewFragmentAccumulator()
	a.Add(types.FragmentEntry{Source: "z", Target: "z.txt", Content: []byte("z"), ComposeMode: types.ComposeReplace, Priority: 1})
	a.Add(types.FragmentEntry{Source: "a", Target: "a.txt", Content: []byte("a"), ComposeMode: types.ComposeReplace, Priority: 1})
	a.Add(types.FragmentEntry{Source: "m", Target: "m.txt", Content: []byte("m"), ComposeMode: types.ComposeReplace, Priority: 1})

	files, err := a.Resolve()
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if len(files) != 3 {
		t.Fatalf("expected 3 files, got %d", len(files))
	}
	if files[0].Path != "a.txt" || files[1].Path != "m.txt" || files[2].Path != "z.txt" {
		t.Fatalf("expected files sorted by path, got %s, %s, %s",
			files[0].Path, files[1].Path, files[2].Path)
	}
}

func TestResolve_Empty(t *testing.T) {
	t.Parallel()

	a := NewFragmentAccumulator()
	files, err := a.Resolve()
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if files != nil {
		t.Fatalf("expected nil for empty resolve, got %v", files)
	}
}

func TestFragmentSet_ReturnsCopy(t *testing.T) {
	t.Parallel()

	a := NewFragmentAccumulator()
	a.Add(types.FragmentEntry{Source: "test", Target: "a.txt", Content: []byte("a")})

	set := a.FragmentSet()
	set[0].Source = "modified"

	original := a.FragmentSet()
	if original[0].Source != "test" {
		t.Fatal("modifying returned slice affected accumulator internal state")
	}
}

func TestResolve_MixedComposeMode(t *testing.T) {
	t.Parallel()

	a := NewFragmentAccumulator()
	a.Add(types.FragmentEntry{
		Source:      "a",
		Target:      "file.txt",
		Content:     []byte("a"),
		ComposeMode: types.ComposeReplace,
	})
	a.Add(types.FragmentEntry{
		Source:      "b",
		Target:      "file.txt",
		Content:     []byte("b"),
		ComposeMode: types.ComposeAppend,
	})

	_, err := a.Resolve()
	if err == nil {
		t.Fatal("expected error for mixed compose modes, got nil")
	}
	if !strings.Contains(err.Error(), "mixed compose modes") {
		t.Fatalf("expected 'mixed compose modes' error, got: %v", err)
	}
}
