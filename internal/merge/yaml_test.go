package merge

import (
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// composeBase is a generated docker-compose fragment as recorded in state.
const composeBase = `services:
  gateway:
    image: example/gw:1.0.0
    ports:
      - "8765:8765"
    read_only: true
    restart: unless-stopped
`

func decodeYAML(t *testing.T, data []byte) any {
	t.Helper()
	var v any
	if err := yaml.Unmarshal(data, &v); err != nil {
		t.Fatalf("result is not valid YAML: %v\n%s", err, data)
	}
	return v
}

func TestMergeYAML(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		base   string
		theirs string
		ours   string
		want   string
	}{
		{
			name:   "unmodified file takes the generator's changes",
			base:   composeBase,
			theirs: composeBase,
			ours:   strings.Replace(composeBase, "gw:1.0.0", "gw:1.1.0", 1),
			want:   strings.Replace(composeBase, "gw:1.0.0", "gw:1.1.0", 1),
		},
		{
			name:   "user edit to one key survives a generator change to another",
			base:   composeBase,
			theirs: strings.Replace(composeBase, `"8765:8765"`, `"9000:8765"`, 1),
			ours:   strings.Replace(composeBase, "gw:1.0.0", "gw:1.1.0", 1),
			want: strings.Replace(strings.Replace(composeBase, `"8765:8765"`, `"9000:8765"`, 1),
				"gw:1.0.0", "gw:1.1.0", 1),
		},
		{
			name:   "conflicting edit keeps the user's value",
			base:   composeBase,
			theirs: strings.Replace(composeBase, "gw:1.0.0", "mirror/gw:pinned", 1),
			ours:   strings.Replace(composeBase, "gw:1.0.0", "gw:1.1.0", 1),
			want:   strings.Replace(composeBase, "gw:1.0.0", "mirror/gw:pinned", 1),
		},
		{
			name:   "user-added keys are kept and new generated keys added",
			base:   composeBase,
			theirs: composeBase + "    mem_limit: 512m\n",
			ours:   composeBase + "    security_opt:\n      - no-new-privileges:true\n",
			want:   composeBase + "    mem_limit: 512m\n    security_opt:\n      - no-new-privileges:true\n",
		},
		{
			name:   "a generated key the user deleted stays deleted",
			base:   composeBase,
			theirs: strings.Replace(composeBase, "    restart: unless-stopped\n", "", 1),
			ours:   strings.Replace(composeBase, "gw:1.0.0", "gw:1.1.0", 1),
			want: strings.Replace(strings.Replace(composeBase, "    restart: unless-stopped\n", "", 1),
				"gw:1.0.0", "gw:1.1.0", 1),
		},
		{
			name:   "a key the generator dropped is removed when the user left it alone",
			base:   composeBase,
			theirs: composeBase,
			ours:   strings.Replace(composeBase, "    restart: unless-stopped\n", "", 1),
			want:   strings.Replace(composeBase, "    restart: unless-stopped\n", "", 1),
		},
		{
			name:   "a key the generator dropped is kept when the user changed it",
			base:   composeBase,
			theirs: strings.Replace(composeBase, "unless-stopped", "always", 1),
			ours:   strings.Replace(composeBase, "    restart: unless-stopped\n", "", 1),
			want:   strings.Replace(composeBase, "unless-stopped", "always", 1),
		},
		{
			name:   "without a recorded base the existing file wins conflicts",
			base:   "",
			theirs: strings.Replace(composeBase, `"8765:8765"`, `"9000:8765"`, 1),
			ours:   composeBase + "    security_opt:\n      - no-new-privileges:true\n",
			want: strings.Replace(composeBase, `"8765:8765"`, `"9000:8765"`, 1) +
				"    security_opt:\n      - no-new-privileges:true\n",
		},
		{
			name:   "sequences merge as a whole value",
			base:   composeBase,
			theirs: strings.Replace(composeBase, `      - "8765:8765"`, "      - \"8765:8765\"\n      - \"9443:9443\"", 1),
			ours:   strings.Replace(composeBase, `"8765:8765"`, `"8800:8800"`, 1),
			want:   strings.Replace(composeBase, `      - "8765:8765"`, "      - \"8765:8765\"\n      - \"9443:9443\"", 1),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var base []byte
			if tt.base != "" {
				base = []byte(tt.base)
			}
			got, err := MergeYAML(base, []byte(tt.theirs), []byte(tt.ours))
			if err != nil {
				t.Fatalf("MergeYAML: %v", err)
			}
			if g, w := decodeYAML(t, got), decodeYAML(t, []byte(tt.want)); !reflect.DeepEqual(g, w) {
				t.Errorf("merged = %v\nwant     %v\nraw:\n%s", g, w, got)
			}
		})
	}
}

