package check

import (
	"fmt"
	"io"
	"os"
)

// IsGitHubActions returns true if running inside GitHub Actions.
func IsGitHubActions() bool {
	return os.Getenv("GITHUB_ACTIONS") == "true"
}

// EmitGitHubAnnotations writes GitHub Actions annotation commands for
// failed and warned checks.
func EmitGitHubAnnotations(results []CheckResult, w io.Writer) {
	for _, r := range results {
		switch r.Status {
		case StatusFail:
			if r.FilePath != "" {
				fmt.Fprintf(w, "::error file=%s::%s: %s\n", r.FilePath, r.Name, r.Message)
			} else {
				fmt.Fprintf(w, "::error::%s: %s\n", r.Name, r.Message)
			}
		case StatusWarn:
			if r.FilePath != "" {
				fmt.Fprintf(w, "::warning file=%s::%s: %s\n", r.FilePath, r.Name, r.Message)
			} else {
				fmt.Fprintf(w, "::warning::%s: %s\n", r.Name, r.Message)
			}
		}
	}
}
