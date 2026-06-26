package toolutil_test

import (
	"reflect"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools/toolutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestDetectedLanguages proves the single shared detection->language-labels
// helper reports the COMPLETE language surface — including java (maven),
// java/kotlin (gradle), and dotnet, the labels the old per-adapter copies were
// missing — and appends detected versions.
func TestDetectedLanguages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   types.DetectedProject
		want []string
	}{
		{
			name: "empty project has no languages",
			in:   types.DetectedProject{},
			want: nil,
		},
		{
			name: "go with version is suffixed",
			in:   types.DetectedProject{HasGoMod: true, GoVersion: "1.22"},
			want: []string{"go 1.22"},
		},
		{
			name: "go without version is bare",
			in:   types.DetectedProject{HasGoMod: true},
			want: []string{"go"},
		},
		{
			name: "jvm and dotnet are included (the previously-missing labels)",
			in:   types.DetectedProject{HasPomXML: true, HasBuildGradle: true, HasCsproj: true},
			want: []string{"java (maven)", "java/kotlin (gradle)", "dotnet"},
		},
		{
			name: "full polyglot surface in canonical order",
			in: types.DetectedProject{
				HasGoMod: true, GoVersion: "1.22",
				HasPackageJSON: true, NodeVersion: "20",
				HasCargoToml: true,
				HasPyProject: true, PythonVersion: "3.12",
				HasPomXML: true, HasBuildGradle: true, HasCsproj: true,
			},
			want: []string{
				"go 1.22", "node 20", "rust", "python 3.12",
				"java (maven)", "java/kotlin (gradle)", "dotnet",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := toolutil.DetectedLanguages(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DetectedLanguages() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestEmptyObjectSchema proves the shared no-argument schema is a permissive
// empty object and that a fresh, non-shared map is returned per call.
func TestEmptyObjectSchema(t *testing.T) {
	t.Parallel()

	got := toolutil.EmptyObjectSchema()
	if got["type"] != "object" {
		t.Errorf("type = %v, want object", got["type"])
	}
	props, ok := got["properties"].(map[string]any)
	if !ok || len(props) != 0 {
		t.Errorf("properties = %v, want empty map", got["properties"])
	}

	// Mutating one result must not affect a subsequent call.
	got["type"] = "tampered"
	if fresh := toolutil.EmptyObjectSchema(); fresh["type"] != "object" {
		t.Errorf("EmptyObjectSchema returned a shared map; second call type = %v", fresh["type"])
	}
}
