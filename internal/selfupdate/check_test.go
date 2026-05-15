package selfupdate

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func testConfig(t *testing.T) Config {
	t.Helper()
	return Config{
		GitHubOwner:   "test-owner",
		GitHubRepo:    "test-repo",
		BinaryName:    "qsdev",
		CheckInterval: 7 * 24 * time.Hour,
		CacheDir:      t.TempDir(),
	}
}

func TestCheckForUpdate_NewerVersionAvailable(t *testing.T) {
	gh := githubRelease{
		TagName: "v2.0.0",
		HTMLURL: "https://github.com/test/releases/v2.0.0",
		Body:    "New features",
		Assets: []githubAsset{
			{Name: "qsdev_2.0.0_Linux_x86_64.tar.gz", BrowserDownloadURL: "https://example.com/qsdev.tar.gz"},
			{Name: "checksums.txt", BrowserDownloadURL: "https://example.com/checksums.txt"},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(gh)
	}))
	defer srv.Close()

	oldBase := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = oldBase }()

	cfg := testConfig(t)
	release, err := CheckForUpdate(cfg, "1.0.0")
	if err != nil {
		t.Fatalf("CheckForUpdate() error: %v", err)
	}
	if release == nil {
		t.Fatal("expected a release, got nil")
	}
	if release.Version != "2.0.0" {
		t.Errorf("Version = %q, want %q", release.Version, "2.0.0")
	}
	if len(release.Assets) != 2 {
		t.Errorf("len(Assets) = %d, want 2", len(release.Assets))
	}
}

func TestCheckForUpdate_AlreadyUpToDate(t *testing.T) {
	gh := githubRelease{
		TagName: "v1.0.0",
		HTMLURL: "https://github.com/test/releases/v1.0.0",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(gh)
	}))
	defer srv.Close()

	oldBase := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = oldBase }()

	cfg := testConfig(t)
	release, err := CheckForUpdate(cfg, "1.0.0")
	if err != nil {
		t.Fatalf("CheckForUpdate() error: %v", err)
	}
	if release != nil {
		t.Errorf("expected nil release when up to date, got %+v", release)
	}
}

func TestCheckForUpdate_DevVersion(t *testing.T) {
	cfg := testConfig(t)

	for _, ver := range []string{"", "dev", "(devel)"} {
		release, err := CheckForUpdate(cfg, ver)
		if err != nil {
			t.Errorf("CheckForUpdate(%q) error: %v", ver, err)
		}
		if release != nil {
			t.Errorf("CheckForUpdate(%q) should return nil for dev versions", ver)
		}
	}
}

func TestCheckForUpdate_CacheHit(t *testing.T) {
	requestCount := 0
	gh := githubRelease{
		TagName: "v2.0.0",
		HTMLURL: "https://github.com/test/releases/v2.0.0",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(gh)
	}))
	defer srv.Close()

	oldBase := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = oldBase }()

	cfg := testConfig(t)

	// First call should hit the API.
	release, err := CheckForUpdate(cfg, "1.0.0")
	if err != nil {
		t.Fatalf("first CheckForUpdate() error: %v", err)
	}
	if release == nil {
		t.Fatal("first call: expected release, got nil")
	}
	if requestCount != 1 {
		t.Errorf("expected 1 API request, got %d", requestCount)
	}

	// Second call should use cache.
	release2, err := CheckForUpdate(cfg, "1.0.0")
	if err != nil {
		t.Fatalf("second CheckForUpdate() error: %v", err)
	}
	if release2 == nil {
		t.Fatal("second call: expected release from cache, got nil")
	}
	if requestCount != 1 {
		t.Errorf("expected 1 API request (cached), got %d", requestCount)
	}
}

func TestCheckForUpdate_CacheExpired(t *testing.T) {
	requestCount := 0
	gh := githubRelease{
		TagName: "v2.0.0",
		HTMLURL: "https://github.com/test/releases/v2.0.0",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(gh)
	}))
	defer srv.Close()

	oldBase := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = oldBase }()

	cfg := testConfig(t)
	cfg.CheckInterval = 1 * time.Millisecond // Very short interval for testing.

	// First call.
	_, err := CheckForUpdate(cfg, "1.0.0")
	if err != nil {
		t.Fatalf("first CheckForUpdate() error: %v", err)
	}

	// Wait for cache to expire.
	time.Sleep(5 * time.Millisecond)

	// Second call should hit API again because cache expired.
	_, err = CheckForUpdate(cfg, "1.0.0")
	if err != nil {
		t.Fatalf("second CheckForUpdate() error: %v", err)
	}
	if requestCount != 2 {
		t.Errorf("expected 2 API requests (cache expired), got %d", requestCount)
	}
}

