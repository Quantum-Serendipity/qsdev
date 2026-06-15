package contentsign

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

// IngestDevDocs sanitizes the DevDocs db.json in dir in place (atomic rewrite),
// stripping invisible/tag/control characters before the external devdocs server
// indexes it. index.json and meta.json are left untouched. Returns the sanitize
// report. opts controls the sanitizer (use DefaultSanitizeOptions for the
// lossless download-time profile — invisible/control/HTML stripping, NFKC off).
func IngestDevDocs(ctx context.Context, dir string, opts SanitizeOptions) (SanitizeReport, error) {
	dbPath := filepath.Join(dir, "db.json")
	raw, err := os.ReadFile(dbPath)
	if err != nil {
		if os.IsNotExist(err) {
			return SanitizeReport{}, nil
		}
		return SanitizeReport{}, fmt.Errorf("reading %s: %w", dbPath, err)
	}

	clean, report, err := SanitizeJSONStrings(ctx, raw, opts)
	if err != nil {
		return SanitizeReport{}, fmt.Errorf("sanitizing %s: %w", dbPath, err)
	}

	if !bytes.Equal(clean, raw) {
		if err := fileutil.WriteFileAtomic(dbPath, clean, fileutil.ModeReadWrite); err != nil {
			return SanitizeReport{}, fmt.Errorf("writing %s: %w", dbPath, err)
		}
	}

	return report, nil
}
