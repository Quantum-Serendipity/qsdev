package logcmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// writeSessionLog writes a minimal session log named "<prefix>-<id>.jsonl"
// whose opening record starts at started, followed by extra records.
func writeSessionLog(t *testing.T, dir, id, command string, started time.Time, extra ...string) {
	t.Helper()
	lines := append([]string{fmt.Sprintf(`{"time":%q,"level":"INFO","msg":"session started","session":%q}`,
		started.Format(time.RFC3339Nano), id),
		fmt.Sprintf(`{"time":%q,"level":"INFO","msg":"command starting","session":%q,"command":%q}`,
			started.Format(time.RFC3339Nano), id, command),
	}, extra...)
	name := "qsdev-" + started.Format("2006-01-02T15-04-05") + "-" + id + ".jsonl"
	if err := os.WriteFile(filepath.Join(dir, name), []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// runLogs executes the logs command tree with args against the global log dir.
func runLogs(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := Command()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestRunList_JSONHonorsSinceAndRedacts(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(branding.Get().EnvLogDirVar, dir)

	now := time.Now()
	writeSessionLog(t, dir, "old111", "qsdev init", now.Add(-3*time.Hour))
	writeSessionLog(t, dir, "new222", "qsdev enable --token "+awsExampleKey, now.Add(-10*time.Minute))

	out, err := runLogs(t, "list", "--global", "--json", "--since", "1h")
	if err != nil {
		t.Fatalf("list --json failed: %v\n%s", err, out)
	}
	var got []sessionInfo
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("decoding JSON: %v\n%s", err, out)
	}
	if len(got) != 1 || got[0].ID != "new222" {
		t.Fatalf("list --json --since 1h = %+v, want only new222", got)
	}
	if strings.Contains(out, awsExampleKey) {
		t.Errorf("list --json leaked a secret from the command field:\n%s", out)
	}
}

func TestRunList_OverflowHint(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(branding.Get().EnvLogDirVar, dir)

	now := time.Now()
	total := listDefaultLimit + 3
	for i := range total {
		writeSessionLog(t, dir, fmt.Sprintf("s%05d", i), "qsdev status", now.Add(-time.Duration(i)*time.Minute))
	}

	out, err := runLogs(t, "list", "--global")
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if !strings.Contains(out, "... 3 more (use --since or --all)") {
		t.Errorf("expected overflow hint for 3 more sessions:\n%s", out)
	}

	out, err = runLogs(t, "list", "--global", "--all")
	if err != nil {
		t.Fatalf("list --all failed: %v", err)
	}
	if strings.Contains(out, "more (use") {
		t.Errorf("list --all should not truncate:\n%s", out)
	}
	if !strings.Contains(out, fmt.Sprintf("s%05d", total-1)) {
		t.Errorf("list --all missing the oldest session:\n%s", out)
	}
}

func TestRunShow_RendersAttributes(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(branding.Get().EnvLogDirVar, dir)

	started := time.Now().Add(-time.Minute)
	writeSessionLog(t, dir, "attr01", "qsdev enable semgrep", started,
		fmt.Sprintf(`{"time":%q,"level":"WARN","msg":"file write failed","session":"attr01","path":"/tmp/x y","error":"permission denied","password":"hunter2"}`,
			started.Format(time.RFC3339Nano)))

	out, err := runLogs(t, "show", "--global", "attr01")
	if err != nil {
		t.Fatalf("show failed: %v", err)
	}
	for _, want := range []string{`error="permission denied"`, `path="/tmp/x y"`, "WRN file write failed"} {
		if !strings.Contains(out, want) {
			t.Errorf("show output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "hunter2") {
		t.Errorf("show leaked a secret attribute:\n%s", out)
	}
	if strings.Contains(out, "session=attr01") {
		t.Errorf("show should not repeat the session attribute:\n%s", out)
	}
}

func TestRunClean_ReportsAccurately(t *testing.T) {
	t.Run("all counts removed files", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv(branding.Get().EnvLogDirVar, dir)
		now := time.Now()
		writeSessionLog(t, dir, "c1", "qsdev status", now)
		writeSessionLog(t, dir, "c2", "qsdev status", now.Add(-time.Second))

		out, err := runLogs(t, "clean", "--global", "--all", "--force")
		if err != nil {
			t.Fatalf("clean failed: %v", err)
		}
		if !strings.Contains(out, "Deleted 2 log file(s)") {
			t.Errorf("unexpected clean output:\n%s", out)
		}
	})

	t.Run("older-than reports count", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv(branding.Get().EnvLogDirVar, dir)
		writeSessionLog(t, dir, "o1", "qsdev status", time.Now())
		old := filepath.Join(dir, "qsdev-2000-01-01T00-00-00-o2.jsonl")
		if err := os.WriteFile(old, []byte("{}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		past := time.Now().Add(-90 * 24 * time.Hour)
		if err := os.Chtimes(old, past, past); err != nil {
			t.Fatal(err)
		}

		out, err := runLogs(t, "clean", "--global", "--older-than", "30d", "--force")
		if err != nil {
			t.Fatalf("clean failed: %v", err)
		}
		if !strings.Contains(out, "Deleted 1 log file(s) older than 30d") {
			t.Errorf("unexpected clean output:\n%s", out)
		}
	})

	t.Run("unreadable log dir is an error", func(t *testing.T) {
		notDir := filepath.Join(t.TempDir(), "file")
		if err := os.WriteFile(notDir, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Setenv(branding.Get().EnvLogDirVar, notDir)

		if _, err := runLogs(t, "clean", "--global", "--all", "--force"); err == nil {
			t.Error("expected an error when the log directory cannot be read")
		}
	})
}

func TestFindSessionFile_ExactID(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	for _, name := range []string{"qsdev-2024-01-01T00-00-00-xabc12.jsonl", "qsdev-2024-01-02T00-00-00-abc123.jsonl"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name    string
		id      string
		want    string
		wantErr bool
	}{
		{name: "exact id", id: "abc123", want: "qsdev-2024-01-02T00-00-00-abc123.jsonl"},
		{name: "empty id", id: "", wantErr: true},
		{name: "partial id", id: "abc", wantErr: true},
		{name: "path separator", id: "../abc123", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := findSessionFile(dir, tt.id)
			if tt.wantErr {
				if err == nil {
					t.Errorf("findSessionFile(%q) = %q, want error", tt.id, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("findSessionFile(%q): %v", tt.id, err)
			}
			if filepath.Base(got) != tt.want {
				t.Errorf("findSessionFile(%q) = %q, want %q", tt.id, filepath.Base(got), tt.want)
			}
		})
	}
}

func TestWriteExcerpt_Limits(t *testing.T) {
	t.Parallel()

	input := strings.Repeat(`{"level":"INFO","msg":"0123456789"}`+"\n", 10)
	lineLen := len(`{"level":"INFO","msg":"0123456789"}`) + 1

	tests := []struct {
		name          string
		lim           ExcerptLimits
		wantLines     int
		wantTruncated bool
	}{
		{name: "no limits", lim: ExcerptLimits{}, wantLines: 10},
		{name: "line limit", lim: ExcerptLimits{MaxLines: 4}, wantLines: 4, wantTruncated: true},
		{name: "line limit exactly met", lim: ExcerptLimits{MaxLines: 10}, wantLines: 10},
		{name: "byte limit", lim: ExcerptLimits{MaxBytes: 3*lineLen + 1}, wantLines: 3, wantTruncated: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			res, err := WriteExcerpt(&buf, strings.NewReader(input), tt.lim)
			if err != nil {
				t.Fatalf("WriteExcerpt: %v", err)
			}
			if res.Lines != tt.wantLines || res.Truncated != tt.wantTruncated {
				t.Errorf("WriteExcerpt = %+v, want lines=%d truncated=%v", res, tt.wantLines, tt.wantTruncated)
			}
			if res.Bytes != buf.Len() {
				t.Errorf("Bytes = %d, wrote %d", res.Bytes, buf.Len())
			}
			if tt.lim.MaxBytes > 0 && buf.Len() > tt.lim.MaxBytes {
				t.Errorf("wrote %d bytes, budget %d", buf.Len(), tt.lim.MaxBytes)
			}
		})
	}
}
