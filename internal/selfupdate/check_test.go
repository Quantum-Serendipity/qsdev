package selfupdate

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
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
	release, err := CheckForUpdate(context.Background(), cfg, "1.0.0")
	if err != nil {
		t.Fatalf("CheckForUpdate() error: %v", err)
	}
	if release == nil {
		t.Fatal("expected a release, got nil")
		return
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
	release, err := CheckForUpdate(context.Background(), cfg, "1.0.0")
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
		release, err := CheckForUpdate(context.Background(), cfg, ver)
		if err != nil {
			t.Errorf("CheckForUpdate(%q) error: %v", ver, err)
		}
		if release != nil {
			t.Errorf("CheckForUpdate(%q) should return nil for dev versions", ver)
		}
	}
}

// countingReleaseServer serves gh at every path and counts requests.
func countingReleaseServer(t *testing.T, gh githubRelease) *atomic.Int32 {
	t.Helper()
	var count atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(gh)
	}))
	t.Cleanup(srv.Close)
	oldBase := apiBaseURL
	apiBaseURL = srv.URL
	t.Cleanup(func() { apiBaseURL = oldBase })
	return &count
}

func TestCheckForUpdate_CacheHit(t *testing.T) {
	gh := githubRelease{
		TagName: "v2.0.0",
		HTMLURL: "https://github.com/test/releases/v2.0.0",
		Body:    "notes",
		Assets:  []githubAsset{{Name: "checksums.txt", BrowserDownloadURL: "https://example.com/c"}},
	}

	t.Run("notice path is served from the cache", func(t *testing.T) {
		requests := countingReleaseServer(t, gh)
		cfg := testConfig(t)
		if _, err := CheckForUpdate(context.Background(), cfg, "1.0.0"); err != nil {
			t.Fatalf("first CheckForUpdate() error: %v", err)
		}
		release, err := checkForUpdateNotice(context.Background(), cfg, "1.0.0")
		if err != nil || release == nil {
			t.Fatalf("cached notice check = %v, %v; want a release", release, err)
		}
		if release.Version != "2.0.0" {
			t.Errorf("Version = %q, want 2.0.0", release.Version)
		}
		if got := requests.Load(); got != 1 {
			t.Errorf("expected 1 API request (cached), got %d", got)
		}
	})

	t.Run("install path returns a complete release", func(t *testing.T) {
		countingReleaseServer(t, gh)
		cfg := testConfig(t)
		if _, err := CheckForUpdate(context.Background(), cfg, "1.0.0"); err != nil {
			t.Fatalf("first CheckForUpdate() error: %v", err)
		}
		release, err := CheckForUpdate(context.Background(), cfg, "1.0.0")
		if err != nil || release == nil {
			t.Fatalf("second CheckForUpdate() = %v, %v; want a release", release, err)
		}
		if release.TagName != "v2.0.0" || len(release.Assets) != 1 || release.Body != "notes" {
			t.Errorf("cache hit returned an incomplete release: %+v", release)
		}
	})

	t.Run("cached up-to-date answer makes no request", func(t *testing.T) {
		requests := countingReleaseServer(t, gh)
		cfg := testConfig(t)
		for range 2 {
			if release, err := CheckForUpdate(context.Background(), cfg, "2.0.0"); err != nil || release != nil {
				t.Fatalf("CheckForUpdate() = %v, %v; want nil, nil", release, err)
			}
		}
		if got := requests.Load(); got != 1 {
			t.Errorf("expected 1 API request, got %d", got)
		}
	})
}

func TestCheckForUpdateNotice_BacksOffAfterAttempt(t *testing.T) {
	tests := []struct {
		name         string
		attemptedAgo time.Duration
		wantRequests int32
	}{
		{name: "recent unfinished attempt suppresses the request", attemptedAgo: time.Minute, wantRequests: 0},
		{name: "old attempt allows a new request", attemptedAgo: 2 * attemptBackoff, wantRequests: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := countingReleaseServer(t, githubRelease{TagName: "v2.0.0"})
			cfg := testConfig(t)
			// An expired successful check plus an attempt that never
			// completed (e.g. a hook process that exited mid-request).
			if err := saveCache(cfg, &cachedCheck{
				CheckedAt:   time.Now().Add(-2 * cfg.CheckInterval),
				AttemptedAt: time.Now().Add(-tt.attemptedAgo),
				Owner:       cfg.GitHubOwner,
				Repo:        cfg.GitHubRepo,
			}); err != nil {
				t.Fatal(err)
			}
			if _, err := checkForUpdateNotice(context.Background(), cfg, "1.0.0"); err != nil {
				t.Fatalf("checkForUpdateNotice() error: %v", err)
			}
			if got := requests.Load(); got != tt.wantRequests {
				t.Errorf("requests = %d, want %d", got, tt.wantRequests)
			}
		})
	}
}