func TestMergeYAML_KeepsUserComments(t *testing.T) {
	t.Parallel()
	theirs := "# my gateway overrides\n" + strings.Replace(composeBase, `"8765:8765"`, `"9000:8765" # host port clash`, 1)
	got, err := MergeYAML([]byte(composeBase), []byte(theirs), []byte(strings.Replace(composeBase, "gw:1.0.0", "gw:1.1.0", 1)))
	if err != nil {
		t.Fatalf("MergeYAML: %v", err)
	}
	for _, want := range []string{"# my gateway overrides", "# host port clash", "gw:1.1.0"} {
		if !strings.Contains(string(got), want) {
			t.Errorf("merged output lost %q:\n%s", want, got)
		}
	}
}

func TestMergeYAML_Errors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name               string
		base, theirs, ours string
	}{
		{"invalid theirs", composeBase, "services: [\n", composeBase},
		{"invalid ours", composeBase, composeBase, "services: [\n"},
		{"invalid base", "services: [\n", composeBase, composeBase},
		{"non-mapping theirs", composeBase, "- a\n- b\n", composeBase},
		{"non-mapping ours", composeBase, composeBase, "just a string\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := MergeYAML([]byte(tt.base), []byte(tt.theirs), []byte(tt.ours)); err == nil {
				t.Error("expected an error, got nil")
			}
		})
	}
}

func TestDispatch_RoutesYAMLThreeWayMerge(t *testing.T) {
	t.Parallel()
	for _, path := range []string{"docker-compose.gateway.yaml", "sub/compose.yml"} {
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			theirs := strings.Replace(composeBase, `"8765:8765"`, `"9000:8765"`, 1)
			got, err := Dispatch(path, types.ThreeWayMerge, []byte(composeBase), []byte(theirs), []byte(composeBase))
			if err != nil {
				t.Fatalf("Dispatch: %v", err)
			}
			if !strings.Contains(string(got), "9000:8765") {
				t.Errorf("user edit lost:\n%s", got)
			}
		})
	}
}

func TestMergeYAML_UnmodifiedFileTakesGeneratedBytes(t *testing.T) {
	t.Parallel()
	ours := "# generated\n" + strings.Replace(composeBase, "gw:1.0.0", "gw:1.1.0", 1)
	got, err := MergeYAML([]byte(composeBase), []byte(composeBase), []byte(ours))
	if err != nil {
		t.Fatalf("MergeYAML: %v", err)
	}
	if string(got) != ours {
		t.Errorf("unmodified file = %q, want the generated content verbatim %q", got, ours)
	}
}

func TestMergeYAML_KeepsAnchorsOnReplacedValues(t *testing.T) {
	t.Parallel()
	theirs := strings.Replace(composeBase, "image: example/gw:1.0.0", "image: &img example/gw:1.0.0", 1) +
		"  sidecar:\n    image: *img\n"
	got, err := MergeYAML([]byte(composeBase), []byte(theirs), []byte(strings.Replace(composeBase, "gw:1.0.0", "gw:1.1.0", 1)))
	if err != nil {
		t.Fatalf("MergeYAML: %v", err)
	}
	want := map[string]any{"services": map[string]any{
		"gateway": map[string]any{"image": "example/gw:1.1.0", "ports": []any{"8765:8765"}, "read_only": true, "restart": "unless-stopped"},
		"sidecar": map[string]any{"image": "example/gw:1.1.0"},
	}}
	if g := decodeYAML(t, got); !reflect.DeepEqual(g, want) {
		t.Errorf("merged = %v\nwant     %v\nraw:\n%s", g, want, got)
	}
}

func TestMergeYAML_RejectsUnsafeResults(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name               string
		base, theirs, ours string
	}{
		{
			// Re-marshaling only the first document would drop the rest.
			name:   "multi-document file",
			base:   composeBase,
			theirs: strings.Replace(composeBase, "gw:1.0.0", "gw:2", 1) + "---\nextra: doc\n",
			ours:   composeBase,
		},
		{
			// The dropped generated key carried an anchor the user aliased.
			name:   "alias left dangling",
			base:   composeBase,
			theirs: strings.Replace(composeBase, "restart: unless-stopped", "restart: &r unless-stopped", 1) + "  sidecar:\n    restart: *r\n",
			ours:   strings.Replace(composeBase, "    restart: unless-stopped\n", "", 1),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got, err := MergeYAML([]byte(tt.base), []byte(tt.theirs), []byte(tt.ours)); err == nil {
				t.Errorf("expected an error, got nil and:\n%s", got)
			}
		})
	}
}
