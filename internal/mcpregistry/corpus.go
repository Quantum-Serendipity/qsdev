package mcpregistry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

// DocSetType classifies a documentation set by its format.
type DocSetType int

const (
	DocSetDevDocs DocSetType = iota
	DocSetZIM
	DocSetManPages
)

// String returns a human-readable label for the documentation set type.
func (d DocSetType) String() string {
	switch d {
	case DocSetDevDocs:
		return "devdocs"
	case DocSetZIM:
		return "zim"
	case DocSetManPages:
		return "manpages"
	default:
		return "unknown"
	}
}

// DocSetEntry tracks a single downloaded documentation set.
type DocSetEntry struct {
	Type        DocSetType `json:"type"`
	Slug        string     `json:"slug"`
	Version     string     `json:"version"`
	InstalledAt time.Time  `json:"installed_at"`
	SizeBytes   int64      `json:"size_bytes"`
	SHA256      string     `json:"sha256"`
	Files       []string   `json:"files"`
}

// DocsManifest records all documentation sets managed by qsdev.
type DocsManifest struct {
	DocSets map[string]*DocSetEntry `json:"doc_sets"`
}

// CleanOptions controls which documentation sets are removed by Clean.
type CleanOptions struct {
	ZIMOnly     bool
	DevDocsOnly bool
	All         bool
}

// OutdatedEntry describes a documentation set that has a newer version available.
type OutdatedEntry struct {
	Slug             string
	Type             DocSetType
	InstalledVersion string
	AvailableVersion string
}

// HTTPClient abstracts HTTP requests for testability.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// DocsCorpusManager handles downloading, tracking, and cleaning local
// documentation sets (DevDocs, ZIM archives, man pages).
type DocsCorpusManager struct {
	DataDir    string
	HTTPClient HTTPClient

	// Ingest, when non-nil, is invoked after a DevDocs set's files are downloaded
	// and before the manifest entry is recorded. It may rewrite files in dir
	// (e.g. sanitize db.json); the manifest hash is computed from the resulting
	// on-disk files. A nil Ingest preserves the prior behavior.
	Ingest func(ctx context.Context, dir string) error
}

// DefaultDocsDataDir returns the default directory for local documentation
// data under the user-global ~/.qsdev/docs/ directory.
func DefaultDocsDataDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".qsdev", "docs")
}

// NewDocsCorpusManager creates a DocsCorpusManager with the given data
// directory and HTTP client.
func NewDocsCorpusManager(dataDir string, client HTTPClient) *DocsCorpusManager {
	return &DocsCorpusManager{
		DataDir:    dataDir,
		HTTPClient: client,
	}
}

// manifestPath returns the path to the manifest.json file.
func (m *DocsCorpusManager) manifestPath() string {
	return filepath.Join(m.DataDir, "manifest.json")
}

// LoadManifest reads the documentation manifest from disk. If the file does
// not exist, an empty manifest is returned.
func (m *DocsCorpusManager) LoadManifest() (*DocsManifest, error) {
	data, err := os.ReadFile(m.manifestPath())
	if err != nil {
		if os.IsNotExist(err) {
			return &DocsManifest{DocSets: make(map[string]*DocSetEntry)}, nil
		}
		return nil, fmt.Errorf("reading manifest: %w", err)
	}

	var manifest DocsManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}
	if manifest.DocSets == nil {
		manifest.DocSets = make(map[string]*DocSetEntry)
	}
	return &manifest, nil
}

// SaveManifest writes the manifest atomically to disk.
func (m *DocsCorpusManager) SaveManifest(manifest *DocsManifest) error {
	if err := os.MkdirAll(m.DataDir, fileutil.ModeDirDefault); err != nil {
		return fmt.Errorf("creating data dir: %w", err)
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling manifest: %w", err)
	}

	tmp := m.manifestPath() + ".tmp"
	if err := os.WriteFile(tmp, data, fileutil.ModeReadWrite); err != nil {
		return fmt.Errorf("writing temp manifest: %w", err)
	}

	if err := os.Rename(tmp, m.manifestPath()); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("renaming manifest: %w", err)
	}

	return nil
}

