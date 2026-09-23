package drift

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

const categoryLockfileDrift = "Lock File Drift"

// detectLockfileDrift checks whether lockfiles are up-to-date relative to
// their manifest files by comparing modification times. A manifest is reported
// as unlocked only when none of its alternative lockfiles exists, and staleness
// is judged only against the lockfiles that are actually present.
func detectLockfileDrift(projectDir string) Category {
	cat := Category{Name: categoryLockfileDrift}

	for _, group := range ecosystem.GroupedManifestLockfiles() {
		manifestInfo, err := os.Stat(filepath.Join(projectDir, group.Manifest))
		if err != nil {
			// Manifest doesn't exist; nothing to lock.
			continue
		}

		present := 0
		for _, lockfile := range group.Lockfiles {
			lockfileInfo, err := os.Stat(filepath.Join(projectDir, lockfile))
			if err != nil {
				continue
			}
			present++
			if lockfileInfo.ModTime().Before(manifestInfo.ModTime()) {
				cat.Findings = append(cat.Findings, staleLockfileFinding(group.Manifest, lockfile, manifestInfo, lockfileInfo))
			}
		}

		if present == 0 {
			cat.Findings = append(cat.Findings, missingLockfileFinding(group))
		}
	}

	return cat
}

// missingLockfileFinding reports a manifest none of whose lockfiles exists. With
// a single possible lockfile the finding names it directly; with alternatives
// it names the manifest, since no one lockfile is the "expected" one.
func missingLockfileFinding(group ecosystem.ManifestLockfiles) Finding {
	if len(group.Lockfiles) == 1 {
		lockfile := group.Lockfiles[0]
		return Finding{
			Category:    categoryLockfileDrift,
			Severity:    Error,
			Subject:     lockfile,
			Description: fmt.Sprintf("Manifest %q exists but lockfile %q is missing", group.Manifest, lockfile),
			Expected:    lockfile,
			Remediation: fmt.Sprintf("Run the package manager to generate %s", lockfile),
		}
	}
	alternatives := strings.Join(group.Lockfiles, ", ")
	return Finding{
		Category:    categoryLockfileDrift,
		Severity:    Error,
		Subject:     group.Manifest,
		Description: fmt.Sprintf("Manifest %q exists but no lockfile is present (expected one of: %s)", group.Manifest, alternatives),
		Expected:    "one of: " + alternatives,
		Remediation: fmt.Sprintf("Run the package manager to generate a lockfile for %s", group.Manifest),
	}
}

// staleLockfileFinding reports a lockfile last written before its manifest.
func staleLockfileFinding(manifest, lockfile string, manifestInfo, lockfileInfo os.FileInfo) Finding {
	return Finding{
		Category:    categoryLockfileDrift,
		Severity:    Warning,
		Subject:     lockfile,
		Description: fmt.Sprintf("Lockfile %q is older than manifest %q", lockfile, manifest),
		Expected:    fmt.Sprintf("%s modified after %s", lockfile, manifest),
		Actual: fmt.Sprintf("%s last modified at %s, %s at %s",
			lockfile, lockfileInfo.ModTime().Format("2006-01-02 15:04:05"),
			manifest, manifestInfo.ModTime().Format("2006-01-02 15:04:05")),
		Remediation: fmt.Sprintf("Run the package manager to update %s", lockfile),
	}
}
