package fileutil

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// ErrOutsideRoot is returned by WriteFileAtomicInRoot when the destination,
// after resolving symbolic links, lies outside the root directory, or when a
// symbolic link redirects the write into a .git metadata directory.
var ErrOutsideRoot = errors.New("path resolves outside root")

// gitDirName is the repository metadata directory. A committed symbolic link
// must never redirect a write into it (for example CLAUDE.md -> .git/config).
const gitDirName = ".git"

// maxSymlinkHops bounds symlink resolution, matching the common kernel limit.
const maxSymlinkHops = 40

// groupOtherReadWrite are the permission bits a user typically removes to make
// a file private; an atomic rewrite never grants them back.
const groupOtherReadWrite os.FileMode = 0o066

// WriteFileAtomic writes content to path atomically by creating a temporary
// file in the same directory (guaranteeing same-filesystem rename), writing
// and syncing the content, and renaming it into place. Parent directories are
// created as needed. On failure, the temporary file is cleaned up.
//
// When path is a symbolic link to a file inside the link's own directory tree
// (such as CLAUDE.md -> AGENTS.md), the file it points to is replaced and the
// link is kept. A link that resolves outside that tree, or into a .git
// directory within it, is replaced by a regular file instead, so a write never
// lands outside the directory it names; use WriteFileAtomicInRoot to confine
// writes to a whole project.
//
// A new file is created with mode subject to the process umask, like
// os.WriteFile. When the destination already exists, the rewrite never grants
// group/other read or write access the existing file does not have, so a file
// the user made private stays private; other bits follow mode.
func WriteFileAtomic(path string, content []byte, mode os.FileMode) error {
	target, err := resolveWriteTarget(path)
	if err != nil {
		return err
	}
	if target != path && !linkTargetWithin(filepath.Dir(path), target) {
		target = path
	}
	return writeAtomic(target, content, mode)
}

// linkTargetWithin reports whether the resolved symlink target lies inside
// dir, and outside any .git directory beneath it, once all symlinks in both
// are resolved.
func linkTargetWithin(dir, target string) bool {
	resolvedDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return false
	}
	resolvedTargetDir, err := resolveExistingPrefix(filepath.Dir(target))
	if err != nil {
		return false
	}
	return isWithin(resolvedDir, resolvedTargetDir) && !redirectsIntoGitDir(resolvedDir, resolvedTargetDir, ".")
}

// WriteFileAtomicInRoot writes rel, a path relative to root, atomically with
// the same semantics as WriteFileAtomic, but refuses to write outside root:
// rel must be a local path, and if a symlinked parent directory or a symlinked
// destination resolves outside root, it returns an error wrapping
// ErrOutsideRoot and writes nothing. Use it for every project-relative write
// so a repository cannot redirect writes (for example by committing a
// symlinked .claude directory) to files outside the project.
func WriteFileAtomicInRoot(root, rel string, content []byte, mode os.FileMode) error {
	if !filepath.IsLocal(rel) {
		return fmt.Errorf("writing %q under %s: %w", rel, root, ErrOutsideRoot)
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return fmt.Errorf("resolving root %s: %w", root, err)
	}
	target, err := resolveWriteTarget(filepath.Join(root, rel))
	if err != nil {
		return err
	}
	resolvedDir, err := resolveExistingPrefix(filepath.Dir(target))
	if err != nil {
		return err
	}
	if !isWithin(resolvedRoot, resolvedDir) || redirectsIntoGitDir(resolvedRoot, resolvedDir, filepath.Dir(rel)) {
		return fmt.Errorf("writing %q under %s (resolves to %s): %w", rel, root, resolvedDir, ErrOutsideRoot)
	}
	return writeAtomic(filepath.Join(resolvedDir, filepath.Base(target)), content, mode)
}

