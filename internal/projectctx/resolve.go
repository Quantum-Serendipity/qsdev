package projectctx

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// Mode selects how Resolve picks the project root.
type Mode int

const (
	// Enclosing walks up from the start directory to the nearest trusted
	// project marker. It is the mode of every command that acts on an
	// existing project.
	Enclosing Mode = iota
	// Here takes the start directory itself as the root, for commands that
	// create a project where they are run.
	Here
)

// DataDirName returns the name of the per-project data directory,
// "."+AppName, which holds the project's policy, logs and catalog defaults.
// It is not a project marker (see Resolve).
func DataDirName() string {
	return "." + branding.Get().AppName
}

// ErrRelativeStart reports a Resolve start directory that is not absolute.
var ErrRelativeStart = errors.New("start directory is not absolute")

// gitEntryName is the entry that marks a repository toplevel: a directory in
// an ordinary clone, a gitlink file in a worktree or submodule.
const gitEntryName = ".git"

// Context is the resolved project a command acts on.
type Context struct {
	// Start is the cleaned absolute directory resolution started from.
	Start string
	// Root is the project root: the directory holding the trusted marker, or
	// Start when none was found (or in Here mode).
	Root string
	// GitTop is the repository toplevel the walk stopped at, or "" when the
	// walk ended before reaching one.
	GitTop string
	// Found reports whether a trusted marker was found.
	Found bool
	// Ignored lists the markers the walk skipped as untrusted, nearest first.
	Ignored []string
}

// stat and lstat are the package's only filesystem probes, kept as seams so
// tests can fabricate owners and devices and count calls per level.
var (
	stat  = os.Stat
	lstat = os.Lstat
)

// Resolve resolves the project for the absolute directory start.
//
// In Here mode the root is start itself; only start is examined, so Found and
// GitTop describe that directory alone.
//
// In Enclosing mode Resolve walks from start toward the volume root and takes
// the first directory holding a trusted marker: the branding config file as a
// regular file, or the branding state directory as a directory. The bare
// "."+AppName data directory is not a marker. The walk ends:
//
//   - after the first directory holding a .git entry of any kind, recorded as
//     GitTop; an entry that cannot be inspected counts as one (fails closed);
//   - before a parent on a different device than its child, as git does
//     without GIT_DISCOVERY_ACROSS_FILESYSTEM;
//   - at the volume root, or at a parent that cannot be inspected.
//
// A marker that fails the trust check (see considerMarker and trusted) is
// skipped, the walk goes on, and its path is recorded in Ignored. When no
// trusted marker is found, Root is start and Found is false, so commands that
// work before a project is initialised still have a directory to act on.
//
// start must exist; Resolve returns its stat error otherwise.
func Resolve(start string, mode Mode) (Context, error) {
	if !filepath.IsAbs(start) {
		return Context{}, fmt.Errorf("resolving project root from %q: %w", start, ErrRelativeStart)
	}
	start = filepath.Clean(start)
	c := Context{Start: start, Root: start}

	dirInfo, err := stat(start)
	if err != nil {
		return Context{}, fmt.Errorf("resolving project root from %q: %w", start, err)
	}
	b := branding.Get()
	walk(start, dirInfo, true, func(dir string, info fs.FileInfo) bool {
		return c.examine(dir, info, b) || c.GitTop != "" || mode == Here
	})
	return c, nil
}

// walk is the package's one upward traversal. It calls visit with dir (whose
// FileInfo is info) and then with each ancestor toward the volume root, until
// visit reports true. It stops at the volume root and before a parent that
// cannot be inspected; when sameDev is set it also stops before a parent on a
// different device than its child. Each step costs one stat of the parent.
func walk(dir string, info fs.FileInfo, sameDev bool, visit func(dir string, info fs.FileInfo) bool) {
	for !visit(dir, info) {
		parent := filepath.Dir(dir)
		if parent == dir {
			return
		}
		parentInfo, err := stat(parent)
		if err != nil || (sameDev && !sameDevice(info, parentInfo)) {
			return
		}
		dir, info = parent, parentInfo
	}
}

