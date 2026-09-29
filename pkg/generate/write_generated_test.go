package generate

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
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

// TestWriteGeneratedFile_RefusesSymlinkEscape verifies the writer used by
// update, enable/disable, repair, auto-fix and the claude subcommands never
// follows a committed symlink (a symlinked .claude directory or file) to a
// location outside the project root.
func TestWriteGeneratedFile_RefusesSymlinkEscape(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privileges on Windows")
	}
	tests := []struct {
		name string
		link string // project-relative path made a symlink to outside
		path string // generated file path
	}{
		{"symlinked parent directory", ".claude", ".claude/settings.json"},
		{"symlinked parent, missing subdirectory", ".claude", ".claude/hooks/guard.sh"},
		{"symlinked file", "CLAUDE.md", "CLAUDE.md"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			outside := t.TempDir()
			target := outside
			if tt.link == tt.path {
				target = filepath.Join(outside, "global.md")
				if err := os.WriteFile(target, []byte("user content"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.Symlink(target, filepath.Join(root, tt.link)); err != nil {
				t.Fatal(err)
			}

			err := WriteGeneratedFile(root, types.GeneratedFile{Path: tt.path, Content: []byte("generated"), SkipValidation: true})
			if !errors.Is(err, fileutil.ErrOutsideRoot) {
				t.Fatalf("err = %v, want ErrOutsideRoot", err)
			}
			assertUntouched(t, outside, "user content")
		})
	}
}

// assertUntouched fails when anything under dir other than a file holding
// want was written.
func assertUntouched(t *testing.T, dir, want string) {
	t.Helper()
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, readErr := os.ReadFile(p)
		if readErr != nil {
			return readErr
		}
		if string(data) != want {
			t.Errorf("%s written outside the project root: %q", p, data)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
