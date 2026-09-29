package logcmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/logging"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// CommandName is the name of the "logs" command.
const CommandName = "logs"

// Command returns the "logs" cobra command tree.
func Command() *cobra.Command {
	app := branding.Get().AppName
	cmd := &cobra.Command{
		Use:   CommandName,
		Short: "Browse and manage " + app + " log files",
		Long: fmt.Sprintf(`Browse and manage structured log files from %s operations.

Inside a %s project, shows project-scoped logs (.%s/logs/) by default.
Use --global to view global logs (~/.%s/logs/).
Outside a project, global logs are shown.`, app, app, app, app),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd)
		},
	}

	var global bool
	cmd.PersistentFlags().BoolVar(&global, "global", false, fmt.Sprintf("Use global logs (~/.%s/logs/) instead of project logs", app))

	list := &cobra.Command{
		Use:   "list",
		Short: "List recent log sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd)
		},
	}

	var since string
	var jsonOut bool
	var listAll bool
	list.Flags().StringVar(&since, "since", "", "Show sessions since duration (e.g. 1h, 24h)")
	list.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	list.Flags().BoolVar(&listAll, "all", false, fmt.Sprintf("Show all sessions instead of the latest %d", listDefaultLimit))

	show := &cobra.Command{
		Use:   "show <session-id>",
		Short: "Display a session's log entries",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runShow(cmd, args[0])
		},
	}

	var level string
	var raw bool
	show.Flags().StringVar(&level, "level", "", "Filter by level (debug, info, warn, error)")
	show.Flags().BoolVar(&raw, "raw", false, "Output raw JSONL without formatting")

	path := &cobra.Command{
		Use:   "path",
		Short: "Print the active log directory path",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPath(cmd)
		},
	}

	clean := &cobra.Command{
		Use:   "clean",
		Short: "Delete old log files",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runClean(cmd)
		},
	}

	var olderThan string
	var all bool
	var force bool
	clean.Flags().StringVar(&olderThan, "older-than", "30d", "Delete logs older than duration (e.g. 7d, 24h)")
	clean.Flags().BoolVar(&all, "all", false, "Delete all logs")
	clean.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")

	cmd.AddCommand(list, show, path, clean)
	return cmd
}

// listDefaultLimit caps how many sessions `logs list` prints when neither
// --since nor --all is given.
const listDefaultLimit = 20

// showOmittedKeys are record keys rendered by `logs show` in its fixed prefix
// (or repeated on every record), so they are not repeated as attributes.
var showOmittedKeys = map[string]bool{"time": true, "level": true, "msg": true, "session": true}

type sessionInfo struct {
	ID       string    `json:"id"`
	Command  string    `json:"command"`
	Started  time.Time `json:"started"`
	Duration int64     `json:"duration_ms"`
	Size     int64     `json:"size_bytes"`
	File     string    `json:"file"`
}

func resolveLogDir(cmd *cobra.Command) string {
	global, _ := cmd.Flags().GetBool("global")
	if global {
		return logging.GlobalLogDir()
	}
	projectRoot := logging.DetectProjectRoot()
	if projectRoot != "" {
		return logging.ProjectLogDir(projectRoot)
	}
	return logging.GlobalLogDir()
}

func discoverSessions(dir string) ([]sessionInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var sessions []sessionInfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}

		path := filepath.Join(dir, e.Name())
		si := sessionInfo{
			Size: info.Size(),
			File: path,
		}

		if lines, last, err := readHeadAndTail(path, 3); err == nil {
			for _, line := range lines {
				if si.ID == "" {
					si.ID = recordSessionID(line)
				}
				if si.Command == "" {
					si.Command = jsonField(line, "command")
				}
				if si.Started.IsZero() {
					if ts := jsonField(line, "time"); ts != "" {
						si.Started, _ = time.Parse(time.RFC3339Nano, ts)
					}
				}
			}
			if ms := jsonField(last, "duration_ms"); ms != "" {
				_, _ = fmt.Sscanf(ms, "%d", &si.Duration)
			}
		}

		if si.ID == "" {
			name := strings.TrimSuffix(e.Name(), ".jsonl")
			parts := strings.Split(name, "-")
			if len(parts) >= 2 {
				si.ID = parts[len(parts)-1]
			}
		}

		sessions = append(sessions, si)
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Started.After(sessions[j].Started)
	})

	return sessions, nil
}

