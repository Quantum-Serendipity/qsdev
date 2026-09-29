package merge

import (
	"errors"
	"testing"
)

func TestReplaceTaggedSection(t *testing.T) {
	t.Parallel()

	section := func(tag, body string) string {
		return SectionBeginLine(tag) + "\n" + body + "\n" + EndMarker + "\n"
	}

	tests := []struct {
		name     string
		existing string
		tag      string
		block    string
		want     string
		wantErr  error
	}{
		{
			name:     "replaces only the matching tag",
			existing: "# base\n" + section("t1", "AAA") + "middle\n" + section("t2", "BBB") + "tail\n",
			tag:      "t2",
			block:    section("t2", "NEW"),
			want:     "# base\n" + section("t1", "AAA") + "middle\n" + section("t2", "NEW") + "tail\n",
		},
		{
			name:     "first section untouched when replacing it",
			existing: section("t1", "AAA") + section("t2", "BBB"),
			tag:      "t1",
			block:    section("t1", "X"),
			want:     section("t1", "X") + section("t2", "BBB"),
		},
		{
			name:     "tag is matched exactly, not by prefix",
			existing: section("t10", "AAA"),
			tag:      "t1",
			wantErr:  ErrMarkersNotFound,
		},
		{
			name:     "missing tag",
			existing: "# base\n",
			tag:      "t1",
			wantErr:  ErrMarkersNotFound,
		},
		{
			name:     "begin without end",
			existing: SectionBeginLine("t1") + "\nbody\n",
			tag:      "t1",
			wantErr:  ErrMalformedMarkers,
		},
		{
			name:     "CRLF line endings",
			existing: SectionBeginLine("t1") + "\r\nold\r\n" + EndMarker + "\r\nrest\r\n",
			tag:      "t1",
			block:    section("t1", "new"),
			want:     section("t1", "new") + "rest\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ReplaceTaggedSection([]byte(tt.existing), tt.tag, []byte(tt.block))
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("got:\n%q\nwant:\n%q", got, tt.want)
			}
		})
	}
}

func TestSectionBeginLine(t *testing.T) {
	t.Parallel()

	if got, want := SectionBeginLine("hooks"), "<!-- BEGIN GENERATED SECTION — hooks -->"; got != want {
		t.Errorf("SectionBeginLine = %q, want %q", got, want)
	}
}
