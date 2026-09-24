package policy

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox/denylist"
)

// nixStore is the one location outside the project a policy may mount: its
// content is immutable and world-readable already.
const nixStore = "/nix/store"

// runtimeDirs hold sockets and other live endpoints of processes outside the
// sandbox (the systemd user bus under /run/user/$UID, ssh-agent, gpg-agent,
// docker.sock). Binding one would let a hook drive those processes, so a
// policy mount under them is rejected even when the project lives there.
var runtimeDirs = []string{"/run", "/var/run"}

// ValidateMountDecl checks that a MountDecl is safe to use in a sandbox policy.
// The policy comes from the repository the sandbox contains, so its mounts are
// allowlisted: both the source and the target must lie inside projectDir or
// under /nix/store (an empty projectDir allows only /nix/store). It also
// rejects non-absolute paths, the root filesystem, anything under /run or
// /var/run, deny-list paths, and a source that is a socket, pipe or device.
// Both the cleaned and symlink-resolved forms of each path are checked, so a
// symlink cannot lead a mount outside the allowlist.
func ValidateMountDecl(m MountDecl, projectDir string) error {
	if !filepath.IsAbs(m.Source) {
		return fmt.Errorf("mount source must be absolute: %q", m.Source)
	}
	if !filepath.IsAbs(m.Target) {
		return fmt.Errorf("mount target must be absolute: %q", m.Target)
	}
	if filepath.Clean(m.Source) == "/" {
		return fmt.Errorf("mount source must not be root filesystem: %q", m.Source)
	}
	if filepath.Clean(m.Target) == "/" {
		return fmt.Errorf("mount target must not be root filesystem: %q", m.Target)
	}

	roots := allowedMountRoots(projectDir)
	for _, p := range []struct{ path, role string }{{m.Source, "source"}, {m.Target, "target"}} {
		if err := checkAllowlist(p.path, p.role, roots); err != nil {
			return err
		}
		if err := checkDenyList(p.path, p.role); err != nil {
			return err
		}
	}
	return checkSourceType(m.Source)
}

// allowedMountRoots returns the directories policy mounts may lie in: the
// project directory (literal and symlink-resolved) and /nix/store.
func allowedMountRoots(projectDir string) []string {
	roots := []string{nixStore}
	if projectDir != "" && filepath.IsAbs(projectDir) && filepath.Clean(projectDir) != "/" {
		roots = append(roots, denylist.CandidatePaths(projectDir)...)
	}
	return roots
}

// checkAllowlist requires every candidate form of path to lie within one of
// roots and outside the runtime directories.
func checkAllowlist(path, role string, roots []string) error {
	for _, candidate := range denylist.CandidatePaths(path) {
		for _, rt := range runtimeDirs {
			if denylist.Overlaps(candidate, rt) {
				return fmt.Errorf("mount %s %q is under runtime directory %s, which holds host sockets", role, path, rt)
			}
		}
		if !withinAny(candidate, roots) {
			return fmt.Errorf("mount %s %q is outside the project directory and %s", role, path, nixStore)
		}
	}
	return nil
}

// withinAny reports whether path equals or descends from one of roots.
func withinAny(path string, roots []string) bool {
	for _, root := range roots {
		if denylist.Overlaps(path, root) {
			return true
		}
	}
	return false
}

// checkSourceType rejects a mount source that is a socket, named pipe or
// device: those are endpoints of processes or hardware outside the sandbox.
// A source that does not exist is left to the backend, which fails to bind it.
func checkSourceType(source string) error {
	info, err := os.Stat(source)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("checking mount source %q: %w", source, err)
	}
	if info.Mode()&(fs.ModeSocket|fs.ModeNamedPipe|fs.ModeDevice|fs.ModeCharDevice) != 0 {
		return fmt.Errorf("mount source %q is a socket, pipe or device", source)
	}
	return nil
}

// checkDenyList checks a single path against the shared deny list, using both
// the cleaned path and its symlink-resolved form.
func checkDenyList(path, role string) error {
	denyPaths := denylist.AllDenyPaths()
	for _, candidate := range denylist.CandidatePaths(path) {
		for _, deny := range denyPaths {
			// Reject the deny path itself, any descendant of it, AND any ancestor
			// of it. Rejecting ancestors prevents binding e.g. $HOME (which
			// contains ~/.ssh) or /etc (which contains /etc/shadow), which would
			// otherwise re-expose the sensitive descendant inside the sandbox.
			if denylist.Overlaps(candidate, deny) || denylist.IsStrictAncestor(candidate, deny) {
				return fmt.Errorf("mount %s %q overlaps sensitive path %q", role, path, deny)
			}
		}
	}

	return nil
}
