package mcpregistry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
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
}

// DefaultDocsDataDir returns the default directory for local documentation
// data, respecting the XDG_DATA_HOME convention.
func DefaultDocsDataDir() string {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "qsdev", "docs")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "qsdev", "docs")
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
	if err := os.MkdirAll(m.DataDir, 0o755); err != nil {
		return fmt.Errorf("creating data dir: %w", err)
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling manifest: %w", err)
	}

	tmp := m.manifestPath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("writing temp manifest: %w", err)
	}

	if err := os.Rename(tmp, m.manifestPath()); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("renaming manifest: %w", err)
	}

	return nil
}

// DownloadDevDocs fetches a DevDocs documentation set (index.json, db.json,
// meta.json) and records it in the manifest.
func (m *DocsCorpusManager) DownloadDevDocs(ctx context.Context, slug string) error {
	dir := filepath.Join(m.DataDir, "devdocs", slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating devdocs dir for %q: %w", slug, err)
	}

	files := []string{"index.json", "db.json", "meta.json"}
	var totalSize int64
	var allPaths []string
	combinedHasher := sha256.New()

	for _, f := range files {
		url := fmt.Sprintf("https://devdocs.io/docs/%s/%s", slug, f)
		destPath := filepath.Join(dir, f)

		size, hash, err := m.downloadFile(ctx, url, destPath)
		if err != nil {
			return fmt.Errorf("downloading %s for %q: %w", f, slug, err)
		}

		totalSize += size
		allPaths = append(allPaths, destPath)
		_, _ = combinedHasher.Write([]byte(hash))
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
		SHA256:      hex.EncodeToString(combinedHasher.Sum(nil)),
		Files:       allPaths,
	}

	return m.SaveManifest(manifest)
}

// DownloadZIM fetches a ZIM archive and verifies its SHA256 hash against
// the expected value in the catalog entry.
func (m *DocsCorpusManager) DownloadZIM(ctx context.Context, entry ZIMEntry) error {
	dir := filepath.Join(m.DataDir, "zim")
	if err := os.MkdirAll(dir, 0o755); err != nil {
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

// CheckOutdated compares installed ZIM entries against the builtin catalog
// and returns entries that have newer versions available.
func (m *DocsCorpusManager) CheckOutdated() ([]OutdatedEntry, error) {
	manifest, err := m.LoadManifest()
	if err != nil {
		return nil, fmt.Errorf("loading manifest: %w", err)
	}

	var outdated []OutdatedEntry
	for _, catalogEntry := range BuiltinZIMCatalog {
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
