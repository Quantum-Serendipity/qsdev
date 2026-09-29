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
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
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
	// and before the manifest entry is recorded. dir is the staging directory
	// holding the complete new set, which replaces the installed set afterwards.
	// It may rewrite files in dir (e.g. sanitize db.json); the manifest hash is
	// computed from the resulting on-disk files. A nil Ingest preserves the
	// prior behavior.
	Ingest func(ctx context.Context, dir string) error
}

// ErrDocsDataDirUnset is returned by DocsCorpusManager operations when no data
// directory is configured, typically because DefaultDocsDataDir could not
// determine the user's home directory.
var ErrDocsDataDirUnset = errors.New("documentation data directory is not set: cannot determine the home directory (set HOME)")

// Download size caps. A ZIM archive may exceed its catalog size estimate, so it
// is capped at twice the estimate plus slack; a size-less entry and each
// DevDocs file get a fixed ceiling that only stops a runaway stream.
const (
	zimSizeSlack        int64 = 256 << 20 // 256 MiB
	maxUnsizedZIMBytes  int64 = 256 << 30 // 256 GiB
	maxDevDocsFileBytes int64 = 1 << 30   // 1 GiB
)

// manifestMu serialises every load-modify-save of a docs manifest within the
// process, so concurrent downloads cannot lose each other's entries.
var manifestMu sync.Mutex

// zimReleaseSuffix matches the _YYYY-MM release date that ends a ZIM slug.
var zimReleaseSuffix = regexp.MustCompile(`_[0-9]{4}-[0-9]{2}$`)

// zimArchiveName returns the stable identity of a ZIM archive: its slug
// without the release date, so unix.stackexchange.com_en_all_2025-06 and
// unix.stackexchange.com_en_all_2026-02 are two releases of the same archive.
func zimArchiveName(slug string) string {
	return zimReleaseSuffix.ReplaceAllString(slug, "")
}

// DefaultDocsDataDir returns the default directory for local documentation
// data under the user-global ~/.qsdev/docs/ directory. It returns "" when the
// home directory is unknown (e.g. HOME unset in CI) rather than a
// working-directory-relative path, and DocsCorpusManager then fails with
// ErrDocsDataDirUnset instead of writing archives into the current directory.
func DefaultDocsDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil || !filepath.IsAbs(home) {
		return ""
	}
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

// checkDataDir reports ErrDocsDataDirUnset when the manager has no data
// directory, so no operation falls back to the working directory.
func (m *DocsCorpusManager) checkDataDir() error {
	if m.DataDir == "" {
		return ErrDocsDataDirUnset
	}
	return nil
}

// manifestPath returns the path to the manifest.json file.
func (m *DocsCorpusManager) manifestPath() string {
	return filepath.Join(m.DataDir, "manifest.json")
}

// LoadManifest reads the documentation manifest from disk. If the file does
// not exist, an empty manifest is returned.
func (m *DocsCorpusManager) LoadManifest() (*DocsManifest, error) {
	if err := m.checkDataDir(); err != nil {
		return nil, err
	}
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

// SaveManifest writes the manifest atomically to disk (unique temp file,
// fsync, rename).
func (m *DocsCorpusManager) SaveManifest(manifest *DocsManifest) error {
	if err := m.checkDataDir(); err != nil {
		return err
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling manifest: %w", err)
	}

	if err := fileutil.WriteFileAtomic(m.manifestPath(), data, fileutil.ModeReadWrite); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}
	return nil
}

// updateManifest loads the manifest, applies fn and saves the result while
// holding manifestMu, so concurrent updates in this process are not lost.
func (m *DocsCorpusManager) updateManifest(fn func(*DocsManifest) error) error {
	manifestMu.Lock()
	defer manifestMu.Unlock()

	manifest, err := m.LoadManifest()
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}
	if err := fn(manifest); err != nil {
		return err
	}
	return m.SaveManifest(manifest)
}

