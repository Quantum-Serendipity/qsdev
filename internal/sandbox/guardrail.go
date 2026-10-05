package sandbox

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/claudesettings"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpconfig"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/canon"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// ErrGuardrailModified marks a hook run after which a guardrail path (see
// GuardrailPaths) was created, replaced or removed. The read-only overlays
// cover only what they can pin: a path absent at launch cannot be bound (bwrap
// would create it on the host through the read-write project bind), nor can a
// symlink. Such a change is caught after the hook, undone as far as possible
// (GuardrailSnapshot.Enforce) and blocks: the hook's verdict is not trusted
// once the control plane changed.
var ErrGuardrailModified = errors.New("sandboxed hook modified a protected project file")

// GuardrailPaths returns the absolute paths, inside projectDir, of the
// project's control plane: the files that make tools run code on the host or
// that steer qsdev, Claude Code, git and devenv. A hook in a category with a
// writable worktree must still never change them. The list is derived from
// the owners of each kind of file:
//   - git's code-executing control files (gitGuardrails);
//   - .claude as a whole (the settings, hooks, agents, skills and rules that
//     steer the agent), the MCP server config and the project config file;
//   - every location selfprotect's canon protects wherever it appears (the
//     project data and state directories, the answers copy, .envrc),
//     devenv's configuration files, which devenv runs on every shell entry,
//     and the package-manager and pre-commit files whose protective settings
//     gate-dodge checks;
//   - devenv.lock, which pins the inputs devenv builds the shell from;
//   - every path a checkout keeps for itself (state.LocalOnlyEntries): the
//     local config overrides, and the devenv and direnv caches (.devenv,
//     .direnv), whose generated shell scripts direnv and devenv run on the
//     next shell entry without asking for a new `direnv allow`.
//
// An entry under another one is dropped: protecting the ancestor covers it.
// It returns nil when projectDir is empty: there is no project to protect.
func GuardrailPaths(projectDir string) []string {
	if projectDir == "" {
		return nil
	}
	rel := slices.Concat(
		gitGuardrails(projectDir),
		[]string{
			path.Dir(claudesettings.ProjectRelPath),
			mcpconfig.FileName,
			branding.Get().ConfigFile,
			devenvLock,
		},
		canon.Segments(),
		canon.DevenvSourceFiles(),
		canon.GuardedConfigFiles(),
		state.LocalOnlyEntries(),
	)
	for i, r := range rel {
		rel[i] = path.Clean(r)
	}
	slices.Sort(rel)
	rel = slices.Compact(rel)
	out := make([]string, 0, len(rel))
	for _, r := range rel {
		covered := slices.ContainsFunc(rel, func(a string) bool {
			return strings.HasPrefix(r, a+"/")
		})
		if !covered {
			out = append(out, filepath.Join(projectDir, filepath.FromSlash(r)))
		}
	}
	return out
}

// gitDir is the name of a repository's git directory, or of the file that
// points a linked worktree or submodule at it.
const gitDir = ".git"

// devenvLock is the lock file pinning devenv's inputs: a changed pin points
// the next shell at other Nix code.
const devenvLock = "devenv.lock"

// gitControl are the entries of a git directory that make git run code or
// decide where the repository lives: hooks; config and config.worktree
// (core.hooksPath, filter and diff drivers, aliases); commondir, which moves
// both; info (attributes that select drivers, exclude); and modules, holding
// each submodule's own hooks and config. Objects, refs and the index stay
// writable, so a formatter can still `git add`.
var gitControl = []string{"hooks", "config", "config.worktree", "commondir", "info", "modules"}

// gitGuardrails returns the slash-separated guardrail entries for git: the
// gitControl entries when .git is a directory, otherwise .git itself (a
// linked worktree's or submodule's gitdir pointer, or absent, so that
// creating one is caught).
func gitGuardrails(projectDir string) []string {
	if fi, err := os.Stat(filepath.Join(projectDir, gitDir)); err != nil || !fi.IsDir() {
		return []string{gitDir}
	}
	out := make([]string, len(gitControl))
	for i, c := range gitControl {
		out[i] = path.Join(gitDir, c)
	}
	return out
}

