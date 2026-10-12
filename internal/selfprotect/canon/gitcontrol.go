package canon

import (
	"path/filepath"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/pathmatch"
)

// GitControlCategory is the protected category of the git files that decide
// whether the commit gate runs and what code git runs: the hooks, the
// repository and global config (core.hooksPath, filter and diff drivers) and
// the attributes that bind filter drivers to paths.
const GitControlCategory = "git-control"

// xdgConfigHomeEnv names the variable git reads its XDG config directory
// from (default ~/.config).
const xdgConfigHomeEnv = "XDG_CONFIG_HOME"

// gitControlEntries are the git-control locations inside a .git directory,
// slash-separated (a directory ends in "/"), as representative probes for a
// check that cannot see the filesystem (FindProbes). isGitControl is the
// matcher; it also finds them in the nested git dirs of submodules and linked
// worktrees.
var gitControlEntries = []string{
	".git/hooks/", ".git/config", ".git/config.worktree", ".git/info/attributes",
}

// gitControlSubstrings are the raw-command fragments of the git-control
// locations, for ContainsProtectedPath: those inside a .git directory
// (".git/config" also covers config.worktree; the modules and worktrees
// fragments cover their nested git dirs), and the global git config files
// ~/.gitconfig and ~/.config/git/{config,attributes}. A bare ".git" is not
// one: it would make every file directly in .git (index.lock, info/exclude)
// count as protected, and a cd into .git is still resolved by path.
var gitControlSubstrings = []string{
	".git/hooks/", ".git/config", ".git/info/attributes",
	".git/modules/", ".git/worktrees/",
	".gitconfig", ".config/git/config", ".config/git/attributes",
}

// gitGlobalConfigEntries returns the protected entries for the global git
// files that apply to every repository and can set core.hooksPath or bind
// filter drivers: ~/.gitconfig and the git config and attributes under
// $XDG_CONFIG_HOME and under its ~/.config default (git reads the default one
// when a command unsets the variable). Each is protected under its absolute
// and symlink-resolved spellings. getenv resolves XDG_CONFIG_HOME.
func gitGlobalConfigEntries(home string, getenv func(string) string) []protectedEntry {
	files := []string{filepath.Join(home, ".gitconfig")}
	for _, cfg := range []string{filepath.Join(home, ".config"), getenv(xdgConfigHomeEnv)} {
		if cfg == "" {
			continue
		}
		files = append(files, filepath.Join(cfg, "git", "config"), filepath.Join(cfg, "git", "attributes"))
	}
	var entries []protectedEntry
	for _, f := range files {
		for _, p := range spellings(f) {
			entries = append(entries, protectedEntry{p, GitControlCategory})
		}
	}
	return entries
}

// isGitControl reports whether the slash-separated path key (already
// case-folded and stripped of Windows aliases by pathmatch) names a
// git-control location inside a .git directory component (see
// GitControlReach). It scans key in place, since it runs on every path a rule
// checks.
func isGitControl(key string) bool {
	if !strings.Contains(key, ".git") {
		return false
	}
	control, _ := GitControlReach(key, equalName)
	return control
}

// MentionsGitDir reports whether the text s names a .git path component
// (`.git`, `.git/index.lock`, `d=.git`), not a longer name such as .gitignore
// or .github, for a scan that must notice a git directory a command may build
// a git-control path from (`d=.git; rm $d/hooks/pre-commit`).
func MentionsGitDir(s string) bool {
	return containsSegment(foldIf(filepath.ToSlash(s), pathmatch.Platform.FoldCase), ".git")
}

// equalName is the GitControlReach matcher of a literal path.
func equalName(component, name string) bool { return component == name }

// GitControlReach reports whether the slash-separated path p names a
// git-control location (control) or a directory that holds one (holder). A
// component of p stands for a name when match(component, name) holds: a
// literal path passes equality, a shell glob a glob matcher, so `.gi?/hooks/x`
// and `.git/*/pre-commit` reach a hook.
//
// The control locations are, inside any .git directory component, the hooks
// directory and anything in it, config and config.worktree, and info/attributes
// and the info directory that holds it, also in the git dir of a submodule
// (modules/<name>/, where the name may have any number of components) or a
// linked worktree (worktrees/<name>/). Refs and logs are not: their entries
// are branch names, so a branch named config or hooks/x is not protected. The
// holders are the .git directory itself, its modules and worktrees
// directories and the nested git dirs: deleting, moving or replacing one
// removes or replaces the hooks inside it without naming them.
func GitControlReach(p string, match func(component, name string) bool) (control, holder bool) {
	for rest := p; ; {
		comp, tail, more := strings.Cut(rest, "/")
		if match(comp, ".git") {
			c, h := gitDirReach(tail, match)
			control, holder = control || c, holder || h
		}
		if !more {
			return control, holder
		}
		rest = tail
	}
}

// gitDirReach is GitControlReach for rem, the slash-separated path below a
// git directory.
func gitDirReach(rem string, match func(component, name string) bool) (control, holder bool) {
	if rem == "" {
		return false, true // the git directory itself
	}
	first, rest, _ := strings.Cut(rem, "/")
	last := rest == ""
	if match(first, "hooks") ||
		last && (match(first, "config") || match(first, "config.worktree") || match(first, "info")) {
		control = true
	}
	if match(first, "info") {
		if second, tail, _ := strings.Cut(rest, "/"); tail == "" && match(second, "attributes") {
			control = true
		}
	}
	nested := func(c, h bool) { control, holder = control || c, holder || h }
	if match(first, "worktrees") {
		// worktrees/<name>/ is a linked worktree's git dir.
		_, inner, _ := strings.Cut(rest, "/")
		nested(gitDirReach(inner, match))
	}
	if match(first, "modules") {
		if last {
			holder = true
		} else {
			nested(submoduleReach(rest, match))
		}
	}
	return control, holder
}

// submoduleReach is gitDirReach for names, the path below .git/modules/. The
// submodule name is the first component and any further ones before the
// control location; a refs or logs component among those further ones means
// the remainder is a branch name inside the submodule's git dir
// (modules/sub/refs/heads/config), not a deeper submodule. A single component
// is a holder: it is, or holds, a submodule's git dir. A longer path is one
// only through its control locations, since its last component may as well
// be a file of the git dir above it (modules/sub/index.lock).
func submoduleReach(names string, match func(component, name string) bool) (control, holder bool) {
	i := strings.IndexByte(names, '/')
	if i < 0 {
		return false, true
	}
	for i >= 0 {
		if _, after, _ := strings.Cut(names[:i], "/"); hasComponent(after, "refs") || hasComponent(after, "logs") {
			return control, holder
		}
		c, h := gitDirReach(names[i+1:], match)
		control, holder = control || c, holder || h
		j := strings.IndexByte(names[i+1:], '/')
		if j < 0 {
			break
		}
		i += 1 + j
	}
	return control, holder
}

// hasComponent reports whether the slash-separated path p has the component
// comp.
func hasComponent(p, comp string) bool {
	for p != "" {
		var c string
		c, p, _ = strings.Cut(p, "/")
		if c == comp {
			return true
		}
	}
	return false
}