// DownloadDevDocs fetches a DevDocs documentation set (index.json, db.json,
// meta.json) and records it in the manifest. The baseURL parameter specifies
// the DevDocs CDN root (e.g. "https://documents.devdocs.io").
func (m *DocsCorpusManager) DownloadDevDocs(ctx context.Context, slug, baseURL string) error {
	if baseURL == "" {
		baseURL = "https://documents.devdocs.io"
	}
	dir := filepath.Join(m.DataDir, "devdocs", slug)
	if err := os.MkdirAll(dir, fileutil.ModeDirDefault); err != nil {
		return fmt.Errorf("creating devdocs dir for %q: %w", slug, err)
	}

	files := []string{"index.json", "db.json", "meta.json"}
	var allPaths []string

	for _, f := range files {
		url := fmt.Sprintf("%s/%s/%s", baseURL, slug, f)
		destPath := filepath.Join(dir, f)

		if _, _, err := m.downloadFile(ctx, url, destPath); err != nil {
			return fmt.Errorf("downloading %s for %q: %w", f, slug, err)
		}

		allPaths = append(allPaths, destPath)
	}

	if m.Ingest != nil {
		if err := m.Ingest(ctx, dir); err != nil {
			return fmt.Errorf("ingesting devdocs %q: %w", slug, err)
		}
	}

	// Compute the combined hash and total size from the FINAL on-disk files so
	// the manifest always reflects post-ingest content.
	sha, totalSize, err := combinedHashAndSize(allPaths)
	if err != nil {
		return fmt.Errorf("hashing devdocs %q: %w", slug, err)
	}

	manifest, err := m.LoadManifest()
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}

	manifest.DocSets["devdocs:"+slug] = &DocSetEntry{
		Type:        DocSetDevDocs,
		Slug:        slug,
		Version:     "latest",
		InstalledAt: time.Now(),
		SizeBytes:   totalSize,
		SHA256:      sha,
		Files:       allPaths,
	}

	return m.SaveManifest(manifest)
}

// combinedHashAndSize recomputes, from the on-disk files, the same combined
// digest and total size that DownloadDevDocs records: for each path in order it
// SHA-256s the file content, writes that file's hex digest into a combined
// hasher, and accumulates the byte count. The returned sha256hex is the hex of
// the combined hasher's sum. Iterating the paths in the same order with the
// same per-file algorithm yields the identical value to the download-time hash
// when content is unchanged. Files are streamed, so a multi-MB db.json is never
// buffered whole.
func combinedHashAndSize(paths []string) (sha256hex string, total int64, err error) {
	combinedHasher := sha256.New()
	for _, p := range paths {
		size, fileHash, err := hashFile(p)
		if err != nil {
			return "", 0, err
		}
		_, _ = combinedHasher.Write([]byte(fileHash))
		total += size
	}
	return hex.EncodeToString(combinedHasher.Sum(nil)), total, nil
}

// hashFile streams the file at path and returns its byte count and lowercase-hex
// SHA-256 digest. The open error is wrapped with %w so callers can detect a
// missing file via errors.Is(err, os.ErrNotExist).
func hashFile(path string) (size int64, sha256hex string, err error) {
	f, err := os.Open(path) //nolint:gosec // path is a manifest-controlled corpus path.
	if err != nil {
		return 0, "", fmt.Errorf("reading %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return 0, "", fmt.Errorf("hashing %s: %w", path, err)
	}
	return n, hex.EncodeToString(h.Sum(nil)), nil
}

// VerifyHash recomputes the combined SHA-256 over the entry's files and reports
// whether it matches the recorded entry.SHA256, returning the computed digest.
// A missing file is a verification failure (ok=false) rather than a hard error —
// it yields ok=false, computed="", err=nil so callers can report it cleanly. Any
// other I/O error is returned as err.
func (m *DocsCorpusManager) VerifyHash(entry *DocSetEntry) (ok bool, computed string, err error) {
	sum, _, err := combinedHashAndSize(entry.Files)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, "", nil
		}
		return false, "", fmt.Errorf("hashing doc set %q: %w", entry.Slug, err)
	}
	return strings.EqualFold(sum, entry.SHA256), sum, nil
}

