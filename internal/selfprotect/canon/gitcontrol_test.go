package canon

import (
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/pathmatch"
)

// allMatchOptions are the four pathmatch.Options a protected-path check can
// run under.
var allMatchOptions = []pathmatch.Options{
	{},
	{FoldCase: true},
	{WindowsAliases: true},
	{FoldCase: true, WindowsAliases: true},
}

// TestGitControlProtected pins the structural git-control matcher: inside a
// .git directory (and the nested git dirs of submodules and linked worktrees)
// the hooks subtree, config, config.worktree and info/attributes are
// protected, because each can turn off the commit gate or make git run code.
// Refs and logs are not, so a branch named config or hooks/x stays writable.
func TestGitControlProtected(t *testing.T) {
	t.Parallel()
	resetProtectedPaths(t)

	tests := []struct {
		rel  string // slash-separated, below /work/repo
		want bool
	}{
		{".git/hooks/pre-commit", true},
		{".git/hooks", true},
		{".git/hooks/sub/x", true},
		{".git/config", true},
		{".git/config.worktree", true},
		{".git/info/attributes", true},
		{".git/info", true},
		{".git/modules/sub/hooks/pre-commit", true},
		{".git/modules/sub/config", true},
		{".git/modules/a/b/config", true},
		{".git/modules/a/b/hooks/x", true},
		{".git/modules/a/modules/b/config", true},
		{".git/modules/refs/hooks/x", true}, // a submodule named refs
		{".git/worktrees/w/config.worktree", true},
		{".git/worktrees/w/config", true},
		{".git/worktrees/w/modules/s/hooks/x", true},
		{"vendor/x/.git/hooks/pre-commit", true},

		{".git", false},
		{".git/HEAD", false},
		{".git/index.lock", false},
		{".git/info/exclude", false},
		{".git/refs/heads/config", false},
		{".git/refs/heads/hooks/x", false},
		{".git/logs/HEAD", false},
		{".git/logs/refs/heads/hooks/x", false},
		{".git/logs/refs/heads/config", false},
		{".git/modules/sub/refs/heads/config", false},
		{".git/modules/sub/logs/refs/heads/hooks/x", false},
		{".git/modules/sub/HEAD", false},
		{".git/worktrees/w/HEAD", false},
		{".git/worktrees/w", false},
		{".git/config.bak", false},
		{".git/hooksx/pre-commit", false},
		{"src/hooks/pre-commit", false},
		{"config", false},
		{"foo.git/hooks/pre-commit", false},
		{".github/hooks/pre-commit", false},
		{"x.git/config", false},
	}
	for _, opts := range allMatchOptions {
		for _, tt := range tests {
			t.Run(tt.rel+"/"+optsName(opts), func(t *testing.T) {
				t.Parallel()
				p := filepath.Join(filepath.FromSlash("/work/repo"), filepath.FromSlash(tt.rel))
				got, cat := isProtected(p, opts)
				if got != tt.want {
					t.Fatalf("isProtected(%q, %+v) = (%v, %q), want %v", p, opts, got, cat, tt.want)
				}
				if got && cat != GitControlCategory {
					t.Errorf("isProtected(%q, %+v) category = %q, want %q", p, opts, cat, GitControlCategory)
				}
			})
		}
	}

	// Spellings a case-folding filesystem or Windows resolves to a protected
	// file are protected only under the options that model it.
	exact := pathmatch.Options{}
	fold := pathmatch.Options{FoldCase: true}
	win := pathmatch.Options{FoldCase: true, WindowsAliases: true}
	aliases := []struct {
		rel  string
		opts pathmatch.Options
		want bool
	}{
		{".GIT/HOOKS/pre-commit", fold, true},
		{".Git/Config", fold, true},
		{".GIT/modules/Sub/CONFIG", fold, true},
		{".GIT/HOOKS/pre-commit", exact, false},
		{".git/config::$DATA", win, true},
		{".git./hooks /pre-commit", win, true},
		{".git/info./attributes.", win, true},
		{".git/config::$DATA", fold, false},
		{".git/config::$DATA", exact, false},
	}
	for _, tt := range aliases {
		t.Run("alias "+tt.rel+"/"+optsName(tt.opts), func(t *testing.T) {
			t.Parallel()
			p := filepath.Join(filepath.FromSlash("/work/repo"), filepath.FromSlash(tt.rel))
			if got, _ := isProtected(p, tt.opts); got != tt.want {
				t.Errorf("isProtected(%q, %+v) = %v, want %v", p, tt.opts, got, tt.want)
			}
		})
	}
}

