package teardown

import (
	"fmt"
	"path/filepath"
)

// checkTrackedPath rejects a state-recorded path that is not a plain relative
// path inside the project (absolute, empty, or containing ".." components).
// State files can be committed to a repository, so their contents are
// untrusted input. Symlink containment is fileutil.ResolveInRoot's job.
func checkTrackedPath(relPath string) error {
	if !filepath.IsLocal(filepath.FromSlash(relPath)) {
		return fmt.Errorf("unsafe tracked path %q: must be relative and inside the project", relPath)
	}
	return nil
}