// HookLogDir returns the absolute directory, inside projectDir, the generated
// Claude Code hooks append their logs to (canon.HookLogDir), or "" when
// projectDir is empty. It lies inside the read-only .claude guardrail; the
// sandbox keeps it writable for a hook with a writable worktree, as hooks are
// its only writers.
func HookLogDir(projectDir string) string {
	if projectDir == "" {
		return ""
	}
	return filepath.Join(projectDir, filepath.FromSlash(canon.HookLogDir))
}

// devenvStateDir is the project-relative, slash-separated directory devenv
// keeps mutable toolchain state in (its default DEVENV_STATE): GOPATH and its
// module cache, the Python venv and other language state. It lies inside the
// read-only .devenv guardrail, whose generated shell scripts devenv runs on
// the next shell entry.
const devenvStateDir = ".devenv/state"

// DevenvStateDir returns the absolute devenv state directory (devenvStateDir)
// inside projectDir, or "" when projectDir is empty. The sandbox keeps it
// writable for a hook with a writable worktree, so a test runner or generator
// can still fill the module cache or sync the venv.
func DevenvStateDir(projectDir string) string {
	if projectDir == "" {
		return ""
	}
	return filepath.Join(projectDir, filepath.FromSlash(devenvStateDir))
}

// UsesDevenv reports whether projectDir holds one of devenv's configuration
// files (canon.DevenvSourceFiles), so devenv will create its state directory.
func UsesDevenv(projectDir string) bool {
	return projectDir != "" && slices.ContainsFunc(canon.DevenvSourceFiles(), func(name string) bool {
		_, err := os.Lstat(filepath.Join(projectDir, name))
		return err == nil
	})
}

// WritableGuardrailDirs returns the directories inside projectDir's
// guardrails that a hook with a writable worktree may still write: the hook
// log directory (HookLogDir) and devenv's state directory (DevenvStateDir).
// It returns nil when projectDir is empty.
func WritableGuardrailDirs(projectDir string) []string {
	if projectDir == "" {
		return nil
	}
	return []string{HookLogDir(projectDir), DevenvStateDir(projectDir)}
}

// GuardrailSnapshot records the identity of every guardrail path of a
// project, so a change made while a hook ran can be detected, and undone,
// afterwards.
type GuardrailSnapshot struct {
	entries []guardrailEntry
}

// guardrailEntry is one guardrail path with its state: the entry itself
// (link) and, through any symlink, what it resolves to (target); a nil
// FileInfo means absent or dangling. linkDest is a symlink's own content,
// so the symlink can be put back.
type guardrailEntry struct {
	path         string
	link, target fs.FileInfo
	linkDest     string
}

// SnapshotGuardrails records the current state of every guardrail path in
// projectDir. A path that cannot be examined (other than not existing) is an
// error: its later state could not be compared.
func SnapshotGuardrails(projectDir string) (*GuardrailSnapshot, error) {
	snap := &GuardrailSnapshot{}
	for _, p := range GuardrailPaths(projectDir) {
		e, err := readGuardrail(p)
		if err != nil {
			return nil, err
		}
		snap.entries = append(snap.entries, e)
	}
	return snap, nil
}

// Enforce compares every guardrail path with the snapshot and puts the
// project back in a safe state where one changed: created, removed, replaced
// by another file or, for a symlink, now resolving to something new. Nothing
// is deleted, so a legitimate writer outside the sandbox loses no work:
//   - whatever now stands at a changed path, or at a symlink's newly created
//     or replaced target, is moved aside to a quarantine name
//     (QuarantineName), where the tool that would run it no longer looks;
//   - a symlinked guardrail (a pre-commit configuration linked into the Nix
//     store, say) is re-created pointing where it did, since the read-only
//     overlays cannot pin a symlink in the writable project.
//
// It returns an error wrapping ErrGuardrailModified naming every changed
// path and what was done with it, or nil when nothing changed. A path that
// can no longer be examined counts as changed, so the check fails closed.
func (s *GuardrailSnapshot) Enforce() error {
	suffix := time.Now().UTC().Format("20060102T150405.000000000Z")
	var errs []error
	for _, before := range s.entries {
		after, err := readGuardrail(before.path)
		if err != nil {
			errs = append(errs, fmt.Errorf("%w: %w", ErrGuardrailModified, err))
			continue
		}
		if before.sameLink(after) && sameFile(before.target, after.target) {
			continue
		}
		errs = append(errs, fmt.Errorf("%w: %s changed while the hook ran: %s; review it before trusting the project",
			ErrGuardrailModified, before.path, strings.Join(before.reconcile(after, suffix), "; ")))
	}
	return errors.Join(errs...)
}

