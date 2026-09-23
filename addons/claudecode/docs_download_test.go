package claudecode

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpregistry"
)

// newDocsTestServer serves ok for paths under /ok/ and 404 for everything else.
func newDocsTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/ok/") {
			_, _ = w.Write([]byte("{}"))
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestDownloadHelpers_ReportFailures pins F098: a failed set must make the
// download fail instead of being printed and then reported as complete.
func TestDownloadHelpers_ReportFailures(t *testing.T) {
	t.Parallel()
	srv := newDocsTestServer(t)

	tests := []struct {
		name       string
		slugs      map[string][]string
		zim        []mcpregistry.ZIMEntry
		wantFailed int
	}{
		{name: "all succeed", slugs: map[string][]string{"go": {"ok"}}, zim: []mcpregistry.ZIMEntry{{Slug: "a", DisplayName: "A", URL: srv.URL + "/ok/a.zim"}}},
		{name: "devdocs failure", slugs: map[string][]string{"go": {"ok", "missing"}}, wantFailed: 1},
		{name: "zim failure", zim: []mcpregistry.ZIMEntry{{Slug: "b", DisplayName: "B", URL: srv.URL + "/missing/b.zim"}}, wantFailed: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := mcpregistry.NewDocsCorpusManager(t.TempDir(), srv.Client())
			var out bytes.Buffer
			var tally downloadTally
			downloadDevDocsSets(context.Background(), &out, mgr, tt.slugs, nil, srv.URL, &tally)
			downloadZIMEntries(context.Background(), &out, mgr, tt.zim, &tally)

			err := tally.err()
			if tt.wantFailed == 0 {
				if err != nil {
					t.Fatalf("unexpected error: %v\n%s", err, out.String())
				}
				return
			}
			if !errors.Is(err, errDocsDownloadFailed) {
				t.Fatalf("err = %v, want errDocsDownloadFailed\n%s", err, out.String())
			}
			if tally.failed != tt.wantFailed {
				t.Errorf("failed = %d, want %d", tally.failed, tt.wantFailed)
			}
		})
	}
}

// TestUpdateOutdatedSets pins F097/F098: every outdated set is attempted, a
// failure or a version missing from the catalog fails the command, and the
// DevDocs path downloads by slug.
func TestUpdateOutdatedSets(t *testing.T) {
	t.Parallel()
	srv := newDocsTestServer(t)
	zimEntries := []mcpregistry.ZIMEntry{
		{Slug: "site_2026-08", URL: srv.URL + "/ok/site_2026-08.zim"},
		{Slug: "broken_2026-08", URL: srv.URL + "/missing/broken_2026-08.zim"},
	}

	tests := []struct {
		name     string
		outdated []mcpregistry.OutdatedEntry
		wantErr  bool
	}{
		{name: "zim ok", outdated: []mcpregistry.OutdatedEntry{{Slug: "site_2026-08", Type: mcpregistry.DocSetZIM, AvailableVersion: "site_2026-08"}}},
		{name: "devdocs ok", outdated: []mcpregistry.OutdatedEntry{{Slug: "ok", Type: mcpregistry.DocSetDevDocs}}},
		{name: "zim download fails", outdated: []mcpregistry.OutdatedEntry{{Slug: "broken_2026-08", Type: mcpregistry.DocSetZIM, AvailableVersion: "broken_2026-08"}}, wantErr: true},
		{name: "zim version not in catalog", outdated: []mcpregistry.OutdatedEntry{{Slug: "gone", Type: mcpregistry.DocSetZIM, AvailableVersion: "gone_2026-08"}}, wantErr: true},
		{name: "devdocs download fails", outdated: []mcpregistry.OutdatedEntry{{Slug: "missing", Type: mcpregistry.DocSetDevDocs}}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := mcpregistry.NewDocsCorpusManager(t.TempDir(), srv.Client())
			var out bytes.Buffer
			err := updateOutdatedSets(context.Background(), &out, mgr, tt.outdated, zimEntries, srv.URL)
			if tt.wantErr != (err != nil) {
				t.Fatalf("err = %v, wantErr %v\n%s", err, tt.wantErr, out.String())
			}
			if tt.wantErr && !errors.Is(err, errDocsDownloadFailed) {
				t.Errorf("err = %v, want errDocsDownloadFailed", err)
			}
		})
	}
}
