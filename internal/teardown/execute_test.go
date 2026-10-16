package teardown

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/testutil"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestExecute_RemoveExclusiveFiles(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "remove-me.txt")
	if err := os.WriteFile(filePath, []byte("delete this"), 0o644); err != nil {
		t.Fatal(err)
	}

	plan := &TeardownPlan{
		Remove: []FileAction{
			{Path: "remove-me.txt", Reason: "exclusive"},
		},
	}

	opts := TeardownOptions{ProjectRoot: dir}
	registry := toolreg.NewRegistry()

	result, err := Execute(plan, opts, registry)
	if err != nil {
		t.Fatal(err)
	}

	// File should be removed.
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Errorf("file should have been removed, but still exists")
	}

	if len(result.Removed) != 1 {
		t.Errorf("Removed = %d, want 1", len(result.Removed))
	}
}

func TestExecute_CleanSharedFiles(t *testing.T) {
	dir := t.TempDir()
	content := []byte("# Project\n<!-- qsdev:test-tool -->\ngenerated stuff\n<!-- /qsdev:test-tool -->\n\nUser content\n")
	mdPath := filepath.Join(dir, "README.md")
	if err := os.WriteFile(mdPath, content, 0o644); err != nil {
		t.Fatal(err)
	}

	registry := toolreg.NewRegistry()
	_ = registry.Register(toolreg.Tool{
		Name:     "test-tool",
		Category: toolreg.CategorySecurity,
		OwnedFiles: []toolreg.FileOwnership{
			{Path: "README.md", Ownership: toolreg.Shared, SectionID: "test-tool"},
		},
	})

	plan := &TeardownPlan{
		Clean: []FileAction{
			{Path: "README.md", Reason: "remove qsdev sections"},
		},
	}

	opts := TeardownOptions{ProjectRoot: dir}
	result, err := Execute(plan, opts, registry)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Cleaned) != 1 {
		t.Errorf("Cleaned = %d, want 1", len(result.Cleaned))
	}

	// Read the file back and verify the section was removed.
	updated, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatal(err)
	}

	if contains(updated, "<!-- qsdev:test-tool -->") {
		t.Errorf("expected qsdev section markers to be removed from README.md")
	}
	if !contains(updated, "User content") {
		t.Errorf("expected user content to be preserved in README.md")
	}
}