func runList(cmd *cobra.Command) error {
	dir := resolveLogDir(cmd)
	sessions, err := discoverSessions(dir)
	if err != nil {
		return fmt.Errorf("reading log directory: %w", err)
	}

	sinceStr, _ := cmd.Flags().GetString("since")
	sessions, err = filterSessionsSince(sessions, sinceStr, time.Now())
	if err != nil {
		return err
	}

	// Defense-in-depth: re-scrub the command field before ANY output format in
	// case a log captured a secret in its argv (F-CAP-29.5-1).
	red := logging.NewRedactor()
	for i := range sessions {
		sessions[i].Command = red.RedactString(sessions[i].Command)
	}

	jsonOut, _ := cmd.Flags().GetBool("json")
	if jsonOut {
		if sessions == nil {
			sessions = []sessionInfo{}
		}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(sessions)
	}

	if len(sessions) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "No log sessions found in %s\n", dir)
		return nil
	}

	listAll, _ := cmd.Flags().GetBool("all")
	limit := len(sessions)
	if sinceStr == "" && !listAll {
		limit = min(limit, listDefaultLimit)
	}

	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "LOGS (%s)\n", dir)
	fmt.Fprintf(w, "%-10s %-20s %-22s %10s %8s\n",
		"SESSION", "COMMAND", "STARTED", "DURATION", "SIZE")

	for _, s := range sessions[:limit] {
		duration := "-"
		if s.Duration > 0 {
			duration = formatDuration(s.Duration)
		}

		started := "-"
		if !s.Started.IsZero() {
			started = s.Started.Format("2006-01-02 15:04:05")
		}

		fmt.Fprintf(w, "%-10s %-20s %-22s %10s %8s\n",
			s.ID, truncate(s.Command, 20), started, duration, formatBytes(s.Size))
	}
	if more := len(sessions) - limit; more > 0 {
		fmt.Fprintf(w, "... %d more (use --since or --all)\n", more)
	}

	return nil
}

// filterSessionsSince keeps only sessions started within the --since window.
// An empty since keeps every session.
func filterSessionsSince(sessions []sessionInfo, since string, now time.Time) ([]sessionInfo, error) {
	if since == "" {
		return sessions, nil
	}
	d, err := parseDuration(since)
	if err != nil {
		return nil, fmt.Errorf("invalid --since value: %w", err)
	}
	cutoff := now.Add(-d)
	kept := make([]sessionInfo, 0, len(sessions))
	for _, s := range sessions {
		if !s.Started.Before(cutoff) {
			kept = append(kept, s)
		}
	}
	return kept, nil
}

func runShow(cmd *cobra.Command, sessionID string) error {
	dir := resolveLogDir(cmd)
	file, err := findSessionFile(dir, sessionID)
	if err != nil {
		globalDir := logging.GlobalLogDir()
		if dir != globalDir {
			file, err = findSessionFile(globalDir, sessionID)
		}
		if err != nil {
			return fmt.Errorf("session %q not found", sessionID)
		}
	}

	f, err := os.Open(file)
	if err != nil {
		return fmt.Errorf("opening session log: %w", err)
	}
	defer f.Close()

	raw, _ := cmd.Flags().GetBool("raw")
	levelFilter, _ := cmd.Flags().GetString("level")

	// Defense-in-depth: re-scrub at display time. Write-time redaction already
	// runs, but a hand-edited or externally-produced log file may still contain
	// secrets, so never emit an unredacted line to the terminal (F-CAP-29.5-1).
	red := logging.NewRedactor()

	scanner := logging.NewLineScanner(f)
	w := cmd.OutOrStdout()
	for scanner.Scan() {
		line := scanner.Text()

		if levelFilter != "" {
			lvl := jsonField(line, "level")
			if !strings.EqualFold(lvl, levelFilter) {
				continue
			}
		}

		if raw {
			fmt.Fprintln(w, red.RedactString(line))
			continue
		}

		fmt.Fprintln(w, formatShowLine(line, red))
	}

	return scanner.Err()
}

