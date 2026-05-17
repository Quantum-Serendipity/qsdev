package logging

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/sysinfo"
	"github.com/Quantum-Serendipity/qsdev/internal/version"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// Session represents a single CLI invocation's logging context.
type Session struct {
	ID        string
	Command   string
	StartedAt time.Time
	LogDir    string
	logFile   *os.File
}

// Config controls logger initialization.
type Config struct {
	Level         slog.Level
	ProjectRoot   string
	ProjectScoped bool
	MaxFiles      int
	StderrToo     bool
}

// Init initializes the logging system and returns a Session.
// It creates the log directory, opens a session log file, sets
// slog.SetDefault(), and prunes old logs.
func Init(cfg Config) (*Session, error) {
	if isDisabled() {
		return nil, nil
	}

	if cfg.Level == 0 {
		cfg.Level = LevelFromEnv()
	}
	if cfg.MaxFiles == 0 {
		cfg.MaxFiles = defaultMaxFiles
	}

	logDir := ResolveLogDir(cfg.ProjectRoot, cfg.ProjectScoped)
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating log directory %s: %w", logDir, err)
	}

	sessionID := generateSessionID()
	now := time.Now()
	filename := fmt.Sprintf("%s%s-%s.jsonl",
		branding.Get().LogFilePrefix,
		now.Format("2006-01-02T15-04-05"),
		sessionID,
	)

	logPath := filepath.Join(logDir, filename)
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("opening log file %s: %w", logPath, err)
	}

	session := &Session{
		ID:        sessionID,
		StartedAt: now,
		LogDir:    logDir,
		logFile:   f,
	}

	fileHandler := slog.NewJSONHandler(f, &slog.HandlerOptions{Level: cfg.Level})
	redactedFile := NewRedactingHandler(fileHandler)

	var handler slog.Handler
	if cfg.StderrToo {
		stderrHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: cfg.Level})
		redactedStderr := NewRedactingHandler(stderrHandler)
		handler = NewTeeHandler(redactedFile, redactedStderr)
	} else {
		handler = redactedFile
	}

	logger := slog.New(handler).With("session", sessionID)
	slog.SetDefault(logger)

	writeOpeningRecord(session)

	go pruneExcessLogs(logDir, cfg.MaxFiles)

	return session, nil
}

// Close writes a closing record and closes the log file.
func (s *Session) Close() {
	if s == nil || s.logFile == nil {
		return
	}
	slog.Info("session complete",
		"duration_ms", time.Since(s.StartedAt).Milliseconds(),
	)
	s.logFile.Close()
}

// LevelFromEnv reads the QSDEV_LOG environment variable and returns
// the corresponding slog.Level. Returns slog.LevelInfo if unset or invalid.
func LevelFromEnv() slog.Level {
	switch strings.ToLower(os.Getenv(branding.Get().EnvLogVar)) {
	case "debug":
		return slog.LevelDebug
	case "info", "":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func isDisabled() bool {
	return strings.ToLower(os.Getenv(branding.Get().EnvLogVar)) == "off"
}

func generateSessionID() string {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%06x", time.Now().UnixNano()&0xFFFFFF)
	}
	return hex.EncodeToString(b)
}

func writeOpeningRecord(s *Session) {
	bi := version.Info()
	osInfo := sysinfo.DetectOS()
	slog.Info("session started",
		"version", bi.Version,
		"commit", bi.Commit,
		"os", osInfo.OS,
		"arch", osInfo.Arch,
		"family", osInfo.Family,
		"shell", osInfo.Shell,
	)
}
