package bugreport

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const (
	repoOwner     = "Quantum-Serendipity"
	repoName      = "qsdev"
	browserMaxLen = 8000
)

// SubmitMethod represents how the report will be delivered.
type SubmitMethod int

const (
	SubmitGH SubmitMethod = iota
	SubmitBrowser
	SubmitFile
	SubmitCancel
)

// CheckGH verifies gh CLI is installed and authenticated.
func CheckGH() error {
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("gh CLI not found — install with: nix-env -iA nixpkgs.gh")
	}
	cmd := exec.Command("gh", "auth", "status")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh not authenticated — run: gh auth login")
	}
	return nil
}

// SubmitViaGH creates a GitHub issue using the gh CLI.
func SubmitViaGH(title, body string) error {
	cmd := exec.Command("gh", "issue", "create",
		"--repo", repoOwner+"/"+repoName,
		"--title", title,
		"--body", body,
		"--label", "bug",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// BrowserURL returns a pre-filled GitHub new issue URL.
func BrowserURL(title, body string) string {
	u := fmt.Sprintf("https://github.com/%s/%s/issues/new", repoOwner, repoName)
	params := url.Values{
		"title":  {title},
		"labels": {"bug"},
	}
	if len(body) <= browserMaxLen {
		params.Set("body", body)
	} else {
		params.Set("body", body[:browserMaxLen]+"\n\n... (truncated — full report saved locally)")
	}
	return u + "?" + params.Encode()
}

// SaveToFile writes the report to ~/.qsdev/ and returns the path.
func SaveToFile(title, body string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}
	dir := filepath.Join(home, ".qsdev")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	filename := fmt.Sprintf("bug-report-%s.md", time.Now().Format("2006-01-02T15-04-05"))
	path := filepath.Join(dir, filename)

	content := fmt.Sprintf("# %s\n\n%s", title, body)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", err
	}
	return path, nil
}
