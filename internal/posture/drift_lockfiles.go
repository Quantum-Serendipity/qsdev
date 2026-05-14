package posture

import (
	"fmt"
	"os"
	"path/filepath"
)

const categoryLockfileDrift = "Lock File Drift"

// lockfilePairs maps manifest files to their corresponding lock files.
var lockfilePairs = []struct{ Manifest, Lockfile string }{
	{"package.json", "package-lock.json"},
	{"package.json", "pnpm-lock.yaml"},
	{"package.json", "yarn.lock"},
	{"package.json", "bun.lockb"},
	{"pyproject.toml", "uv.lock"},
	{"pyproject.toml", "poetry.lock"},
	{"go.mod", "go.sum"},
	{"Cargo.toml", "Cargo.lock"},
}

// detectLockfileDrift checks whether lockfiles are up-to-date relative to
// their manifest files by comparing modification times.
func detectLockfileDrift(projectDir string) DriftCategory {
	cat := DriftCategory{Name: categoryLockfileDrift}

	for _, pair := range lockfilePairs {
		manifestPath := filepath.Join(projectDir, pair.Manifest)
		lockfilePath := filepath.Join(projectDir, pair.Lockfile)

		manifestInfo, err := os.Stat(manifestPath)
		if err != nil {
			// Manifest doesn't exist; skip this pair.
			continue
		}

		lockfileInfo, err := os.Stat(lockfilePath)
		if err != nil {
			if os.IsNotExist(err) {
				cat.Findings = append(cat.Findings, DriftFinding{
					Category:    categoryLockfileDrift,
					Severity:    DriftError,
					Subject:     pair.Lockfile,
					Description: fmt.Sprintf("Manifest %q exists but lockfile %q is missing", pair.Manifest, pair.Lockfile),
					Expected:    pair.Lockfile,
					Remediation: fmt.Sprintf("Run the package manager to generate %s", pair.Lockfile),
				})
			}
			continue
		}

		if lockfileInfo.ModTime().Before(manifestInfo.ModTime()) {
			cat.Findings = append(cat.Findings, DriftFinding{
				Category:    categoryLockfileDrift,
				Severity:    DriftWarning,
				Subject:     pair.Lockfile,
				Description: fmt.Sprintf("Lockfile %q is older than manifest %q", pair.Lockfile, pair.Manifest),
				Expected:    fmt.Sprintf("%s modified after %s", pair.Lockfile, pair.Manifest),
				Actual:      fmt.Sprintf("%s last modified at %s, %s at %s", pair.Lockfile, lockfileInfo.ModTime().Format("2006-01-02 15:04:05"), pair.Manifest, manifestInfo.ModTime().Format("2006-01-02 15:04:05")),
				Remediation: fmt.Sprintf("Run the package manager to update %s", pair.Lockfile),
			})
		}
	}

	return cat
}
