package bugreport

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/extlog"
	"github.com/Quantum-Serendipity/qsdev/internal/logcmd"
	"github.com/Quantum-Serendipity/qsdev/internal/logging"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

const (
	// issueBodyMaxBytes keeps the composed issue body under GitHub's
	// 65,536-character limit with headroom for markdown framing.
	issueBodyMaxBytes = 60000
	// logExcerptMaxLines and logExcerptMaxBytes bound the qsdev log excerpt.
	logExcerptMaxLines = 200
	logExcerptMaxBytes = 32 * 1024
	// extLogMaxEntries and extLogProviderMaxBytes bound each external log
	// provider's excerpt.
	extLogMaxEntries       = 50
	extLogProviderMaxBytes = 8 * 1024
)

// Submit method values used by the submit form.
const (
	methodGH      = "gh"
	methodBrowser = "browser"
	methodFile    = "file"
	methodCancel  = "cancel"
)

type wizardState struct {
	title          string
	description    string
	steps          string
	severity       string
	category       string
	includeEnv     bool
	logWindow      string
	includeExtLogs bool
	submitMethod   string
	confirmPublish bool
}

// RunWizard walks the user through creating a bug report. The report content
// is collected first, then the full issue body (including any log excerpts) is
// shown, and only then is the user asked how to submit it, with an explicit
// confirmation before anything is posted publicly.
func RunWizard(projectRoot string) error {
	env := CollectEnvironment(projectRoot)
	ws := &wizardState{
		includeEnv: true,
		severity:   "major",
		logWindow:  "last-command",
	}

	if err := buildContentForm(ws, projectRoot, env).Run(); err != nil {
		return formError(err)
	}

	report := buildReport(ws, projectRoot, env)
	body := fitIssueBody(&report, issueBodyMaxBytes)

	fmt.Println("\n--- Bug Report Preview ---")
	fmt.Println(body)
	fmt.Println("--- End Preview ---")

	if isatty.IsTerminal(os.Stdin.Fd()) || isatty.IsCygwinTerminal(os.Stdin.Fd()) {
		if err := buildSubmitForm(ws).Run(); err != nil {
			return formError(err)
		}
	} else {
		// Without a terminal there is no way to confirm a public post.
		ws.submitMethod = methodFile
	}

	d := defaultDelivery()
	d.browserBody = func() string { return browserIssueBody(report) }
	return d.deliver(ws.submitMethod, ws.confirmPublish, ws.title, body)
}

// formError maps a huh form error to the wizard's result: an abort cancels
// the report quietly, anything else is a failure.
func formError(err error) error {
	if errors.Is(err, huh.ErrUserAborted) {
		fmt.Println("Bug report cancelled.")
		return nil
	}
	return fmt.Errorf("wizard error: %w", err)
}

// buildReport assembles the report from the wizard answers and the collected
// log excerpts.
func buildReport(ws *wizardState, projectRoot string, env Environment) BugReport {
	report := BugReport{
		Title:       ws.title,
		Description: ws.description,
		Steps:       ws.steps,
		Severity:    ws.severity,
		Category:    ws.category,
		Environment: env,
		IncludeEnv:  ws.includeEnv,
	}

	report.LogExcerpt, report.SessionInfo = collectLogs(ws.logWindow, logDirs(projectRoot), time.Now())

	if ws.includeExtLogs {
		window := extlog.DefaultWindow(60)
		entries, summaries := extlog.CollectAll(projectRoot, userHomeDir(), window)
		for _, s := range summaries {
			for _, e := range s.CollectionErrors {
				fmt.Fprintf(os.Stderr, "warning: %s logs incomplete: %s\n", s.Provider, e)
			}
		}
		if len(entries) > 0 {
			report.ExtLogExcerpt = formatExtLogExcerpt(entries)
		}
	}
	return report
}

// fitIssueBody renders the report, shrinking its log excerpts until the body
// fits maxBytes: external logs are dropped first, then the qsdev log excerpt
// is cut to fit. The report is updated in place to match the returned body.
func fitIssueBody(r *BugReport, maxBytes int) string {
	body := r.FormatIssueBody()
	if len(body) <= maxBytes {
		return body
	}
	if r.ExtLogExcerpt != "" {
		r.ExtLogExcerpt = ""
		body = r.FormatIssueBody()
		if len(body) <= maxBytes {
			return body
		}
	}
	if r.LogExcerpt != "" {
		r.SessionInfo += ", truncated to fit"
		over := len(r.FormatIssueBody()) - maxBytes
		r.LogExcerpt = truncateLines(r.LogExcerpt, max(len(r.LogExcerpt)-over, 0))
		body = r.FormatIssueBody()
	}
	return body
}

