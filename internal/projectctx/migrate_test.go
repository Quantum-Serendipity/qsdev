package projectctx

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// writeLog creates dir/name holding body, creating dir.
func writeLog(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// readLog returns the contents of dir/name, or "" when dir is not a directory
// or dir/name does not exist.
func readLog(t *testing.T, dir, name string) string {
	t.Helper()
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(dir, name))
	if errors.Is(err, fs.ErrNotExist) {
		return ""
	}
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestMigrateLegacyLogs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		// setup plants files below the legacy and state directories.
		setup      func(t *testing.T, d Dirs)
		wantMoved  bool
		wantLegacy string // contents of Legacy/logs/s.jsonl afterwards
		wantState  string // contents of State/logs/s.jsonl afterwards
	}{
		{
			name:      "renames-once",
			setup:     func(t *testing.T, d Dirs) { writeLog(t, filepath.Join(d.Legacy, "logs"), "s.jsonl", "old") },
			wantMoved: true,
			wantState: "old",
		},
		{
			name: "colliding-entry-kept",
			setup: func(t *testing.T, d Dirs) {
				writeLog(t, filepath.Join(d.Legacy, "logs"), "s.jsonl", "old")
				writeLog(t, filepath.Join(d.State, "logs"), "s.jsonl", "new")
			},
			wantLegacy: "old",
			wantState:  "new",
		},
		{
			name: "merges-into-empty-target-dir",
			setup: func(t *testing.T, d Dirs) {
				writeLog(t, filepath.Join(d.Legacy, "logs"), "s.jsonl", "old")
				if err := os.MkdirAll(filepath.Join(d.State, "logs"), 0o700); err != nil {
					t.Fatal(err)
				}
			},
			wantMoved: true,
			wantState: "old",
		},
		{
			name: "target-not-a-directory-untouched",
			setup: func(t *testing.T, d Dirs) {
				writeLog(t, filepath.Join(d.Legacy, "logs"), "s.jsonl", "old")
				writeLog(t, d.State, "logs", "file")
			},
			wantLegacy: "old",
		},
		{
			name:  "missing-legacy-no-op",
			setup: func(*testing.T, Dirs) {},
		},
		{
			name: "legacy-logs-not-a-directory",
			setup: func(t *testing.T, d Dirs) {
				writeLog(t, d.Legacy, "logs", "file")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base := t.TempDir()
			d := Dirs{
				Home:   base,
				State:  filepath.Join(base, "state", "app"),
				Cache:  filepath.Join(base, "cache", "app"),
				Legacy: filepath.Join(base, ".app"),
			}
			tt.setup(t, d)

			moved, err := MigrateLegacyLogs(d)
			if err != nil {
				t.Fatalf("MigrateLegacyLogs: %v", err)
			}
			if moved != tt.wantMoved {
				t.Errorf("moved = %v, want %v", moved, tt.wantMoved)
			}
			if got := readLog(t, filepath.Join(d.Legacy, "logs"), "s.jsonl"); got != tt.wantLegacy {
				t.Errorf("legacy log = %q, want %q", got, tt.wantLegacy)
			}
			if got := readLog(t, filepath.Join(d.State, "logs"), "s.jsonl"); got != tt.wantState {
				t.Errorf("state log = %q, want %q", got, tt.wantState)
			}

			// A second run never moves anything: the migration is one-time.
			if moved, err := MigrateLegacyLogs(d); err != nil || moved {
				t.Errorf("second MigrateLegacyLogs = (%v, %v), want (false, nil)", moved, err)
			}
		})
	}
}

// TestMigrateLegacyLogs_NeverCopies pins that a rename that fails (here: the
// state directory's parent is a file) leaves the legacy logs in place and
// writes nothing at the target, instead of falling back to a copy.
func TestMigrateLegacyLogs_NeverCopies(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	blocker := filepath.Join(base, "state")
	if err := os.WriteFile(blocker, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	d := Dirs{Home: base, State: filepath.Join(blocker, "app"), Legacy: filepath.Join(base, ".app")}
	writeLog(t, filepath.Join(d.Legacy, "logs"), "s.jsonl", "old")

	moved, err := MigrateLegacyLogs(d)
	if err == nil || moved {
		t.Fatalf("MigrateLegacyLogs = (%v, %v), want an error and no move", moved, err)
	}
	if got := readLog(t, filepath.Join(d.Legacy, "logs"), "s.jsonl"); got != "old" {
		t.Errorf("legacy log = %q, want it left in place", got)
	}
}

// TestMigrateLegacyLogs_MergesAfterAutomatedSession covers the order Claude
// Code produces after an upgrade: a hook process (which never migrates) opens
// a session in State/logs/automated first, then a person runs a command. The
// legacy logs, including the legacy automated sub-tier, still move; nothing
// at the target is overwritten; the emptied legacy directory is removed.
func TestMigrateLegacyLogs_MergesAfterAutomatedSession(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	d := Dirs{Home: base, State: filepath.Join(base, "state", "app"), Legacy: filepath.Join(base, ".app")}
	legacy, target := filepath.Join(d.Legacy, "logs"), filepath.Join(d.State, "logs")
	writeLog(t, legacy, "old.jsonl", "old")
	writeLog(t, filepath.Join(legacy, "automated"), "hook-old.jsonl", "hook-old")
	writeLog(t, filepath.Join(target, "automated"), "hook-new.jsonl", "hook-new")

	moved, err := MigrateLegacyLogs(d)
	if err != nil || !moved {
		t.Fatalf("MigrateLegacyLogs = (%v, %v), want (true, nil)", moved, err)
	}
	for _, f := range []struct{ dir, name, want string }{
		{target, "old.jsonl", "old"},
		{filepath.Join(target, "automated"), "hook-old.jsonl", "hook-old"},
		{filepath.Join(target, "automated"), "hook-new.jsonl", "hook-new"},
	} {
		if got := readLog(t, f.dir, f.name); got != f.want {
			t.Errorf("%s = %q, want %q", filepath.Join(f.dir, f.name), got, f.want)
		}
	}
	if _, err := os.Lstat(legacy); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("legacy logs dir after the merge: Lstat error = %v, want it removed", err)
	}
	if moved, err := MigrateLegacyLogs(d); err != nil || moved {
		t.Errorf("second MigrateLegacyLogs = (%v, %v), want (false, nil)", moved, err)
	}
}