// TestGitControlProtected_Relative pins that a relative key (a path the
// rules resolve from an unknown working directory) is matched the same way.
func TestGitControlProtected_Relative(t *testing.T) {
	t.Parallel()
	resetProtectedPaths(t)
	for _, rel := range []string{".git/hooks/pre-commit", ".git/config", "sub/.git/info/attributes"} {
		if got, _ := IsProtected(filepath.FromSlash(rel)); !got {
			t.Errorf("IsProtected(%q) = false, want true", rel)
		}
	}
}

func optsName(o pathmatch.Options) string {
	var parts []string
	if o.FoldCase {
		parts = append(parts, "fold")
	}
	if o.WindowsAliases {
		parts = append(parts, "win")
	}
	if len(parts) == 0 {
		return "exact"
	}
	return strings.Join(parts, "+")
}

// TestGitGlobalConfigEntries pins the home-anchored git files that apply to
// every repository: ~/.gitconfig, and the XDG git config and attributes under
// both $XDG_CONFIG_HOME and the ~/.config default (a command can unset the
// variable to make git read the default one).
func TestGitGlobalConfigEntries(t *testing.T) {
	t.Parallel()
	home := filepath.FromSlash("/home/alice")
	xdg := filepath.FromSlash("/xdg/cfg")
	tests := []struct {
		name string
		xdg  string
		want []string
	}{
		{"xdg unset", "", []string{
			filepath.Join(home, ".gitconfig"),
			filepath.Join(home, ".config", "git", "config"),
			filepath.Join(home, ".config", "git", "attributes"),
		}},
		{"xdg set", xdg, []string{
			filepath.Join(home, ".gitconfig"),
			filepath.Join(home, ".config", "git", "config"),
			filepath.Join(home, ".config", "git", "attributes"),
			filepath.Join(xdg, "git", "config"),
			filepath.Join(xdg, "git", "attributes"),
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			getenv := func(k string) string {
				if k == "XDG_CONFIG_HOME" {
					return tt.xdg
				}
				return ""
			}
			var got []string
			for _, e := range gitGlobalConfigEntries(home, getenv) {
				if e.category != GitControlCategory {
					t.Errorf("entry %q category = %q, want %q", e.path, e.category, GitControlCategory)
				}
				got = append(got, e.path)
			}
			for _, w := range tt.want {
				w, err := filepath.Abs(w)
				if err != nil {
					t.Fatalf("Abs(%q): %v", w, err)
				}
				if !slices.Contains(got, w) {
					t.Errorf("entries %q lack %q", got, w)
				}
			}
		})
	}
}