func TestExecute_RemoveDirs(t *testing.T) {
	dir := t.TempDir()
	devinitDir := filepath.Join(dir, ".devinit")
	if err := os.MkdirAll(devinitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Put a file in the directory.
	if err := os.WriteFile(filepath.Join(devinitDir, "state.yaml"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	plan := &TeardownPlan{
		Dirs: []string{".devinit"},
	}

	opts := TeardownOptions{ProjectRoot: dir}
	registry := toolreg.NewRegistry()

	result, err := Execute(plan, opts, registry)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(devinitDir); !os.IsNotExist(err) {
		t.Errorf("directory .devinit should have been removed")
	}

	if len(result.DirsRemoved) != 1 {
		t.Errorf("DirsRemoved = %d, want 1", len(result.DirsRemoved))
	}
}

func TestExecute_DryRun(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "keep-me.txt")
	if err := os.WriteFile(filePath, []byte("keep this"), 0o644); err != nil {
		t.Fatal(err)
	}
	devinitDir := filepath.Join(dir, ".devinit")
	if err := os.MkdirAll(devinitDir, 0o755); err != nil {
		t.Fatal(err)
	}

	plan := &TeardownPlan{
		Remove: []FileAction{
			{Path: "keep-me.txt", Reason: "exclusive"},
		},
		Dirs: []string{".devinit"},
	}

	opts := TeardownOptions{ProjectRoot: dir, DryRun: true}
	registry := toolreg.NewRegistry()

	result, err := Execute(plan, opts, registry)
	if err != nil {
		t.Fatal(err)
	}

	// DryRun: nothing should be changed.
	if _, err := os.Stat(filePath); err != nil {
		t.Errorf("file should still exist in dry-run mode")
	}
	if _, err := os.Stat(devinitDir); err != nil {
		t.Errorf("directory should still exist in dry-run mode")
	}

	// But result should list what would be done.
	if len(result.Removed) != 1 {
		t.Errorf("Removed = %d, want 1 (dry-run still populates result)", len(result.Removed))
	}
	if len(result.DirsRemoved) != 1 {
		t.Errorf("DirsRemoved = %d, want 1 (dry-run still populates result)", len(result.DirsRemoved))
	}
}

func TestExecute_RemoveEmptyDirs(t *testing.T) {
	dir := t.TempDir()

	// Create .claude/skills/ with a single file that we will remove.
	skillsDir := filepath.Join(dir, ".claude", "skills")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	skillFile := filepath.Join(skillsDir, "test.md")
	if err := os.WriteFile(skillFile, []byte("skill"), 0o644); err != nil {
		t.Fatal(err)
	}

	plan := &TeardownPlan{
		Remove: []FileAction{
			{Path: ".claude/skills/test.md", Reason: "exclusive"},
		},
	}

	opts := TeardownOptions{ProjectRoot: dir}
	registry := toolreg.NewRegistry()

	_, err := Execute(plan, opts, registry)
	if err != nil {
		t.Fatal(err)
	}

	// .claude/skills/ should be removed (now empty).
	if _, err := os.Stat(skillsDir); !os.IsNotExist(err) {
		t.Errorf(".claude/skills/ should be removed after becoming empty")
	}

	// .claude/ should also be removed (now empty).
	claudeDir := filepath.Join(dir, ".claude")
	if _, err := os.Stat(claudeDir); !os.IsNotExist(err) {
		t.Errorf(".claude/ should be removed after becoming empty")
	}
}

func TestExecute_NonexistentFileRemoval(t *testing.T) {
	dir := t.TempDir()

	plan := &TeardownPlan{
		Remove: []FileAction{
			{Path: "doesnt-exist.txt", Reason: "exclusive"},
		},
	}

	opts := TeardownOptions{ProjectRoot: dir}
	registry := toolreg.NewRegistry()

	result, err := Execute(plan, opts, registry)
	if err != nil {
		t.Fatal(err)
	}

	// Nonexistent files should be silently skipped, not counted as removed.
	if len(result.Removed) != 0 {
		t.Errorf("Removed = %d, want 0 (nonexistent files should be skipped)", len(result.Removed))
	}
}

func contains(data []byte, needle string) bool {
	for i := 0; i <= len(data)-len(needle); i++ {
		if string(data[i:i+len(needle)]) == needle {
			return true
		}
	}
	return false
}

// scaffoldOnly is a Markdown file holding only qsdev's title for name and its
// generated block: cleaning it leaves nothing of the user's, so teardown
// deletes it.
func scaffoldOnly(name string) []byte {
	return []byte("# " + name + "\n<!-- BEGIN GENERATED SECTION -->\ngen\n<!-- END GENERATED SECTION -->\n")
}

// gitRedirectProject returns a project whose .claude/hooks is a committed
// symlink into .git/hooks, holding a scaffold-only .git/hooks/pre-commit, so
// an unconfined clean would delete it as well as an unconfined remove.
func gitRedirectProject(t *testing.T) (root, gitHook string) {
	t.Helper()
	root = t.TempDir()
	gitHook = filepath.Join(root, ".git", "hooks", "pre-commit")
	writeFile(t, gitHook, scaffoldOnly("pre-commit"), 0o755)
	testutil.SymlinkOrSkip(t, filepath.Join("..", ".git", "hooks"), filepath.Join(root, ".claude", "hooks"))
	return root, gitHook
}

// requireNoChange fails unless execution reported no errors, removals or
// cleanings.
func requireNoChange(t *testing.T, result *TeardownResult) {
	t.Helper()
	if len(result.Errors) != 0 || len(result.Removed) != 0 || len(result.Cleaned) != 0 {
		t.Errorf("Removed = %v, Cleaned = %v, Errors = %v, want none", result.Removed, result.Cleaned, result.Errors)
	}
}

// TestExecuteRefusesGitRedirect is the teardown half of U14-07: a tracked
// path under .claude/hooks -> ../.git/hooks is neither removed nor cleaned,
// the refusal is recorded, and the git hook survives untouched.
func TestExecuteRefusesGitRedirect(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		plan       *TeardownPlan
		wantPrefix string
	}{
		{"remove", &TeardownPlan{Remove: []FileAction{{Path: ".claude/hooks/pre-commit"}}}, "refusing to remove .claude/hooks/pre-commit: "},
		{"clean", &TeardownPlan{Clean: []FileAction{{Path: ".claude/hooks/pre-commit"}}}, "cleaning .claude/hooks/pre-commit: "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root, gitHook := gitRedirectProject(t)
			result, err := Execute(tt.plan, TeardownOptions{ProjectRoot: root}, testRegistry())
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Errors) != 1 || !errors.Is(result.Errors[0], fileutil.ErrRedirectsIntoGit) {
				t.Fatalf("Errors = %v, want one wrapping ErrRedirectsIntoGit", result.Errors)
			}
			if msg := result.Errors[0].Error(); !strings.HasPrefix(msg, tt.wantPrefix) {
				t.Errorf("error = %q, want prefix %q", msg, tt.wantPrefix)
			}
			if len(result.Removed) != 0 || len(result.Cleaned) != 0 {
				t.Errorf("Removed = %v, Cleaned = %v, want none", result.Removed, result.Cleaned)
			}
			got, err := os.ReadFile(gitHook)
			if err != nil {
				t.Fatalf(".git/hooks/pre-commit was removed: %v", err)
			}
			if string(got) != string(scaffoldOnly("pre-commit")) {
				t.Errorf(".git/hooks/pre-commit changed: %q", got)
			}
		})
	}
}

