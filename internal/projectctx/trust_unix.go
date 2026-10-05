//go:build unix

package projectctx

import (
	"io/fs"
	"os"
	"syscall"
)

// worldWritable is the permission bit that lets any local user write.
const worldWritable fs.FileMode = 0o002

// trusted reports whether a marker with info (its target, for a symlink),
// inside a directory with dirInfo, can only have been planted by the current
// user or root: the marker and the directory holding it must both be owned by
// the effective uid or by uid 0, and neither may be world-writable. The
// directory-owner rule is git's safe.directory rule: another user's directory
// is not this user's project, whatever its content. The world-writable
// directory rule is what rejects a marker in a shared sticky directory such as
// /tmp even when the current user owns it. Group-writable is accepted, so
// projects created under a user-private-group umask (002) stay trusted.
func trusted(info, dirInfo fs.FileInfo) bool {
	return ownedByUserOrRoot(info) && ownedByUserOrRoot(dirInfo) &&
		info.Mode().Perm()&worldWritable == 0 && dirInfo.Mode().Perm()&worldWritable == 0
}

// trustedLink reports whether a symlink marker entry, as Lstat reports it,
// was placed by the current user or root. Only its owner counts: a symlink's
// own permission bits are meaningless (always 0777 on Linux).
func trustedLink(link fs.FileInfo) bool {
	return ownedByUserOrRoot(link)
}

// ownedByUserOrRoot reports whether info is owned by the effective uid or by
// uid 0. Missing ownership information fails closed.
func ownedByUserOrRoot(info fs.FileInfo) bool {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return false
	}
	uid := uint64(st.Uid)
	return uid == 0 || uid == uint64(os.Geteuid())
}

// sameDevice reports whether a and b live on the same device. Missing device
// information fails closed: the walk stops.
func sameDevice(a, b fs.FileInfo) bool {
	sa, okA := a.Sys().(*syscall.Stat_t)
	sb, okB := b.Sys().(*syscall.Stat_t)
	return okA && okB && sa.Dev == sb.Dev
}
