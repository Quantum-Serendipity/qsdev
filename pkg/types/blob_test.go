package types_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// blobHolder mirrors how FileState embeds a Blob.
type blobHolder struct {
	Content types.Blob `yaml:"content,omitempty" json:"content,omitempty"`
}

// blobSamples cover the scalar styles yaml.v3 can choose and the content it
// must not normalise: line-break kinds, leading and trailing whitespace,
// YAML indicators, control bytes and invalid UTF-8.
var blobSamples = []string{
	"",
	"x",
	`{"permissions": {"allow": ["Read"]}}`,
	"line one\nline two\n",
	"no trailing newline\nsecond",
	"trailing blank lines\n\n\n",
	"\n\nleading blank lines",
	"  indented first line\nnext\n",
	"trailing spaces  \nnext  \n",
	"crlf\r\nline\r\n",
	"lone cr\rhere",
	"tab\tseparated\n\tindented\n",
	"- looks like a list\n- item\n",
	"key: value\n# comment\n",
	"null",
	"~",
	"true",
	"0x1F",
	"'quoted' \"both\"",
	"\ufeffbom first",
	"nul\x00byte",
	"\u0085nel and \u2028ls",
	"\xff\xfe invalid utf-8",
	"emoji \U0001F600\n",
}

func TestBlobYAMLRoundTrip(t *testing.T) {
	t.Parallel()
	for _, s := range blobSamples {
		t.Run(strings.ToValidUTF8(s, "?"), func(t *testing.T) {
			t.Parallel()
			assertBlobRoundTrip(t, []byte(s))
		})
	}
}

func FuzzBlobYAMLRoundTrip(f *testing.F) {
	for _, s := range blobSamples {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		assertBlobRoundTrip(t, b)
	})
}

func assertBlobRoundTrip(t *testing.T, want []byte) {
	t.Helper()
	data, err := yaml.Marshal(blobHolder{Content: want})
	if err != nil {
		t.Fatalf("yaml.Marshal: %v", err)
	}
	var got blobHolder
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("yaml.Unmarshal(%q): %v", data, err)
	}
	if !bytes.Equal(got.Content, want) {
		t.Fatalf("round trip of %q via %q = %q", want, data, got.Content)
	}
}

// TestBlobYAMLScalarForm pins the encoding: one base64 !!binary scalar,
// wrapped at 76 columns, never the per-byte integer sequence a plain []byte
// produces.
func TestBlobYAMLScalarForm(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("a", 60) // 80 base64 characters: one full line and a short one
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{name: "short", content: `{"a":1}`, want: "content: !!binary eyJhIjoxfQ==\n"},
		{name: "invalid utf-8", content: "\xff\x00", want: "content: !!binary /wA=\n"},
		{
			name:    "long content wraps in a literal block",
			content: long,
			want:    "content: !!binary |\n    " + strings.Repeat("YWFh", 19) + "\n    YWFh\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := yaml.Marshal(blobHolder{Content: types.Blob(tt.content)})
			if err != nil {
				t.Fatalf("yaml.Marshal: %v", err)
			}
			if string(data) != tt.want {
				t.Errorf("yaml.Marshal = %q, want %q", data, tt.want)
			}
		})
	}
}

func TestBlobYAMLDecode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		doc     string
		want    types.Blob
		wantErr bool
	}{
		{name: "legacy integer list", doc: "content:\n    - 123\n    - 10\n    - 125\n", want: types.Blob("{\n}")},
		{name: "legacy flow list", doc: "content: [104, 105]\n", want: types.Blob("hi")},
		{name: "plain string", doc: "content: hi\n", want: types.Blob("hi")},
		{name: "wrapped binary", doc: "content: !!binary |\n    aG\n    k=\n", want: types.Blob("hi")},
		{name: "binary", doc: "content: !!binary aGk=\n", want: types.Blob("hi")},
		{name: "null", doc: "content: null\n", want: nil},
		{name: "absent", doc: "{}\n", want: nil},
		{name: "mapping is rejected", doc: "content: {a: 1}\n", wantErr: true},
		{name: "out of range byte is rejected", doc: "content: [300]\n", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got blobHolder
			err := yaml.Unmarshal([]byte(tt.doc), &got)
			if (err != nil) != tt.wantErr {
				t.Fatalf("yaml.Unmarshal error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if !bytes.Equal(got.Content, tt.want) || (got.Content == nil) != (tt.want == nil) {
				t.Errorf("Content = %q (nil %t), want %q (nil %t)", got.Content, got.Content == nil, tt.want, tt.want == nil)
			}
		})
	}
}

// TestBlobJSONIsBase64 pins that the JSON form is unchanged by the YAML
// methods: encoding/json still encodes the []byte underlying type as base64.
func TestBlobJSONIsBase64(t *testing.T) {
	t.Parallel()
	data, err := json.Marshal(blobHolder{Content: types.Blob("hi")})
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if string(data) != `{"content":"aGk="}` {
		t.Errorf("json.Marshal = %s, want base64", data)
	}
	var got blobHolder
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if string(got.Content) != "hi" {
		t.Errorf("json round trip = %q, want %q", got.Content, "hi")
	}
}
