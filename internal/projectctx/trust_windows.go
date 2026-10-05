//go:build !unix

package projectctx

import "io/fs"

// untrustedReason always reports "" (trusted) off unix. Windows ownership and
// write access are ACL-based, and evaluating ACLs is out of scope; the
// git-toplevel ceiling still bounds the walk.
func untrustedReason(_, _ fs.FileInfo) string { return "" }

// linkUntrustedReason always reports "" off unix, for the same reason as
// untrustedReason.
func linkUntrustedReason(fs.FileInfo) string { return "" }

// sameDevice always reports true off unix, where the FileInfo carries no
// device number; the walk is still bounded by the volume root.
func sameDevice(_, _ fs.FileInfo) bool { return true }
