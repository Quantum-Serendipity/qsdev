package toolutil_test

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools/toolutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestConfineToRoot covers the shared path-containment primitive: lexical
// traversal rejection plus the symlink-aware check that an in-root symlink whose
// target escapes the root is rejected while a not-yet-created in-root path and an
// in-root symlink staying inside are both allowed.
func TestConfineToRoot(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	t.Run("lexical", func(t *testing.T) {
		t.Parallel()
		// A sibling temp dir is genuinely absolute-and-outside on every platform.
		outsideAbs := filepath.Join(t.TempDir(), "go.sum")
		cases := []struct {
			name      string
			candidate string
			wantOK    bool
		}{
			{"in-root relative", "go.sum", true},
			{"in-root nested relative", "sub/dir/go.sum", true},
			{"root itself", ".", true},
			{"relative escape", "../../../etc/passwd", false},
			{"sneaky middle escape", "sub/../../outside/go.sum", false},
			{"absolute outside root", outsideAbs, false},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()
				if _, ok := toolutil.ConfineToRoot(root, c.candidate); ok != c.wantOK {
					t.Errorf("ConfineToRoot(root, %q) ok = %v, want %v", c.candidate, ok, c.wantOK)
				}
			})
		}
	})

	t.Run("symlink", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("symlink creation requires privilege on Windows")
		}
		linkRoot := t.TempDir()

		// An in-root symlink whose target escapes the root must be rejected even
		// though the link path is lexically inside root.
		if err := os.Symlink(t.TempDir(), filepath.Join(linkRoot, "escape")); err != nil {
			t.Fatalf("symlink: %v", err)
		}
		if _, ok := toolutil.ConfineToRoot(linkRoot, "escape/go.sum"); ok {
			t.Error("escape/go.sum resolves outside root via symlink; want rejection")
		}

		// An in-root symlink whose target stays inside the root is allowed.
		inside := filepath.Join(linkRoot, "real")
		if err := os.Mkdir(inside, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.Symlink(inside, filepath.Join(linkRoot, "alias")); err != nil {
			t.Fatalf("symlink: %v", err)
		}
		if _, ok := toolutil.ConfineToRoot(linkRoot, "alias/go.sum"); !ok {
			t.Error("alias/go.sum stays inside root via symlink; want allow")
		}

		// A not-yet-created in-root path must still pass (lexical leaf).
		if _, ok := toolutil.ConfineToRoot(linkRoot, "does/not/exist/go.sum"); !ok {
			t.Error("a not-yet-created in-root path must be allowed")
		}
	})
}

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