// TestGlobalGitConfigProtected pins that the global git configuration, which
// can set core.hooksPath for every repository, is protected under the HOME
// and XDG_CONFIG_HOME the hook sees, while its neighbours stay writable.
//
// It is sequential because it re-runs ensureInit with a different
// environment; the parallel tests resume only after its cleanup.
func TestGlobalGitConfigProtected(t *testing.T) {
	resetInit := func() {
		initOnce = sync.Once{}
		initErr = nil
		protectedPrefixes = nil
		protectedSuffixes = nil
	}
	t.Cleanup(resetInit)

	for _, xdgSet := range []bool{true, false} {
		name := "xdg-unset"
		if xdgSet {
			name = "xdg-set"
		}
		t.Run(name, func(t *testing.T) {
			base, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatalf("resolving temp dir: %v", err)
			}
			home := filepath.Join(base, "home")
			xdg := filepath.Join(base, "xdg")
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			if xdgSet {
				t.Setenv("XDG_CONFIG_HOME", xdg)
			} else {
				t.Setenv("XDG_CONFIG_HOME", "")
			}
			resetInit()

			protected := []string{
				filepath.Join(home, ".gitconfig"),
				filepath.Join(home, ".config", "git", "config"),
				filepath.Join(home, ".config", "git", "attributes"),
			}
			if got := ShellPathVars()["XDG_CONFIG_HOME"]; xdgSet && got != xdg || !xdgSet && got != "" {
				t.Errorf("ShellPathVars()[XDG_CONFIG_HOME] = %q, want the hook's value", got)
			}
			if xdgSet {
				protected = append(protected,
					filepath.Join(xdg, "git", "config"),
					filepath.Join(xdg, "git", "attributes"))
			}
			for _, p := range protected {
				if got, cat := IsProtected(p); !got || cat != GitControlCategory {
					t.Errorf("IsProtected(%q) = (%v, %q), want (true, %q)", p, got, cat, GitControlCategory)
				}
			}
			for _, p := range []string{
				filepath.Join(home, ".gitconfig.bak"),
				filepath.Join(home, ".config", "git", "ignore"),
				filepath.Join(xdg, "git", "ignore"),
			} {
				if got, cat := IsProtected(p); got {
					t.Errorf("IsProtected(%q) = (true, %q), want false", p, cat)
				}
			}
		})
	}
}

// gitControlExamples are representative git-control locations, below a
// project (relative) or the home directory (~/), for the raw-command
// coverage check.
var gitControlExamples = []string{
	".git/hooks/pre-commit",
	".git/config",
	".git/config.worktree",
	".git/info/attributes",
	".git/modules/sub/hooks/pre-commit",
	".git/modules/a/b/config",
	".git/worktrees/w/config.worktree",
	"~/.gitconfig",
	"~/.config/git/config",
	"~/.config/git/attributes",
}

// checkGitControlCoveredByCommandScan is the git-control part of
// TestProtectedSegmentsCoveredByCommandScan: the structural matcher protects
// these locations rather than a segment, and each must also be recognised by
// the raw-command scan, or a shell command could reach a file the Write tool
// cannot.
func checkGitControlCoveredByCommandScan(t *testing.T) {
	t.Parallel()
	resetProtectedPaths(t)
	for _, ex := range gitControlExamples {
		t.Run(ex, func(t *testing.T) {
			t.Parallel()
			p := ex
			if !strings.HasPrefix(ex, "~/") {
				p = "/work/repo/" + ex
			}
			if !ContainsProtectedPath("echo x > " + p) {
				t.Errorf("ContainsProtectedPath(%q) = false, but %q is git-control", p, ex)
			}
			if strings.HasPrefix(ex, "~/") {
				return
			}
			if got, _ := IsProtected(filepath.FromSlash(p)); !got {
				t.Errorf("IsProtected(%q) = false; the example is not git-control", p)
			}
		})
	}
	// Mentions of a git dir that name no control file are not a trigger.
	for _, s := range []string{
		"ls .git", "cat .git/HEAD", "du -sh .git", "git --git-dir=.git status",
		"echo x >> .git/info/exclude", "rm -f .git/index.lock",
	} {
		if ContainsProtectedPath(s) {
			t.Errorf("ContainsProtectedPath(%q) = true, want false", s)
		}
	}
}

// TestFindProbes_CoverGitControl pins that a find name pattern is tried
// against the git-control files: the hooks directory with a pre-commit hook
// in it, and the config and attributes files.
func TestFindProbes_CoverGitControl(t *testing.T) {
	t.Parallel()
	relative, _ := FindProbes()
	for _, want := range []string{".git/hooks/pre-commit", ".git/hooks", ".git/config", ".git/config.worktree", ".git/info/attributes"} {
		if !slices.Contains(relative, want) {
			t.Errorf("relative probes lack %q", want)
		}
	}
}