// DownloadDevDocs fetches a DevDocs documentation set (index.json, db.json,
// meta.json) and records it in the manifest. The baseURL parameter specifies
// the DevDocs CDN root (e.g. "https://documents.devdocs.io"). The set is
// downloaded and ingested in a staging directory that replaces the installed
// set only once every file is complete, so a failed download leaves the
// previous set intact and never mixes old and new files.
func (m *DocsCorpusManager) DownloadDevDocs(ctx context.Context, slug, baseURL string) error {
	if err := m.checkDataDir(); err != nil {
		return err
	}
	if baseURL == "" {
		baseURL = "https://documents.devdocs.io"
	}
	parent := filepath.Join(m.DataDir, "devdocs")
	if err := os.MkdirAll(parent, fileutil.ModeDirDefault); err != nil {
		return fmt.Errorf("creating devdocs dir for %q: %w", slug, err)
	}
	dir := filepath.Join(parent, slug)

	stage, err := os.MkdirTemp(parent, "."+slug+".download-*")
	if err != nil {
		return fmt.Errorf("creating staging dir for %q: %w", slug, err)
	}
	// After a successful swap the staging dir no longer exists, so this only
	// cleans up after a failure.
	defer func() { _ = os.RemoveAll(stage) }()

	files := []string{"index.json", "db.json", "meta.json"}
	stagedPaths := make([]string, 0, len(files))
	finalPaths := make([]string, 0, len(files))

	for _, f := range files {
		url := fmt.Sprintf("%s/%s/%s", baseURL, slug, f)
		staged := filepath.Join(stage, f)
		if err := m.downloadFile(ctx, url, staged, maxDevDocsFileBytes); err != nil {
			return fmt.Errorf("downloading %s for %q: %w", f, slug, err)
		}
		stagedPaths = append(stagedPaths, staged)
		finalPaths = append(finalPaths, filepath.Join(dir, f))
	}

	if m.Ingest != nil {
		if err := m.Ingest(ctx, stage); err != nil {
			return fmt.Errorf("ingesting devdocs %q: %w", slug, err)
		}
	}

	// Compute the combined hash and total size from the FINAL (post-ingest)
	// files. The swap moves them unchanged, so the digest matches finalPaths.
	sha, totalSize, err := combinedHashAndSize(stagedPaths)
	if err != nil {
		return fmt.Errorf("hashing devdocs %q: %w", slug, err)
	}

	backup, err := swapDir(stage, dir)
	if err != nil {
		return fmt.Errorf("installing devdocs %q: %w", slug, err)
	}

	err = m.updateManifest(func(manifest *DocsManifest) error {
		manifest.DocSets["devdocs:"+slug] = &DocSetEntry{
			Type:        DocSetDevDocs,
			Slug:        slug,
			Version:     "latest",
			InstalledAt: time.Now(),
			SizeBytes:   totalSize,
			SHA256:      sha,
			Files:       finalPaths,
		}
		return nil
	})
	if err != nil {
		return err
	}
	if backup != "" {
		_ = os.RemoveAll(backup)
	}
	return nil
}