// truncateLines returns the longest prefix of s made of whole lines that is at
// most n bytes.
func truncateLines(s string, n int) string {
	if len(s) <= n {
		return s
	}
	cut := strings.LastIndexByte(s[:n], '\n')
	if cut < 0 {
		return ""
	}
	return s[:cut+1]
}

// browserIssueBody renders the report without log excerpts, for the browser
// URL: a URL cannot carry them, so the user is told to attach the saved file.
func browserIssueBody(r BugReport) string {
	if r.LogExcerpt == "" && r.ExtLogExcerpt == "" {
		return r.FormatIssueBody()
	}
	r.LogExcerpt = ""
	r.ExtLogExcerpt = ""
	return r.FormatIssueBody() + "\n_Log excerpts omitted from this URL; the full report was saved locally — please attach it._\n"
}

// delivery performs the chosen submit action. Its dependencies are fields so
// tests can exercise every path without gh or a browser.
type delivery struct {
	out, errOut io.Writer
	checkGH     func() error
	submitGH    func(title, body string) error
	save        func(title, body string) (string, error)
	// browserBody returns the body to encode in the browser URL; nil means
	// the full body.
	browserBody func() string
}

func defaultDelivery() delivery {
	return delivery{
		out:      os.Stdout,
		errOut:   os.Stderr,
		checkGH:  CheckGH,
		submitGH: SubmitViaGH,
		save:     SaveToFile,
	}
}

// deliver submits, prints, or saves the report. Public submission (gh or
// browser) happens only when confirmed; a declined or failed submission saves
// the report locally so the user's work is never lost.
func (d delivery) deliver(method string, confirmed bool, title, body string) error {
	switch method {
	case methodGH, methodBrowser:
		if !confirmed {
			fmt.Fprintln(d.out, "Not submitted.")
			return d.saveAndPrint(title, body)
		}
	case methodFile:
		return d.saveAndPrint(title, body)
	default:
		fmt.Fprintln(d.out, "Bug report cancelled.")
		return nil
	}

	if method == methodBrowser {
		return d.deliverBrowser(title, body)
	}

	if err := d.checkGH(); err != nil {
		fmt.Fprintf(d.errOut, "Cannot submit via gh: %v\n", err)
		fmt.Fprintln(d.out, "Falling back to file save.")
		return d.saveAndPrint(title, body)
	}
	if err := d.submitGH(title, body); err != nil {
		fmt.Fprintf(d.errOut, "Submitting via gh failed: %v\n", err)
		if saveErr := d.saveAndPrint(title, body); saveErr != nil {
			return errors.Join(fmt.Errorf("submitting via gh: %w", err), saveErr)
		}
		return fmt.Errorf("submitting via gh: %w", err)
	}
	return nil
}

// deliverBrowser prints a pre-filled issue URL. Whenever the URL cannot carry
// the whole report, the full report is saved locally first so the promise in
// the URL's note holds.
func (d delivery) deliverBrowser(title, body string) error {
	urlBody := body
	if d.browserBody != nil {
		urlBody = d.browserBody()
	}
	u, truncated := BrowserURL(title, urlBody)
	if truncated || urlBody != body {
		if err := d.saveAndPrint(title, body); err != nil {
			return err
		}
	}
	fmt.Fprintf(d.out, "Open this URL in your browser:\n%s\n", u)
	return nil
}

func (d delivery) saveAndPrint(title, body string) error {
	path, err := d.save(title, body)
	if err != nil {
		return fmt.Errorf("saving bug report: %w", err)
	}
	fmt.Fprintf(d.out, "Bug report saved to: %s\n", path)
	return nil
}

