package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/projectctx"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// userDirVars are the variables IsolateUserDirs must point away from the
// host's values.
var userDirVars = []string{
	"HOME", "USERPROFILE",
	"XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_STATE_HOME", "XDG_CACHE_HOME",
	"APPDATA", "LocalAppData",
	"TMPDIR", "TMP", "TEMP",
	"GIT_CONFIG_GLOBAL",
}

// plantTMPDIR makes a trusted (0755, own-user) directory holding the project
// markers a shared machine may leave in a temp directory, and points
// TMPDIR, TMP and TEMP at it, so every later t.TempDir of the calling test
// lands below the markers. It must run before the test's first t.TempDir:
// testing creates the per-test temp root once, on first use.
func plantTMPDIR(t *testing.T) string {
	t.Helper()
	planted, err := os.MkdirTemp("", "planted-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(planted) })
	if planted, err = filepath.EvalSymlinks(planted); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(planted, 0o755); err != nil {
		t.Fatal(err)
	}
	b := branding.Get()
	if err := os.WriteFile(filepath.Join(planted, b.ConfigFile), []byte("version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(planted, b.StateDir), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, v := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(v, planted)
	}
	return planted
}

// requireUnder fails t unless path lies strictly below dir.
func requireUnder(t *testing.T, what, path, dir string) {
	t.Helper()
	rel, err := filepath.Rel(dir, path)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		t.Fatalf("%s = %q, want a path below %q", what, path, dir)
	}
}

func requireDir(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		t.Fatalf("%s is not a directory (err %v)", path, err)
	}
}

func requireAbsent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("%s exists, want it absent (err %v)", path, err)
	}
}

// TestIsolatedDir_IgnoresPlantedAncestor is the XA-WS12 core regression: a
// temp directory below a trusted ancestor holding project markers resolves
// to no project, because IsolatedDir puts a git ceiling between them.
func TestIsolatedDir_IgnoresPlantedAncestor(t *testing.T) {
	planted := plantTMPDIR(t)
	dir := IsolatedDir(t)
	requireUnder(t, "IsolatedDir()", dir, planted)

	for _, mode := range []projectctx.Mode{projectctx.Enclosing, projectctx.Here} {
		pc, err := projectctx.Resolve(dir, mode)
		if err != nil {
			t.Fatalf("Resolve(%q, %v): %v", dir, mode, err)
		}
		if pc.Found || pc.Root != dir {
			t.Errorf("Resolve(%q, %v) = {Root: %q, Found: %v}, want {Root: %q, Found: false}", dir, mode, pc.Root, pc.Found, dir)
		}
	}
}

// TestIsolatedDir_ReturnsResolvedDirUnderGitCeiling pins the shape tests
// rely on: an empty, symlink-resolved directory without a .git of its own,
// whose parent is the repository toplevel that ends every walk.
func TestIsolatedDir_ReturnsResolvedDirUnderGitCeiling(t *testing.T) {
	t.Parallel()
	dir := IsolatedDir(t)
	requireDir(t, dir)
	if resolved, err := filepath.EvalSymlinks(dir); err != nil || resolved != dir {
		t.Errorf("IsolatedDir() = %q, want its symlink-resolved form %q (err %v)", dir, resolved, err)
	}
	if entries, err := os.ReadDir(dir); err != nil || len(entries) != 0 {
		t.Errorf("IsolatedDir() is not empty: %v (err %v)", entries, err)
	}
	requireAbsent(t, filepath.Join(dir, ".git"))
	ceiling := filepath.Dir(dir)
	for _, d := range []string{"objects", "refs"} {
		requireDir(t, filepath.Join(ceiling, ".git", d))
	}
	if head, err := os.ReadFile(filepath.Join(ceiling, ".git", "HEAD")); err != nil || !strings.HasPrefix(string(head), "ref: refs/heads/") {
		t.Errorf("ceiling HEAD = %q (err %v), want a symbolic ref", head, err)
	}

	pc, err := projectctx.Resolve(dir, projectctx.Enclosing)
	if err != nil {
		t.Fatal(err)
	}
	if pc.GitTop != ceiling || pc.Found {
		t.Errorf("Resolve(%q) = {GitTop: %q, Found: %v}, want {GitTop: %q, Found: false}", dir, pc.GitTop, pc.Found, ceiling)
	}
}

// TestIsolatedDir_ParallelSafe checks that IsolatedDir touches no
// process-wide state (t.Setenv and t.Chdir panic in parallel tests) and
// hands every caller its own directory.
func TestIsolatedDir_ParallelSafe(t *testing.T) {
	t.Parallel()
	dirs := make(chan string, 4)
	t.Run("group", func(t *testing.T) {
		for _, name := range []string{"a", "b", "c", "d"} {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				dir := IsolatedDir(t)
				if err := os.WriteFile(filepath.Join(dir, branding.Get().ConfigFile), []byte("version: 1\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				pc, err := projectctx.Resolve(dir, projectctx.Enclosing)
				if err != nil || !pc.Found || pc.Root != dir {
					t.Errorf("Resolve(%q) = %+v (err %v), want the directory's own marker", dir, pc, err)
				}
				dirs <- dir
			})
		}
	})
	close(dirs)
	seen := map[string]bool{}
	for d := range dirs {
		if seen[d] {
			t.Errorf("IsolatedDir returned %q twice", d)
		}
		seen[d] = true
	}
}

