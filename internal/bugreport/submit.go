package bugreport

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

// browserMaxURLLen caps the length of the pre-filled new-issue URL. GitHub
// rejects (414) or truncates URLs much beyond ~8 KB, and the limit applies to
// the percent-encoded form, so the body is fitted against the encoded length.
const browserMaxURLLen = 8000

// browserTruncationNote is appended to a body cut to fit the browser URL. The
// wizard saves the full report locally whenever this note is used.
const browserTruncationNote = "\n\n... (truncated — full report saved locally; please attach it to this issue)"

// bugLabel is the label applied to issues filed via gh.
const bugLabel = "bug"

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
		return fmt.Errorf("gh CLI not found (install with: %s devenv add-package gh): %w",
			branding.Get().AppName, err)
	}
	cmd := exec.Command("gh", "auth", "status")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh not authenticated (run: gh auth login): %w", err)
	}
	return nil
}

// SubmitViaGH creates a GitHub issue using the gh CLI. The body is streamed on
// stdin (--body-file -) rather than passed as an argument, so large reports do
// not hit the per-argument size limit. When the target repository has no "bug"
// label, the issue is created without it rather than failing.
func SubmitViaGH(title, body string) error {
	stderr, err := ghIssueCreate(title, body, bugLabel)
	if err != nil && strings.Contains(strings.ToLower(stderr), "label") {
		_, err = ghIssueCreate(title, body, "")
	}
	if err != nil {
		return fmt.Errorf("gh issue create: %w", err)
	}
	return nil
}

// ghIssueCreate runs one `gh issue create` and returns its captured stderr
// (which is also shown to the user) for failure classification.
func ghIssueCreate(title, body, label string) (string, error) {
	b := branding.Get()
	args := []string{"issue", "create",
		"--repo", b.GitHubOwner + "/" + b.GitHubRepo,
		"--title", title,
		"--body-file", "-",
	}
	if label != "" {
		args = append(args, "--label", label)
	}
	var errBuf bytes.Buffer
	cmd := exec.Command("gh", args...)
	cmd.Stdin = strings.NewReader(body)
	cmd.Stdout = os.Stdout
	cmd.Stderr = io.MultiWriter(os.Stderr, &errBuf)
	err := cmd.Run()
	return errBuf.String(), err
}

// BrowserURL returns a pre-filled GitHub new issue URL whose total length fits
// browserMaxURLLen, and whether the body had to be truncated to fit. The body
// is cut on a UTF-8 boundary and marked with browserTruncationNote.
func BrowserURL(title, body string) (string, bool) {
	b := branding.Get()
	base := fmt.Sprintf("https://github.com/%s/%s/issues/new", b.GitHubOwner, b.GitHubRepo)
	build := func(body string) string {
		params := url.Values{
			"title":  {title},
			"labels": {bugLabel},
			"body":   {body},
		}
		return base + "?" + params.Encode()
	}

	if u := build(body); len(u) <= browserMaxURLLen {
		return u, false
	}

	// Binary-search the longest prefix whose encoded URL (with the note) fits.
	lo, hi := 0, len(body)
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if len(build(runePrefix(body, mid)+browserTruncationNote)) <= browserMaxURLLen {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return build(runePrefix(body, lo) + browserTruncationNote), true
}

// runePrefix returns the longest prefix of s that is at most n bytes and does
// not split a UTF-8 sequence.
func runePrefix(s string, n int) string {
	if n >= len(s) {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n]
}

// SaveToFile writes the report to the app's home directory and returns the path.
func SaveToFile(title, body string) (string, error) {
	// No shared-temp fallback: a predictable path there could be pre-created
	// or symlinked by another local user.
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locating home directory for the bug report: %w", err)
	}
	dir := filepath.Join(home, "."+branding.Get().AppName)
	if err := os.MkdirAll(dir, fileutil.ModeDirDefault); err != nil {
		return "", fmt.Errorf("creating report directory %s: %w", dir, err)
	}

	filename := fmt.Sprintf("bug-report-%s.md", time.Now().Format("2006-01-02T15-04-05"))
	path := filepath.Join(dir, filename)

	// Owner-only: the report embeds log excerpts and environment details.
	content := fmt.Sprintf("# %s\n\n%s", title, body)
	if err := os.WriteFile(path, []byte(content), fileutil.ModePrivate); err != nil {
		return "", fmt.Errorf("writing bug report %s: %w", path, err)
	}
	return path, nil
}
