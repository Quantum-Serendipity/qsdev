package mcpregistry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

// failingBody yields some bytes and then a read error, like a dropped
// connection mid-transfer.
func failingBody(prefix string) io.ReadCloser {
	return io.NopCloser(io.MultiReader(strings.NewReader(prefix), iotestErrReader{}))
}

type iotestErrReader struct{}

func (iotestErrReader) Read([]byte) (int, error) { return 0, errors.New("connection reset") }

func zimEntry(slug string) ZIMEntry {
	return ZIMEntry{Slug: slug, URL: "https://example.com/" + slug + ".zim"}
}

// contentSHA256 returns the hex SHA-256 of content. Test ZIM entries pin it,
// since DownloadZIM refuses an archive without an expected digest (F263).
func contentSHA256(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

// installZIM downloads a ZIM with the given content through mgr.
func installZIM(t *testing.T, mgr *DocsCorpusManager, entry ZIMEntry, content string) {
	t.Helper()
	entry.ExpectedHash = contentSHA256(content)
	mgr.HTTPClient = &mockHTTPClient{responses: map[string]*http.Response{entry.URL: makeResponse(content)}}
	if err := mgr.DownloadZIM(context.Background(), entry); err != nil {
		t.Fatalf("DownloadZIM(%s): %v", entry.Slug, err)
	}
}

// TestCheckOutdated is the F262 regression: the old lookup keyed the manifest
// by the catalog's dated slug and compared it with itself, so it could never
// report a newer release.
func TestCheckOutdated(t *testing.T) {
	t.Parallel()

	const (
		old    = "unix.stackexchange.com_en_all_2025-06"
		newer  = "unix.stackexchange.com_en_all_2026-02"
		future = "unix.stackexchange.com_en_all_2027-01"
	)

	tests := []struct {
		name      string
		installed []string
		catalog   []string
		want      []OutdatedEntry
	}{
		{name: "same release is up to date", installed: []string{newer}, catalog: []string{newer}},
		{
			name:      "newer catalog release is outdated",
			installed: []string{old},
			catalog:   []string{newer},
			want: []OutdatedEntry{{
				Slug:             "unix.stackexchange.com_en_all",
				Type:             DocSetZIM,
				InstalledVersion: old,
				AvailableVersion: newer,
			}},
		},
		{name: "archive not installed is not reported", catalog: []string{newer}},
		{name: "installed newer than catalog is up to date", installed: []string{future}, catalog: []string{newer}},
		{name: "newest of several installed releases is compared", installed: []string{old, newer}, catalog: []string{newer}},
		{name: "other archive is not matched", installed: []string{"serverfault.com_en_all_2025-06"}, catalog: []string{newer}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mgr := NewDocsCorpusManager(t.TempDir(), nil)
			manifest := &DocsManifest{DocSets: map[string]*DocSetEntry{
				"devdocs:go": {Type: DocSetDevDocs, Slug: "go", Version: "latest"},
			}}
			for _, slug := range tt.installed {
				manifest.DocSets["zim:"+slug] = &DocSetEntry{Type: DocSetZIM, Slug: slug, Version: slug}
			}
			if err := mgr.SaveManifest(manifest); err != nil {
				t.Fatalf("SaveManifest: %v", err)
			}

			var catalog []ZIMEntry
			for _, slug := range tt.catalog {
				catalog = append(catalog, zimEntry(slug))
			}

			got, err := mgr.CheckOutdated(catalog)
			if err != nil {
				t.Fatalf("CheckOutdated: %v", err)
			}
			if fmt.Sprint(got) != fmt.Sprint(tt.want) {
				t.Errorf("CheckOutdated() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestDownloadZIM_NewReleaseReplacesSuperseded(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mgr := NewDocsCorpusManager(dir, nil)
	oldEntry := zimEntry("test.stackexchange.com_en_all_2025-06")
	newEntry := zimEntry("test.stackexchange.com_en_all_2026-02")

	installZIM(t, mgr, oldEntry, "old release")
	installZIM(t, mgr, newEntry, "new release")

	manifest, err := mgr.LoadManifest()
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if _, ok := manifest.DocSets["zim:"+oldEntry.Slug]; ok {
		t.Error("superseded release still recorded in manifest")
	}
	if _, ok := manifest.DocSets["zim:"+newEntry.Slug]; !ok {
		t.Error("new release not recorded in manifest")
	}
	if _, err := os.Stat(filepath.Join(dir, "zim", oldEntry.Slug+".zim")); !os.IsNotExist(err) {
		t.Errorf("superseded archive not removed: stat err = %v", err)
	}

	outdated, err := mgr.CheckOutdated([]ZIMEntry{newEntry})
	if err != nil {
		t.Fatalf("CheckOutdated: %v", err)
	}
	if len(outdated) != 0 {
		t.Errorf("CheckOutdated after update = %+v, want none", outdated)
	}
}

// TestDownloadZIM_FailureKeepsInstalledCopy is the F271 regression: the old
// download truncated the installed archive before any byte was verified.
func TestDownloadZIM_FailureKeepsInstalledCopy(t *testing.T) {
	t.Parallel()

	const good = "good installed archive"

	tests := []struct {
		name     string
		resp     func() *http.Response
		expected string
	}{
		{
			name: "connection drops mid-transfer",
			resp: func() *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: failingBody("partial")}
			},
		},
		{
			name:     "hash mismatch",
			resp:     func() *http.Response { return makeResponse("tampered archive") },
			expected: strings.Repeat("0", 64),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			mgr := NewDocsCorpusManager(dir, nil)
			entry := zimEntry("test_2025-06")
			installZIM(t, mgr, entry, good)

			entry.ExpectedHash = tt.expected
			if entry.ExpectedHash == "" {
				entry.ExpectedHash = contentSHA256("complete new archive")
			}
			mgr.HTTPClient = &mockHTTPClient{responses: map[string]*http.Response{entry.URL: tt.resp()}}
			if err := mgr.DownloadZIM(context.Background(), entry); err == nil {
				t.Fatal("DownloadZIM succeeded, want error")
			}

			data, err := os.ReadFile(filepath.Join(dir, "zim", entry.Slug+".zim"))
			if err != nil {
				t.Fatalf("installed archive missing after failed re-download: %v", err)
			}
			if string(data) != good {
				t.Errorf("installed archive = %q, want %q", data, good)
			}
			assertNoTempFiles(t, filepath.Join(dir, "zim"))
		})
	}
}

// TestDownloadDevDocs_FailureKeepsInstalledSet covers the DevDocs half of F271:
// a failure on db.json must not leave a new index.json beside the old files.
func TestDownloadDevDocs_FailureKeepsInstalledSet(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	base := "https://documents.devdocs.io"
	mgr := NewDocsCorpusManager(dir, &mockHTTPClient{responses: map[string]*http.Response{
		base + "/go/index.json": makeResponse(`{"v":1}`),
		base + "/go/db.json":    makeResponse(`{"v":1}`),
		base + "/go/meta.json":  makeResponse(`{"v":1}`),
	}})
	if err := mgr.DownloadDevDocs(context.Background(), "go", base); err != nil {
		t.Fatalf("initial DownloadDevDocs: %v", err)
	}

	mgr.HTTPClient = &mockHTTPClient{responses: map[string]*http.Response{
		base + "/go/index.json": makeResponse(`{"v":2}`),
		base + "/go/db.json":    {StatusCode: http.StatusOK, Body: failingBody(`{"v":`)},
		base + "/go/meta.json":  makeResponse(`{"v":2}`),
	}}
	if err := mgr.DownloadDevDocs(context.Background(), "go", base); err == nil {
		t.Fatal("DownloadDevDocs succeeded, want error")
	}

	for _, f := range []string{"index.json", "db.json", "meta.json"} {
		data, err := os.ReadFile(filepath.Join(dir, "devdocs", "go", f))
		if err != nil {
			t.Fatalf("reading %s: %v", f, err)
		}
		if string(data) != `{"v":1}` {
			t.Errorf("%s = %q, want the previously installed content", f, data)
		}
	}
	assertNoTempFiles(t, filepath.Join(dir, "devdocs"))

	manifest, err := mgr.LoadManifest()
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	ok, _, err := mgr.VerifyHash(manifest.DocSets["devdocs:go"])
	if err != nil || !ok {
		t.Errorf("installed set no longer verifies: ok=%v err=%v", ok, err)
	}
}

func TestDownloadToTemp_RejectsOversizedBody(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	url := "https://example.com/big"
	mgr := NewDocsCorpusManager(dir, &mockHTTPClient{responses: map[string]*http.Response{
		url: makeResponse(strings.Repeat("x", 100)),
	}})

	if _, _, _, err := mgr.downloadToTemp(context.Background(), url, dir, 10); err == nil {
		t.Fatal("downloadToTemp accepted a body over the limit")
	}
	assertNoTempFiles(t, dir)
}

// TestUpdateManifest_ConcurrentDownloadsKeepAllEntries is the F277 regression:
// unlocked load-modify-save cycles lost each other's manifest entries.
func TestUpdateManifest_ConcurrentDownloadsKeepAllEntries(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	const n = 8
	responses := make(map[string]*http.Response, n)
	entries := make([]ZIMEntry, n)
	for i := range entries {
		entries[i] = zimEntry(fmt.Sprintf("site%d.stackexchange.com_en_all_2026-02", i))
		content := fmt.Sprintf("archive %d", i)
		entries[i].ExpectedHash = contentSHA256(content)
		responses[entries[i].URL] = makeResponse(content)
	}
	client := &lockedClient{inner: &mockHTTPClient{responses: responses}}

	var wg sync.WaitGroup
	errs := make(chan error, n)
	for _, e := range entries {
		wg.Add(1)
		go func(e ZIMEntry) {
			defer wg.Done()
			errs <- NewDocsCorpusManager(dir, client).DownloadZIM(context.Background(), e)
		}(e)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("DownloadZIM: %v", err)
		}
	}

	manifest, err := NewDocsCorpusManager(dir, nil).LoadManifest()
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if len(manifest.DocSets) != n {
		t.Errorf("manifest has %d entries, want %d", len(manifest.DocSets), n)
	}
	if _, err := os.Stat(filepath.Join(dir, "manifest.json.tmp")); !os.IsNotExist(err) {
		t.Errorf("fixed-name manifest temp file left behind: stat err = %v", err)
	}
}

// lockedClient serialises access to the non-thread-safe mock's map.
type lockedClient struct {
	mu    sync.Mutex
	inner HTTPClient
}

func (c *lockedClient) Do(req *http.Request) (*http.Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.inner.Do(req)
}

// TestDocsDataDir_NoHome is the F278 regression: with HOME unset the default
// data dir used to become the cwd-relative ".qsdev/docs".
func TestDocsDataDir_NoHome(t *testing.T) {
	if runtime.GOOS == "windows" || runtime.GOOS == "plan9" {
		t.Skip("os.UserHomeDir reads HOME only on Unix")
	}
	t.Setenv("HOME", "")
	cwd := t.TempDir()
	t.Chdir(cwd)

	got := DefaultDocsDataDir()
	if got != "" {
		t.Fatalf("DefaultDocsDataDir() = %q with HOME unset, want empty", got)
	}

	mgr := NewDocsCorpusManager(got, &mockHTTPClient{responses: map[string]*http.Response{
		"https://example.com/x_2026-02.zim": makeResponse("zim"),
	}})
	if _, err := mgr.LoadManifest(); !errors.Is(err, ErrDocsDataDirUnset) {
		t.Errorf("LoadManifest error = %v, want ErrDocsDataDirUnset", err)
	}
	if err := mgr.DownloadZIM(context.Background(), zimEntry("x_2026-02")); !errors.Is(err, ErrDocsDataDirUnset) {
		t.Errorf("DownloadZIM error = %v, want ErrDocsDataDirUnset", err)
	}
	if err := mgr.DownloadDevDocs(context.Background(), "go", ""); !errors.Is(err, ErrDocsDataDirUnset) {
		t.Errorf("DownloadDevDocs error = %v, want ErrDocsDataDirUnset", err)
	}
	if err := mgr.Clean(CleanOptions{All: true}); !errors.Is(err, ErrDocsDataDirUnset) {
		t.Errorf("Clean error = %v, want ErrDocsDataDirUnset", err)
	}

	entries, err := os.ReadDir(cwd)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("working directory gained %d entries, want none", len(entries))
	}
}

// assertNoTempFiles fails when dir (recursively) holds download temp files or
// staging directories.
func assertNoTempFiles(t *testing.T, dir string) {
	t.Helper()
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if strings.Contains(d.Name(), ".download-") || strings.HasSuffix(d.Name(), ".old") {
			t.Errorf("leftover temp path %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", dir, err)
	}
}
