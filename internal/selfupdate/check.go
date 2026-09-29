package selfupdate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/mod/semver"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

// cachedCheck stores the result of the most recent update check.
type cachedCheck struct {
	CheckedAt time.Time `json:"checked_at"`
	// AttemptedAt records when a GitHub request was last started, whether or
	// not it completed. It backs off notice-only checks from processes that
	// exit before the request finishes (hooks) or that are offline or
	// rate-limited, which never reach the CheckedAt write.
	AttemptedAt time.Time `json:"attempted_at,omitzero"`
	Version     string    `json:"version,omitempty"`
	URL         string    `json:"url,omitempty"`
	Owner       string    `json:"owner,omitempty"`
	Repo        string    `json:"repo,omitempty"`
}

// githubRelease is the subset of the GitHub API release response we need.
type githubRelease struct {
	TagName    string        `json:"tag_name"`
	HTMLURL    string        `json:"html_url"`
	Body       string        `json:"body"`
	Assets     []githubAsset `json:"assets"`
	Prerelease bool          `json:"prerelease"`
	Draft      bool          `json:"draft"`
}

// githubAsset is the subset of the GitHub API asset response we need.
type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// apiBaseURL can be overridden in tests.
var apiBaseURL = "https://api.github.com"

var errReleaseNotFound = errors.New("release not found")

// attemptBackoff is the minimum interval between GitHub requests started by
// notice-only (background) checks when no successful check is cached within
// CheckInterval.
const attemptBackoff = time.Hour

// ErrDowngrade is returned by ResolveForcedUpdate when the latest published
// release is older than the running version.
var ErrDowngrade = errors.New("latest release is older than the current version")

// CheckForUpdate queries GitHub for the latest release and returns it if
// the latest version is newer than currentVersion. The returned release is
// always complete (tag, assets, release notes), so it can be passed straight
// to DoUpdate. Check results are cached so that an up-to-date answer does not
// hit the API more than once per CheckInterval.
//
// Returns nil (with no error) if:
//   - the current version is already up-to-date
//   - a recent cache entry indicates the current version is up-to-date
//   - the current version string is empty, "dev", or not a semantic version
func CheckForUpdate(ctx context.Context, cfg Config, currentVersion string) (*Release, error) {
	return checkForUpdate(ctx, cfg, currentVersion, false)
}

// checkForUpdateNotice is the cache-first variant used for the background
// "new version available" notice. A cache hit yields a stub Release carrying
// only Version and URL (enough for the notice, NOT for DoUpdate), and request
// attempts are backed off by attemptBackoff.
func checkForUpdateNotice(ctx context.Context, cfg Config, currentVersion string) (*Release, error) {
	return checkForUpdate(ctx, cfg, currentVersion, true)
}

func checkForUpdate(ctx context.Context, cfg Config, currentVersion string, noticeOnly bool) (*Release, error) {
	current, ok := comparableVersion(currentVersion)
	if !ok {
		// Dev builds and unparseable versions cannot be compared.
		return nil, nil
	}

	cached := loadRepoCache(cfg)
	if cached != nil && time.Since(cached.CheckedAt) < cfg.CheckInterval {
		if !isNewerVersion(cached.Version, current) {
			return nil, nil
		}
		if noticeOnly {
			return &Release{Version: cached.Version, URL: cached.URL}, nil
		}
		// An install needs the full release (tag, assets, notes), which the
		// cache does not hold: fall through and fetch it.
	} else if noticeOnly && cached != nil && time.Since(cached.AttemptedAt) < attemptBackoff {
		return nil, nil
	}

	recordAttempt(cfg, cached)

	release, err := FetchLatestRelease(ctx, cfg)
	if err != nil {
		return nil, err
	}

	if release == nil {
		return nil, nil
	}

	// Save to cache regardless of whether an update is available.
	now := time.Now()
	_ = saveCache(cfg, &cachedCheck{
		CheckedAt:   now,
		AttemptedAt: now,
		Version:     release.Version,
		URL:         release.URL,
		Owner:       cfg.GitHubOwner,
		Repo:        cfg.GitHubRepo,
	})

	if !isNewerVersion(release.Version, current) {
		return nil, nil
	}

	return release, nil
}

// ResolveForcedUpdate fetches the latest release without consulting the cache
// or requiring it to be newer (for --force reinstalls), but refuses with
// ErrDowngrade when it is older than currentVersion, so a forced reinstall
// never silently moves a newer (e.g. prerelease) build back. Installing a
// specific older release is done explicitly via FetchRelease. Returns nil,
// nil if no release exists.
func ResolveForcedUpdate(ctx context.Context, cfg Config, currentVersion string) (*Release, error) {
	release, err := FetchLatestRelease(ctx, cfg)
	if err != nil || release == nil {
		return release, err
	}
	if IsDowngrade(release.Version, currentVersion) {
		return nil, fmt.Errorf("%w: latest is v%s, running v%s (install a specific version explicitly to roll back)",
			ErrDowngrade, release.Version, strings.TrimPrefix(currentVersion, "v"))
	}
	return release, nil
}

// IsDowngrade reports whether installing target would move currentVersion
// backwards. Unparseable versions (dev builds) are never reported as a
// downgrade.
func IsDowngrade(target, currentVersion string) bool {
	current, ok := comparableVersion(currentVersion)
	if !ok {
		return false
	}
	t, ok := canonicalVersion(target)
	return ok && semver.Compare(t, current) < 0
}

