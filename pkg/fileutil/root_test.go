package fileutil_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/testutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

// projectDir is the name of the project root newRootLayout creates, so a
// relative link can step out of the root and back in.
const projectDir = "project"

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

// newRootLayout returns a fresh project root and a sibling directory outside
// it.
func newRootLayout(t *testing.T) (root, outside string) {
	t.Helper()
	base := t.TempDir()
	root = filepath.Join(base, projectDir)
	outside = filepath.Join(base, "outside")
	mustMkdir(t, root)
	mustMkdir(t, outside)
	return root, outside
}

// gitHooksRedirect commits the layout .claude/hooks -> ../.git/hooks with a
// pre-commit hook behind it, and returns the hook's path.
func gitHooksRedirect(t *testing.T, root string) string {
	t.Helper()
	hook := filepath.Join(root, ".git", "hooks", "pre-commit")
	mustWrite(t, hook, "hook")
	testutil.SymlinkOrSkip(t, filepath.Join("..", ".git", "hooks"), filepath.Join(root, ".claude", "hooks"))
	return hook
}

// aliasedRelativeLink builds a -> sub/deep and sub/deep/link -> ../target,
// so the OS resolves a/link/v to sub/target/v. It writes that file ("real")
// and a decoy target/v ("decoy") where a lexical join of the relative link
// onto a/ would land.
func aliasedRelativeLink(t *testing.T, root string) {
	t.Helper()
	mustWrite(t, filepath.Join(root, "sub", "target", "v"), "real")
	mustWrite(t, filepath.Join(root, "target", "v"), "decoy")
	testutil.SymlinkOrSkip(t, filepath.Join("sub", "deep"), filepath.Join(root, "a"))
	testutil.SymlinkOrSkip(t, filepath.Join("..", "target"), filepath.Join(root, "sub", "deep", "link"))
}

// joinRaw joins elements with the separator without cleaning, so a spelling
// such as "d/link/../x" reaches the function under test as written.
func joinRaw(elem ...string) string {
	return strings.Join(elem, string(filepath.Separator))
}