// isGitTop reports whether dir holds a .git entry of any kind (a directory in
// an ordinary clone, a gitlink file in a worktree or submodule). It fails
// closed: an entry that cannot be inspected, such as one behind a permission
// error, counts as a toplevel, so it never lets a walk escape a repository.
func isGitTop(dir string) bool {
	_, err := lstat(filepath.Join(dir, gitEntryName))
	return !errors.Is(err, fs.ErrNotExist)
}

// examine checks one directory level: it reports whether dir holds a trusted
// marker (setting Root and Found), records untrusted markers in Ignored, and
// sets GitTop when dir is a repository toplevel. It makes at most three
// filesystem calls when no marker is a symlink, so with the caller's stat of
// dir a level costs at most four calls; each symlinked marker costs one more
// (stat of its target), so the worst case is six.
func (c *Context) examine(dir string, dirInfo fs.FileInfo, b branding.Config) bool {
	found := c.considerMarker(filepath.Join(dir, b.ConfigFile), fs.FileMode.IsRegular, dirInfo)
	if !found {
		found = c.considerMarker(filepath.Join(dir, b.StateDir), fs.FileMode.IsDir, dirInfo)
	}
	if found {
		c.Root, c.Found = dir, true
	}
	if isGitTop(dir) {
		c.GitTop = dir
	}
	return found
}

// considerMarker reports whether path is a marker of the right kind that
// passes the trust check. Trust is decided on the marker entry itself, not
// only on what it points at: the entry is Lstat'ed, and a symlink is
// followed only when the link itself was placed by the current user or root
// (trustedLink), since a foreign link to a file the user owns would otherwise
// pass for the user's own marker. A marker of the right kind that fails a
// check, and any foreign symlink, is appended to Ignored. A path that cannot
// be inspected is no marker. A regular marker costs one call; a symlink, two.
func (c *Context) considerMarker(path string, kind func(fs.FileMode) bool, dirInfo fs.FileInfo) bool {
	info, err := lstat(path)
	if err != nil {
		return false
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		if !trustedLink(info) {
			c.Ignored = append(c.Ignored, path)
			return false
		}
		if info, err = stat(path); err != nil {
			return false
		}
	}
	if !kind(info.Mode()) {
		return false
	}
	if !trusted(info, dirInfo) {
		c.Ignored = append(c.Ignored, path)
		return false
	}
	return true
}

type contextKey struct{}

// WithContext returns a copy of ctx carrying the resolved project pc.
func WithContext(ctx context.Context, pc Context) context.Context {
	return context.WithValue(ctx, contextKey{}, pc)
}

// FromContext returns the project stored by WithContext, reporting false when
// ctx carries none.
func FromContext(ctx context.Context) (Context, bool) {
	pc, ok := ctx.Value(contextKey{}).(Context)
	return pc, ok
}

// ResolveWorkingDir resolves the project for the process working directory
// (see Resolve).
func ResolveWorkingDir(mode Mode) (Context, error) {
	wd, err := WorkingDir()
	if err != nil {
		return Context{}, err
	}
	return Resolve(wd, mode)
}

// WorkingDir returns the process working directory. It is the only reader of
// the working directory outside the process entry point.
func WorkingDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("reading working directory: %w", err)
	}
	return wd, nil
}

// HomeDir returns the user's home directory. It is the only reader of the
// home directory in the tree; it fails when the home directory is unknown or
// not absolute, so no per-user path is ever resolved against the working
// directory.
func HomeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("reading home directory: %w", err)
	}
	if !filepath.IsAbs(home) {
		return "", fmt.Errorf("reading home directory: %q: %w", home, ErrNoUserDir)
	}
	return home, nil
}