// TestExecuteRemovesThroughInRootDirAlias verifies a tracked file reached
// through a directory symlink that resolves inside the project, written
// absolute or stepping out of the root and back in, is still removed, as on
// base.
func TestExecuteRemovesThroughInRootDirAlias(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		target func(root string) string
	}{
		{"absolute", func(root string) string { return filepath.Join(root, "inside") }},
		{"relative step-out", func(root string) string {
			return filepath.Join("..", filepath.Base(root), "inside")
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := filepath.Join(t.TempDir(), "proj")
			file := filepath.Join(root, "inside", "x")
			writeFile(t, file, []byte("x\n"), 0o644)
			testutil.SymlinkOrSkip(t, tt.target(root), filepath.Join(root, "alias"))

			plan := &TeardownPlan{Remove: []FileAction{{Path: "alias/x"}}}
			result, err := Execute(plan, TeardownOptions{ProjectRoot: root}, toolreg.NewRegistry())
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Errors) != 0 || len(result.Removed) != 1 {
				t.Fatalf("Removed = %v, Errors = %v, want alias/x removed", result.Removed, result.Errors)
			}
			if _, err := os.Stat(file); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("inside/x still present (stat err=%v)", err)
			}
		})
	}
}

// TestExecuteDanglingAncestorIsSilent verifies a tracked file behind a
// dangling directory symlink, which therefore cannot exist, is skipped with
// no error, as on base.
func TestExecuteDanglingAncestorIsSilent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		link   string // the dangling symlink, relative to the project
		target func(t *testing.T) string
	}{
		{"in root", ".claude", func(*testing.T) string { return "missing" }},
		{"outside", ".claude", func(t *testing.T) string { return filepath.Join(t.TempDir(), "missing") }},
		// A parent dangling into .git cannot hold the file either: it is
		// skipped like any dangling parent, not refused as a redirect.
		{"parent into .git", filepath.Join(".claude", "hooks"), func(*testing.T) string {
			return filepath.Join("..", ".git", "newdir")
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
				t.Fatal(err)
			}
			testutil.SymlinkOrSkip(t, tt.target(t), filepath.Join(root, tt.link))

			plan := &TeardownPlan{Remove: []FileAction{{Path: ".claude/hooks/pre-commit"}}}
			result, err := Execute(plan, TeardownOptions{ProjectRoot: root}, toolreg.NewRegistry())
			if err != nil {
				t.Fatal(err)
			}
			requireNoChange(t, result)
		})
	}
}

// TestExecuteUnlinksOutsideLeafSymlink verifies a tracked file that is itself
// a symlink out of the project is unlinked, not followed: the link goes and
// its outside target survives.
func TestExecuteUnlinksOutsideLeafSymlink(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "keep.txt")
	writeFile(t, outside, []byte("keep\n"), 0o644)
	link := filepath.Join(root, ".claude", "hooks", "guard.sh")
	testutil.SymlinkOrSkip(t, outside, link)

	plan := &TeardownPlan{Remove: []FileAction{{Path: ".claude/hooks/guard.sh"}}}
	result, err := Execute(plan, TeardownOptions{ProjectRoot: root}, toolreg.NewRegistry())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) != 0 || len(result.Removed) != 1 {
		t.Fatalf("Removed = %v, Errors = %v", result.Removed, result.Errors)
	}
	if _, err := os.Lstat(link); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("link still present (lstat err=%v)", err)
	}
	if _, err := os.Stat(outside); err != nil {
		t.Errorf("outside target was removed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".claude")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("emptied .claude not pruned (stat err=%v)", err)
	}
}