func TestResolveInRoot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		// setup builds the layout and returns the rel path to resolve.
		setup func(t *testing.T, root, outside string) string
		// wantErrs must all match with errors.Is; empty means success.
		wantErrs []error
		// wantMsg and notMsg are substrings the error message must and must
		// not contain.
		wantMsg, notMsg string
		// wantRel is the expected result relative to the resolved root.
		wantRel string
	}{
		{
			name: "symlinked dir redirecting into .git",
			setup: func(t *testing.T, root, _ string) string {
				t.Helper()
				gitHooksRedirect(t, root)
				return filepath.Join(".claude", "hooks", "pre-commit")
			},
			wantErrs: []error{fileutil.ErrRedirectsIntoGit, fileutil.ErrOutsideRoot},
			wantMsg:  "inside .git",
			notMsg:   "outside root",
		},
		{
			name: "symlinked leaf naming the .git dir",
			setup: func(t *testing.T, root, _ string) string {
				t.Helper()
				mustWrite(t, filepath.Join(root, ".git", "config"), "[core]\n")
				testutil.SymlinkOrSkip(t, ".git", filepath.Join(root, "meta"))
				return "meta"
			},
			wantErrs: []error{fileutil.ErrRedirectsIntoGit, fileutil.ErrOutsideRoot},
		},
		{
			name: "cleaned spelling still judged a redirect",
			setup: func(t *testing.T, root, _ string) string {
				t.Helper()
				mustMkdir(t, filepath.Join(root, ".git", "hooks"))
				testutil.SymlinkOrSkip(t, filepath.Join(".git", "hooks"), filepath.Join(root, "x"))
				return joinRaw(".git", "..", "x", "pre-commit")
			},
			wantErrs: []error{fileutil.ErrRedirectsIntoGit},
		},
		{
			name:     "dot-dot traversal",
			setup:    func(*testing.T, string, string) string { return filepath.Join("..", "x") },
			wantErrs: []error{fileutil.ErrOutsideRoot},
		},
		{
			name:     "empty rel",
			setup:    func(*testing.T, string, string) string { return "" },
			wantErrs: []error{fileutil.ErrOutsideRoot},
		},
		{
			name:     "absolute rel",
			setup:    func(_ *testing.T, root, _ string) string { return filepath.Join(root, "x") },
			wantErrs: []error{fileutil.ErrOutsideRoot},
		},
		{
			name: "absolute symlinked parent escaping root",
			setup: func(t *testing.T, root, outside string) string {
				t.Helper()
				testutil.SymlinkOrSkip(t, outside, filepath.Join(root, ".github"))
				return filepath.Join(".github", "workflows", "new", "x.yml")
			},
			wantErrs: []error{fileutil.ErrOutsideRoot},
		},
		{
			name: "symlinked file escaping root",
			setup: func(t *testing.T, root, outside string) string {
				t.Helper()
				mustWrite(t, filepath.Join(outside, "victim.md"), "mine")
				testutil.SymlinkOrSkip(t, filepath.Join(outside, "victim.md"), filepath.Join(root, "linked.md"))
				return "linked.md"
			},
			wantErrs: []error{fileutil.ErrOutsideRoot},
		},
		{
			name: "dangling ancestor dir symlink to outside",
			setup: func(t *testing.T, root, outside string) string {
				t.Helper()
				testutil.SymlinkOrSkip(t, filepath.Join(outside, "missing"), filepath.Join(root, ".claude"))
				return filepath.Join(".claude", "hooks", "guard.py")
			},
			wantErrs: []error{fileutil.ErrOutsideRoot},
		},
		{
			name: "dangling ancestor dir symlink inside root",
			setup: func(t *testing.T, root, _ string) string {
				t.Helper()
				testutil.SymlinkOrSkip(t, "missing", filepath.Join(root, ".claude"))
				return filepath.Join(".claude", "hooks", "guard.py")
			},
			wantErrs: []error{fileutil.ErrOutsideRoot},
		},
		{
			name: "dangling leaf pointing outside root",
			setup: func(t *testing.T, root, outside string) string {
				t.Helper()
				testutil.SymlinkOrSkip(t, filepath.Join(outside, "missing.md"), filepath.Join(root, "dangling.md"))
				return "dangling.md"
			},
			wantErrs: []error{fileutil.ErrOutsideRoot},
		},
		{
			name: "CLAUDE.md -> existing AGENTS.md",
			setup: func(t *testing.T, root, _ string) string {
				t.Helper()
				mustWrite(t, filepath.Join(root, "AGENTS.md"), "orig")
				testutil.SymlinkOrSkip(t, "AGENTS.md", filepath.Join(root, "CLAUDE.md"))
				return "CLAUDE.md"
			},
			wantRel: "AGENTS.md",
		},
		{
			name: "CLAUDE.md -> missing AGENTS.md",
			setup: func(t *testing.T, root, _ string) string {
				t.Helper()
				testutil.SymlinkOrSkip(t, "AGENTS.md", filepath.Join(root, "CLAUDE.md"))
				return "CLAUDE.md"
			},
			wantRel: "AGENTS.md",
		},
		{
			name: "explicit .git path",
			setup: func(t *testing.T, root, _ string) string {
				t.Helper()
				mustMkdir(t, filepath.Join(root, ".git", "info"))
				return filepath.Join(".git", "info", "exclude")
			},
			wantRel: filepath.Join(".git", "info", "exclude"),
		},
		{
			name:    "missing nested dirs",
			setup:   func(*testing.T, string, string) string { return filepath.Join("a", "b", "c", "new.txt") },
			wantRel: filepath.Join("a", "b", "c", "new.txt"),
		},
		{
			name: "in-root absolute dir alias",
			setup: func(t *testing.T, root, _ string) string {
				t.Helper()
				mustMkdir(t, filepath.Join(root, "inside"))
				testutil.SymlinkOrSkip(t, filepath.Join(root, "inside"), filepath.Join(root, "alias"))
				return filepath.Join("alias", "x.txt")
			},
			wantRel: filepath.Join("inside", "x.txt"),
		},
		{
			name: "in-root alias stepping out and back in",
			setup: func(t *testing.T, root, _ string) string {
				t.Helper()
				mustMkdir(t, filepath.Join(root, "inside"))
				testutil.SymlinkOrSkip(t, filepath.Join("..", projectDir, "inside"), filepath.Join(root, "alias"))
				return filepath.Join("alias", "x.txt")
			},
			wantRel: filepath.Join("inside", "x.txt"),
		},
		{
			name: "relative link inside an aliased dir",
			setup: func(t *testing.T, root, _ string) string {
				t.Helper()
				aliasedRelativeLink(t, root)
				return filepath.Join("a", "link", "v")
			},
			wantRel: filepath.Join("sub", "target", "v"),
		},
		{
			name: "relative leaf link inside an aliased dir",
			setup: func(t *testing.T, root, _ string) string {
				t.Helper()
				mustMkdir(t, filepath.Join(root, "sub", "deep"))
				testutil.SymlinkOrSkip(t, filepath.Join("sub", "deep"), filepath.Join(root, "a"))
				testutil.SymlinkOrSkip(t, filepath.Join("..", "t.md"), filepath.Join(root, "sub", "deep", "l.md"))
				return filepath.Join("a", "l.md")
			},
			wantRel: filepath.Join("sub", "t.md"),
		},
		{
			name:    "root itself",
			setup:   func(*testing.T, string, string) string { return "." },
			wantRel: ".",
		},
		{
			name: "link then dot-dot is judged lexically",
			setup: func(t *testing.T, root, _ string) string {
				t.Helper()
				mustMkdir(t, filepath.Join(root, ".git", "hooks"))
				testutil.SymlinkOrSkip(t, filepath.Join("..", ".git", "hooks"), filepath.Join(root, "d", "link"))
				return joinRaw("d", "link", "..", "x")
			},
			wantRel: filepath.Join("d", "x"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root, outside := newRootLayout(t)
			rel := tt.setup(t, root, outside)

			got, err := fileutil.ResolveInRoot(root, rel)
			if len(tt.wantErrs) > 0 {
				if err == nil {
					t.Fatalf("ResolveInRoot(%q) = %q, want error", rel, got)
				}
				for _, want := range tt.wantErrs {
					if !errors.Is(err, want) {
						t.Errorf("ResolveInRoot(%q) error = %v, want errors.Is %v", rel, err, want)
					}
				}
				if tt.wantMsg != "" && !strings.Contains(err.Error(), tt.wantMsg) {
					t.Errorf("error %q does not contain %q", err, tt.wantMsg)
				}
				if tt.notMsg != "" && strings.Contains(err.Error(), tt.notMsg) {
					t.Errorf("error %q contains %q", err, tt.notMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveInRoot(%q): %v", rel, err)
			}
			resolvedRoot, err := filepath.EvalSymlinks(root)
			if err != nil {
				t.Fatal(err)
			}
			if want := filepath.Join(resolvedRoot, tt.wantRel); got != want {
				t.Errorf("ResolveInRoot(%q) = %q, want %q", rel, got, want)
			}
		})
	}
}

// TestResolveInRootThroughSymlinkedRoot verifies a root that is itself a
// symbolic link (as t.TempDir is on macOS) still contains its own entries.
func TestResolveInRootThroughSymlinkedRoot(t *testing.T) {
	t.Parallel()
	target, _ := newRootLayout(t)
	link := filepath.Join(filepath.Dir(target), "link")
	testutil.SymlinkOrSkip(t, target, link)
	resolvedTarget, err := filepath.EvalSymlinks(target)
	if err != nil {
		t.Fatal(err)
	}

	for _, rel := range []string{".", "x.txt", filepath.Join("a", "b")} {
		got, err := fileutil.ResolveInRoot(link, rel)
		if err != nil {
			t.Fatalf("ResolveInRoot(%q): %v", rel, err)
		}
		if want := filepath.Join(resolvedTarget, rel); got != want {
			t.Errorf("ResolveInRoot(%q) = %q, want %q", rel, got, want)
		}
	}
}

func TestRemoveInRootRefusesGitRedirect(t *testing.T) {
	t.Parallel()
	root, _ := newRootLayout(t)
	hook := gitHooksRedirect(t, root)

	err := fileutil.RemoveInRoot(root, filepath.Join(".claude", "hooks", "pre-commit"))
	if !errors.Is(err, fileutil.ErrRedirectsIntoGit) {
		t.Fatalf("RemoveInRoot error = %v, want ErrRedirectsIntoGit", err)
	}
	assertFile(t, hook, []byte("hook"))
}

func TestRemoveInRoot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		// setup builds the layout and returns the rel path to remove.
		setup func(t *testing.T, root, outside string) string
		// wantErrs must all match with errors.Is and notErrs must not; an
		// empty wantErrs means success.
		wantErrs, notErrs []error
		// check verifies the tree afterwards.
		check func(t *testing.T, root, outside string)
	}{
		{
			name: "file and its now-empty dir",
			setup: func(t *testing.T, root, _ string) string {
				t.Helper()
				mustMkdir(t, filepath.Join(root, "a", "b"))
				return filepath.Join("a", "b")
			},
			check: func(t *testing.T, root, _ string) {
				t.Helper()
				assertAbsent(t, filepath.Join(root, "a", "b"))
			},
		},
		{
			name: "in-root absolute dir alias",
			setup: func(t *testing.T, root, _ string) string {
				t.Helper()
				mustWrite(t, filepath.Join(root, "inside", "x"), "x")
				testutil.SymlinkOrSkip(t, filepath.Join(root, "inside"), filepath.Join(root, "alias"))
				return filepath.Join("alias", "x")
			},
			check: func(t *testing.T, root, _ string) {
				t.Helper()
				assertAbsent(t, filepath.Join(root, "inside", "x"))
			},
		},
		{
			name: "in-root alias stepping out and back in",
			setup: func(t *testing.T, root, _ string) string {
				t.Helper()
				mustWrite(t, filepath.Join(root, "inside", "x"), "x")
				testutil.SymlinkOrSkip(t, filepath.Join("..", projectDir, "inside"), filepath.Join(root, "alias"))
				return filepath.Join("alias", "x")
			},
			check: func(t *testing.T, root, _ string) {
				t.Helper()
				assertAbsent(t, filepath.Join(root, "inside", "x"))
			},
		},
		{
			name: "relative link inside an aliased dir",
			setup: func(t *testing.T, root, _ string) string {
				t.Helper()
				aliasedRelativeLink(t, root)
				return filepath.Join("a", "link", "v")
			},
			check: func(t *testing.T, root, _ string) {
				t.Helper()
				assertAbsent(t, filepath.Join(root, "sub", "target", "v"))
				assertFile(t, filepath.Join(root, "target", "v"), []byte("decoy"))
			},
		},
		{
			name: "leaf link to outside is unlinked",
			setup: func(t *testing.T, root, outside string) string {
				t.Helper()
				mustWrite(t, filepath.Join(outside, "victim.md"), "mine")
				testutil.SymlinkOrSkip(t, filepath.Join(outside, "victim.md"), filepath.Join(root, "sub", "linked.md"))
				return filepath.Join("sub", "linked.md")
			},
			check: func(t *testing.T, root, outside string) {
				t.Helper()
				assertAbsent(t, filepath.Join(root, "sub", "linked.md"))
				assertFile(t, filepath.Join(outside, "victim.md"), []byte("mine"))
			},
		},
		{
			name: "leaf link to in-root dir unlinks the link only",
			setup: func(t *testing.T, root, _ string) string {
				t.Helper()
				mustWrite(t, filepath.Join(root, "inside", "keep"), "keep")
				testutil.SymlinkOrSkip(t, "inside", filepath.Join(root, "linkdir"))
				return "linkdir"
			},
			check: func(t *testing.T, root, _ string) {
				t.Helper()
				assertAbsent(t, filepath.Join(root, "linkdir"))
				assertFile(t, filepath.Join(root, "inside", "keep"), []byte("keep"))
			},
		},
		{
			name:     "missing file",
			setup:    func(*testing.T, string, string) string { return "missing.txt" },
			wantErrs: []error{fs.ErrNotExist},
		},
		{
			name:     "missing nested dirs",
			setup:    func(*testing.T, string, string) string { return filepath.Join("no", "such", "x.txt") },
			wantErrs: []error{fs.ErrNotExist},
		},
		{
			name: "dangling ancestor is not exist",
			setup: func(t *testing.T, root, outside string) string {
				t.Helper()
				testutil.SymlinkOrSkip(t, filepath.Join(outside, "missing"), filepath.Join(root, ".claude"))
				return filepath.Join(".claude", "hooks", "pre-commit")
			},
			wantErrs: []error{fs.ErrNotExist},
			notErrs:  []error{fileutil.ErrOutsideRoot},
		},
		{
			name: "dangling in-root ancestor is not exist",
			setup: func(t *testing.T, root, _ string) string {
				t.Helper()
				testutil.SymlinkOrSkip(t, "missing", filepath.Join(root, ".claude"))
				return filepath.Join(".claude", "hooks", "pre-commit")
			},
			wantErrs: []error{fs.ErrNotExist},
			notErrs:  []error{fileutil.ErrOutsideRoot},
		},
		{
			name: "dangling parent link to outside is not exist",
			setup: func(t *testing.T, root, outside string) string {
				t.Helper()
				testutil.SymlinkOrSkip(t, filepath.Join(outside, "missing"), filepath.Join(root, ".claude"))
				return filepath.Join(".claude", "settings.json")
			},
			wantErrs: []error{fs.ErrNotExist},
			notErrs:  []error{fileutil.ErrOutsideRoot},
		},
		{
			name: "dangling parent link into .git is not exist",
			setup: func(t *testing.T, root, _ string) string {
				t.Helper()
				mustMkdir(t, filepath.Join(root, ".git"))
				testutil.SymlinkOrSkip(t, filepath.Join("..", ".git", "newdir"), filepath.Join(root, ".claude", "hooks"))
				return filepath.Join(".claude", "hooks", "x")
			},
			wantErrs: []error{fs.ErrNotExist},
			notErrs:  []error{fileutil.ErrOutsideRoot, fileutil.ErrRedirectsIntoGit},
			check: func(t *testing.T, root, _ string) {
				t.Helper()
				assertAbsent(t, filepath.Join(root, ".git", "newdir"))
			},
		},
		{
			name: "missing dir beneath a .git redirect is not exist",
			setup: func(t *testing.T, root, _ string) string {
				t.Helper()
				gitHooksRedirect(t, root)
				return filepath.Join(".claude", "hooks", "sub", "x")
			},
			wantErrs: []error{fs.ErrNotExist},
			notErrs:  []error{fileutil.ErrOutsideRoot, fileutil.ErrRedirectsIntoGit},
		},
		{
			name: "link then dot-dot removes the lexical path only",
			setup: func(t *testing.T, root, _ string) string {
				t.Helper()
				mustWrite(t, filepath.Join(root, ".git", "hooks", "victim"), "keep")
				testutil.SymlinkOrSkip(t, filepath.Join("..", ".git", "hooks"), filepath.Join(root, "d", "link"))
				mustWrite(t, filepath.Join(root, "d", "victim"), "stale")
				return joinRaw("d", "link", "..", "victim")
			},
			check: func(t *testing.T, root, _ string) {
				t.Helper()
				assertFile(t, filepath.Join(root, ".git", "hooks", "victim"), []byte("keep"))
				assertAbsent(t, filepath.Join(root, "d", "victim"))
			},
		},
		{
			name: "escaping parent link",
			setup: func(t *testing.T, root, outside string) string {
				t.Helper()
				mustWrite(t, filepath.Join(outside, "victim.md"), "mine")
				testutil.SymlinkOrSkip(t, outside, filepath.Join(root, ".github"))
				return filepath.Join(".github", "victim.md")
			},
			wantErrs: []error{fileutil.ErrOutsideRoot},
			check: func(t *testing.T, _, outside string) {
				t.Helper()
				assertFile(t, filepath.Join(outside, "victim.md"), []byte("mine"))
			},
		},
		{
			name:     "dot-dot traversal",
			setup:    func(*testing.T, string, string) string { return filepath.Join("..", "outside", "victim.md") },
			wantErrs: []error{fileutil.ErrOutsideRoot},
		},
		{
			name:     "absolute rel",
			setup:    func(_ *testing.T, _, outside string) string { return filepath.Join(outside, "victim.md") },
			wantErrs: []error{fileutil.ErrOutsideRoot},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root, outside := newRootLayout(t)
			rel := tt.setup(t, root, outside)

			err := fileutil.RemoveInRoot(root, rel)
			if len(tt.wantErrs) == 0 && err != nil {
				t.Fatalf("RemoveInRoot(%q): %v", rel, err)
			}
			for _, want := range tt.wantErrs {
				if !errors.Is(err, want) {
					t.Errorf("RemoveInRoot(%q) error = %v, want errors.Is %v", rel, err, want)
				}
			}
			for _, not := range tt.notErrs {
				if errors.Is(err, not) {
					t.Errorf("RemoveInRoot(%q) error = %v, want not errors.Is %v", rel, err, not)
				}
			}
			if tt.check != nil {
				tt.check(t, root, outside)
			}
		})
	}
}