// TestIsolateUserDirs_PointsEveryUserDirAtTemp checks that every per-user
// directory variable moves to a fresh directory that exists, away from the
// host's value, and that the application's own per-user directories follow.
func TestIsolateUserDirs_PointsEveryUserDirAtTemp(t *testing.T) {
	host := map[string]string{}
	for _, v := range userDirVars {
		host[v] = os.Getenv(v)
	}
	home := IsolateUserDirs(t)
	requireDir(t, home)
	root := filepath.Dir(home)
	for _, v := range userDirVars {
		got := os.Getenv(v)
		if got == host[v] {
			t.Errorf("%s = %q, still the host value", v, got)
		}
		requireUnder(t, v, got, root)
		if v != "GIT_CONFIG_GLOBAL" {
			requireDir(t, got)
		}
	}
	if got := os.Getenv("GIT_CONFIG_NOSYSTEM"); got != "1" {
		t.Errorf("GIT_CONFIG_NOSYSTEM = %q, want 1", got)
	}
	if got := os.TempDir(); got != os.Getenv("TMPDIR") && got != os.Getenv("TMP") {
		t.Errorf("os.TempDir() = %q, want the isolated temp directory", got)
	}

	d, err := projectctx.UserDirs()
	if err != nil {
		t.Fatalf("UserDirs: %v", err)
	}
	if d.Home != home {
		t.Errorf("UserDirs().Home = %q, want %q", d.Home, home)
	}
	for what, p := range map[string]string{"State": d.State, "Cache": d.Cache, "Legacy": d.Legacy} {
		requireUnder(t, "UserDirs()."+what, p, home)
	}
}

// TestProject_IsolatesUserDirsAndChdirs checks the full fixture: an isolated
// directory that is its own repository toplevel, the working directory, and
// isolated per-user directories, even below a planted temp directory.
func TestProject_IsolatesUserDirsAndChdirs(t *testing.T) {
	planted := plantTMPDIR(t)
	hostHome := os.Getenv("HOME")
	dir := Project(t, ProjectOptions{})
	requireUnder(t, "Project()", dir, planted)

	if wd, err := os.Getwd(); err != nil || wd != dir {
		t.Errorf("os.Getwd() = %q (err %v), want %q", wd, err, dir)
	}
	requireDir(t, filepath.Join(dir, ".git"))
	home := os.Getenv("HOME")
	if home == hostHome || home == "" {
		t.Errorf("HOME = %q, not isolated", home)
	}
	if d, err := projectctx.UserDirs(); err != nil || d.Home != home {
		t.Errorf("UserDirs() = %+v (err %v), want Home %q", d, err, home)
	}

	pc, err := projectctx.ResolveWorkingDir(projectctx.Enclosing)
	if err != nil {
		t.Fatal(err)
	}
	if pc.Found || pc.Root != dir || pc.GitTop != dir {
		t.Errorf("ResolveWorkingDir() = %+v, want Root and GitTop %q, Found false", pc, dir)
	}

	// The minimal repository is one git itself accepts as the toplevel.
	if git, err := exec.LookPath("git"); err == nil {
		out, err := exec.Command(git, "rev-parse", "--show-toplevel").Output()
		if err != nil {
			t.Fatalf("git rev-parse --show-toplevel in %s: %v", dir, err)
		}
		if got, _ := filepath.EvalSymlinks(strings.TrimSpace(string(out))); got != dir {
			t.Errorf("git toplevel = %q, want %q", got, dir)
		}
	}
}

// TestProject_NoGit checks that NoGit leaves the project directory without a
// repository of its own; the isolation ceiling above it still ends the walk.
func TestProject_NoGit(t *testing.T) {
	planted := plantTMPDIR(t)
	dir := Project(t, ProjectOptions{NoGit: true})
	requireUnder(t, "Project()", dir, planted)
	requireAbsent(t, filepath.Join(dir, ".git"))
	if wd, err := os.Getwd(); err != nil || wd != dir {
		t.Errorf("os.Getwd() = %q (err %v), want %q", wd, err, dir)
	}
	pc, err := projectctx.ResolveWorkingDir(projectctx.Enclosing)
	if err != nil {
		t.Fatal(err)
	}
	if pc.Found || pc.Root != dir || pc.GitTop != filepath.Dir(dir) {
		t.Errorf("ResolveWorkingDir() = %+v, want Root %q, GitTop its parent, Found false", pc, dir)
	}
}