// DownloadZIM fetches a ZIM archive and verifies its SHA256 hash against
// the expected value in the catalog entry.
func (m *DocsCorpusManager) DownloadZIM(ctx context.Context, entry ZIMEntry) error {
	dir := filepath.Join(m.DataDir, "zim")
	if err := os.MkdirAll(dir, fileutil.ModeDirDefault); err != nil {
		return fmt.Errorf("creating zim dir: %w", err)
	}

	destPath := filepath.Join(dir, entry.Slug+".zim")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, entry.URL, nil)
	if err != nil {
		return fmt.Errorf("creating request for %q: %w", entry.Slug, err)
	}

	resp, err := m.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("downloading zim %q: %w", entry.Slug, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("downloading zim %q: HTTP %d", entry.Slug, resp.StatusCode)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("creating zim file %q: %w", entry.Slug, err)
	}

	hasher := sha256.New()
	written, err := io.Copy(f, io.TeeReader(resp.Body, hasher))
	if closeErr := f.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(destPath)
		return fmt.Errorf("writing zim file %q: %w", entry.Slug, err)
	}

	computedHash := hex.EncodeToString(hasher.Sum(nil))
	if entry.ExpectedHash != "" && computedHash != entry.ExpectedHash {
		_ = os.Remove(destPath)
		return fmt.Errorf("hash mismatch for %q: expected %s, got %s", entry.Slug, entry.ExpectedHash, computedHash)
	}

	manifest, err := m.LoadManifest()
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}

	manifest.DocSets["zim:"+entry.Slug] = &DocSetEntry{
		Type:        DocSetZIM,
		Slug:        entry.Slug,
		Version:     entry.Slug,
		InstalledAt: time.Now(),
		SizeBytes:   written,
		SHA256:      computedHash,
		Files:       []string{destPath},
	}

	return m.SaveManifest(manifest)
}

// CheckOutdated compares installed ZIM entries against the provided catalog
// and returns entries that have newer versions available.
func (m *DocsCorpusManager) CheckOutdated(zimEntries []ZIMEntry) ([]OutdatedEntry, error) {
	manifest, err := m.LoadManifest()
	if err != nil {
		return nil, fmt.Errorf("loading manifest: %w", err)
	}

	var outdated []OutdatedEntry
	for _, catalogEntry := range zimEntries {
		key := "zim:" + catalogEntry.Slug
		installed, ok := manifest.DocSets[key]
		if !ok {
			continue
		}
		if installed.Version != catalogEntry.Slug {
			outdated = append(outdated, OutdatedEntry{
				Slug:             catalogEntry.Slug,
				Type:             DocSetZIM,
				InstalledVersion: installed.Version,
				AvailableVersion: catalogEntry.Slug,
			})
		}
	}

	return outdated, nil
}

// Clean removes documentation files according to the given options and
// updates the manifest.
func (m *DocsCorpusManager) Clean(opts CleanOptions) error {
	manifest, err := m.LoadManifest()
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}

	if opts.All || opts.ZIMOnly {
		zimDir := filepath.Join(m.DataDir, "zim")
		if err := os.RemoveAll(zimDir); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing zim dir: %w", err)
		}
		for key, entry := range manifest.DocSets {
			if entry.Type == DocSetZIM {
				delete(manifest.DocSets, key)
			}
		}
	}

	if opts.All || opts.DevDocsOnly {
		devdocsDir := filepath.Join(m.DataDir, "devdocs")
		if err := os.RemoveAll(devdocsDir); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing devdocs dir: %w", err)
		}
		for key, entry := range manifest.DocSets {
			if entry.Type == DocSetDevDocs {
				delete(manifest.DocSets, key)
			}
		}
	}

	return m.SaveManifest(manifest)
}

// downloadFile fetches a URL and writes it to destPath, returning the file
// size and hex-encoded SHA256 hash.
func (m *DocsCorpusManager) downloadFile(ctx context.Context, url, destPath string) (int64, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, "", fmt.Errorf("creating request: %w", err)
	}

	resp, err := m.HTTPClient.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("fetching %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, "", fmt.Errorf("fetching %s: HTTP %d", url, resp.StatusCode)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return 0, "", fmt.Errorf("creating %s: %w", destPath, err)
	}

	hasher := sha256.New()
	written, err := io.Copy(f, io.TeeReader(resp.Body, hasher))
	if closeErr := f.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(destPath)
		return 0, "", fmt.Errorf("writing %s: %w", destPath, err)
	}

	return written, hex.EncodeToString(hasher.Sum(nil)), nil
}
