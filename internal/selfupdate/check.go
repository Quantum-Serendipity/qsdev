package selfupdate

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/doctor"
)

// cachedCheck stores the result of the most recent update check.
type cachedCheck struct {
	CheckedAt time.Time `json:"checked_at"`
	Version   string    `json:"version,omitempty"`
	URL       string    `json:"url,omitempty"`
	Owner     string    `json:"owner,omitempty"`
	Repo      string    `json:"repo,omitempty"`
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

// CheckForUpdate queries GitHub for the latest release and returns it if
// the latest version is newer than currentVersion. It caches check results
// to avoid hitting the API too frequently.
//
// Returns nil (with no error) if:
//   - the current version is already up-to-date
//   - a recent cache entry indicates we checked recently
//   - the current version string is empty or "dev"
// stripBuildMeta removes semver build metadata (everything after "+") so that
// version comparison works correctly. "0.3.0+cee1fee" → "0.3.0".
func stripBuildMeta(v string) string {
	if idx := strings.Index(v, "+"); idx >= 0 {
		return v[:idx]
	}
	return v
}

func CheckForUpdate(cfg Config, currentVersion string) (*Release, error) {
	// Skip check for dev builds.
	if currentVersion == "" || currentVersion == "dev" || currentVersion == "(devel)" {
		return nil, nil
	}

	cleanCurrent := stripBuildMeta(currentVersion)

	// Check cache first.
	cached, err := loadCache(cfg)
	if err == nil && cached != nil {
		if cached.Owner != cfg.GitHubOwner || cached.Repo != cfg.GitHubRepo {
			cached = nil // stale cache from different repo
		}
	}
	if err == nil && cached != nil {
		if time.Since(cached.CheckedAt) < cfg.CheckInterval {
			// We checked recently. Only return a release if the cached
			// version is newer than current.
			if cached.Version != "" && doctor.CompareVersions(stripBuildMeta(cached.Version), cleanCurrent) > 0 {
				return &Release{
					Version: cached.Version,
					URL:     cached.URL,
				}, nil
			}
			return nil, nil
		}
	}

	// Fetch latest release from GitHub.
	release, err := fetchLatestRelease(cfg)
	if err != nil {
		return nil, err
	}

	if release == nil {
		return nil, nil
	}

	// Save to cache regardless of whether an update is available.
	_ = saveCache(cfg, &cachedCheck{
		CheckedAt: time.Now(),
		Version:   release.Version,
		URL:       release.URL,
		Owner:     cfg.GitHubOwner,
		Repo:      cfg.GitHubRepo,
	})

	// Compare versions.
	if doctor.CompareVersions(stripBuildMeta(release.Version), cleanCurrent) <= 0 {
		return nil, nil
	}

	return release, nil
}

// FetchRelease fetches a specific release by tag from GitHub.
func FetchRelease(cfg Config, tag string) (*Release, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases/tags/%s",
		apiBaseURL, cfg.GitHubOwner, cfg.GitHubRepo, tag)

	return doFetchRelease(cfg, url)
}

// fetchLatestRelease fetches the latest release from GitHub.
// Returns nil, nil if no releases exist (404).
func fetchLatestRelease(cfg Config) (*Release, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest",
		apiBaseURL, cfg.GitHubOwner, cfg.GitHubRepo)

	release, err := doFetchRelease(cfg, url)
	if errors.Is(err, errReleaseNotFound) {
		return nil, nil
	}
	return release, err
}

// doFetchRelease performs the HTTP request and parses the response.
func doFetchRelease(cfg Config, url string) (*Release, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", cfg.BinaryName+"/self-update")

	// Support optional GITHUB_TOKEN for higher rate limits or private repos.
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
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
	if err := os.MkdirAll(cfg.CacheDir, 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(cacheFile(cfg), data, 0o644)
}
