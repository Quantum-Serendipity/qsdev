package bugreport

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/huh"

	"github.com/Quantum-Serendipity/qsdev/internal/extlog"
	"github.com/Quantum-Serendipity/qsdev/internal/logging"
)

type wizardState struct {
	title          string
	description    string
	steps          string
	severity       string
	category       string
	includeEnv     bool
	logSource      string
	logWindow      string
	logSessions    []string
	includeExtLogs bool
	submitMethod   string
}

// RunWizard walks the user through creating a bug report.
func RunWizard(projectRoot string) error {
	env := CollectEnvironment(projectRoot)
	ws := &wizardState{
		includeEnv: true,
		severity:   "major",
		logSource:  "project",
		logWindow:  "last-command",
	}

	form := buildForm(ws, projectRoot, env)
	if err := form.Run(); err != nil {
		if err == huh.ErrUserAborted {
			fmt.Println("Bug report cancelled.")
			return nil
		}
		return fmt.Errorf("wizard error: %w", err)
	}

	report := BugReport{
		Title:       ws.title,
		Description: ws.description,
		Steps:       ws.steps,
		Severity:    ws.severity,
		Category:    ws.category,
		Environment: env,
		IncludeEnv:  ws.includeEnv,
	}

	logExcerpt, sessionInfo := collectLogs(ws, projectRoot)
	report.LogExcerpt = logExcerpt
	report.SessionInfo = sessionInfo

	if ws.includeExtLogs {
		homeDir, _ := os.UserHomeDir()
		window := extlog.DefaultWindow(60)
		entries, _ := extlog.CollectAll(projectRoot, homeDir, window)
		if len(entries) > 0 {
			report.ExtLogExcerpt = formatExtLogExcerpt(entries)
		}
	}

	body := report.FormatIssueBody()

	fmt.Println("\n--- Bug Report Preview ---")
	fmt.Println(body)
	fmt.Println("--- End Preview ---")

	switch ws.submitMethod {
	case "gh":
		if err := CheckGH(); err != nil {
			fmt.Fprintf(os.Stderr, "Cannot submit via gh: %v\n", err)
			fmt.Println("Falling back to file save.")
			return saveAndPrint(ws.title, body)
		}
		return SubmitViaGH(ws.title, body)
	case "browser":
		u := BrowserURL(ws.title, body)
		fmt.Printf("Open this URL in your browser:\n%s\n", u)
		return nil
	case "file":
		return saveAndPrint(ws.title, body)
	default:
		fmt.Println("Bug report cancelled.")
		return nil
	}
}

func buildForm(ws *wizardState, projectRoot string, env Environment) *huh.Form {
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
			Description("Logs are privacy-scrubbed — no secrets or credentials included.").
			Options(logOptions...).
			Value(&ws.logWindow),
		huh.NewConfirm().
			Title("Include external tool logs?").
			Description("Auto-detected logs from npm, nix, devenv (scrubbed for secrets).").
			Value(&ws.includeExtLogs),
	)

	submitGroup := huh.NewGroup(
		huh.NewSelect[string]().
			Title("How to submit?").
			Options(
				huh.NewOption("Submit via gh CLI (requires gh auth)", "gh"),
				huh.NewOption("Print browser URL (copy/paste)", "browser"),
				huh.NewOption("Save to file", "file"),
				huh.NewOption("Cancel", "cancel"),
			).
			Value(&ws.submitMethod),
	)

	return huh.NewForm(summaryGroup, reproGroup, envGroup, logGroup, submitGroup)
}

func collectLogs(ws *wizardState, projectRoot string) (excerpt, sessionInfo string) {
	if ws.logWindow == "none" {
		return "", ""
	}

	logDir := logging.GlobalLogDir()
	if projectRoot != "" {
		logDir = logging.ProjectLogDir(projectRoot)
		if _, err := os.Stat(logDir); os.IsNotExist(err) {
			logDir = logging.GlobalLogDir()
		}
	}

	entries, err := os.ReadDir(logDir)
	if err != nil {
		return "", ""
	}

	type logFile struct {
		path    string
		modTime time.Time
	}
	var files []logFile
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, logFile{
			path:    filepath.Join(logDir, e.Name()),
			modTime: info.ModTime(),
		})
	}
	if len(files) == 0 {
		return "", ""
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.After(files[j].modTime)
	})

	var selected []logFile
	switch ws.logWindow {
	case "last-command":
		selected = files[:1]
	case "1h":
		cutoff := time.Now().Add(-1 * time.Hour)
		for _, f := range files {
			if f.modTime.After(cutoff) {
				selected = append(selected, f)
			}
		}
	case "24h":
		cutoff := time.Now().Add(-24 * time.Hour)
		for _, f := range files {
			if f.modTime.After(cutoff) {
				selected = append(selected, f)
			}
		}
	}

	if len(selected) == 0 {
		return "", ""
	}

	redactor := logging.NewRedactor()
	var buf bytes.Buffer
	totalLines := 0
	maxLines := 200

	for _, f := range selected {
		data, err := os.ReadFile(f.path)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(bytes.NewReader(data))
		for scanner.Scan() {
			if totalLines >= maxLines {
				fmt.Fprintf(&buf, "... (truncated, %d files total)\n", len(selected))
				goto done
			}
			line := redactor.RedactString(scanner.Text())
			buf.WriteString(line)
			buf.WriteByte('\n')
			totalLines++
		}
	}
done:

	info := fmt.Sprintf("%d session(s), %d lines", len(selected), totalLines)
	return buf.String(), info
}

func formatExtLogExcerpt(allEntries map[string][]extlog.LogEntry) string {
	var buf bytes.Buffer
	for provider, entries := range allEntries {
		truncated := extlog.Truncate(entries, 50)
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

func saveAndPrint(title, body string) error {
	path, err := SaveToFile(title, body)
	if err != nil {
		return err
	}
	fmt.Printf("Bug report saved to: %s\n", path)
	return nil
}