// formatShowLine renders one JSONL record as "time LVL msg key=value ...".
// Every attribute beyond time/level/msg is kept (error details are usually in
// one) and redacted structurally before display. Lines that are not JSON
// objects are shown redacted as-is.
func formatShowLine(line string, red *logging.Redactor) string {
	var rec map[string]any
	if err := json.Unmarshal([]byte(line), &rec); err != nil {
		return red.RedactString(line)
	}

	ts, _ := rec["time"].(string)
	if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
		ts = t.Format("15:04:05.000")
	}
	lvl, _ := rec["level"].(string)
	msg, _ := rec["msg"].(string)

	attrs := make(map[string]any, len(rec))
	for k, v := range rec {
		if !showOmittedKeys[k] {
			attrs[k] = v
		}
	}
	redacted, _ := red.RedactStructured(attrs).(map[string]any)
	keys := make([]string, 0, len(redacted))
	for k := range redacted {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	fmt.Fprintf(&b, "%s %s %s", ts, levelPrefix(lvl), red.RedactString(msg))
	for _, k := range keys {
		fmt.Fprintf(&b, " %s=%s", k, formatAttrValue(redacted[k]))
	}
	return b.String()
}

// formatAttrValue renders a decoded JSON attribute value for `logs show`:
// strings are quoted only when they contain whitespace or quotes, other values
// are compact JSON.
func formatAttrValue(v any) string {
	if s, ok := v.(string); ok {
		if s == "" || strings.ContainsAny(s, " \t\n\"") {
			return strconv.Quote(s)
		}
		return s
	}
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(data)
}

func runPath(cmd *cobra.Command) error {
	dir := resolveLogDir(cmd)
	fmt.Fprintln(cmd.OutOrStdout(), dir)
	return nil
}

func runClean(cmd *cobra.Command) error {
	dir := resolveLogDir(cmd)
	all, _ := cmd.Flags().GetBool("all")
	force, _ := cmd.Flags().GetBool("force")
	olderThan, _ := cmd.Flags().GetString("older-than")

	if !force {
		scope := fmt.Sprintf("older than %s", olderThan)
		if all {
			scope = "ALL"
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "This will delete %s logs in %s. Use --force to confirm.\n", scope, dir)
		return nil
	}

	if all {
		count, err := removeLogFiles(dir)
		fmt.Fprintf(cmd.OutOrStdout(), "Deleted %d log file(s)\n", count)
		return err
	}

	d, err := parseDuration(olderThan)
	if err != nil {
		return fmt.Errorf("invalid --older-than value: %w", err)
	}

	before, err := countLogFiles(dir)
	if err != nil {
		return err
	}
	if err := logging.CleanOldLogs(dir, d); err != nil {
		return fmt.Errorf("cleaning logs older than %s: %w", olderThan, err)
	}
	after, err := countLogFiles(dir)
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Deleted %d log file(s) older than %s\n", before-after, olderThan)
	return nil
}

// logFileNames lists the .jsonl session logs in dir. A missing directory has
// no logs; any other read failure is returned. A path that is not a directory
// is checked explicitly: on Windows os.ReadDir of a regular file does not
// reliably fail, which would report a misconfigured log dir as empty.
func logFileNames(dir string) ([]string, error) {
	info, err := os.Stat(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading log directory %s: %w", dir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("reading log directory %s: not a directory", dir)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading log directory %s: %w", dir, err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".jsonl") {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

func countLogFiles(dir string) (int, error) {
	names, err := logFileNames(dir)
	return len(names), err
}

// removeLogFiles deletes every session log in dir and returns how many were
// actually removed, together with any removal failures.
func removeLogFiles(dir string) (int, error) {
	names, err := logFileNames(dir)
	if err != nil {
		return 0, err
	}
	count := 0
	var errs []error
	for _, name := range names {
		if err := os.Remove(filepath.Join(dir, name)); err != nil {
			errs = append(errs, fmt.Errorf("removing %s: %w", name, err))
			continue
		}
		count++
	}
	return count, errors.Join(errs...)
}

// findSessionFile returns the log file whose name ends in "-<sessionID>.jsonl".
// The ID must match the filename's final segment exactly, so a short or empty
// ID never selects an arbitrary file.
func findSessionFile(dir, sessionID string) (string, error) {
	if sessionID == "" || strings.ContainsAny(sessionID, `/\`) {
		return "", fmt.Errorf("invalid session ID %q", sessionID)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("reading log directory %s: %w", dir, err)
	}
	suffix := "-" + sessionID + ".jsonl"
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), suffix) {
			return filepath.Join(dir, e.Name()), nil
		}
	}
	return "", fmt.Errorf("session %q not found in %s", sessionID, dir)
}

// sessionHeaderLines is how many leading records are searched for the
// session's command (it is logged by the "command starting" record, right
// after the "session started" record).
const sessionHeaderLines = 3

// SessionCommand returns the command path recorded in a session log's opening
// records (e.g. "qsdev enable semgrep"), or "" when none is recorded.
func SessionCommand(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("opening session log: %w", err)
	}
	defer f.Close()

	scanner := logging.NewLineScanner(f)
	for i := 0; i < sessionHeaderLines && scanner.Scan(); i++ {
		if cmd := jsonField(scanner.Text(), "command"); cmd != "" {
			return cmd, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("reading session log %s: %w", path, err)
	}
	return "", nil
}

func readHeadAndTail(path string, headCount int) (head []string, last string, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()

	scanner := logging.NewLineScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if len(head) < headCount {
			head = append(head, line)
		}
		last = line
	}
	return head, last, scanner.Err()
}

// recordSessionID returns the session ID a log record is tagged with. Logs
// written before the ID moved to logging.SessionAttrKey carry it under the
// legacy "session" key, which the redacting handler replaced with the
// redaction marker; such a value is ignored so the caller falls back to the
// ID in the file name.
func recordSessionID(line string) string {
	if id := jsonField(line, logging.SessionAttrKey); id != "" {
		return id
	}
	if id := jsonField(line, "session"); id != logging.RedactionMarker {
		return id
	}
	return ""
}

func jsonField(line, key string) string {
	var m map[string]json.RawMessage
	if err := json.Unmarshal([]byte(line), &m); err != nil {
		return ""
	}
	raw, ok := m[key]
	if !ok {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return strings.Trim(string(raw), `"`)
	}
	return s
}

func levelPrefix(level string) string {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return "DBG"
	case "INFO":
		return "INF"
	case "WARN", "WARNING":
		return "WRN"
	case "ERROR":
		return "ERR"
	default:
		return "???"
	}
}

func truncate(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max-1]) + "…"
}

