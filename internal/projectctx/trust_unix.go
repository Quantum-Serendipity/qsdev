//go:build unix

package projectctx

import (
	"fmt"
	"io/fs"
	"os"
	"syscall"
)

// worldWritable is the permission bit that lets any local user write.
const worldWritable fs.FileMode = 0o002

// untrustedReason is the package's one trust rule. It reports why an entry
// with info (its target, for a symlink), inside a directory with dirInfo,
// could have been planted by someone other than the current user or root, or
// "" when it could not: the entry and the directory holding it must both be
// owned by the effective uid or by uid 0, and neither may be world-writable.
// The directory-owner rule is git's safe.directory rule: another user's
// directory is not this user's project, whatever its content. The
// world-writable directory rule is what rejects an entry in a shared sticky
// directory such as /tmp even when the current user owns it. Group-writable
// is accepted, so projects created under a user-private-group umask (002)
// stay trusted.
func untrustedReason(info, dirInfo fs.FileInfo) string {
	if r := ownerReason(info); r != "" {
		return r
	}
	if info.Mode().Perm()&worldWritable != 0 {
		return "world-writable"
	}
	if r := ownerReason(dirInfo); r != "" {
		return "parent directory " + r
	}
	if dirInfo.Mode().Perm()&worldWritable != 0 {
		return "parent directory is world-writable"
	}
	return ""
}

// linkUntrustedReason reports why a symlink entry, as Lstat reports it, may
// not have been placed by the current user or root, or "" when it was. Only
// its owner counts: a symlink's own permission bits are meaningless (always
// 0777 on Linux).
func linkUntrustedReason(link fs.FileInfo) string {
	if r := ownerReason(link); r != "" {
		return "symlink " + r
	}
	return ""
}

// ownerReason reports, as "owned by uid N", an info owned by neither the
// effective uid nor uid 0, or "" for one that is. Missing ownership
// information fails closed.
func ownerReason(info fs.FileInfo) string {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return "owner unknown"
	}
	uid := uint64(st.Uid)
	if uid == 0 || uid == uint64(os.Geteuid()) {
		return ""
	}
	return fmt.Sprintf("owned by uid %d", uid)
}

// sameDevice reports whether a and b live on the same device. Missing device
// information fails closed: the walk stops.
func sameDevice(a, b fs.FileInfo) bool {
	sa, okA := a.Sys().(*syscall.Stat_t)
	sb, okB := b.Sys().(*syscall.Stat_t)
	return okA && okB && sa.Dev == sb.Dev
}
