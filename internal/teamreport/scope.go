package teamreport

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/posture"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// LoadScopeFile reads and validates a scope file from the given path.
// The scope file defines which repositories should be included in the
// team report when using the scope-based collection method.
func LoadScopeFile(path string) (*ScopeFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading scope file: %w", err)
	}

	var scope ScopeFile
	if err := json.Unmarshal(data, &scope); err != nil {
		return nil, fmt.Errorf("parsing scope file: %w", err)
	}

	if len(scope.Projects) == 0 {
		return nil, fmt.Errorf("scope file has no projects defined")
	}

	for i, p := range scope.Projects {
		if p.Repo == "" {
			return nil, fmt.Errorf("project at index %d has empty repo", i)
		}
	}

	return &scope, nil
}

// CollectFromScope reads the scope file, downloads the posture artifact
// from each repository's latest CI run using `gh run download`, and returns
// the deserialized PostureReports along with any warnings.
func CollectFromScope(scopePath string) ([]*posture.PostureReport, []string, error) {
	scope, err := LoadScopeFile(scopePath)
	if err != nil {
		return nil, nil, err
	}

	tmpDir, err := os.MkdirTemp("", branding.Get().AppName+"-team-report-*")
	if err != nil {
		return nil, nil, fmt.Errorf("creating temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	var reports []*posture.PostureReport
	var warnings []string

	for _, proj := range scope.Projects {
		projectDir := filepath.Join(tmpDir, sanitizeRepoName(proj.Repo))
		if err := os.MkdirAll(projectDir, 0o755); err != nil {
			warnings = append(warnings, fmt.Sprintf("failed to create dir for %s: %v", proj.Repo, err))
			continue
		}

		// Download the posture-report artifact from the latest workflow run.
		args := []string{
			"run", "download",
			"--repo", proj.Repo,
			"--name", "posture-report",
			"--dir", projectDir,
		}

		cmd := exec.Command("gh", args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			warnings = append(warnings,
				fmt.Sprintf("failed to download artifact from %s: %v\noutput: %s",
					proj.Repo, err, strings.TrimSpace(string(output))))
			continue
		}

		// Load reports from the downloaded directory.
		dirReports, dirWarnings, err := LoadPostureReports(projectDir)
		warnings = append(warnings, dirWarnings...)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("error loading reports from %s: %v", proj.Repo, err))
			continue
		}

		reports = append(reports, dirReports...)
	}

	return reports, warnings, nil
}

// sanitizeRepoName converts "owner/repo" to "owner-repo" for safe directory names.
func sanitizeRepoName(repo string) string {
	return strings.ReplaceAll(repo, "/", "-")
}