// TestExecuteSymlinkDotDotIntoGit verifies a tracked path that spells a
// symlink followed by ".." (.claude/hooks/../config with .claude/hooks ->
// ../.git/hooks) removes the lexically cleaned .claude/config, never
// .git/config.
func TestExecuteSymlinkDotDotIntoGit(t *testing.T) {
	t.Parallel()
	root, _ := gitRedirectProject(t)
	gitConfig := filepath.Join(root, ".git", "config")
	writeFile(t, gitConfig, []byte("[core]\n"), 0o644)
	claudeConfig := filepath.Join(root, ".claude", "config")
	writeFile(t, claudeConfig, []byte("x\n"), 0o644)

	plan := &TeardownPlan{Remove: []FileAction{{Path: ".claude/hooks/../config"}}}
	result, err := Execute(plan, TeardownOptions{ProjectRoot: root}, testRegistry())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Errors = %v, want none", result.Errors)
	}
	if _, err := os.Stat(gitConfig); err != nil {
		t.Errorf(".git/config was removed: %v", err)
	}
	if _, err := os.Stat(claudeConfig); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf(".claude/config still present (stat err=%v)", err)
	}
}

// TestExecuteRemoveErrorWording verifies only a containment refusal is
// reported as "refusing to remove"; an ordinary removal failure (here a
// non-empty directory) says "removing".
func TestExecuteRemoveErrorWording(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "full", "child.txt"), []byte("x\n"), 0o644)

	plan := &TeardownPlan{Remove: []FileAction{{Path: "full"}}}
	result, err := Execute(plan, TeardownOptions{ProjectRoot: root}, testRegistry())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) != 1 {
		t.Fatalf("Errors = %v, want one", result.Errors)
	}
	if msg := result.Errors[0].Error(); !strings.HasPrefix(msg, "removing full: ") {
		t.Errorf("error = %q, want prefix %q", msg, "removing full: ")
	}
	if errors.Is(result.Errors[0], fileutil.ErrOutsideRoot) {
		t.Errorf("error %v wraps ErrOutsideRoot", result.Errors[0])
	}
}

// TestCleanSharedFileDanglingOutsideLinkUnchanged verifies a shared file that
// is a dangling symlink out of the project is left alone with no error, as on
// base: there is no content behind it to clean.
func TestCleanSharedFileDanglingOutsideLinkUnchanged(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	testutil.SymlinkOrSkip(t, filepath.Join(t.TempDir(), "missing.md"), filepath.Join(root, "shared.md"))

	got, err := cleanSharedFile(root, FileAction{Path: "shared.md"}, testRegistry())
	if err != nil {
		t.Fatalf("cleanSharedFile: %v", err)
	}
	if got != cleanUnchanged {
		t.Errorf("outcome = %v, want cleanUnchanged", got)
	}
}

// TestCleanSharedFileScaffoldOnlyRemovesSymlinkTarget verifies a scaffold-only
// CLAUDE.md -> AGENTS.md deletes AGENTS.md, the file holding the content, as
// on base.
func TestCleanSharedFileScaffoldOnlyRemovesSymlinkTarget(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	agents := filepath.Join(root, "AGENTS.md")
	writeFile(t, agents, scaffoldOnly("CLAUDE.md"), 0o644)
	testutil.SymlinkOrSkip(t, "AGENTS.md", filepath.Join(root, "CLAUDE.md"))

	got, err := cleanSharedFile(root, FileAction{Path: "CLAUDE.md"}, testRegistry())
	if err != nil {
		t.Fatalf("cleanSharedFile: %v", err)
	}
	if got != cleanRemoved {
		t.Errorf("outcome = %v, want cleanRemoved", got)
	}
	if _, err := os.Stat(agents); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("AGENTS.md still present (stat err=%v)", err)
	}
}

// TestCollectAllFilePathsArchivesReadableFiles verifies the --archive list
// keeps every tracked file it can read inside the project, including one
// behind a .git redirect that teardown refuses to remove (as on base) and one
// behind a dangling link (the archive skips it), and drops only unsafe paths
// and those whose parent escapes the project.
func TestCollectAllFilePathsArchivesReadableFiles(t *testing.T) {
	t.Parallel()
	root, _ := gitRedirectProject(t)
	outside := t.TempDir()
	testutil.SymlinkOrSkip(t, outside, filepath.Join(root, ".github"))
	testutil.SymlinkOrSkip(t, "missing", filepath.Join(root, "gone"))
	writeFile(t, filepath.Join(root, "plain.txt"), []byte("x\n"), 0o644)

	st := types.GeneratedState{Files: map[string]types.FileState{
		"plain.txt":                {},
		".claude/hooks/pre-commit": {},
		"gone/x":                   {},
		".github/escape.yml":       {},
		"../outside.txt":           {},
	}}
	got := collectAllFilePaths(st, root)
	slices.Sort(got)
	want := []string{".claude/hooks/pre-commit", "gone/x", "plain.txt"}
	if !slices.Equal(got, want) {
		t.Errorf("collectAllFilePaths = %v, want %v", got, want)
	}
}

