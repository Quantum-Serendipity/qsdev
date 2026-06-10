package canon

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

func TestExpandTilde(t *testing.T) {
	t.Parallel()

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("getting home dir: %v", err)
	}

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"bare tilde", "~", home},
		{"tilde with subpath", "~/foo", filepath.Join(home, "foo")},
		{"absolute path unchanged", "/abs/path", "/abs/path"},
		{"empty string unchanged", "", ""},
		{"tilde nested subpath", "~/a/b/c", filepath.Join(home, "a", "b", "c")},
		{"no tilde prefix", "foo/bar", "foo/bar"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ExpandTilde(tt.input)
			if err != nil {
				t.Fatalf("ExpandTilde(%q) returned error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("ExpandTilde(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCanonicalize_AbsolutePath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := filepath.Join(dir, "existing.txt")
	if err := os.WriteFile(file, []byte("test"), 0o644); err != nil {
		t.Fatalf("creating test file: %v", err)
	}

	got, err := Canonicalize(file)
	if err != nil {
		t.Fatalf("Canonicalize(%q) returned error: %v", file, err)
	}

	// Resolve dir itself in case TempDir uses symlinks (e.g. /tmp -> /private/tmp).
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("resolving temp dir: %v", err)
	}
	want := filepath.Join(resolved, "existing.txt")

	if got != want {
		t.Errorf("Canonicalize(%q) = %q, want %q", file, got, want)
	}
}

func TestCanonicalize_RelativePath(t *testing.T) {
	t.Parallel()

	got, err := Canonicalize(".")
	if err != nil {
		t.Fatalf("Canonicalize(\".\") returned error: %v", err)
	}

	if !filepath.IsAbs(got) {
		t.Errorf("Canonicalize(\".\") = %q, expected absolute path", got)
	}
}

func TestCanonicalize_NonExistentPath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	nonExistent := filepath.Join(dir, "nonexistent", "deeply", "nested", "file.txt")

	got, err := Canonicalize(nonExistent)
	if err != nil {
		t.Fatalf("Canonicalize(%q) returned error: %v", nonExistent, err)
	}

	// The existing ancestor (dir) should be resolved; the rest appended lexically.
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("resolving temp dir: %v", err)
	}
	want := filepath.Join(resolved, "nonexistent", "deeply", "nested", "file.txt")

	if got != want {
		t.Errorf("Canonicalize(%q) = %q, want %q", nonExistent, got, want)
	}
}

func TestCanonicalize_SymlinkResolution(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("symlinks require elevated privileges on Windows")
	}

	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("creating target dir: %v", err)
	}
	realFile := filepath.Join(target, "file.txt")
	if err := os.WriteFile(realFile, []byte("data"), 0o644); err != nil {
		t.Fatalf("creating target file: %v", err)
	}

	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("creating symlink: %v", err)
	}

	linkedFile := filepath.Join(link, "file.txt")
	got, err := Canonicalize(linkedFile)
	if err != nil {
		t.Fatalf("Canonicalize(%q) returned error: %v", linkedFile, err)
	}

	resolvedDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("resolving temp dir: %v", err)
	}
	want := filepath.Join(resolvedDir, "target", "file.txt")

	if got != want {
		t.Errorf("Canonicalize(%q) = %q, want %q", linkedFile, got, want)
	}
}

func TestCanonicalize_SymlinkToNonExistent(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("symlinks require elevated privileges on Windows")
	}

	dir := t.TempDir()
	target := filepath.Join(dir, "real")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("creating target dir: %v", err)
	}

	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("creating symlink: %v", err)
	}

	// Path through symlink to a file that doesn't exist.
	path := filepath.Join(link, "does-not-exist.txt")
	got, err := Canonicalize(path)
	if err != nil {
		t.Fatalf("Canonicalize(%q) returned error: %v", path, err)
	}

	resolvedDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("resolving temp dir: %v", err)
	}
	// The symlink "link" -> "real" should be resolved, then the non-existent tail appended.
	want := filepath.Join(resolvedDir, "real", "does-not-exist.txt")

	if got != want {
		t.Errorf("Canonicalize(%q) = %q, want %q", path, got, want)
	}
}

