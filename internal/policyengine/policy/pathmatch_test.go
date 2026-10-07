package policy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/pathmatch"
)

// TestCompilePathMatcher_NativeSeparatorPattern is the regression for patterns
// built from native paths: on Windows a pattern such as C:\Users\me\x\* had its
// backslash separators read as glob escapes, so it never matched the
// forward-slash path forms and denied paths went unprotected. The pattern is
// built with the host's separator, so on Unix it covers the same cases with `/`.
func TestCompilePathMatcher_NativeSeparatorPattern(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := filepath.Join(dir, "secret.env")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "other.txt")

	tests := []struct {
		name    string
		pattern string
		path    string
		want    bool
	}{
		{"exact native path", file, file, true},
		{"native dir with slash glob", dir + "/*", file, true},
		{"native dir with native glob", dir + string(filepath.Separator) + "*", file, true},
		{"path outside the pattern", dir + string(filepath.Separator) + "*", outside, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m, err := CompilePathMatcher(tt.pattern)
			if err != nil {
				t.Fatalf("CompilePathMatcher(%q): %v", tt.pattern, err)
			}
			if got := m.Match(tt.path, ""); got != tt.want {
				t.Errorf("CompilePathMatcher(%q).Match(%q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
			}
		})
	}
}

// TestPathMatcher_FilesystemKey checks that path forms are keyed as the
// filesystem names files (pathmatch.Key): with Windows options, an alias of a
// protected file (other case, trailing dot, stream suffix) still matches,
// while an exact-name filesystem treats them as different files.
func TestPathMatcher_FilesystemKey(t *testing.T) {
	t.Parallel()
	windows := pathmatch.Options{FoldCase: true, WindowsAliases: true}
	tests := []struct {
		name string
		path string
		opts pathmatch.Options
		want bool
	}{
		{"exact", "C:/Users/me/.ssh/id_rsa", windows, true},
		{"aliases", "C:/Users/me/.ssh./id_rsa::$DATA", windows, true},
		{"case", "c:/USERS/Me/.SSH/ID_RSA", windows, true},
		{"trailing space", "C:/Users/me/.ssh /id_rsa", windows, true},
		{"other dir", "C:/Users/me/.sshx/id_rsa", windows, false},
		{"aliases on exact-name fs", "C:/Users/me/.ssh./id_rsa", pathmatch.Options{}, false},
		{"case on exact-name fs", "C:/Users/me/.SSH/id_rsa", pathmatch.Options{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m, err := compilePathMatcher("C:/Users/me/.ssh/**", tt.opts)
			if err != nil {
				t.Fatal(err)
			}
			if got := m.MatchForms([]string{tt.path}); got != tt.want {
				t.Errorf("MatchForms(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