// aliasedRelativeLinkProject returns a project with a -> sub/deep and
// sub/deep/link -> ../target, so the OS resolves a/link/<name> to
// sub/target/<name>. That file holds real and a decoy target/<name>, where a
// lexical join of the relative link onto a/ would land, holds "decoy".
func aliasedRelativeLinkProject(t *testing.T, name string, real []byte) (root, realPath, decoyPath string) {
	t.Helper()
	root = t.TempDir()
	realPath = filepath.Join(root, "sub", "target", name)
	decoyPath = filepath.Join(root, "target", name)
	writeFile(t, realPath, real, 0o644)
	writeFile(t, decoyPath, []byte("decoy\n"), 0o644)
	testutil.SymlinkOrSkip(t, filepath.Join("sub", "deep"), filepath.Join(root, "a"))
	testutil.SymlinkOrSkip(t, filepath.Join("..", "target"), filepath.Join(root, "sub", "deep", "link"))
	return root, realPath, decoyPath
}

// requireFile fails unless path holds want.
func requireFile(t *testing.T, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	if string(got) != string(want) {
		t.Errorf("%s = %q, want %q", path, got, want)
	}
}

// TestExecuteRemovesThroughRelativeLinkInAliasedDir verifies a tracked path
// whose parent is a relative symlink inside a symlinked directory removes the
// file the OS resolves it to, as on base, and leaves the decoy a lexical join
// would name alone.
func TestExecuteRemovesThroughRelativeLinkInAliasedDir(t *testing.T) {
	t.Parallel()
	root, realPath, decoyPath := aliasedRelativeLinkProject(t, "v", []byte("tracked\n"))

	plan := &TeardownPlan{Remove: []FileAction{{Path: "a/link/v"}}}
	result, err := Execute(plan, TeardownOptions{ProjectRoot: root}, toolreg.NewRegistry())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) != 0 || len(result.Removed) != 1 {
		t.Fatalf("Removed = %v, Errors = %v, want a/link/v removed", result.Removed, result.Errors)
	}
	if _, err := os.Stat(realPath); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("sub/target/v still present (stat err=%v)", err)
	}
	requireFile(t, decoyPath, []byte("decoy\n"))
}

// TestCleanSharedFileThroughRelativeLinkInAliasedDir verifies a nested shared
// file that is itself a relative symlink inside a symlinked directory
// (a -> sub/deep, sub/deep/NOTES.md -> ../target/NOTES.md) is cleaned, or
// deleted when only the scaffold is left, at the file the OS resolves it to
// (sub/target/NOTES.md), as on base; the decoy target/NOTES.md is untouched.
func TestCleanSharedFileThroughRelativeLinkInAliasedDir(t *testing.T) {
	t.Parallel()
	const name = "NOTES.md"
	sharedLink := func(t *testing.T, content []byte) (root, realPath, decoyPath string) {
		t.Helper()
		root, realPath, decoyPath = aliasedRelativeLinkProject(t, name, content)
		testutil.SymlinkOrSkip(t, filepath.Join("..", "target", name), filepath.Join(root, "sub", "deep", name))
		return root, realPath, decoyPath
	}
	block := "<!-- BEGIN GENERATED SECTION -->\ngen\n<!-- END GENERATED SECTION -->\n"
	tests := []struct {
		name     string
		content  []byte
		want     cleanOutcome
		wantRest string // "" means the file is deleted
	}{
		{"scaffold only is deleted", scaffoldOnly(name), cleanRemoved, ""},
		{"user content is rewritten", []byte("user notes\n" + block), cleanRewritten, "user notes"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root, realPath, decoyPath := sharedLink(t, tt.content)

			got, err := cleanSharedFile(root, FileAction{Path: "a/" + name}, testRegistry())
			if err != nil {
				t.Fatalf("cleanSharedFile: %v", err)
			}
			if got != tt.want {
				t.Errorf("outcome = %v, want %v", got, tt.want)
			}
			requireFile(t, decoyPath, []byte("decoy\n"))
			if tt.wantRest == "" {
				if _, err := os.Stat(realPath); !errors.Is(err, fs.ErrNotExist) {
					t.Errorf("sub/target/%s still present (stat err=%v)", name, err)
				}
				return
			}
			content, err := os.ReadFile(realPath)
			if err != nil {
				t.Fatal(err)
			}
			if !contains(content, tt.wantRest) || contains(content, "GENERATED") {
				t.Errorf("sub/target/%s = %q, want the user content without the generated block", name, content)
			}
		})
	}
}