// swapDir replaces dir with the fully-populated stage directory. An existing
// dir is first moved aside and restored if the swap fails; the returned backup
// path (empty when dir did not exist) is for the caller to remove once the new
// set is recorded.
func swapDir(stage, dir string) (backup string, err error) {
	if _, statErr := os.Lstat(dir); statErr == nil {
		backup = stage + ".old"
		if err := os.Rename(dir, backup); err != nil {
			return "", fmt.Errorf("moving aside %s: %w", dir, err)
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return "", fmt.Errorf("checking %s: %w", dir, statErr)
	}

	if err := os.Rename(stage, dir); err != nil {
		if backup != "" {
			_ = os.Rename(backup, dir)
		}
		return "", fmt.Errorf("moving %s into place: %w", stage, err)
	}
	return backup, nil
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

// DownloadZIM fetches a ZIM archive and verifies its SHA-256 digest before
// installing it. The expected digest is the catalog entry's ExpectedHash or,
// when the catalog pins none, the digest the publisher serves beside the
// archive (see expectedZIMHash). The download fails closed when no expected
// digest can be obtained: an unverified archive is never trusted. The archive
// is downloaded to a temporary file and only renamed over the installed copy
// once complete and verified. Installing a new release removes the superseded
// releases of the same archive from the manifest and disk.
func (m *DocsCorpusManager) DownloadZIM(ctx context.Context, entry ZIMEntry) error {
	if err := m.checkDataDir(); err != nil {
		return err
	}
	expectedHash, err := m.expectedZIMHash(ctx, entry)
	if err != nil {
		return fmt.Errorf("verifying zim %q: %w", entry.Slug, err)
	}

	dir := filepath.Join(m.DataDir, "zim")
	if err := os.MkdirAll(dir, fileutil.ModeDirDefault); err != nil {
		return fmt.Errorf("creating zim dir: %w", err)
	}

	destPath := filepath.Join(dir, entry.Slug+".zim")

	tmp, written, computedHash, err := m.downloadToTemp(ctx, entry.URL, dir, zimSizeCap(entry))
	if err != nil {
		return fmt.Errorf("downloading zim %q: %w", entry.Slug, err)
	}
	if !strings.EqualFold(computedHash, expectedHash) {
		_ = os.Remove(tmp)
		return fmt.Errorf("hash mismatch for %q: expected %s, got %s", entry.Slug, expectedHash, computedHash)
	}
	if err := os.Rename(tmp, destPath); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("installing zim file %q: %w", entry.Slug, err)
	}

	var superseded []string
	err = m.updateManifest(func(manifest *DocsManifest) error {
		name := zimArchiveName(entry.Slug)
		for key, installed := range manifest.DocSets {
			if installed.Type != DocSetZIM || installed.Slug == entry.Slug || zimArchiveName(installed.Slug) != name {
				continue
			}
			delete(manifest.DocSets, key)
			for _, f := range installed.Files {
				if f != destPath {
					superseded = append(superseded, f)
				}
			}
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
		return nil
	})
	if err != nil {
		return err
	}
	for _, f := range superseded {
		_ = os.Remove(f)
	}
	return nil
}

// zimSizeCap returns the maximum number of bytes accepted for a ZIM download.
func zimSizeCap(entry ZIMEntry) int64 {
	if entry.SizeBytes <= 0 {
		return maxUnsizedZIMBytes
	}
	return 2*entry.SizeBytes + zimSizeSlack
}

// zimHashSuffix is appended to a ZIM archive URL to fetch the SHA-256 digest
// that download.kiwix.org publishes for every archive it hosts.
const zimHashSuffix = ".sha256"

// maxZIMHashBytes bounds the published-digest response ("<hex>  <name>\n").
const maxZIMHashBytes = 4 << 10

// expectedZIMHash returns the lowercase hex SHA-256 a ZIM download must match:
// the catalog-pinned ExpectedHash when set, otherwise the digest published at
// entry.URL + ".sha256". The published digest is only accepted over HTTPS from
// the archive's own host or one of its subdomains (download.kiwix.org answers
// from lb.download.kiwix.org): archive downloads are redirected to third-party
// mirrors, and a digest served by the same mirror as the archive would verify
// nothing. It never returns an empty digest without an error.
func (m *DocsCorpusManager) expectedZIMHash(ctx context.Context, entry ZIMEntry) (string, error) {
	if entry.ExpectedHash != "" {
		if !isSHA256Hex(entry.ExpectedHash) {
			return "", fmt.Errorf("catalog hash %q is not a hex SHA-256 digest", entry.ExpectedHash)
		}
		return strings.ToLower(entry.ExpectedHash), nil
	}

	hashURL, err := url.Parse(entry.URL + zimHashSuffix)
	if err != nil {
		return "", fmt.Errorf("parsing archive URL: %w", err)
	}
	if hashURL.Scheme != "https" {
		return "", fmt.Errorf("no pinned hash, and the published hash at %s is not served over HTTPS", hashURL)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, hashURL.String(), nil)
	if err != nil {
		return "", fmt.Errorf("creating hash request: %w", err)
	}
	resp, err := m.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching published hash: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetching published hash %s: HTTP %d", hashURL, resp.StatusCode)
	}
	if resp.Request != nil && !sameOrigin(hashURL, resp.Request.URL) {
		return "", fmt.Errorf("published hash %s was redirected to %s; refusing a digest not served by %s over HTTPS",
			hashURL, resp.Request.URL.Redacted(), hashURL.Host)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxZIMHashBytes))
	if err != nil {
		return "", fmt.Errorf("reading published hash: %w", err)
	}
	fields := strings.Fields(string(body))
	if len(fields) == 0 || !isSHA256Hex(fields[0]) {
		return "", fmt.Errorf("published hash %s is not a SHA-256 digest", hashURL)
	}
	return strings.ToLower(fields[0]), nil
}

// sameOrigin reports whether final, the URL a request ended at after
// redirects, is still HTTPS on the host of orig or a subdomain of it, i.e.
// under the control of the same publisher rather than a third-party mirror.
func sameOrigin(orig, final *url.URL) bool {
	if final.Scheme != "https" {
		return false
	}
	host, finalHost := strings.ToLower(orig.Hostname()), strings.ToLower(final.Hostname())
	return finalHost == host || strings.HasSuffix(finalHost, "."+host)
}

