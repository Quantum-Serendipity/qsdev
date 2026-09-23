package merge

import (
	"bytes"
	"errors"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestSectionMarkers_BasicMerge(t *testing.T) {
	existing := []byte("# CLAUDE.md\n\n<!-- BEGIN GENERATED SECTION — do not edit -->\n\nOld generated content.\n\n<!-- END GENERATED SECTION -->\n\n## My Custom Notes\n\nI added this myself.\n")

	newGenerated := []byte("# CLAUDE.md\n\n<!-- BEGIN GENERATED SECTION — do not edit -->\n\nNew generated content with more stuff.\n\n<!-- END GENERATED SECTION -->\n\n## Custom Instructions\n\nAdd your instructions here.\n")

	want := []byte("# CLAUDE.md\n\n<!-- BEGIN GENERATED SECTION — do not edit -->\n\nNew generated content with more stuff.\n\n<!-- END GENERATED SECTION -->\n\n## My Custom Notes\n\nI added this myself.\n")

	got, err := SectionMarkers(existing, newGenerated)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestSectionMarkers_PreservesContentBeforeMarkers(t *testing.T) {
	existing := []byte("# My Project\n\nCustom header added by user.\n\n<!-- BEGIN GENERATED SECTION -->\nold stuff\n<!-- END GENERATED SECTION -->\n")

	newGenerated := []byte("<!-- BEGIN GENERATED SECTION -->\nnew stuff\n<!-- END GENERATED SECTION -->\n")

	want := []byte("# My Project\n\nCustom header added by user.\n\n<!-- BEGIN GENERATED SECTION -->\nnew stuff\n<!-- END GENERATED SECTION -->\n")

	got, err := SectionMarkers(existing, newGenerated)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestSectionMarkers_PreservesContentAfterMarkers(t *testing.T) {
	existing := []byte("<!-- BEGIN GENERATED SECTION -->\nold\n<!-- END GENERATED SECTION -->\n\nUser notes below.\nDo not remove.\n")

	newGenerated := []byte("<!-- BEGIN GENERATED SECTION -->\nnew\n<!-- END GENERATED SECTION -->\n")

	want := []byte("<!-- BEGIN GENERATED SECTION -->\nnew\n<!-- END GENERATED SECTION -->\n\nUser notes below.\nDo not remove.\n")

	got, err := SectionMarkers(existing, newGenerated)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestSectionMarkers_MarkersNotFound(t *testing.T) {
	existing := []byte("# CLAUDE.md\n\nNo markers here.\n")
	newGenerated := []byte("<!-- BEGIN GENERATED SECTION -->\nstuff\n<!-- END GENERATED SECTION -->\n")

	_, err := SectionMarkers(existing, newGenerated)
	if !errors.Is(err, ErrMarkersNotFound) {
		t.Errorf("expected ErrMarkersNotFound, got %v", err)
	}
}

func TestSectionMarkersOrAppend(t *testing.T) {
	gen := []byte("# CLAUDE.md\n\n<!-- BEGIN GENERATED SECTION -->\nnew\n<!-- END GENERATED SECTION -->\n\n## Custom\n")
	block := "<!-- BEGIN GENERATED SECTION -->\nnew\n<!-- END GENERATED SECTION -->\n"

	tests := []struct {
		name     string
		existing string
		want     string
		wantErr  error
	}{
		{
			name:     "handwritten_file_is_kept_and_block_appended",
			existing: "# my handwritten notes\n",
			want:     "# my handwritten notes\n\n" + block,
		},
		{
			name:     "no_trailing_newline",
			existing: "notes",
			want:     "notes\n\n" + block,
		},
		{
			name:     "already_blank_line_separated",
			existing: "notes\n\n",
			want:     "notes\n\n" + block,
		},
		{
			name:     "empty_existing",
			existing: "",
			want:     block,
		},
		{
			name:     "existing_markers_are_replaced_not_appended",
			existing: "top\n<!-- BEGIN GENERATED SECTION -->\nold\n<!-- END GENERATED SECTION -->\nbottom\n",
			want:     "top\n" + block + "bottom\n",
		},
		{
			name:     "malformed_markers_still_error",
			existing: "<!-- BEGIN GENERATED SECTION -->\nuser text\n",
			wantErr:  ErrMalformedMarkers,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SectionMarkersOrAppend([]byte(tt.existing), gen)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("got:\n%q\nwant:\n%q", got, tt.want)
			}
			// Idempotent: a second run replaces the appended section in place.
			again, err := SectionMarkersOrAppend(got, gen)
			if err != nil {
				t.Fatalf("second run: %v", err)
			}
			if !bytes.Equal(again, got) {
				t.Errorf("second run changed content:\n%q\nwant:\n%q", again, got)
			}
		})
	}
}

func TestDispatch_SectionMarkerAppendsToHandwrittenFile(t *testing.T) {
	existing := []byte("# my handwritten notes\n")
	gen := []byte("<!-- BEGIN GENERATED SECTION -->\ngen\n<!-- END GENERATED SECTION -->\n")
	got, err := Dispatch("CLAUDE.md", types.SectionMarker, nil, existing, gen)
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if !bytes.HasPrefix(got, existing) {
		t.Errorf("user content lost: %q", got)
	}
	if !bytes.Contains(got, gen) {
		t.Errorf("generated section missing: %q", got)
	}
}

func TestSectionMarkers_BeginWithoutEnd(t *testing.T) {
	existing := []byte("<!-- BEGIN GENERATED SECTION -->\nstuff here\n")
	newGenerated := []byte("<!-- BEGIN GENERATED SECTION -->\nstuff\n<!-- END GENERATED SECTION -->\n")

	_, err := SectionMarkers(existing, newGenerated)
	if !errors.Is(err, ErrMalformedMarkers) {
		t.Errorf("expected ErrMalformedMarkers, got %v", err)
	}
}

func TestSectionMarkers_EndWithoutBegin(t *testing.T) {
	existing := []byte("some text\n<!-- END GENERATED SECTION -->\n")
	newGenerated := []byte("<!-- BEGIN GENERATED SECTION -->\nstuff\n<!-- END GENERATED SECTION -->\n")

	_, err := SectionMarkers(existing, newGenerated)
	if !errors.Is(err, ErrMalformedMarkers) {
		t.Errorf("expected ErrMalformedMarkers, got %v", err)
	}
}

func TestSectionMarkers_NewGeneratedMissingMarkers(t *testing.T) {
	existing := []byte("<!-- BEGIN GENERATED SECTION -->\nold\n<!-- END GENERATED SECTION -->\n")
	newGenerated := []byte("No markers in the new content.\n")

	_, err := SectionMarkers(existing, newGenerated)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSectionMarkers_EmptyContentBetweenMarkers(t *testing.T) {
	existing := []byte("header\n<!-- BEGIN GENERATED SECTION -->\n<!-- END GENERATED SECTION -->\nfooter\n")

	newGenerated := []byte("<!-- BEGIN GENERATED SECTION -->\nnew content\n<!-- END GENERATED SECTION -->\n")

	want := []byte("header\n<!-- BEGIN GENERATED SECTION -->\nnew content\n<!-- END GENERATED SECTION -->\nfooter\n")

	got, err := SectionMarkers(existing, newGenerated)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestSectionMarkers_EmptyAfterEndMarker(t *testing.T) {
	existing := []byte("<!-- BEGIN GENERATED SECTION -->\nold\n<!-- END GENERATED SECTION -->\n")

	newGenerated := []byte("<!-- BEGIN GENERATED SECTION -->\nnew\n<!-- END GENERATED SECTION -->\n")

	want := []byte("<!-- BEGIN GENERATED SECTION -->\nnew\n<!-- END GENERATED SECTION -->\n")

	got, err := SectionMarkers(existing, newGenerated)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestSectionMarkers_IdenticalContent(t *testing.T) {
	content := []byte("# Title\n\n<!-- BEGIN GENERATED SECTION -->\ngenerated\n<!-- END GENERATED SECTION -->\n\nuser notes\n")

	got, err := SectionMarkers(content, content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("expected idempotent result\ngot:\n%s\nwant:\n%s", got, content)
	}
}

func TestSectionMarkers_AnnotationVariation(t *testing.T) {
	existing := []byte("<!-- BEGIN GENERATED SECTION — do not edit between markers -->\nold\n<!-- END GENERATED SECTION -->\n")

	newGenerated := []byte("<!-- BEGIN GENERATED SECTION — auto-generated -->\nnew\n<!-- END GENERATED SECTION -->\n")

	want := []byte("<!-- BEGIN GENERATED SECTION — auto-generated -->\nnew\n<!-- END GENERATED SECTION -->\n")

	got, err := SectionMarkers(existing, newGenerated)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestSectionMarkers_NoTrailingNewline(t *testing.T) {
	existing := []byte("header\n<!-- BEGIN GENERATED SECTION -->\nold\n<!-- END GENERATED SECTION -->")

	newGenerated := []byte("<!-- BEGIN GENERATED SECTION -->\nnew\n<!-- END GENERATED SECTION -->")

	want := []byte("header\n<!-- BEGIN GENERATED SECTION -->\nnew\n<!-- END GENERATED SECTION -->")

	got, err := SectionMarkers(existing, newGenerated)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("mismatch\ngot:\n%q\nwant:\n%q", got, want)
	}
}

func TestIndexLinePrefix_Found(t *testing.T) {
	data := []byte("line one\nline two\n<!-- BEGIN GENERATED SECTION -->\nline four\n")
	idx := indexLinePrefix(data, []byte(BeginMarkerPrefix))
	if idx < 0 {
		t.Fatal("expected to find prefix, got -1")
	}
	// Should point to the start of the line containing the prefix.
	if !bytes.HasPrefix(data[idx:], []byte(BeginMarkerPrefix)) {
		t.Errorf("offset %d does not start with prefix; got %q", idx, data[idx:idx+30])
	}
}

func TestIndexLinePrefix_FoundAtStart(t *testing.T) {
	data := []byte("<!-- BEGIN GENERATED SECTION -->\nrest\n")
	idx := indexLinePrefix(data, []byte(BeginMarkerPrefix))
	if idx != 0 {
		t.Errorf("expected offset 0, got %d", idx)
	}
}

func TestIndexLinePrefix_NotFound(t *testing.T) {
	data := []byte("line one\nline two\nno marker here\n")
	idx := indexLinePrefix(data, []byte(BeginMarkerPrefix))
	if idx != -1 {
		t.Errorf("expected -1, got %d", idx)
	}
}

func TestRemoveSection(t *testing.T) {
	t.Parallel()
	const block = BeginMarkerPrefix + " -->\ngenerated\n" + EndMarker + "\n"
	tests := []struct {
		name, in, want string
		wantErr        bool
	}{
		{"only the block", block, "", false},
		{"scaffold title kept for the caller", "# CLAUDE.md\n\n" + block, "# CLAUDE.md\n", false},
		{"user text before, appended block", "# Mine\n\nnotes\n\n" + block, "# Mine\n\nnotes\n", false},
		{"user text after", block + "after\n", "after\n", false},
		{"no markers", "plain\n", "plain\n", false},
		{"end before begin", EndMarker + "\n" + BeginMarkerPrefix + " -->\n", "", true},
		{"begin without end", BeginMarkerPrefix + " -->\nx\n", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := RemoveSection([]byte(tt.in))
			if (err != nil) != tt.wantErr {
				t.Fatalf("RemoveSection error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && string(got) != tt.want {
				t.Errorf("RemoveSection = %q, want %q", got, tt.want)
			}
		})
	}
}
