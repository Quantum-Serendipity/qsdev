package update

import "os"

// CleanupSidecar removes a sidecar file if it exists. Returns nil if the file does not exist.
func CleanupSidecar(path string) error {
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
