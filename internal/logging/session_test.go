package logging

import (
	"bufio"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/secrets"
)

// initTestSession runs Init against a temporary project and restores the
// process-global default logger afterwards. Tests using it must not be
// parallel: Init replaces slog's default logger.
func initTestSession(t *testing.T, cfg Config) *Session {
	t.Helper()
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })
	t.Setenv("QSDEV_LOG", "")

	s, err := Init(cfg)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if s == nil {
		t.Fatal("Init returned a nil session")
	}
	return s
}

// sessionFiles returns the .jsonl files directly inside dir.
func sessionFiles(t *testing.T, dir string) []string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	return matches
}

// TestSessionAttrKeyIsNotRedacted guards the regression where the session ID
// attribute ("session") matched the sensitive-name canon and was written as
// [REDACTED] in every record, breaking `logs list` / `logs show`.
func TestSessionAttrKeyIsNotRedacted(t *testing.T) {
	if secrets.IsSensitiveName(SessionAttrKey) {
		t.Fatalf("SessionAttrKey %q is a sensitive name and would be redacted", SessionAttrKey)
	}

	root := t.TempDir()
	s := initTestSession(t, Config{ProjectRoot: root, ProjectScoped: true})
	s.Close()

	files := sessionFiles(t, ProjectLogDir(root))
	if len(files) != 1 {
		t.Fatalf("got %d session files, want 1", len(files))
	}
	f, err := os.Open(files[0])
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	records := 0
	for sc.Scan() {
		var rec map[string]any
		if err := json.Unmarshal(sc.Bytes(), &rec); err != nil {
			t.Fatalf("record is not JSON: %s", sc.Text())
		}
		if rec[SessionAttrKey] != s.ID {
			t.Errorf("record %s = %v, want session ID %q", SessionAttrKey, rec[SessionAttrKey], s.ID)
		}
		records++
	}
	if records == 0 {
		t.Fatal("no records written")
	}
	if !strings.HasSuffix(strings.TrimSuffix(filepath.Base(files[0]), ".jsonl"), "-"+s.ID) {
		t.Errorf("file name %q does not end with session ID %q", files[0], s.ID)
	}
}

// TestInit_PrivatePermissions proves session logs are owner-only, including
// the tier's log directory when an automated session is the first to create it.
func TestInit_PrivatePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not enforced on Windows")
	}
	tests := []struct {
		name      string
		automated bool
	}{
		{name: "user session"},
		{name: "automated session", automated: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			s := initTestSession(t, Config{ProjectRoot: root, ProjectScoped: true, Automated: tt.automated})
			s.Close()

			dirs := []string{ProjectLogDir(root)}
			if tt.automated {
				dirs = append(dirs, filepath.Join(dirs[0], AutomatedLogSubdir))
			}
			for _, dir := range dirs {
				info, err := os.Stat(dir)
				if err != nil {
					t.Fatal(err)
				}
				if perm := info.Mode().Perm(); perm != modeDirPrivate {
					t.Errorf("%s mode = %o, want %o", dir, perm, modeDirPrivate)
				}
			}
			files := sessionFiles(t, dirs[len(dirs)-1])
			if len(files) != 1 {
				t.Fatalf("got %d session files, want 1", len(files))
			}
			fi, err := os.Stat(files[0])
			if err != nil {
				t.Fatal(err)
			}
			if perm := fi.Mode().Perm(); perm != 0o600 {
				t.Errorf("log file mode = %o, want 600", perm)
			}
		})
	}
}

// TestInit_NoHomeRefusesTempFallback proves a missing home directory yields an
// error instead of a predictable shared-temp log path.
func TestInit_NoHomeRefusesTempFallback(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	t.Setenv("home", "")
	t.Setenv("QSDEV_LOG_DIR", "")
	t.Setenv("QSDEV_LOG", "")

	if dir := GlobalLogDir(); dir != "" {
		t.Fatalf("GlobalLogDir() = %q, want empty without a home directory", dir)
	}
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })
	if s, err := Init(Config{}); err == nil {
		s.Close()
		t.Fatal("Init succeeded without a resolvable log directory")
	}
}

// TestInit_AutomatedSessionsDoNotEvictUserLogs proves a burst of automated
// (hook / MCP server) sessions is kept and pruned apart from user-command logs.
func TestInit_AutomatedSessionsDoNotEvictUserLogs(t *testing.T) {
	root := t.TempDir()
	user := initTestSession(t, Config{ProjectRoot: root, ProjectScoped: true, MaxFiles: 3})
	user.Close()

	for i := 0; i < 5; i++ {
		s := initTestSession(t, Config{ProjectRoot: root, ProjectScoped: true, Automated: true, MaxFiles: 3})
		s.Close()
	}

	userDir := ProjectLogDir(root)
	if got := sessionFiles(t, userDir); len(got) != 1 {
		t.Errorf("user log dir has %d files, want the 1 user session to survive", len(got))
	}
	if got := sessionFiles(t, filepath.Join(userDir, AutomatedLogSubdir)); len(got) != 3 {
		t.Errorf("automated log dir has %d files, want 3 (its own cap)", len(got))
	}
}
