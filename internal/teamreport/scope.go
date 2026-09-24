package teamreport

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/posture"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

// ghWaitDelay bounds how long a cancelled gh invocation may keep its output
// pipes open before it is abandoned.
const ghWaitDelay = 5 * time.Second

// runGH runs the gh CLI with args and returns its combined output. The child
// is killed when ctx is cancelled. It is a variable so tests can stub gh.
var runGH = func(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "gh", args...)
	cmd.WaitDelay = ghWaitDelay
	return cmd.CombinedOutput()
}

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

// CollectFromScope reads the scope file, downloads the latest posture report
// artifact from each repository using `gh run download`, and returns the
// deserialized PostureReports, each tagged with the repository it came from,
// along with any warnings. Cancelling ctx stops the collection.
func CollectFromScope(ctx context.Context, scopePath string) ([]*posture.PostureReport, []string, error) {
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
		if err := ctx.Err(); err != nil {
			return reports, warnings, fmt.Errorf("collecting posture reports: %w", err)
		}

		projectDir := filepath.Join(tmpDir, sanitizeRepoName(proj.Repo))
		if err := os.MkdirAll(projectDir, fileutil.ModeDirDefault); err != nil {
			warnings = append(warnings, fmt.Sprintf("failed to create dir for %s: %v", proj.Repo, err))
			continue
		}

		output, err := runGH(ctx, scopeDownloadArgs(proj.Repo, projectDir)...)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return reports, warnings, fmt.Errorf("collecting posture reports: %w", ctxErr)
			}
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

		// The scope entry names the repository the artifact was downloaded
		// from, which is where issues for this project belong.
		for _, r := range dirReports {
			r.Repository = proj.Repo
		}
		reports = append(reports, dirReports...)
	}

	return reports, warnings, nil
}

// scopeDownloadArgs builds the `gh run download` arguments that fetch the
// latest posture report artifact uploaded by the per-project CI steps.
func scopeDownloadArgs(repo, dir string) []string {
	return []string{
		"run", "download",
		"--repo", repo,
		"--pattern", postureArtifactPattern,
		"--dir", dir,
	}
}

// sanitizeRepoName converts "owner/repo" to "owner-repo" for safe directory names.
func sanitizeRepoName(repo string) string {
	return strings.ReplaceAll(repo, "/", "-")
}