func formatDuration(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	return fmt.Sprintf("%.1fs", float64(ms)/1000)
}

func formatBytes(b int64) string {
	switch {
	case b < 1024:
		return fmt.Sprintf("%dB", b)
	case b < 1024*1024:
		return fmt.Sprintf("%.1fKB", float64(b)/1024)
	default:
		return fmt.Sprintf("%.1fMB", float64(b)/(1024*1024))
	}
}

func parseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "d") {
		var days int
		if _, err := fmt.Sscanf(s, "%dd", &days); err == nil {
			return time.Duration(days) * 24 * time.Hour, nil
		}
	}
	return time.ParseDuration(s)
}

// ExcerptLimits bounds a log excerpt. Zero values mean "no limit".
type ExcerptLimits struct {
	// LevelFilter keeps only records with this level (case-insensitive).
	LevelFilter string
	// MaxLines caps the number of lines written.
	MaxLines int
	// MaxBytes caps the number of bytes written, including newlines.
	MaxBytes int
}

// ExcerptResult reports what WriteExcerpt wrote.
type ExcerptResult struct {
	Lines int
	Bytes int
	// Truncated is true when a qualifying line was left out because a limit
	// was reached.
	Truncated bool
}

// WriteExcerpt copies log records from r to w, re-scrubbing every line and
// stopping once a line or byte limit would be exceeded. It is the extractor
// behind the bug report's log excerpt.
//
// Entries are re-scrubbed at extraction time as defense-in-depth: a bug report
// is a shareable artifact, so no excerpt should carry a secret even if write-time
// redaction missed it or the file was edited by hand (F-CAP-29.5-1).
func WriteExcerpt(w io.Writer, r io.Reader, lim ExcerptLimits) (ExcerptResult, error) {
	var res ExcerptResult
	red := logging.NewRedactor()
	scanner := logging.NewLineScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if lim.LevelFilter != "" {
			lvl := jsonField(line, "level")
			if !strings.EqualFold(lvl, lim.LevelFilter) {
				continue
			}
		}
		out := red.RedactString(line) + "\n"
		if (lim.MaxLines > 0 && res.Lines >= lim.MaxLines) ||
			(lim.MaxBytes > 0 && res.Bytes+len(out) > lim.MaxBytes) {
			res.Truncated = true
			break
		}
		n, err := io.WriteString(w, out)
		res.Bytes += n
		if err != nil {
			return res, fmt.Errorf("writing log excerpt: %w", err)
		}
		res.Lines++
	}
	if err := scanner.Err(); err != nil {
		return res, fmt.Errorf("reading log: %w", err)
	}
	return res, nil
}
