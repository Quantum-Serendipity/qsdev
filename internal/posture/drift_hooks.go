package posture

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const categoryHookDrift = "Pre-Commit Hook Drift"

// detectHookDrift checks that git hooks are installed and properly configured.
func detectHookDrift(projectDir string, enabledTools map[string]bool) DriftCategory {
	cat := DriftCategory{Name: categoryHookDrift}

	gitDir := filepath.Join(projectDir, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		if os.IsNotExist(err) {
			cat.Findings = append(cat.Findings, DriftFinding{
				Category:    categoryHookDrift,
				Severity:    DriftInfo,
				Subject:     ".git",
				Description: "Project is not a git repository",
			})
		}
		return cat
	}

	// Check pre-commit hook.
	preCommitPath := filepath.Join(gitDir, "hooks", "pre-commit")
	preCommitInfo, err := os.Stat(preCommitPath)
	if err != nil {
		if os.IsNotExist(err) {
			cat.Findings = append(cat.Findings, DriftFinding{
				Category:    categoryHookDrift,
				Severity:    DriftWarning,
				Subject:     "pre-commit",
				Description: "Git pre-commit hook is not installed",
				Remediation: "Run qsdev update to install the pre-commit hook",
			})
		}
	} else if runtime.GOOS != "windows" {
		// Check executable bit on non-Windows systems.
		if preCommitInfo.Mode().Perm()&0o111 == 0 {
			cat.Findings = append(cat.Findings, DriftFinding{
				Category:    categoryHookDrift,
				Severity:    DriftWarning,
				Subject:     "pre-commit",
				Description: "Git pre-commit hook exists but is not executable",
				Remediation: fmt.Sprintf("Run: chmod +x %s", preCommitPath),
			})
		}
	}

	// If commitlint is enabled, check commit-msg hook.
	if enabledTools["commitlint"] {
		commitMsgPath := filepath.Join(gitDir, "hooks", "commit-msg")
		if _, err := os.Stat(commitMsgPath); err != nil {
			if os.IsNotExist(err) {
				cat.Findings = append(cat.Findings, DriftFinding{
					Category:    categoryHookDrift,
					Severity:    DriftWarning,
					Subject:     "commit-msg",
					Description: "Commitlint is enabled but the commit-msg hook is not installed",
					Remediation: "Run qsdev update to install the commit-msg hook",
				})
			}
		}
	}

	return cat
}
