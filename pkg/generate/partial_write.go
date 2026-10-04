package generate

import "fmt"

// PartialWriteError reports the files a write failed on and the command that
// finishes the setup once their errors are fixed. recorded is the number of
// files whose state was saved; rerun names the command to re-run. It returns
// nil when nothing failed. The failures stay reachable through errors.Is/As.
func PartialWriteError(r WriteResult, recorded int, rerun string) error {
	err := r.Err()
	if err == nil {
		return nil
	}
	return fmt.Errorf("partial write: %d files failed (state saved for %d successful files); fix the errors below and re-run %s to finish setup:\n%w",
		r.Failed, recorded, rerun, err)
}
