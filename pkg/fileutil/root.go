package fileutil

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// ErrRedirectsIntoGit is returned, together with ErrOutsideRoot, when a
// symbolic link redirects a project-relative path into a .git metadata
// directory the path itself does not name (for example a committed
// .claude/hooks -> ../.git/hooks).
var ErrRedirectsIntoGit = errors.New("path redirects into .git")

// errDanglingSymlink marks a refusal caused by a symlink whose target is
// missing. Nothing can exist behind such a link, so RemoveInRoot reports it
// as fs.ErrNotExist instead of a containment failure.
var errDanglingSymlink = errors.New("dangling symlink")

// gitRedirectError reports a symlink redirect into .git metadata inside the
// root. It matches both ErrRedirectsIntoGit and ErrOutsideRoot with
// errors.Is, while its message does not claim the path left the root.
type gitRedirectError struct {
	rel, root, resolved string
}

func (e *gitRedirectError) Error() string {
	return fmt.Sprintf("%q under %s resolves to %s inside .git: refusing a symlink redirect into repository metadata",
		e.rel, e.root, e.resolved)
}

func (e *gitRedirectError) Unwrap() []error {
	return []error{ErrRedirectsIntoGit, ErrOutsideRoot}
}

// ResolveInRoot returns the absolute, symlink-resolved path that a write of
// rel, a path relative to root, lands on. It is the single containment rule
// for project-relative writes, removals and their pre-checks.
//
// rel must be local (relative, non-empty, no ".." escape) and is cleaned
// lexically first, so "d/link/../x" is judged, and resolved, as "d/x". A
// symlinked leaf is followed to the file it points to, which may not exist
// yet, so an in-root link such as CLAUDE.md -> AGENTS.md resolves to
// AGENTS.md. The result must lie inside root once every symlink is resolved,
// or the error wraps ErrOutsideRoot. A symlink that redirects rel into a .git
// directory rel does not name wraps both ErrRedirectsIntoGit and
// ErrOutsideRoot; an explicit path such as .git/info/exclude is allowed. A
// dangling symlink among the ancestors of the result is refused (wrapping
// ErrOutsideRoot): its target cannot be checked, and creating directories
// through it would land wherever it points.
func ResolveInRoot(root, rel string) (string, error) {
	_, resolved, err := resolveInRoot(root, rel)
	return resolved, err
}

// resolveInRoot implements ResolveInRoot and also returns the resolved root.
func resolveInRoot(root, rel string) (resolvedRoot, resolved string, err error) {
	if !filepath.IsLocal(rel) {
		return "", "", fmt.Errorf("resolving %q under %s: %w", rel, root, ErrOutsideRoot)
	}
	rel = filepath.Clean(rel)
	resolvedRoot, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", "", fmt.Errorf("resolving root %s: %w", root, err)
	}
	if rel == "." {
		// The root itself: following it as a leaf link would step to its
		// parent directory.
		return resolvedRoot, resolvedRoot, nil
	}
	joined := filepath.Join(root, rel)
	target, err := resolveWriteTarget(joined)
	if err != nil {
		return "", "", err
	}
	if err := refuseDanglingAncestors(filepath.Dir(target), rel); err != nil {
		return "", "", err
	}
	resolvedDir, err := resolveExistingPrefix(filepath.Dir(target))
	if err != nil {
		return "", "", err
	}
	resolved = filepath.Join(resolvedDir, filepath.Base(target))
	if !isWithin(resolvedRoot, resolvedDir) {
		if target != joined && !exists(target) {
			return "", "", fmt.Errorf("%q under %s is a dangling symlink to %s: %w: %w",
				rel, root, resolved, errDanglingSymlink, ErrOutsideRoot)
		}
		return "", "", fmt.Errorf("%q under %s resolves to %s: %w", rel, root, resolved, ErrOutsideRoot)
	}
	if redirectsIntoGitDir(resolvedRoot, resolvedDir, filepath.Dir(rel)) ||
		redirectsIntoGitDir(resolvedRoot, resolved, rel) {
		return "", "", &gitRedirectError{rel: rel, root: root, resolved: resolved}
	}
	return resolvedRoot, resolved, nil
}