// isSHA256Hex reports whether s is a hex-encoded SHA-256 digest.
func isSHA256Hex(s string) bool {
	if len(s) != hex.EncodedLen(sha256.Size) {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}

// CheckOutdated compares installed ZIM entries against the provided catalog
// and returns entries that have newer versions available. Installed and
// catalog archives are matched by their stable name (the slug without its
// release date), so a catalog release newer than every installed release of
// the same archive is reported as outdated. OutdatedEntry.Slug is that stable
// name; AvailableVersion is the catalog slug to download.
func (m *DocsCorpusManager) CheckOutdated(zimEntries []ZIMEntry) ([]OutdatedEntry, error) {
	manifest, err := m.LoadManifest()
	if err != nil {
		return nil, fmt.Errorf("loading manifest: %w", err)
	}

	// Newest installed release (dated slug) per archive name. The release date
	// is a fixed-width YYYY-MM suffix, so slugs order chronologically.
	installed := make(map[string]string)
	for _, e := range manifest.DocSets {
		if e.Type != DocSetZIM {
			continue
		}
		name := zimArchiveName(e.Slug)
		if e.Slug > installed[name] {
			installed[name] = e.Slug
		}
	}

	var outdated []OutdatedEntry
	for _, catalogEntry := range zimEntries {
		name := zimArchiveName(catalogEntry.Slug)
		current, ok := installed[name]
		if !ok || current >= catalogEntry.Slug {
			continue
		}
		outdated = append(outdated, OutdatedEntry{
			Slug:             name,
			Type:             DocSetZIM,
			InstalledVersion: current,
			AvailableVersion: catalogEntry.Slug,
		})
	}

	return outdated, nil
}

// Clean removes documentation files according to the given options and
// updates the manifest.
func (m *DocsCorpusManager) Clean(opts CleanOptions) error {
	return m.updateManifest(func(manifest *DocsManifest) error {
		return m.clean(manifest, opts)
	})
}

// clean removes the selected documentation directories and their manifest
// entries.
func (m *DocsCorpusManager) clean(manifest *DocsManifest, opts CleanOptions) error {
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

	return nil
}

// downloadFile fetches a URL into destPath via a temporary file in the same
// directory, renaming it into place only once the whole body (at most maxBytes)
// has been written.
func (m *DocsCorpusManager) downloadFile(ctx context.Context, url, destPath string, maxBytes int64) error {
	tmp, _, _, err := m.downloadToTemp(ctx, url, filepath.Dir(destPath), maxBytes)
	if err != nil {
		return err
	}
	if err := os.Rename(tmp, destPath); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("renaming %s to %s: %w", tmp, destPath, err)
	}
	return nil
}

// downloadToTemp streams url into a new temporary file in dir, rejecting a
// body larger than maxBytes, and returns the synced, closed file's path, size
// and hex SHA-256. The caller renames it into place or removes it; on error
// nothing is left behind.
func (m *DocsCorpusManager) downloadToTemp(ctx context.Context, url, dir string, maxBytes int64) (tmpPath string, size int64, sha256hex string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", 0, "", fmt.Errorf("creating request: %w", err)
	}

	resp, err := m.HTTPClient.Do(req)
	if err != nil {
		return "", 0, "", fmt.Errorf("fetching %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", 0, "", fmt.Errorf("fetching %s: HTTP %d", url, resp.StatusCode)
	}
	if resp.ContentLength > maxBytes {
		return "", 0, "", fmt.Errorf("fetching %s: size %d exceeds limit %d", url, resp.ContentLength, maxBytes)
	}

	f, err := os.CreateTemp(dir, ".download-*")
	if err != nil {
		return "", 0, "", fmt.Errorf("creating temp file in %s: %w", dir, err)
	}
	defer func() {
		if err != nil {
			_ = f.Close()
			_ = os.Remove(f.Name())
		}
	}()

	hasher := sha256.New()
	size, err = io.Copy(io.MultiWriter(f, hasher), io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return "", 0, "", fmt.Errorf("writing %s: %w", f.Name(), err)
	}
	if size > maxBytes {
		err = fmt.Errorf("fetching %s: response exceeds limit %d", url, maxBytes)
		return "", 0, "", err
	}
	if err = f.Sync(); err != nil {
		return "", 0, "", fmt.Errorf("syncing %s: %w", f.Name(), err)
	}
	if err = f.Close(); err != nil {
		return "", 0, "", fmt.Errorf("closing %s: %w", f.Name(), err)
	}
	if err = os.Chmod(f.Name(), fileutil.ModeReadWrite); err != nil {
		return "", 0, "", fmt.Errorf("chmod %s: %w", f.Name(), err)
	}
	return f.Name(), size, hex.EncodeToString(hasher.Sum(nil)), nil
}