func buildContentForm(ws *wizardState, projectRoot string, env Environment) *huh.Form {
	summaryGroup := huh.NewGroup(
		huh.NewInput().
			Title("Bug title").
			Description("Brief summary of the issue").
			Value(&ws.title).
			Validate(func(s string) error {
				if len(s) < 10 {
					return fmt.Errorf("title must be at least 10 characters")
				}
				return nil
			}),
		huh.NewText().
			Title("Description").
			Description("What happened? What did you expect?").
			Value(&ws.description).
			Validate(func(s string) error {
				if len(s) < 10 {
					return fmt.Errorf("description must be at least 10 characters")
				}
				return nil
			}),
	)

	reproGroup := huh.NewGroup(
		huh.NewText().
			Title("Steps to reproduce").
			Description("Optional — numbered steps or a brief description").
			Value(&ws.steps),
		huh.NewSelect[string]().
			Title("Severity").
			Options(
				huh.NewOption("Cosmetic — visual/formatting issue", "cosmetic"),
				huh.NewOption("Minor — workaround exists", "minor"),
				huh.NewOption("Major — blocks intended workflow", "major"),
				huh.NewOption("Critical — data loss or security issue", "critical"),
			).
			Value(&ws.severity),
		huh.NewSelect[string]().
			Title("Category").
			Options(
				huh.NewOption("Init wizard", "init-wizard"),
				huh.NewOption("devenv generation", "devenv-generation"),
				huh.NewOption("Claude Code config", "claude-code"),
				huh.NewOption("Security hardening", "security-hardening"),
				huh.NewOption("Self-update", "self-update"),
				huh.NewOption("CLI behavior", "cli-behavior"),
				huh.NewOption("Other", "other"),
			).
			Value(&ws.category),
	)

	envGroup := huh.NewGroup(
		huh.NewNote().
			Title("Auto-collected environment").
			Description(env.FormatTable()),
		huh.NewConfirm().
			Title("Include environment info in report?").
			Value(&ws.includeEnv),
	)

	logOptions := []huh.Option[string]{
		huh.NewOption("No logs", "none"),
		huh.NewOption("Last command only", "last-command"),
		huh.NewOption("Last hour", "1h"),
		huh.NewOption("Last 24 hours", "24h"),
	}

	logGroup := huh.NewGroup(
		huh.NewSelect[string]().
			Title("Attach qsdev log excerpt?").
			Description("Logs are privacy-scrubbed and shown in full for review before anything is sent.").
			Options(logOptions...).
			Value(&ws.logWindow),
		huh.NewConfirm().
			Title("Include external tool logs?").
			Description(extLogDescription(extlog.DefaultRegistry(), projectRoot, userHomeDir())).
			Value(&ws.includeExtLogs),
	)

	return huh.NewForm(summaryGroup, reproGroup, envGroup, logGroup)
}

// buildSubmitForm asks how to deliver the previewed report and, for public
// destinations, requires an explicit confirmation (default: no).
func buildSubmitForm(ws *wizardState) *huh.Form {
	b := branding.Get()
	methodGroup := huh.NewGroup(
		huh.NewSelect[string]().
			Title("How to submit the report shown above?").
			Options(
				huh.NewOption("Submit via gh CLI (requires gh auth)", methodGH),
				huh.NewOption("Print browser URL (copy/paste)", methodBrowser),
				huh.NewOption("Save to file", methodFile),
				huh.NewOption("Cancel", methodCancel),
			).
			Value(&ws.submitMethod),
	)
	confirmGroup := huh.NewGroup(
		huh.NewConfirm().
			Title(fmt.Sprintf("Post this report publicly to github.com/%s/%s?", b.GitHubOwner, b.GitHubRepo)).
			Description("The issue will contain the preview shown above (a browser URL leaves out the log excerpts).").
			Affirmative("Yes, post it").
			Negative("No, save it locally").
			Value(&ws.confirmPublish),
	).WithHideFunc(func() bool {
		return ws.submitMethod != methodGH && ws.submitMethod != methodBrowser
	})
	return huh.NewForm(methodGroup, confirmGroup)
}

// extLogDescription names the external log sources that currently have logs to
// attach, derived from the registered providers' own detection rather than a
// fixed list, so the prompt never advertises a source with nothing to offer.
func extLogDescription(reg *extlog.Registry, projectRoot, homeDir string) string {
	var names []string
	for _, p := range reg.DetectAll(projectRoot, homeDir) {
		names = append(names, p.DisplayName())
	}
	if len(names) == 0 {
		return "No external tool logs detected."
	}
	sort.Strings(names)
	return "Auto-detected logs from " + strings.Join(names, ", ") + " (scrubbed for secrets)."
}

// userHomeDir returns the user's home directory, or "" when it is unknown
// (providers that read from it then detect nothing).
func userHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

// logDirs returns the log directories to draw excerpts from: the project tier
// (when inside a project) and the global tier. Both are searched because
// commands log to one tier or the other depending on the command.
func logDirs(projectRoot string) []string {
	dirs := []string{}
	if projectRoot != "" {
		dirs = append(dirs, logging.ProjectLogDir(projectRoot))
	}
	global := logging.GlobalLogDir()
	for _, d := range dirs {
		if filepath.Clean(d) == filepath.Clean(global) {
			return dirs
		}
	}
	return append(dirs, global)
}

// excludedSessionCommands are subcommands whose sessions never explain a bug:
// the bug report itself, log browsing, and shell completion requests.
func excludedSessionCommands() map[string]bool {
	return map[string]bool{
		commandName:                     true,
		logcmd.CommandName:              true,
		cobra.ShellCompRequestCmd:       true,
		cobra.ShellCompNoDescRequestCmd: true,
	}
}

// isExcludedSession reports whether a session's recorded command path (e.g.
// "qsdev report bug") belongs to an excluded subcommand.
func isExcludedSession(command string, excluded map[string]bool) bool {
	fields := strings.Fields(command)
	return len(fields) >= 2 && excluded[fields[1]]
}