// describeSuffixRe matches the prerelease part that `git describe --tags
// --dirty` appends to a tag: "-<n>-g<hash>", "-dirty", or both.
var describeSuffixRe = regexp.MustCompile(`^-(?:[0-9]+-g[0-9a-f]+(?:-dirty)?|dirty)$`)

// canonicalVersion normalizes a version string ("1.2.3", "v1.2.3",
// "1.2.3-rc.1+meta") to canonical semver with a single leading "v" and no
// build metadata. It reports false for anything that is not a semantic
// version ("", "dev", "(devel)", a bare commit hash).
func canonicalVersion(v string) (string, bool) {
	if i := strings.IndexByte(v, '+'); i >= 0 {
		v = v[:i]
	}
	v = "v" + strings.TrimPrefix(v, "v")
	if !semver.IsValid(v) {
		return "", false
	}
	return semver.Canonical(v), true
}

// comparableVersion is canonicalVersion for the RUNNING binary. A `git
// describe` build ("v0.8.0-3-gabc123-dirty") is a commit AFTER its base tag,
// not a prerelease of it, so it is reduced to that base: the base release
// itself is then not offered as an update.
func comparableVersion(v string) (string, bool) {
	c, ok := canonicalVersion(v)
	if !ok {
		return "", false
	}
	if pre := semver.Prerelease(c); pre != "" && describeSuffixRe.MatchString(pre) {
		c = strings.TrimSuffix(c, pre)
	}
	return c, true
}

// isNewerVersion reports whether candidate is strictly newer than current,
// which must already be a comparableVersion. Prereleases order before their
// release, per semver.
func isNewerVersion(candidate, current string) bool {
	c, ok := canonicalVersion(candidate)
	return ok && semver.Compare(c, current) > 0
}

// FetchRelease fetches a specific release by tag from GitHub.
func FetchRelease(ctx context.Context, cfg Config, tag string) (*Release, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases/tags/%s",
		apiBaseURL, cfg.GitHubOwner, cfg.GitHubRepo, tag)

	return doFetchRelease(ctx, cfg, url)
}

// FetchLatestRelease fetches the latest release from GitHub.
// Returns nil, nil if no releases exist (404).
func FetchLatestRelease(ctx context.Context, cfg Config) (*Release, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest",
		apiBaseURL, cfg.GitHubOwner, cfg.GitHubRepo)

	release, err := doFetchRelease(ctx, cfg, url)
	if errors.Is(err, errReleaseNotFound) {
		return nil, nil
	}
	return release, err
}

// doFetchRelease performs the HTTP request and parses the response.
func doFetchRelease(ctx context.Context, cfg Config, url string) (*Release, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", cfg.BinaryName+"/self-update")

	// Support optional GITHUB_TOKEN for higher rate limits or private repos.
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, errReleaseNotFound
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(body))
	}

	var gh githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&gh); err != nil {
		return nil, fmt.Errorf("decoding release: %w", err)
	}

	release := &Release{
		Version: strings.TrimPrefix(gh.TagName, "v"),
		TagName: gh.TagName,
		URL:     gh.HTMLURL,
		Body:    gh.Body,
		Assets:  make([]Asset, 0, len(gh.Assets)),
	}

	for _, a := range gh.Assets {
		release.Assets = append(release.Assets, Asset{
			Name: a.Name,
			URL:  a.BrowserDownloadURL,
		})
	}

	return release, nil
}

// cacheFile returns the path to the update check cache file.
func cacheFile(cfg Config) string {
	return filepath.Join(cfg.CacheDir, "update-check.json")
}

// loadRepoCache returns the cached check for cfg's repository, or nil when
// there is none (missing, unreadable, or recorded for a different repo).
func loadRepoCache(cfg Config) *cachedCheck {
	cached, err := loadCache(cfg)
	if err != nil || cached == nil {
		return nil
	}
	if cached.Owner != cfg.GitHubOwner || cached.Repo != cfg.GitHubRepo {
		return nil // stale cache from different repo
	}
	return cached
}

// recordAttempt stamps AttemptedAt on the cache BEFORE a request is started,
// so a process that exits (or fails) before the request completes still backs
// off subsequent notice-only checks.
func recordAttempt(cfg Config, cached *cachedCheck) {
	c := cachedCheck{Owner: cfg.GitHubOwner, Repo: cfg.GitHubRepo}
	if cached != nil {
		c = *cached
	}
	c.AttemptedAt = time.Now()
	_ = saveCache(cfg, &c)
}

// loadCache reads the cached update check result.
func loadCache(cfg Config) (*cachedCheck, error) {
	data, err := os.ReadFile(cacheFile(cfg))
	if err != nil {
		return nil, err
	}
	var c cachedCheck
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// saveCache writes the update check result to the cache file.
func saveCache(cfg Config, c *cachedCheck) error {
	if err := os.MkdirAll(cfg.CacheDir, fileutil.ModeDirDefault); err != nil {
		return err
	}
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	// Atomic, so concurrent processes never read a half-written cache.
	return fileutil.WriteFileAtomic(cacheFile(cfg), data, fileutil.ModeReadWrite)
}