// reconcile undoes the change from e to after (see Enforce) and returns what
// it did, for the error.
func (e guardrailEntry) reconcile(after guardrailEntry, suffix string) []string {
	var notes []string
	if e.sameLink(after) {
		// Only what the unchanged symlink resolves to changed.
		if after.target != nil {
			resolved, err := filepath.EvalSymlinks(e.path)
			if err != nil {
				return append(notes, fmt.Sprintf("its new target could not be located: %v", err))
			}
			notes = append(notes, quarantine(resolved, suffix))
		}
		return notes
	}
	if after.link != nil {
		notes = append(notes, quarantine(e.path, suffix))
	}
	switch {
	case e.linkDest != "":
		if err := os.Symlink(e.linkDest, e.path); err != nil {
			notes = append(notes, fmt.Sprintf("the symlink to %s could not be restored: %v", e.linkDest, err))
		} else {
			notes = append(notes, "the symlink to "+e.linkDest+" was restored")
		}
	case e.link != nil:
		notes = append(notes, "the original was moved or removed; restore it from version control or regenerate it")
	}
	return notes
}

// QuarantineName returns the name Enforce moves a changed guardrail p to:
// p with a ".<app>-quarantined-<suffix>" extension, which no tool that reads
// p looks at.
func QuarantineName(p, suffix string) string {
	return p + "." + branding.Get().AppName + "-quarantined-" + suffix
}

// quarantine moves p aside to its QuarantineName and describes the outcome.
func quarantine(p, suffix string) string {
	q := QuarantineName(p, suffix)
	if _, err := os.Lstat(q); err == nil {
		return fmt.Sprintf("it could not be moved aside: %s already exists", q)
	}
	if err := os.Rename(p, q); err != nil {
		return fmt.Sprintf("it could not be moved aside: %v", err)
	}
	return "it was moved to " + q
}

// readGuardrail returns p's entry and resolved state. Not existing, a
// dangling symlink or a parent that is not a directory is a nil FileInfo,
// not an error.
func readGuardrail(p string) (guardrailEntry, error) {
	e := guardrailEntry{path: p}
	link, err := os.Lstat(p)
	if err != nil {
		if absent(err) {
			return e, nil
		}
		return e, fmt.Errorf("examining guardrail %s: %w", p, err)
	}
	e.link = pinIdentity(link)
	if link.Mode()&fs.ModeSymlink == 0 {
		e.target = e.link
		return e, nil
	}
	if e.linkDest, err = os.Readlink(p); err != nil {
		return e, fmt.Errorf("examining guardrail %s: %w", p, err)
	}
	target, err := os.Stat(p)
	if err != nil {
		if absent(err) {
			return e, nil
		}
		return e, fmt.Errorf("examining guardrail %s: %w", p, err)
	}
	e.target = pinIdentity(target)
	return e, nil
}

// absent reports whether err means the path does not exist, including a
// parent that is no longer a directory.
func absent(err error) bool {
	return errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR)
}

// pinIdentity fixes fi's file identity now. On Windows os.SameFile reads the
// identity lazily, by path, on first use, which would compare a replaced
// file against itself; comparing fi with itself loads it immediately.
func pinIdentity(fi fs.FileInfo) fs.FileInfo {
	_ = os.SameFile(fi, fi)
	return fi
}

// sameLink reports whether after's entry is still e's own: the same file of
// the same type and, for a symlink, pointing to the same place. The type and
// symlink content are compared as well because a file system may give a new
// file the inode number of one just removed.
func (e guardrailEntry) sameLink(after guardrailEntry) bool {
	if !sameFile(e.link, after.link) {
		return false
	}
	return e.link == nil || e.link.Mode().Type() == after.link.Mode().Type() && e.linkDest == after.linkDest
}

// sameFile reports whether a and b are both absent or the same file.
func sameFile(a, b fs.FileInfo) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return os.SameFile(a, b)
}
