package generate

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestWriteGeneratedFile covers the validated writer the non-WriteFiles
// commands (update, enable/disable, repair, auto-fix, claude subcommands) use:
// content WriteFiles would reject must not be written by them either.
func TestWriteGeneratedFile(t *testing.T) {
	t.Parallel()
	const existing = `{"ok": true}`
	tests := []struct {
		name        string
		file        types.GeneratedFile
		wantInvalid bool
		wantContent string
	}{
		{"valid json is written", types.GeneratedFile{Path: "cfg/a.json", Content: []byte(`{"a": 1}`)}, false, `{"a": 1}`},
		{"invalid json is refused", types.GeneratedFile{Path: "cfg/a.json", Content: []byte(`{"a": `)}, true, existing},
		{"invalid yaml is refused", types.GeneratedFile{Path: "cfg/a.yaml", Content: []byte("a: [1\n")}, true, ""},
		{"skip validation is honored", types.GeneratedFile{Path: "cfg/a.json", Content: []byte(`{`), SkipValidation: true}, false, `{`},
		{"unvalidated type is written", types.GeneratedFile{Path: "cfg/a.toml", Content: []byte(`x = [`)}, false, `x = [`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			target := filepath.Join(root, filepath.FromSlash(tt.file.Path))
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				t.Fatal(err)
			}
			if filepath.Ext(target) == ".json" {
				if err := os.WriteFile(target, []byte(existing), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			err := WriteGeneratedFile(root, tt.file)
			if got := errors.Is(err, ErrInvalidContent); got != tt.wantInvalid {
				t.Fatalf("err = %v, want invalid=%v", err, tt.wantInvalid)
			}
			if !tt.wantInvalid && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			data, readErr := os.ReadFile(target)
			if tt.wantContent == "" {
				if !os.IsNotExist(readErr) {
					t.Errorf("refused file was created (err=%v)", readErr)
				}
				return
			}
			if string(data) != tt.wantContent {
				t.Errorf("content = %q, want %q", data, tt.wantContent)
			}
		})
	}
}
