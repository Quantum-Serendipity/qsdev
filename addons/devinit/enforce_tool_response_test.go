package devinit

import (
	"encoding/json"
	"strings"
	"testing"
)

// bracket is a stand-in hardening transform whose output json.Marshal leaves
// unescaped, so the expected responses stay readable.
func bracket(s string) string { return "[" + s + "]" }

func TestHardenToolResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		raw         string
		harden      func(string) string
		wantChanged bool
		want        string
	}{
		{
			name:        "string",
			raw:         `"hello"`,
			wantChanged: true,
			want:        `"[hello]"`,
		},
		{
			// F291: each text block is hardened on its own and stays where it
			// was, so a JSON block is never merged into a neighbouring one.
			name:        "every text block hardened in place",
			raw:         `[{"type":"text","text":"a"},{"type":"image","data":"AA==","mimeType":"image/png"},{"type":"text","text":"b"}]`,
			wantChanged: true,
			want:        `[{"text":"[a]","type":"text"},{"data":"AA==","mimeType":"image/png","type":"image"},{"text":"[b]","type":"text"}]`,
		},
		{
			name:        "text block keeps its other fields",
			raw:         `[{"type":"text","text":"a","annotations":{"audience":["user"]}}]`,
			wantChanged: true,
			want:        `[{"annotations":{"audience":["user"]},"text":"[a]","type":"text"}]`,
		},
		{
			name:        "result object keeps structuredContent",
			raw:         `{"content":[{"type":"text","text":"a"}],"isError":false,"structuredContent":{"k":"v w"}}`,
			wantChanged: true,
			want:        `{"content":[{"text":"[a]","type":"text"}],"isError":false,"structuredContent":{"k":"v w"}}`,
		},
		{
			name:        "single text block",
			raw:         `{"type":"text","text":"a","_meta":{"x":1}}`,
			wantChanged: true,
			want:        `{"_meta":{"x":1},"text":"[a]","type":"text"}`,
		},
		{
			name: "image-only content has no text",
			raw:  `[{"type":"image","data":"AA==","mimeType":"image/png"}]`,
		},
		{
			name:        "unknown object is hardened as its JSON",
			raw:         `{"rows":[1,2]}`,
			wantChanged: true,
			want:        `"[{\"rows\":[1,2]}]"`,
		},
		{
			name:        "array of plain rows is not a content array",
			raw:         `[{"id":1,"name":"x"},{"type":"event"}]`,
			wantChanged: true,
			want:        `"[[{\"id\":1,\"name\":\"x\"},{\"type\":\"event\"}]]"`,
		},
		{
			// A text block whose text is not a string is not a content
			// block, so the response is hardened as JSON instead of the
			// block being skipped unhardened.
			name:        "malformed text block is hardened as JSON",
			raw:         `[{"type":"text","text":{"x":"y"}}]`,
			wantChanged: true,
			want:        `"[[{\"type\":\"text\",\"text\":{\"x\":\"y\"}}]]"`,
		},
		{
			name:   "unchanged text reports no change",
			raw:    `[{"type":"text","text":"a"}]`,
			harden: func(s string) string { return s },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			harden := tt.harden
			if harden == nil {
				harden = bracket
			}
			got, changed, err := hardenToolResponse(json.RawMessage(tt.raw), harden)
			if err != nil {
				t.Fatalf("hardenToolResponse: %v", err)
			}
			if changed != tt.wantChanged {
				t.Fatalf("changed = %v, want %v (output %s)", changed, tt.wantChanged, got)
			}
			if changed && string(got) != tt.want {
				t.Errorf("hardenToolResponse =\n %s\nwant\n %s", got, tt.want)
			}
		})
	}
}

// TestHardenToolResponse_TextBlocksHardenedSeparately checks harden sees each
// text block's own text, never several blocks joined together.
func TestHardenToolResponse_TextBlocksHardenedSeparately(t *testing.T) {
	t.Parallel()

	var seen []string
	harden := func(s string) string {
		seen = append(seen, s)
		return strings.ToUpper(s)
	}
	raw := `[{"type":"text","text":"{\"a\": 1}"},{"type":"text","text":"prose"}]`
	if _, _, err := hardenToolResponse(json.RawMessage(raw), harden); err != nil {
		t.Fatalf("hardenToolResponse: %v", err)
	}
	if want := []string{`{"a": 1}`, "prose"}; strings.Join(seen, "|") != strings.Join(want, "|") {
		t.Errorf("harden saw %q, want %q", seen, want)
	}
}
