//go:build !unix

package check

import "os"

// canExecute reports whether the file at path has an execute bit. Hook
// scripts are checked for it only where the kernel enforces it (unix).
func canExecute(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.Mode().Perm()&0o111 != 0
}
