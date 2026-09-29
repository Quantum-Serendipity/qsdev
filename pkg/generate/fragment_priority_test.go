package generate

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/merge"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestResolve_PriorityBeatsSourceName uses source names that sort opposite to
// their priorities, so a source-first ordering would pick the wrong winner.
func TestResolve_PriorityBeatsSourceName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		mode        types.ComposeMode
		high, low   string
		wantContent string
	}{
		{
			name: "replace", mode: types.ComposeReplace,
			high: "HIGH", low: "LOW",
			wantContent: "HIGH",
		},
		{
			name: "merge json", mode: types.ComposeMergeJSON,
			high: `{"k":"HIGH"}`, low: `{"k":"LOW"}`,
			wantContent: `"k": "HIGH"`,
		},
		{
			name: "merge yaml", mode: types.ComposeMergeYAML,
			high: "k: HIGH\n", low: "k: LOW\n",
			wantContent: "k: HIGH",
		},
		{
			name: "append order", mode: types.ComposeAppend,
			high: "HIGH", low: "LOW",
			wantContent: "HIGH\nLOW",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a := NewFragmentAccumulator()
			a.Add(types.FragmentEntry{
				Source: "alpha", Target: "out", Priority: 10, ComposeMode: tt.mode,
				Content: []byte(tt.low), Strategy: types.Overwrite, Owner: "alpha",
			})
			a.Add(types.FragmentEntry{
				Source: "zeta", Target: "out", Priority: 5000, ComposeMode: tt.mode,
				Content: []byte(tt.high), Strategy: types.ThreeWayMerge, Owner: "zeta",
			})

			files, err := a.Resolve()
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if len(files) != 1 {
				t.Fatalf("expected 1 file, got %d", len(files))
			}
			f := files[0]
			if !strings.Contains(string(f.Content), tt.wantContent) {
				t.Errorf("content = %q, want it to contain %q", f.Content, tt.wantContent)
			}
			if f.Owner != "zeta" || f.Strategy != types.ThreeWayMerge {
				t.Errorf("owner/strategy = %q/%s, want zeta/%s (from the highest-priority fragment)",
					f.Owner, f.Strategy, types.ThreeWayMerge)
			}
		})
	}
}

// TestResolve_ComposeSectionKeepsEveryTag verifies that each tagged fragment
// lands in its own section: splicing one tag must not overwrite another.
func TestResolve_ComposeSectionKeepsEveryTag(t *testing.T) {
	t.Parallel()

	a := NewFragmentAccumulator()
	a.Add(types.FragmentEntry{Source: "base", Target: "CLAUDE.md", Priority: 100, ComposeMode: types.ComposeSection, Content: []byte("# base\n")})
	a.Add(types.FragmentEntry{Source: "one", Target: "CLAUDE.md", Priority: 50, ComposeMode: types.ComposeSection, Tag: "t1", Content: []byte("AAA")})
	a.Add(types.FragmentEntry{Source: "two", Target: "CLAUDE.md", Priority: 40, ComposeMode: types.ComposeSection, Tag: "t2", Content: []byte("BBB")})

	files, err := a.Resolve()
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	got := string(files[0].Content)

	for _, want := range []string{
		"# base\n",
		merge.SectionBeginLine("t1") + "\nAAA\n" + merge.EndMarker,
		merge.SectionBeginLine("t2") + "\nBBB\n" + merge.EndMarker,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("resolved content missing %q:\n%s", want, got)
		}
	}
}

// TestResolve_ComposeSectionReplacesExistingTag verifies a tagged fragment
// replaces the matching section already present in the base document.
func TestResolve_ComposeSectionReplacesExistingTag(t *testing.T) {
	t.Parallel()

	base := "# base\n" +
		merge.SectionBeginLine("t1") + "\nold1\n" + merge.EndMarker + "\n" +
		merge.SectionBeginLine("t2") + "\nold2\n" + merge.EndMarker + "\n"

	a := NewFragmentAccumulator()
	a.Add(types.FragmentEntry{Source: "base", Target: "CLAUDE.md", Priority: 100, ComposeMode: types.ComposeSection, Content: []byte(base)})
	a.Add(types.FragmentEntry{Source: "two", Target: "CLAUDE.md", Priority: 50, ComposeMode: types.ComposeSection, Tag: "t2", Content: []byte("new2")})

	files, err := a.Resolve()
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	want := "# base\n" +
		merge.SectionBeginLine("t1") + "\nold1\n" + merge.EndMarker + "\n" +
		merge.SectionBeginLine("t2") + "\nnew2\n" + merge.EndMarker + "\n"
	if got := string(files[0].Content); got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}
