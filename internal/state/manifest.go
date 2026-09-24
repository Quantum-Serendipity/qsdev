package state

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// ManifestFile returns the path, relative to the project root, of the
// committed manifest of machine-owned generated files. It lives at the project
// root, beside the project config, because the state directory and the
// project dot-directory are gitignored: a CI checkout has neither, but it does
// have this file.
func ManifestFile() string {
	return "." + branding.Get().AppName + "-generated.sha256"
}

// Manifest maps the slash-separated project-relative path of each
// machine-owned generated file to its content hash ("sha256:<hex>").
type Manifest map[string]string

// BuildManifest returns the manifest entries for st: every tracked file whose
// strategy is machine-owned. Human-edited files (see
// MergeStrategy.IsHumanEdited) are left out, because their divergence from the
// generated content is expected rather than drift.
func BuildManifest(st types.GeneratedState) Manifest {
	m := make(Manifest, len(st.Files))
	for relPath, fs := range st.Files {
		if fs.Strategy.IsHumanEdited() {
			continue
		}
		m[relPath] = fs.Hash
	}
	return m
}

// Marshal renders the manifest in the sha256sum text format ("<hex>  <path>"
// per line, sorted by path), so CI can also verify it with
// `sha256sum --check --strict`.
func (m Manifest) Marshal() ([]byte, error) {
	var b bytes.Buffer
	for _, relPath := range slices.Sorted(maps.Keys(m)) {
		if err := validateManifestPath(relPath); err != nil {
			return nil, err
		}
		sum, ok := strings.CutPrefix(m[relPath], HashPrefix)
		if !ok || !isSHA256Hex(sum) {
			return nil, fmt.Errorf("manifest entry %s: hash %q is not a %s digest", relPath, m[relPath], HashPrefix)
		}
		fmt.Fprintf(&b, "%s  %s\n", sum, relPath)
	}
	return b.Bytes(), nil
}

// ParseManifest parses a manifest in the sha256sum text format. Blank lines
// and lines starting with '#' are ignored. The file is committed and so
// untrusted: every path must be a clean, slash-separated path inside the
// project, and a path listed twice is rejected.
func ParseManifest(data []byte) (Manifest, error) {
	m := make(Manifest)
	for i, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSuffix(line, "\r")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		sum, relPath, ok := strings.Cut(line, " ")
		// sha256sum writes "<hex>  <path>" (text mode) or "<hex> *<path>"
		// (binary mode); both hash the same bytes on the platforms we run on.
		if !ok || (!strings.HasPrefix(relPath, " ") && !strings.HasPrefix(relPath, "*")) {
			return nil, fmt.Errorf("line %d: expected \"<sha256>  <path>\"", i+1)
		}
		relPath = relPath[1:]
		if !isSHA256Hex(sum) {
			return nil, fmt.Errorf("line %d: %q is not a lowercase SHA-256 hex digest", i+1, sum)
		}
		if err := validateManifestPath(relPath); err != nil {
			return nil, fmt.Errorf("line %d: %w", i+1, err)
		}
		if _, dup := m[relPath]; dup {
			return nil, fmt.Errorf("line %d: %s is listed more than once", i+1, relPath)
		}
		m[relPath] = HashPrefix + sum
	}
	return m, nil
}

// LoadManifest reads and parses the manifest at path. A missing file is
// returned as an error wrapping os.ErrNotExist.
func LoadManifest(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading manifest %s: %w", path, err)
	}
	m, err := ParseManifest(data)
	if err != nil {
		return nil, fmt.Errorf("parsing manifest %s: %w", path, err)
	}
	return m, nil
}

// WriteManifest writes m to ManifestFile under projectRoot atomically.
func WriteManifest(projectRoot string, m Manifest) error {
	data, err := m.Marshal()
	if err != nil {
		return fmt.Errorf("rendering manifest: %w", err)
	}
	if err := fileutil.WriteFileAtomicInRoot(projectRoot, ManifestFile(), data, fileutil.ModeReadWrite); err != nil {
		return fmt.Errorf("writing manifest %s: %w", ManifestFile(), err)
	}
	return nil
}

// SaveInitState persists st as the project's init state file and rewrites the
// committed manifest from it, so the two never disagree about what qsdev
// generated.
func SaveInitState(projectRoot string, st types.GeneratedState) error {
	if err := SaveStateToFile(filepath.Join(projectRoot, InitStateFile()), st); err != nil {
		return err
	}
	return WriteManifest(projectRoot, BuildManifest(st))
}

// validateManifestPath rejects a manifest path that is not a clean,
// slash-separated path inside the project, or that the one-line-per-entry
// format cannot carry.
func validateManifestPath(relPath string) error {
	switch {
	case relPath == "":
		return errors.New("empty path")
	case strings.ContainsAny(relPath, "\n\r\\"):
		return fmt.Errorf("path %q contains a newline or backslash", relPath)
	case !filepath.IsLocal(filepath.FromSlash(relPath)) || filepath.ToSlash(filepath.Clean(filepath.FromSlash(relPath))) != relPath:
		return fmt.Errorf("unsafe path %q: must be a clean relative path inside the project", relPath)
	}
	return nil
}

func isSHA256Hex(s string) bool {
	if len(s) != hex.EncodedLen(32) {
		return false
	}
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}