// RemoveInRoot removes rel, a file or empty directory relative to root, after
// applying the containment rule of ResolveInRoot to its parent directory. The
// leaf itself is unlinked, never followed, so removing a symlink whose target
// lies outside root removes only the link. A missing rel returns an error
// wrapping fs.ErrNotExist, as os.Remove does; so does a rel behind a dangling
// symlink, or in a missing directory a symlink redirects into .git, since
// nothing can exist there. CheckRemoveInRoot runs the same check alone.
//
// rel is cleaned lexically before both the check and the removal, and the
// removal goes through the parent directory the check resolved, so the path
// removed is exactly the path checked and an in-root alias (absolute, or
// stepping out of root and back in) still works. The removal is made through
// os.Root, so a directory swapped for an escaping symlink between the check
// and the removal is still refused. A swap in that window that redirects into
// .git inside root is not caught; the check only rules out links present when
// it runs.
func RemoveInRoot(root, rel string) error {
	rel, resolvedRoot, parent, err := checkRemoveInRoot(root, rel)
	if err != nil {
		return err
	}
	parentRel, err := filepath.Rel(resolvedRoot, parent)
	if err != nil {
		return fmt.Errorf("removing %q under %s: %w", rel, root, err)
	}
	r, err := os.OpenRoot(resolvedRoot)
	if err != nil {
		return fmt.Errorf("opening root %s: %w", resolvedRoot, err)
	}
	defer func() { _ = r.Close() }()
	if err := r.Remove(filepath.Join(parentRel, filepath.Base(rel))); err != nil {
		return fmt.Errorf("removing %q under %s: %w", rel, root, err)
	}
	return nil
}

// CheckRemoveInRoot returns the error RemoveInRoot(root, rel) would report
// before removing anything: an error wrapping ErrOutsideRoot (and
// ErrRedirectsIntoGit for a .git redirect) when rel's parent directory leaves
// root, or one wrapping fs.ErrNotExist when the parent cannot exist. It lets
// a caller vet every path of a batch before the first removal. A nil result
// does not mean rel exists.
func CheckRemoveInRoot(root, rel string) error {
	_, _, _, err := checkRemoveInRoot(root, rel)
	return err
}

// checkRemoveInRoot implements CheckRemoveInRoot and returns the cleaned rel,
// the resolved root and the resolved parent directory of rel.
//
// A parent that cannot exist (behind a dangling symlink, or missing beneath a
// .git redirect) reports fs.ErrNotExist: there is nothing to remove, which
// callers already treat as benign, and nothing is created there either.
func checkRemoveInRoot(root, rel string) (cleanRel, resolvedRoot, parent string, err error) {
	if !filepath.IsLocal(rel) {
		return "", "", "", fmt.Errorf("removing %q under %s: %w", rel, root, ErrOutsideRoot)
	}
	rel = filepath.Clean(rel)
	resolvedRoot, parent, err = resolveInRoot(root, filepath.Dir(rel))
	if errors.Is(err, errDanglingSymlink) ||
		(errors.Is(err, ErrRedirectsIntoGit) && !exists(filepath.Join(root, filepath.Dir(rel)))) {
		return "", "", "", fmt.Errorf("removing %q under %s: %w", rel, root, fs.ErrNotExist)
	}
	if err != nil {
		// Already names the path and root; callers add their own context.
		return "", "", "", err
	}
	return rel, resolvedRoot, parent, nil
}

// refuseDanglingAncestors walks up from dir through the components that do
// not resolve and refuses the first one that exists as a directory entry,
// i.e. a symlink whose target is missing. resolveExistingPrefix alone would
// treat such a link as a missing directory it can append to.
func refuseDanglingAncestors(dir, rel string) error {
	for current := filepath.Clean(dir); ; {
		_, err := os.Stat(current)
		if !errors.Is(err, fs.ErrNotExist) {
			// Resolves (or fails for another reason, which
			// resolveExistingPrefix reports).
			return nil
		}
		if err := refuseDanglingSymlink(current, rel); err != nil {
			return err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
		current = parent
	}
}

// refuseDanglingSymlink reports an error when p exists as a directory entry
// even though resolving it failed with "not exist", i.e. p is a symlink whose
// target is missing.
func refuseDanglingSymlink(p, rel string) error {
	if _, err := os.Lstat(p); err == nil {
		return fmt.Errorf("%q passes through %s: %w: %w", rel, p, errDanglingSymlink, ErrOutsideRoot)
	}
	return nil
}

// exists reports whether path resolves to an existing file or directory.
func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