// TestCheckRemoveInRoot verifies the check alone reports what RemoveInRoot
// would, and removes nothing.
func TestCheckRemoveInRoot(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		// setup builds the layout and returns rel and the file that must
		// survive the check.
		setup    func(t *testing.T, root, outside string) (rel, keep string)
		wantErrs []error
	}{
		{
			name: "in-root file",
			setup: func(t *testing.T, root, _ string) (string, string) {
				t.Helper()
				mustWrite(t, filepath.Join(root, "d", "x"), "x")
				return filepath.Join("d", "x"), filepath.Join(root, "d", "x")
			},
		},
		{
			name: "missing file",
			setup: func(*testing.T, string, string) (string, string) {
				return "missing.txt", ""
			},
		},
		{
			name: "git redirect",
			setup: func(t *testing.T, root, _ string) (string, string) {
				t.Helper()
				hook := gitHooksRedirect(t, root)
				return filepath.Join(".claude", "hooks", "pre-commit"), hook
			},
			wantErrs: []error{fileutil.ErrRedirectsIntoGit, fileutil.ErrOutsideRoot},
		},
		{
			name: "escaping parent",
			setup: func(t *testing.T, root, outside string) (string, string) {
				t.Helper()
				mustWrite(t, filepath.Join(outside, "victim.md"), "mine")
				testutil.SymlinkOrSkip(t, outside, filepath.Join(root, ".github"))
				return filepath.Join(".github", "victim.md"), filepath.Join(outside, "victim.md")
			},
			wantErrs: []error{fileutil.ErrOutsideRoot},
		},
		{
			// An in-root dangling parent resolves to its missing target and
			// passes; RemoveInRoot then finds nothing there.
			name: "dangling parent to outside",
			setup: func(t *testing.T, root, outside string) (string, string) {
				t.Helper()
				testutil.SymlinkOrSkip(t, filepath.Join(outside, "missing"), filepath.Join(root, ".claude"))
				return filepath.Join(".claude", "x"), ""
			},
			wantErrs: []error{fs.ErrNotExist},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root, outside := newRootLayout(t)
			rel, keep := tt.setup(t, root, outside)

			err := fileutil.CheckRemoveInRoot(root, rel)
			if len(tt.wantErrs) == 0 && err != nil {
				t.Fatalf("CheckRemoveInRoot(%q): %v", rel, err)
			}
			for _, want := range tt.wantErrs {
				if !errors.Is(err, want) {
					t.Errorf("CheckRemoveInRoot(%q) error = %v, want errors.Is %v", rel, err, want)
				}
			}
			if keep != "" {
				if _, err := os.Stat(keep); err != nil {
					t.Errorf("check removed %s: %v", keep, err)
				}
			}
		})
	}
}

// TestInRootRefusalIsNotRewrapped verifies the write and remove entry points
// return the containment error as is, so the path and the refusal are stated
// once.
func TestInRootRefusalIsNotRewrapped(t *testing.T) {
	t.Parallel()
	root, _ := newRootLayout(t)
	gitHooksRedirect(t, root)
	rel := filepath.Join(".claude", "hooks", "guard.sh")

	tests := []struct {
		name string
		err  error
	}{
		{"write", fileutil.WriteFileAtomicInRoot(root, rel, []byte("x"), 0o644)},
		{"remove", fileutil.RemoveInRoot(root, rel)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if !errors.Is(tt.err, fileutil.ErrRedirectsIntoGit) {
				t.Fatalf("error = %v, want ErrRedirectsIntoGit", tt.err)
			}
			if n := strings.Count(tt.err.Error(), "refusing"); n != 1 {
				t.Errorf("message %q states the refusal %d times, want 1", tt.err, n)
			}
		})
	}
	assertAbsent(t, filepath.Join(root, ".git", "hooks", "guard.sh"))
}
