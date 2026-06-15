package contentsign

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

// maxIngestBytes bounds the db.json that IngestDevDocs will sanitize. Value-level
// JSON sanitization must parse the whole document into an in-memory tree and
// re-encode it (~3-4x file size peak), unlike the one-pass streaming
// download/hash/sign paths. db.json is a per-slug documentation index (MB-scale
// in practice), so this generous ceiling never rejects legitimate content — it
// only converts a pathological/oversized file into a clear error instead of an
// out-of-memory crash.
const maxIngestBytes int64 = 512 << 20 // 512 MiB

// IngestDevDocs sanitizes the DevDocs db.json in dir in place (atomic rewrite),
// stripping invisible/tag/control characters before the external devdocs server
// indexes it. index.json and meta.json are left untouched. Returns the sanitize
// report. opts controls the sanitizer (use DefaultSanitizeOptions for the
// lossless download-time profile — invisible/control/HTML stripping, NFKC off).
//
// The input is size-bounded (see maxIngestBytes): value-level sanitization needs
// the parsed tree in memory to tell string VALUES from object keys and to
// preserve document structure, so unlike the streaming download/hash/sign paths
// it cannot process the file one pass at a time. An oversized db.json therefore
// yields a clear error rather than an out-of-memory crash.
func IngestDevDocs(ctx context.Context, dir string, opts SanitizeOptions) (SanitizeReport, error) {
	return ingestDevDocs(ctx, dir, opts, maxIngestBytes)
}

// ingestDevDocs is the implementation behind IngestDevDocs with the size bound
// injected so tests can exercise the oversize path without writing a huge file.
func ingestDevDocs(ctx context.Context, dir string, opts SanitizeOptions, maxBytes int64) (SanitizeReport, error) {
	dbPath := filepath.Join(dir, "db.json")
	f, err := os.Open(dbPath) //nolint:gosec // dbPath is a manifest-controlled corpus path.
	if err != nil {
		if os.IsNotExist(err) {
			return SanitizeReport{}, nil
		}
		return SanitizeReport{}, fmt.Errorf("opening %s: %w", dbPath, err)
	}
	defer f.Close()

	raw, err := io.ReadAll(io.LimitReader(f, maxBytes+1))
	if err != nil {
		return SanitizeReport{}, fmt.Errorf("reading %s: %w", dbPath, err)
	}
	if int64(len(raw)) > maxBytes {
		return SanitizeReport{}, fmt.Errorf("refusing to sanitize %s: size exceeds the %d-byte limit", dbPath, maxBytes)
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