func TestIsProtected(t *testing.T) {
	t.Parallel()

	// Reset the init state so tests use the real home dir.
	resetProtectedPaths(t)

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("getting home dir: %v", err)
	}

	tests := []struct {
		name     string
		path     string
		wantProt bool
		wantCat  string
	}{
		{
			"qsdev config dir",
			filepath.Join(home, ".qsdev", "config.yaml"),
			true, "config",
		},
		{
			"gdev config dir",
			filepath.Join(home, ".gdev", "something"),
			true, "config",
		},
		{
			"claude settings.json",
			filepath.Join(home, ".claude", "settings.json"),
			true, "claude-settings",
		},
		{
			"claude settings.local.json",
			filepath.Join(home, ".claude", "settings.local.json"),
			true, "claude-settings",
		},
		{
			"claude managed-settings.json",
			filepath.Join(home, ".claude", "managed-settings.json"),
			true, "claude-settings",
		},
		{
			"random unprotected etc path",
			"/etc/something-else/file.txt",
			false, "",
		},
	}

	if runtime.GOOS != "windows" {
		tests = append(tests,
			struct {
				name     string
				path     string
				wantProt bool
				wantCat  string
			}{"system config etc gdev", "/etc/gdev/config.yaml", true, "system-config"},
			struct {
				name     string
				path     string
				wantProt bool
				wantCat  string
			}{"system config etc claude-code", "/etc/claude-code/policy.json", true, "system-config"},
		)
	}

	tests = append(tests, []struct {
		name     string
		path     string
		wantProt bool
		wantCat  string
	}{
		{
			"audit dir",
			filepath.Join(home, ".qsdev", "audit", "log.json"),
			true, "audit",
		},
		{
			"binary dir",
			filepath.Join(home, ".qsdev", "bin", "qsdev"),
			true, "binary",
		},
		{
			"random unprotected path",
			"/tmp/random/file.txt",
			false, "",
		},
		{
			"home dir itself",
			home,
			false, "",
		},
		{
			"non-matching claude subdir",
			filepath.Join(home, ".claude", "other-file"),
			false, "",
		},
	}...)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotProt, gotCat := IsProtected(tt.path)
			if gotProt != tt.wantProt || gotCat != tt.wantCat {
				t.Errorf("IsProtected(%q) = (%v, %q), want (%v, %q)",
					tt.path, gotProt, gotCat, tt.wantProt, tt.wantCat)
			}
		})
	}
}

func TestIsProtected_McpJson(t *testing.T) {
	t.Parallel()

	resetProtectedPaths(t)

	tests := []struct {
		name     string
		path     string
		wantProt bool
		wantCat  string
	}{
		{
			"mcp.json in project root",
			"/home/user/project/.mcp.json",
			true, "mcp-config",
		},
		{
			"mcp.json in nested dir",
			"/some/deep/path/.mcp.json",
			true, "mcp-config",
		},
		{
			"not mcp.json",
			"/home/user/project/mcp.json",
			false, "",
		},
		{
			"mcp.json with extra suffix",
			"/home/user/.mcp.json.bak",
			false, "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotProt, gotCat := IsProtected(tt.path)
			if gotProt != tt.wantProt || gotCat != tt.wantCat {
				t.Errorf("IsProtected(%q) = (%v, %q), want (%v, %q)",
					tt.path, gotProt, gotCat, tt.wantProt, tt.wantCat)
			}
		})
	}
}

// resetProtectedPaths forces re-initialization of the protected path list.
// This ensures tests pick up the current home directory and don't depend on
// init ordering.
func resetProtectedPaths(t *testing.T) {
	t.Helper()
	initOnce = sync.Once{}
	protectedPrefixes = nil
	protectedSuffixes = nil
	initErr = nil
	if err := ensureInit(); err != nil {
		t.Fatalf("initializing protected paths: %v", err)
	}
}
