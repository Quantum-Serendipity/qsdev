//go:build unix

package check

import "golang.org/x/sys/unix"

// canExecute reports whether this user may execute the file at path, as the
// kernel decides it (owner, group and other bits, and root's rules).
func canExecute(path string) bool {
	return unix.Access(path, unix.X_OK) == nil
}
