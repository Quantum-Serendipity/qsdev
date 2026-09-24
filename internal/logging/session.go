package logging

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/sysinfo"
	"github.com/Quantum-Serendipity/qsdev/internal/version"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

// Session represents a single CLI invocation's logging context.
type Session struct {
	ID        string
	Command   string
	StartedAt time.Time
	LogDir    string
	logFile   *os.File
}

// SessionAttrKey is the attribute key that tags every record with its session
// ID. It must not be a name secrets.IsSensitiveName matches (as "session" and
// "session_id" do), or the RedactingHandler would replace every ID with the
// redaction marker and `logs list` / `logs show` could no longer tell sessions
// apart.
const SessionAttrKey = "run_id"

// modeDirPrivate is the permission for log directories: session logs can hold
// diagnostic detail about the user's environment, so only the owner may list
// or read them.
const modeDirPrivate os.FileMode = 0o700

// Config controls logger initialization.
type Config struct {
	Level         slog.Level
	ProjectRoot   string
	ProjectScoped bool
	// Automated routes the session to the AutomatedLogSubdir of its tier, for
	// commands invoked by tooling rather than the user (see ClassAutomated).
	Automated bool
	MaxFiles  int
	StderrToo bool
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
	if logDir == "" {
		return nil, fmt.Errorf("resolving log directory: no home directory and no %s override", branding.Get().EnvLogDirVar)
	}
	if err := ensurePrivateDir(logDir); err != nil {
		return nil, err
	}
	if cfg.Automated {
		// Create the tier's log directory first (above) so it is private too,
		// rather than a default-mode parent of the automated sub-directory.
		logDir = filepath.Join(logDir, AutomatedLogSubdir)
		if err := ensurePrivateDir(logDir); err != nil {
			return nil, err
		}
	}

	sessionID := generateSessionID()
	now := time.Now()
	filename := fmt.Sprintf("%s%s-%s.jsonl",
		branding.Get().LogFilePrefix,
		now.Format("2006-01-02T15-04-05"),
		sessionID,
	)

	logPath := filepath.Join(logDir, filename)
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, fileutil.ModePrivate)
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

	logger := slog.New(handler).With(SessionAttrKey, sessionID)
	slog.SetDefault(logger)

	writeOpeningRecord(session)

	pruneExcessLogs(logDir, cfg.MaxFiles)

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

// ensurePrivateDir creates dir with owner-only permissions. Missing parents
// (e.g. the project's .qsdev/, which other tooling shares) get the default
// directory mode; only the log directory itself is made private.
func ensurePrivateDir(dir string) error {
	if err := os.MkdirAll(filepath.Dir(dir), fileutil.ModeDirDefault); err != nil {
		return fmt.Errorf("creating log directory parent %s: %w", filepath.Dir(dir), err)
	}
	if err := os.Mkdir(dir, modeDirPrivate); err != nil && !errors.Is(err, fs.ErrExist) {
		return fmt.Errorf("creating log directory %s: %w", dir, err)
	}
	return nil
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
