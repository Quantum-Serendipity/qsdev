package claudecode_test

import (
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/answers"
)

// snapshotTree returns the content of every regular file under root, keyed by
// slash-separated relative path, skipping the directory skip.
func snapshotTree(t *testing.T, root, skip string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path == skip {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = readFile(t, path)
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", root, err)
	}
	return files
}

// TestClaudeInit_NestedRepoWritesToCwd covers U18-V05: `claude init` run in a
// nested repository below a qsdev project configures the nested repository,
// which is its own project boundary, and leaves the ancestor's files alone.
func TestClaudeInit_NestedRepoWritesToCwd(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ancestor := t.TempDir()
	if err := os.Mkdir(filepath.Join(ancestor, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	chdir(t, ancestor)
	mustRunClaude(t, "init", "--yes")

	sub := filepath.Join(ancestor, "sub")
	nested := filepath.Join(sub, "nested")
	if err := os.MkdirAll(filepath.Join(nested, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	before := snapshotTree(t, ancestor, sub)

	chdir(t, nested)
	mustRunClaude(t, "init", "--yes", "--force")

	for _, path := range []string{filepath.Join(nested, filepath.FromSlash(settingsRel)), answers.PrimaryPath(nested)} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("nested repository lacks %s: %v", path, err)
		}
	}
	if !settingsHasSelfprotect(t, nested) {
		t.Errorf("nested settings.json does not register %q", selfprotectCommand())
	}
	if after := snapshotTree(t, ancestor, sub); !maps.Equal(before, after) {
		t.Errorf("claude init in the nested repository changed the ancestor project's files")
	}
}
