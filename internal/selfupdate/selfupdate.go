// Package selfupdate implements self-update functionality for the qsdev binary.
// It checks GitHub releases for newer versions, downloads and verifies archives,
// and replaces the running binary with rollback support.
package selfupdate

import (
	"os"
	"path/filepath"
	"time"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// Release represents a GitHub release.
type Release struct {
	Version string  // Semantic version (without leading "v")
	TagName string  // Original tag name (e.g. "v1.2.3")
	URL     string  // HTML URL to the release page
	Body    string  // Release notes / changelog
	Assets  []Asset // Downloadable assets
}

// Asset represents a downloadable file attached to a release.
type Asset struct {
	Name string // Filename (e.g. "qsdev_1.2.3_Linux_x86_64.tar.gz")
	URL  string // Browser download URL
}

// Config holds configuration for the self-update system.
type Config struct {
	GitHubOwner   string        // GitHub organization or user
	GitHubRepo    string        // GitHub repository name
	BinaryName    string        // Name of the binary (e.g. "qsdev")
	CheckInterval time.Duration // Minimum interval between update checks
	CacheDir      string        // Directory for caching update check results
}

// testConfigOverride, when non-nil, is used by DefaultConfig instead of
// the real config. This prevents tests from polluting ~/.qsdev/.
var testConfigOverride *Config

// DefaultConfig returns the default self-update configuration.
func DefaultConfig() Config {
	if testConfigOverride != nil {
		return *testConfigOverride
	}
	b := branding.Get()
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}
	return Config{
		GitHubOwner:   b.GitHubOwner,
		GitHubRepo:    b.GitHubRepo,
		BinaryName:    b.AppName,
		CheckInterval: 7 * 24 * time.Hour,
		CacheDir:      filepath.Join(home, "."+b.AppName),
	}
}

// osMapping maps GOOS values to archive naming conventions.
var osMapping = map[string]string{
	"linux":   "Linux",
	"darwin":  "Darwin",
	"windows": "Windows",
}

// archMapping maps GOARCH values to archive naming conventions.
var archMapping = map[string]string{
	"amd64": "x86_64",
	"arm64": "arm64",
}

// ArchiveFilename constructs the expected archive filename for a given
// version, OS, and architecture.
func ArchiveFilename(binaryName, version, targetOS, targetArch string) string {
	osName := osMapping[targetOS]
	if osName == "" {
		osName = targetOS
	}
	archName := archMapping[targetArch]
	if archName == "" {
		archName = targetArch
	}

	ext := ".tar.gz"
	if targetOS == "windows" {
		ext = ".zip"
	}

	return binaryName + "_" + version + "_" + osName + "_" + archName + ext
}
