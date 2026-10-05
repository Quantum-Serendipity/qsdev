//go:build !unix

package projectctx

import "io/fs"

// trusted always reports true off unix. Windows ownership and write access are
// ACL-based, and evaluating ACLs is out of scope; the git-toplevel ceiling
// still bounds the walk.
func trusted(_, _ fs.FileInfo) bool { return true }

// trustedLink always reports true off unix, for the same reason as trusted.
func trustedLink(fs.FileInfo) bool { return true }

// sameDevice always reports true off unix, where the FileInfo carries no
// device number; the walk is still bounded by the volume root.
func sameDevice(_, _ fs.FileInfo) bool { return true }
