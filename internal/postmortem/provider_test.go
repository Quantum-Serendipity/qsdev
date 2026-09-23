package postmortem

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func writeSession(t *testing.T, path string, lines ...string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestProvider_ConfinesPaths proves the agent-supplied paths cannot direct the
// tools outside the Claude sessions directory.
func TestProvider_ConfinesPaths(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	outside := t.TempDir()
	inside := filepath.Join(root, "proj", "s.jsonl")
	writeSession(t, inside, `{"type":"user","sessionId":"in"}`)
	writeSession(t, filepath.Join(outside, "o.jsonl"), `{"type":"user","sessionId":"out"}`)

	p := &MCPProvider{SessionsRoot: root}
	ctx := context.Background()

	tests := []struct {
		name    string
		call    func() (string, error)
		wantErr bool
	}{
		{"analyze inside", func() (string, error) {
			return p.handleAnalyzeSession(ctx, map[string]any{"session_path": inside})
		}, false},
		{"analyze outside", func() (string, error) {
			return p.handleAnalyzeSession(ctx, map[string]any{"session_path": filepath.Join(outside, "o.jsonl")})
		}, true},
		{"analyze traversal", func() (string, error) {
			return p.handleAnalyzeSession(ctx, map[string]any{"session_path": filepath.Join(root, "..", filepath.Base(outside), "o.jsonl")})
		}, true},
		{"list default root", func() (string, error) {
			return p.handleListFailurePatterns(ctx, map[string]any{})
		}, false},
		{"list subdirectory", func() (string, error) {
			return p.handleListFailurePatterns(ctx, map[string]any{"sessions_dir": filepath.Join(root, "proj")})
		}, false},
		{"list filesystem root", func() (string, error) {
			return p.handleListFailurePatterns(ctx, map[string]any{"sessions_dir": "/"})
		}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := tt.call()
			if (err != nil) != tt.wantErr {
				t.Errorf("err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProvider_ConfineRejectsSymlinkEscape(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need privileges on Windows")
	}
	root := t.TempDir()
	outside := t.TempDir()
	writeSession(t, filepath.Join(outside, "o.jsonl"), `{}`)
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	p := &MCPProvider{SessionsRoot: root}
	if _, err := p.handleListFailurePatterns(context.Background(), map[string]any{"sessions_dir": filepath.Join(root, "link")}); err == nil {
		t.Error("symlink out of the sessions root was followed")
	}
}

// TestProvider_ListReportsSkipped proves unparseable files are counted rather
// than silently dropped, and that a file cap truncates the scan visibly.
func TestProvider_ListReportsSkipped(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not enforced on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root can read unreadable files")
	}
	root := t.TempDir()
	writeSession(t, filepath.Join(root, "a.jsonl"), `{"type":"user","sessionId":"a"}`)
	bad := filepath.Join(root, "b.jsonl")
	writeSession(t, bad, `{}`)
	if err := os.Chmod(bad, 0o000); err != nil {
		t.Fatal(err)
	}

	out, err := (&MCPProvider{SessionsRoot: root}).handleListFailurePatterns(context.Background(), map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	var report FailureReport
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatal(err)
	}
	if report.TotalSessions != 1 || report.SkippedFiles != 1 {
		t.Errorf("report = %+v, want 1 session and 1 skipped file", report)
	}

	scan, err := FindSessionFiles(root, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(scan.Paths) != 1 || !scan.Truncated {
		t.Errorf("scan with limit 1 = %+v, want 1 path and Truncated", scan)
	}
}

// TestProvider_ChecklistErrors proves a checklist failure is reported instead
// of being replaced by Go commands that do not apply to the project.
func TestProvider_ChecklistErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		fn      func() ([]string, error)
		want    string
		wantErr bool
	}{
		{"no source", nil, "", true},
		{"source error", func() ([]string, error) { return nil, errors.New("no answers file") }, "", true},
		{"no commands", func() ([]string, error) { return nil, nil }, "[]", false},
		{"commands", func() ([]string, error) { return []string{"npm test"}, nil }, "npm test", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out, err := (&MCPProvider{ChecklistFunc: tt.fn}).handleGenerateChecklist(context.Background(), nil)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if strings.Contains(out, "go test") {
				t.Errorf("checklist fell back to Go commands: %s", out)
			}
			if !strings.Contains(out, tt.want) {
				t.Errorf("checklist = %s, want it to contain %q", out, tt.want)
			}
		})
	}
}