// TestGitControlReach pins the matcher rules use for globs and for whole git
// directories: a glob path reaches a control location when its components can
// match one (`.gi?/hooks/x`, `.git/*/pre-commit`), and a holder is a .git
// directory, its modules or worktrees directory or a nested git dir, but not
// a file in one (index.lock) nor a ref.
func TestGitControlReach(t *testing.T) {
	t.Parallel()
	glob := func(component, name string) bool {
		if strings.HasPrefix(name, ".") && !strings.HasPrefix(component, ".") {
			return false // the shell's leading-dot rule
		}
		ok, err := pathmatch.ShellMatch(component, name)
		return ok || err != nil
	}
	tests := []struct {
		path            string
		globbed         bool
		control, holder bool
	}{
		{".git", false, false, true},
		{"/home/u/project/.git", false, false, true},
		{".git/", false, false, true},
		{".git/hooks", false, true, false},
		{".git/hooks/pre-commit", false, true, false},
		{".git/config", false, true, false},
		{".git/info", false, true, false},
		{".git/info/attributes", false, true, false},
		{".git/modules", false, false, true},
		{".git/modules/sub", false, false, true},
		{".git/modules/sub/hooks/pre-commit", false, true, false},
		{".git/modules/a/b/config", false, true, false},
		{".git/worktrees", false, false, true},
		{".git/worktrees/w", false, false, true},
		{".git/worktrees/w/config.worktree", false, true, false},
		{".git/index.lock", false, false, false},
		{".git/HEAD", false, false, false},
		{".git/info/exclude", false, false, false},
		{".git/refs/heads/config", false, false, false},
		{".git/modules/sub/index.lock", false, false, false},
		{".git/modules/sub/refs/heads/config", false, false, false},
		{".gitignore", false, false, false},
		{"repo.git", false, false, false},
		{"hooks/pre-commit", false, false, false},
		{".gi?/hooks/pre-commit", true, true, false},
		{".git/*/pre-commit", true, true, true}, // also .git/modules/pre-commit
		{".git/hoo*/pre-commit", true, true, false},
		{".[g]it/hooks/pre-commit", true, true, false},
		{".[!x]it/hooks/pre-commit", true, true, false},
		{".[[:alpha:]]it/hooks/pre-commit", true, true, false},
		{".[!.]*", true, false, true},
		{".git/*", true, true, true},
		{".gi?", true, false, true},
		{"*/.git", true, false, true},
		{".git/*.lock", true, false, false},
		{"*/hooks/pre-commit", true, false, false},
		{".g*", true, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			match := equalName
			if tt.globbed {
				match = glob
			}
			control, holder := GitControlReach(tt.path, match)
			if control != tt.control || holder != tt.holder {
				t.Errorf("GitControlReach(%q) = (%v, %v), want (%v, %v)", tt.path, control, holder, tt.control, tt.holder)
			}
		})
	}
}

// TestIsGitControl_NoAllocs pins that the literal matcher IsProtected runs on
// every path does not allocate. Not parallel: AllocsPerRun counts every
// allocation in the process.
func TestIsGitControl_NoAllocs(t *testing.T) {
	for _, key := range []string{"/home/u/project/.git/modules/a/b/hooks/pre-commit", "/home/u/project/src/main.go"} {
		if got := testing.AllocsPerRun(100, func() { isGitControl(key) }); got != 0 {
			t.Errorf("isGitControl(%q) allocates %v times, want 0", key, got)
		}
	}
}

// TestMentionsGitDir pins the raw-text trigger for a git directory: a .git
// path component, not a longer name.
func TestMentionsGitDir(t *testing.T) {
	t.Parallel()
	for s, want := range map[string]bool{
		"d=.git; rm $d/hooks/x":     true,
		"rm -f .git/index.lock":     true,
		"cd repo/.git":              true,
		`"$(printf .git)/hooks"`:    true,
		"cat .gitignore":            false,
		"ls .github/workflows":      false,
		"git clone host:x/repo.git": false,
		"git status":                false,
	} {
		if got := MentionsGitDir(s); got != want {
			t.Errorf("MentionsGitDir(%q) = %v, want %v", s, got, want)
		}
	}
}
