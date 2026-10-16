package update

import (
	"errors"
	"io/fs"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

// CleanupSidecar removes the sidecar file rel, relative to root, confined to
// root by fileutil.RemoveInRoot. It returns nil if the file does not exist.
func CleanupSidecar(root, rel string) error {
	if err := fileutil.RemoveInRoot(root, rel); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
