package logcmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/logging"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// Command returns the "logs" cobra command tree.
func Command() *cobra.Command {
	app := branding.Get().AppName
	cmd := &cobra.Command{
		Use:   "logs",
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
	list.Flags().StringVar(&since, "since", "", "Show sessions since duration (e.g. 1h, 24h)")
	list.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")

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
					si.ID = jsonField(line, "session")
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

	if len(sessions) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "No log sessions found in %s\n", dir)
		return nil
	}

	jsonOut, _ := cmd.Flags().GetBool("json")
	if jsonOut {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(sessions)
	}

	sinceStr, _ := cmd.Flags().GetString("since")
	var cutoff time.Time
	if sinceStr != "" {
		d, err := parseDuration(sinceStr)
		if err != nil {
			return fmt.Errorf("invalid --since value: %w", err)
		}
		cutoff = time.Now().Add(-d)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "LOGS (%s)\n", dir)
	fmt.Fprintf(cmd.OutOrStdout(), "%-10s %-20s %-22s %10s %8s\n",
		"SESSION", "COMMAND", "STARTED", "DURATION", "SIZE")

	count := 0
	for _, s := range sessions {
		if !cutoff.IsZero() && s.Started.Before(cutoff) {
			continue
		}
		if count >= 20 && sinceStr == "" {
			fmt.Fprintf(cmd.OutOrStdout(), "... %d more (use --since or logs list --all)\n", len(sessions)-count)
			break
		}

		duration := "-"
		if s.Duration > 0 {
			duration = formatDuration(s.Duration)
		}

		started := "-"
		if !s.Started.IsZero() {
			started = s.Started.Format("2006-01-02 15:04:05")
		}

		fmt.Fprintf(cmd.OutOrStdout(), "%-10s %-20s %-22s %10s %8s\n",
			s.ID, truncate(s.Command, 20), started, duration, formatBytes(s.Size))
		count++
	}

	return nil
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
		return err
	}
	defer f.Close()

	raw, _ := cmd.Flags().GetBool("raw")
	levelFilter, _ := cmd.Flags().GetString("level")

	scanner := bufio.NewScanner(f)
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
			fmt.Fprintln(w, line)
			continue
		}

		ts := jsonField(line, "time")
		lvl := jsonField(line, "level")
		msg := jsonField(line, "msg")

		if ts != "" {
			if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
				ts = t.Format("15:04:05.000")
			}
		}

		prefix := levelPrefix(lvl)
		fmt.Fprintf(w, "%s %s %s\n", ts, prefix, msg)
	}

	return scanner.Err()
}

func runPath(cmd *cobra.Command) error {
	dir := resolveLogDir(cmd)
	fmt.Fprintln(cmd.OutOrStdout(), dir)
	return nil
}

func runClean(cmd *cobra.Command, ) error {
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
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil
		}
		count := 0
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".jsonl") {
				os.Remove(filepath.Join(dir, e.Name()))
				count++
			}
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Deleted %d log file(s)\n", count)
		return nil
	}

	d, err := parseDuration(olderThan)
	if err != nil {
		return fmt.Errorf("invalid --older-than value: %w", err)
	}

	return logging.CleanOldLogs(dir, d)
}

func findSessionFile(dir, sessionID string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), sessionID) && strings.HasSuffix(e.Name(), ".jsonl") {
			return filepath.Join(dir, e.Name()), nil
		}
	}
	return "", fmt.Errorf("not found")
}

func readHeadAndTail(path string, headCount int) (head []string, last string, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if len(head) < headCount {
			head = append(head, line)
		}
		last = line
	}
	return head, last, scanner.Err()
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

// WriteTo writes log entries from the given reader, applying optional level filter,
// to the writer. Used by the bug report system to extract log excerpts.
func WriteTo(w io.Writer, r io.Reader, levelFilter string, maxLines int) (int, error) {
	scanner := bufio.NewScanner(r)
	count := 0
	for scanner.Scan() {
		if maxLines > 0 && count >= maxLines {
			break
		}
		line := scanner.Text()
		if levelFilter != "" {
			lvl := jsonField(line, "level")
			if !strings.EqualFold(lvl, levelFilter) {
				continue
			}
		}
		fmt.Fprintln(w, line)
		count++
	}
	return count, scanner.Err()
}
