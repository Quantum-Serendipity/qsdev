package extlog

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/logging"
)

// lineProvider is a test LogProvider serving one file, parsed one entry per
// line with the shared long-line-safe scanner, optionally failing after the
// lines are read.
type lineProvider struct {
	path     string
	parseErr error
}

func (p *lineProvider) Name() string            { return "fake" }
func (p *lineProvider) DisplayName() string     { return "fake" }
func (p *lineProvider) Detect(_, _ string) bool { return true }
func (p *lineProvider) Discover(_, _ string, _ time.Time) ([]LogFile, error) {
	return []LogFile{{Path: p.path, Provider: "fake"}}, nil
}

func (p *lineProvider) Parse(r io.Reader, sourceFile string) ([]LogEntry, error) {
	sc := logging.NewLineScanner(r)
	var entries []LogEntry
	for sc.Scan() {
		entries = append(entries, LogEntry{Source: "fake", File: sourceFile, Message: sc.Text(), Level: LevelError})
	}
	if err := sc.Err(); err != nil {
		return entries, err
	}
	return entries, p.parseErr
}

func collectFake(t *testing.T, content string, parseErr error) ([]LogEntry, CollectionSummary) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fake.log")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	reg := NewRegistry()
	reg.Register(&lineProvider{path: path, parseErr: parseErr})
	entries, summaries := collectFrom(reg, "", "", DefaultWindow(60))
	if len(summaries) != 1 {
		t.Fatalf("got %d summaries, want 1", len(summaries))
	}
	return entries["fake"], summaries[0]
}

func TestCollectFrom(t *testing.T) {
	t.Parallel()

	t.Run("over-long line does not drop the rest of the file", func(t *testing.T) {
		t.Parallel()
		content := "first\n" + strings.Repeat("x", 2*logging.MaxLogLineBytes) + "\nerror: the real failure\n"
		entries, summary := collectFake(t, content, nil)
		if len(summary.CollectionErrors) != 0 {
			t.Errorf("unexpected collection errors: %v", summary.CollectionErrors)
		}
		if len(entries) != 3 {
			t.Fatalf("got %d entries, want 3", len(entries))
		}
		if entries[2].Message != "error: the real failure" {
			t.Errorf("last entry = %q, want the line after the long one", entries[2].Message)
		}
		if !strings.HasSuffix(entries[1].Message, logging.TruncatedLineSuffix) {
			t.Errorf("long line was not marked truncated")
		}
	})

	t.Run("parse error keeps entries parsed before it", func(t *testing.T) {
		t.Parallel()
		entries, summary := collectFake(t, "one\ntwo\n", errors.New("read failed"))
		if len(entries) != 2 {
			t.Errorf("got %d entries, want the 2 parsed before the error", len(entries))
		}
		if len(summary.CollectionErrors) != 1 {
			t.Errorf("collection errors = %v, want the parse error recorded", summary.CollectionErrors)
		}
	})

	t.Run("multi-line private key is fully redacted", func(t *testing.T) {
		t.Parallel()
		content := strings.Join([]string{
			"-----BEGIN OPENSSH PRIVATE KEY-----",
			"b3BlbnNzaC1rZXktdjEAAAAAsecretbody",
			"-----END OPENSSH PRIVATE KEY-----",
			"after",
		}, "\n")
		entries, _ := collectFake(t, content, nil)
		for _, e := range entries {
			if strings.Contains(e.Message, "secretbody") || strings.Contains(e.Message, "PRIVATE KEY") {
				t.Errorf("key material survived collection: %q", e.Message)
			}
		}
		if got := entries[len(entries)-1].Message; got != "after" {
			t.Errorf("line after the key block = %q, want it untouched", got)
		}
	})
}