func TestCheckForUpdate_CacheUpToDate(t *testing.T) {
	gh := githubRelease{
		TagName: "v1.0.0",
		HTMLURL: "https://github.com/test/releases/v1.0.0",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(gh)
	}))
	defer srv.Close()

	oldBase := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = oldBase }()

	cfg := testConfig(t)

	// First call caches "1.0.0" as latest.
	release, err := CheckForUpdate(cfg, "1.0.0")
	if err != nil {
		t.Fatalf("CheckForUpdate() error: %v", err)
	}
	if release != nil {
		t.Errorf("expected nil (up to date), got %+v", release)
	}

	// Second call should read cache and still return nil.
	release2, err := CheckForUpdate(cfg, "1.0.0")
	if err != nil {
		t.Fatalf("cached CheckForUpdate() error: %v", err)
	}
	if release2 != nil {
		t.Errorf("cached: expected nil (up to date), got %+v", release2)
	}
}

func TestCheckForUpdate_GithubTokenInjected(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(githubRelease{TagName: "v1.0.0"})
	}))
	defer srv.Close()

	oldBase := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = oldBase }()

	t.Setenv("GITHUB_TOKEN", "ghp_testtoken123")

	cfg := testConfig(t)
	_, _ = CheckForUpdate(cfg, "0.9.0")

	if gotAuth != "Bearer ghp_testtoken123" {
		t.Errorf("Authorization header = %q, want %q", gotAuth, "Bearer ghp_testtoken123")
	}
}

func TestCheckForUpdate_NoGithubToken(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(githubRelease{TagName: "v1.0.0"})
	}))
	defer srv.Close()

	oldBase := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = oldBase }()

	t.Setenv("GITHUB_TOKEN", "")

	cfg := testConfig(t)
	_, _ = CheckForUpdate(cfg, "0.9.0")

	if gotAuth != "" {
		t.Errorf("Authorization header should be empty when no token, got %q", gotAuth)
	}
}

func TestCheckForUpdate_NoReleases(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"Not Found"}`))
	}))
	defer srv.Close()

	oldBase := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = oldBase }()

	cfg := testConfig(t)
	release, err := CheckForUpdate(cfg, "1.0.0")
	if err != nil {
		t.Fatalf("expected no error for 404 (no releases), got: %v", err)
	}
	if release != nil {
		t.Fatalf("expected nil release for 404, got: %+v", release)
	}
}

func TestCheckForUpdate_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message":"Internal Server Error"}`))
	}))
	defer srv.Close()

	oldBase := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = oldBase }()

	cfg := testConfig(t)
	_, err := CheckForUpdate(cfg, "1.0.0")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestFetchRelease(t *testing.T) {
	gh := githubRelease{
		TagName: "v1.5.0",
		HTMLURL: "https://github.com/test/releases/v1.5.0",
		Body:    "Specific release",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/test-owner/test-repo/releases/tags/v1.5.0" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(gh)
	}))
	defer srv.Close()

	oldBase := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = oldBase }()

	cfg := testConfig(t)
	release, err := FetchRelease(cfg, "v1.5.0")
	if err != nil {
		t.Fatalf("FetchRelease() error: %v", err)
	}
	if release.Version != "1.5.0" {
		t.Errorf("Version = %q, want %q", release.Version, "1.5.0")
	}
}

func TestCacheFileLocation(t *testing.T) {
	cfg := Config{CacheDir: "/tmp/test-qsdev"}
	expected := filepath.Join("/tmp/test-qsdev", "update-check.json")
	if got := cacheFile(cfg); got != expected {
		t.Errorf("cacheFile() = %q, want %q", got, expected)
	}
}

func TestSaveAndLoadCache(t *testing.T) {
	cfg := testConfig(t)

	c := &cachedCheck{
		CheckedAt: time.Now().Truncate(time.Second),
		Version:   "2.0.0",
		URL:       "https://example.com",
	}

	if err := saveCache(cfg, c); err != nil {
		t.Fatalf("saveCache() error: %v", err)
	}

	loaded, err := loadCache(cfg)
	if err != nil {
		t.Fatalf("loadCache() error: %v", err)
	}

	if loaded.Version != c.Version {
		t.Errorf("loaded Version = %q, want %q", loaded.Version, c.Version)
	}
	if loaded.URL != c.URL {
		t.Errorf("loaded URL = %q, want %q", loaded.URL, c.URL)
	}
}

func TestLoadCache_Missing(t *testing.T) {
	cfg := Config{CacheDir: t.TempDir()}
	_, err := loadCache(cfg)
	if err == nil {
		t.Error("expected error loading missing cache")
	}
}

func TestLoadCache_Invalid(t *testing.T) {
	cfg := Config{CacheDir: t.TempDir()}
	os.WriteFile(cacheFile(cfg), []byte("not json"), 0o644)
	_, err := loadCache(cfg)
	if err == nil {
		t.Error("expected error loading invalid cache")
	}
}
