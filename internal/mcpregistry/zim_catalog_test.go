package mcpregistry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDownloadZIM_UnpinnedHash is the F263 regression: every built-in ZIM
// entry has no pinned hash, and DownloadZIM used to skip verification for
// those and trust whatever a mirror served. Without a pinned hash it must
// verify against the digest published on the archive's own host over HTTPS,
// and refuse the download whenever that digest is unavailable.
func TestDownloadZIM_UnpinnedHash(t *testing.T) {
	t.Parallel()

	const content = "fake zim content"
	sum := sha256.Sum256([]byte(content))
	goodHash := hex.EncodeToString(sum[:])

	const archiveURL = "https://download.example/zim/a.zim"
	hashURL := archiveURL + zimHashSuffix

	redirectedTo := func(scheme, host string) *http.Response {
		resp := makeResponse(goodHash + "  a.zim\n")
		resp.Request = &http.Request{URL: &url.URL{Scheme: scheme, Host: host, Path: "/zim/a.zim.sha256"}}
		return resp
	}

	tests := []struct {
		name      string
		url       string
		hashResp  *http.Response
		wantErr   string
		wantSaved bool
	}{
		{name: "matching published hash", url: archiveURL, hashResp: makeResponse(strings.ToUpper(goodHash) + "  a.zim\n"), wantSaved: true},
		{name: "published hash mismatch", url: archiveURL, hashResp: makeResponse(strings.Repeat("0", 64) + "  a.zim\n"), wantErr: "hash mismatch"},
		{name: "published hash missing", url: archiveURL, wantErr: "HTTP 404"},
		{name: "published hash malformed", url: archiveURL, hashResp: makeResponse("<html>oops</html>"), wantErr: "not a SHA-256"},
		{name: "published hash redirected to own subdomain", url: archiveURL, hashResp: redirectedTo("https", "lb.download.example"), wantSaved: true},
		{name: "published hash redirected to mirror", url: archiveURL, hashResp: redirectedTo("https", "mirror.example"), wantErr: "redirected"},
		{name: "published hash redirected to lookalike host", url: archiveURL, hashResp: redirectedTo("https", "evildownload.example"), wantErr: "redirected"},
		{name: "published hash downgraded to http", url: archiveURL, hashResp: redirectedTo("http", "download.example"), wantErr: "redirected"},
		{name: "plain http archive", url: "http://download.example/zim/a.zim", wantErr: "HTTPS"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			responses := map[string]*http.Response{
				tt.url: makeResponse(content),
			}
			if tt.hashResp != nil {
				responses[hashURL] = tt.hashResp
			}
			mgr := NewDocsCorpusManager(dir, &mockHTTPClient{responses: responses})

			err := mgr.DownloadZIM(context.Background(), ZIMEntry{Slug: "a", URL: tt.url})
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("DownloadZIM: %v", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("DownloadZIM error = %v, want it to contain %q", err, tt.wantErr)
			}

			_, statErr := os.Stat(filepath.Join(dir, "zim", "a.zim"))
			if saved := statErr == nil; saved != tt.wantSaved {
				t.Errorf("archive saved = %v, want %v", saved, tt.wantSaved)
			}
			manifest, mErr := mgr.LoadManifest()
			if mErr != nil {
				t.Fatalf("LoadManifest: %v", mErr)
			}
			if _, recorded := manifest.DocSets["zim:a"]; recorded != tt.wantSaved {
				t.Errorf("manifest entry recorded = %v, want %v", recorded, tt.wantSaved)
			}
		})
	}
}

func TestDownloadZIM_RejectsMalformedPinnedHash(t *testing.T) {
	t.Parallel()

	mgr := NewDocsCorpusManager(t.TempDir(), &mockHTTPClient{})
	err := mgr.DownloadZIM(context.Background(), ZIMEntry{Slug: "a", URL: "https://download.example/a.zim", ExpectedHash: "abc"})
	if err == nil || !strings.Contains(err.Error(), "not a hex SHA-256") {
		t.Errorf("DownloadZIM error = %v, want a malformed-hash error", err)
	}
}
