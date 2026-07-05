package canon

import (
	"errors"
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

func TestIsProtected_HooksAndAgents(t *testing.T) {
	t.Parallel()

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
			// Home config hook script — caught by the home-anchored prefix.
			"home claude hook script",
			filepath.Join(home, ".claude", "hooks", "preToolUse.sh"),
			true, "claude-settings",
		},
		{
			"home claude agent definition",
			filepath.Join(home, ".claude", "agents", "reviewer.md"),
			true, "claude-settings",
		},
		{
			// Project-relative hook path canonicalizes OUTSIDE $HOME; only the
			// segment guard catches it.
			"project claude hook script",
			filepath.Join(home, "project", ".claude", "hooks", "preToolUse.sh"),
			true, "claude-settings",
		},
		{
			"project claude agent definition",
			filepath.Join(home, "work", "repo", ".claude", "agents", "x.md"),
			true, "claude-settings",
		},
		{
			// A settings file that is NOT a hook/agent still resolves via its own
			// prefix, unaffected by the new segment guard.
			"home claude settings",
			filepath.Join(home, ".claude", "settings.json"),
			true, "claude-settings",
		},
		{
			// Non-hook/agent .claude file stays unprotected (no matching prefix).
			"non-matching claude subdir",
			filepath.Join(home, ".claude", "other-file"),
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

func TestContainsProtectedPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  bool
	}{
		// Trailing-path forms (existing substring patterns).
		{".claude/settings.json", true},
		{"cat .claude/hooks/pre.sh", true},
		{"/etc/gdev/policy.yaml", true},
		{"/etc/claude-code/config.json", true},
		// Bare directory names at a path-token boundary (the new match).
		{".claude", true},
		{"rm -rf .claude", true},
		{"rm -rf ~/.claude", true},
		{"rm -rf .qsdev", true},
		{"rm -rf .gdev", true},
		{"rm -rf /etc/gdev", true},
		{"find .claude -delete", true}, // followed by whitespace
		{`rm -rf ".claude"`, true},     // followed by a quote
		{"rm -rf .claude;", true},      // followed by a shell metachar
		// Over-match guards: a longer name that merely embeds a token.
		{"my.claude.bak", false},
		{"foo.claudex", false},
		{"my.claude", false},
		{"settings.claude", false},
		// Unrelated paths.
		{"node_modules", false},
		{"README.md", false},
		{"/tmp/scratch", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			if got := ContainsProtectedPath(tt.input); got != tt.want {
				t.Errorf("ContainsProtectedPath(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// TestIsProtected_FailsClosedOnHomeError verifies the fail-CLOSED behavior of
// IsProtected when the home directory cannot be resolved (regression for
// F-CAP-20.4-3). If os.UserHomeDir fails, the home-anchored protected-prefix
// table cannot be built; a self-protection control must then treat paths as
// protected (deny) rather than returning "not protected" (fail OPEN).
//
// This test is intentionally NOT parallel: it mutates the package-level home
// resolver and init state. All other tests in this package call t.Parallel()
// and therefore resume only after this sequential test (and its cleanup) has
// fully completed, so there is no data race. Cleanup restores a clean,
// re-initializable state so those tests initialize normally.
func TestIsProtected_FailsClosedOnHomeError(t *testing.T) {
	origHome := userHomeDir
	t.Cleanup(func() {
		userHomeDir = origHome
		initOnce = sync.Once{}
		initErr = nil
		protectedPrefixes = nil
		protectedSuffixes = nil
	})

	// Simulate an unresolvable home directory (e.g. HOME/USERPROFILE unset) and
	// force ensureInit to re-run with the failing resolver.
	userHomeDir = func() (string, error) {
		return "", errors.New("home directory unavailable")
	}
	initOnce = sync.Once{}
	initErr = nil
	protectedPrefixes = nil
	protectedSuffixes = nil

	// The fail-closed branch is only exercised if init actually fails.
	if err := ensureInit(); err == nil {
		t.Fatal("ensureInit() succeeded, want an error when home is unavailable")
	}

	// Every path must stay protected: with no prefix table we cannot prove any
	// path is unprotected, so the control must deny. This includes both
	// would-be home-anchored config paths and arbitrary paths.
	paths := []string{
		filepath.FromSlash("/home/alice/.qsdev/config.yaml"),
		filepath.FromSlash("/home/alice/.claude/settings.json"),
		filepath.FromSlash("/etc/gdev/policy.yaml"),
		filepath.FromSlash("/some/arbitrary/path.txt"),
	}
	for _, p := range paths {
		t.Run(p, func(t *testing.T) {
			gotProt, gotCat := IsProtected(p)
			if !gotProt {
				t.Errorf("IsProtected(%q) = (false, %q) on home-resolution failure; want fail-closed (true, ...)", p, gotCat)
			}
			// SP-001 denies Write/Edit only for these categories; the fail-closed
			// return must land in one of them so the operation is actually blocked.
			switch gotCat {
			case "config", "claude-settings", "system-config":
			default:
				t.Errorf("IsProtected(%q) category = %q; want a category SP-001 denies (config/claude-settings/system-config)", p, gotCat)
			}
		})
	}
}

// resetProtectedPaths ensures the protected path list is initialized.
func resetProtectedPaths(t *testing.T) {
	t.Helper()
	if err := ensureInit(); err != nil {
		t.Fatalf("initializing protected paths: %v", err)
	}
}