type sessionLogFile struct {
	path    string
	modTime time.Time
}

// collectLogs selects session logs from dirs for the chosen window and returns
// a scrubbed excerpt plus a short description. Sessions of the report, logs
// and completion commands (including the running `report bug`) are skipped.
func collectLogs(window string, dirs []string, now time.Time) (excerpt, sessionInfo string) {
	if window == "none" {
		return "", ""
	}

	selected := selectSessionLogs(window, listSessionLogs(dirs), now)
	if len(selected) == 0 {
		return "", ""
	}

	var buf bytes.Buffer
	totalLines, used := 0, 0
	truncated := false
	for _, f := range selected {
		lim := logcmd.ExcerptLimits{
			MaxLines: logExcerptMaxLines - totalLines,
			MaxBytes: logExcerptMaxBytes - buf.Len(),
		}
		if lim.MaxLines <= 0 || lim.MaxBytes <= 0 {
			truncated = true
			break
		}
		res, err := writeSessionExcerpt(&buf, f.path, lim)
		totalLines += res.Lines
		if err != nil {
			// Note the failure instead of dropping the session silently. The
			// path-free cause keeps the (public) report free of home paths,
			// and the note is only added while it fits the byte budget.
			note := fmt.Sprintf("... (reading %s failed: %v)\n", filepath.Base(f.path), pathFreeError(err))
			if buf.Len()+len(note) <= logExcerptMaxBytes {
				buf.WriteString(note)
			}
			continue
		}
		used++
		if res.Truncated {
			truncated = true
			break
		}
	}
	if truncated {
		fmt.Fprintf(&buf, "... (truncated, %d of %d session(s) shown)\n", used, len(selected))
	}

	info := fmt.Sprintf("%d session(s), %d lines", used, totalLines)
	return buf.String(), info
}

// pathFreeError returns the underlying cause of a filesystem error without the
// file path it carries.
func pathFreeError(err error) error {
	var pe *fs.PathError
	if errors.As(err, &pe) {
		return pe.Err
	}
	return err
}

func writeSessionExcerpt(w io.Writer, path string, lim logcmd.ExcerptLimits) (logcmd.ExcerptResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return logcmd.ExcerptResult{}, fmt.Errorf("opening session log: %w", err)
	}
	defer f.Close()
	return logcmd.WriteExcerpt(w, f, lim)
}

// listSessionLogs returns the session logs in dirs, newest first, excluding
// sessions of meta commands.
func listSessionLogs(dirs []string) []sessionLogFile {
	excluded := excludedSessionCommands()
	var files []sessionLogFile
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			path := filepath.Join(dir, e.Name())
			if cmd, err := logcmd.SessionCommand(path); err == nil && isExcludedSession(cmd, excluded) {
				continue
			}
			files = append(files, sessionLogFile{path: path, modTime: info.ModTime()})
		}
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.After(files[j].modTime)
	})
	return files
}

// selectSessionLogs applies the window choice to files (newest first).
func selectSessionLogs(window string, files []sessionLogFile, now time.Time) []sessionLogFile {
	var since time.Duration
	switch window {
	case "last-command":
		if len(files) == 0 {
			return nil
		}
		return files[:1]
	case "1h":
		since = time.Hour
	case "24h":
		since = 24 * time.Hour
	default:
		return nil
	}
	cutoff := now.Add(-since)
	var selected []sessionLogFile
	for _, f := range files {
		if f.modTime.After(cutoff) {
			selected = append(selected, f)
		}
	}
	return selected
}

// formatExtLogExcerpt renders external log entries per provider, in provider
// name order, each bounded by entry count and byte budget.
func formatExtLogExcerpt(allEntries map[string][]extlog.LogEntry) string {
	providers := make([]string, 0, len(allEntries))
	for provider := range allEntries {
		providers = append(providers, provider)
	}
	sort.Strings(providers)

	var buf bytes.Buffer
	for _, provider := range providers {
		entries := allEntries[provider]
		truncated := extlog.TruncateWithBudget(entries, extLogMaxEntries, extLogProviderMaxBytes)
		fmt.Fprintf(&buf, "--- %s (%d entries) ---\n", provider, len(entries))
		for _, e := range truncated {
			ts := ""
			if !e.Timestamp.IsZero() {
				ts = e.Timestamp.Format("15:04:05") + " "
			}
			fmt.Fprintf(&buf, "%s[%s] %s\n", ts, e.Level, e.Message)
		}
		if len(entries) > len(truncated) {
			fmt.Fprintf(&buf, "... (%d more entries truncated)\n", len(entries)-len(truncated))
		}
		buf.WriteByte('\n')
	}
	return buf.String()
}
