package policy

import (
	"os"
	"path/filepath"
	"testing"
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