func TestCheckForUpdate_FailedFetchRecordsAttempt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden) // rate limited
	}))
	defer srv.Close()
	oldBase := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = oldBase }()

	cfg := testConfig(t)
	if _, err := checkForUpdateNotice(context.Background(), cfg, "1.0.0"); err == nil {
		t.Fatal("expected an error for a 403 response")
	}
	cached, err := loadCache(cfg)
	if err != nil {
		t.Fatalf("a failed attempt must still be recorded: %v", err)
	}
	if time.Since(cached.AttemptedAt) > time.Minute {
		t.Errorf("AttemptedAt = %v, want now", cached.AttemptedAt)
	}
	if !cached.CheckedAt.IsZero() || cached.Version != "" {
		t.Errorf("a failed attempt must not record a successful check: %+v", cached)
	}
}

func TestIsNewerVersion(t *testing.T) {
	t.Parallel()
	tests := []struct {
		candidate, current string
		want               bool
	}{
		{"1.2.4", "1.2.3", true},
		{"1.2.3", "v1.2.3", false},     // 'v' prefix on the running version
		{"1.2.3", "v1.2.3+abc", false}, // build metadata ignored
		{"2.0.0", "v1.10.0", true},
		{"1.10.0", "1.9.0", true},
		{"0.9.0", "0.9.0-rc.1", true}, // final release supersedes its rc
		{"0.9.0-rc.2", "0.9.0-rc.1", true},
		{"0.9.0-rc.1", "0.9.0", false},
		{"0.8.0", "v0.8.0-3-gabc1234-dirty", false}, // describe build is after its tag
		{"0.8.0", "v0.8.0-dirty", false},
		{"0.8.1", "v0.8.0-3-gabc1234", true},
		// W150: a go-install pseudo-version build of a commit after v0.7.9 must
		// not be offered v0.7.9 (the old dot-split compare saw 0.7.0.0 < 0.7.9).
		{"0.7.9", "v0.7.10-0.20260826231511-039466f3f4e2+dirty", false},
		{"0.7.10", "v0.7.10-0.20260826231511-039466f3f4e2", true},
		{"garbage", "1.0.0", false},
		{"", "1.0.0", false},
	}
	for _, tt := range tests {
		t.Run(tt.candidate+"_vs_"+tt.current, func(t *testing.T) {
			t.Parallel()
			current, ok := comparableVersion(tt.current)
			if !ok {
				t.Fatalf("comparableVersion(%q) not ok", tt.current)
			}
			if got := isNewerVersion(tt.candidate, current); got != tt.want {
				t.Errorf("isNewerVersion(%q, %q) = %v, want %v", tt.candidate, current, got, tt.want)
			}
		})
	}
}

func TestComparableVersion_Unparseable(t *testing.T) {
	t.Parallel()
	for _, v := range []string{"", "dev", "(devel)", "abc1234"} {
		if _, ok := comparableVersion(v); ok {
			t.Errorf("comparableVersion(%q) ok, want not ok (check skipped)", v)
		}
	}
}

func TestResolveForcedUpdate(t *testing.T) {
	tests := []struct {
		name, latest, current string
		wantErr               bool
	}{
		{name: "reinstall same version", latest: "v1.0.0", current: "1.0.0"},
		{name: "newer latest", latest: "v1.1.0", current: "v1.0.0"},
		{name: "latest older than prerelease is refused", latest: "v0.8.5", current: "0.9.0-rc.2", wantErr: true},
		{name: "dev build is not compared", latest: "v0.8.5", current: "dev"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			countingReleaseServer(t, githubRelease{TagName: tt.latest})
			release, err := ResolveForcedUpdate(context.Background(), testConfig(t), tt.current)
			if tt.wantErr {
				if !errors.Is(err, ErrDowngrade) {
					t.Fatalf("error = %v, want ErrDowngrade", err)
				}
				return
			}
			if err != nil || release == nil {
				t.Fatalf("ResolveForcedUpdate() = %v, %v; want a release", release, err)
			}
		})
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
	_, err := CheckForUpdate(context.Background(), cfg, "1.0.0")
	if err != nil {
		t.Fatalf("first CheckForUpdate() error: %v", err)
	}

	// Wait for cache to expire.
	time.Sleep(5 * time.Millisecond)

	// Second call should hit API again because cache expired.
	_, err = CheckForUpdate(context.Background(), cfg, "1.0.0")
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
	release, err := CheckForUpdate(context.Background(), cfg, "1.0.0")
	if err != nil {
		t.Fatalf("CheckForUpdate() error: %v", err)
	}
	if release != nil {
		t.Errorf("expected nil (up to date), got %+v", release)
	}

	// Second call should read cache and still return nil.
	release2, err := CheckForUpdate(context.Background(), cfg, "1.0.0")
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
	_, _ = CheckForUpdate(context.Background(), cfg, "0.9.0")

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
	_, _ = CheckForUpdate(context.Background(), cfg, "0.9.0")

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
	release, err := CheckForUpdate(context.Background(), cfg, "1.0.0")
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
	_, err := CheckForUpdate(context.Background(), cfg, "1.0.0")
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
	release, err := FetchRelease(context.Background(), cfg, "v1.5.0")
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