// writeAtomic performs the temp-file-and-rename write of an already-resolved
// destination path.
func writeAtomic(path string, content []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, ModeDirDefault); err != nil {
		return fmt.Errorf("create parent directories for %s: %w", path, err)
	}

	existing, statErr := os.Stat(path)
	if statErr != nil && !errors.Is(statErr, fs.ErrNotExist) {
		return fmt.Errorf("stat %s: %w", path, statErr)
	}
	// Create the temp file with the final mode so its content is never more
	// widely readable than the destination, even before the rename.
	finalMode := mode
	if existing != nil {
		finalMode = rewriteMode(existing.Mode().Perm(), mode)
	}

	tmp, err := createTemp(dir, finalMode)
	if err != nil {
		return err
	}

	success := false
	defer func() {
		if !success {
			_ = tmp.Close()
			_ = os.Remove(tmp.Name())
		}
	}()

	if _, err := tmp.Write(content); err != nil {
		return fmt.Errorf("write to temp file %s: %w", tmp.Name(), err)
	}

	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("sync temp file %s: %w", tmp.Name(), err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file %s: %w", tmp.Name(), err)
	}

	// A new file keeps the umask-filtered mode it was created with. An
	// existing file's mode is replaced explicitly, without widening it.
	if existing != nil {
		if err := os.Chmod(tmp.Name(), finalMode); err != nil {
			return fmt.Errorf("chmod temp file %s: %w", tmp.Name(), err)
		}
	}

	if err := renameWithRetry(tmp.Name(), path); err != nil {
		return fmt.Errorf("rename %s to %s: %w", tmp.Name(), path, err)
	}

	success = true
	syncDir(dir)
	return nil
}

// rewriteMode returns the mode for rewriting a file whose current permission
// bits are existing: the requested mode, minus any group/other read or write
// access the existing file does not grant.
func rewriteMode(existing, requested os.FileMode) os.FileMode {
	return requested &^ (^existing & groupOtherReadWrite)
}

// createTemp creates a uniquely named temporary file in dir. The file is
// created with mode, so the kernel applies the process umask to it.
func createTemp(dir string, mode os.FileMode) (*os.File, error) {
	prefix := branding.Get().TempPrefix
	for range 10000 {
		var b [8]byte
		if _, err := rand.Read(b[:]); err != nil {
			return nil, fmt.Errorf("generating temp file name: %w", err)
		}
		name := filepath.Join(dir, prefix+hex.EncodeToString(b[:]))
		f, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, mode.Perm())
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("create temp file in %s: %w", dir, err)
		}
		return f, nil
	}
	return nil, fmt.Errorf("create temp file in %s: too many name collisions", dir)
}

// resolveWriteTarget returns the path an atomic write should replace: path
// itself, or, when path is a symbolic link, the file it ultimately points to
// (which may not exist yet), so the rename replaces the target and keeps the
// link instead of replacing the link with a regular file.
func resolveWriteTarget(path string) (string, error) {
	current := path
	for range maxSymlinkHops {
		info, err := os.Lstat(current)
		if errors.Is(err, fs.ErrNotExist) || (err == nil && info.Mode()&fs.ModeSymlink == 0) {
			return current, nil
		}
		if err != nil {
			return "", fmt.Errorf("lstat %s: %w", current, err)
		}
		link, err := os.Readlink(current)
		if err != nil {
			return "", fmt.Errorf("reading symlink %s: %w", current, err)
		}
		if !filepath.IsAbs(link) {
			link = filepath.Join(filepath.Dir(current), link)
		}
		current = link
	}
	return "", fmt.Errorf("resolving %s: too many levels of symbolic links", path)
}

// resolveExistingPrefix resolves symlinks in the longest existing prefix of
// dir and re-appends the components that do not exist yet (which MkdirAll
// will create as real directories).
func resolveExistingPrefix(dir string) (string, error) {
	var missing []string
	current := filepath.Clean(dir)
	for {
		resolved, err := filepath.EvalSymlinks(current)
		if err == nil {
			return filepath.Join(append([]string{resolved}, missing...)...), nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("resolving %s: %w", current, err)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("resolving %s: %w", dir, err)
		}
		missing = append([]string{filepath.Base(current)}, missing...)
		current = parent
	}
}

// redirectsIntoGitDir reports whether resolved, a symlink-resolved directory
// beneath root, lies inside a .git directory that the lexical directory the
// caller asked for (relative to root) does not name, i.e. whether a symbolic
// link redirected the write into repository metadata.
func redirectsIntoGitDir(root, resolved, lexicalRel string) bool {
	rel, err := filepath.Rel(root, resolved)
	if err != nil {
		return false
	}
	return hasGitComponent(rel) && !hasGitComponent(lexicalRel)
}

// hasGitComponent reports whether any element of the relative path rel is a
// .git directory (compared case-insensitively for case-folding filesystems).
func hasGitComponent(rel string) bool {
	for part := range strings.SplitSeq(filepath.ToSlash(rel), "/") {
		if strings.EqualFold(part, gitDirName) {
			return true
		}
	}
	return false
}

// isWithin reports whether path is root or lies beneath it. Both must be
// clean, symlink-resolved paths.
func isWithin(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
