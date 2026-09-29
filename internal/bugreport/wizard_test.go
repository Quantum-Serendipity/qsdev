package bugreport

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/Quantum-Serendipity/qsdev/internal/extlog"
)

type stubProvider struct {
	name     string
	detected bool
}

func (p stubProvider) Name() string            { return p.name }
func (p stubProvider) DisplayName() string     { return p.name + " logs" }
func (p stubProvider) Detect(_, _ string) bool { return p.detected }
func (p stubProvider) Discover(_, _ string, _ time.Time) ([]extlog.LogFile, error) {
	return nil, nil
}
func (p stubProvider) Parse(io.Reader, string) ([]extlog.LogEntry, error) { return nil, nil }

// TestExtLogDescription proves the external-log prompt names only the sources
// that actually have logs, instead of a fixed list that advertised providers
// with nothing to collect.
func TestExtLogDescription(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		providers []stubProvider
		want      string
	}{
		{"none detected", []stubProvider{{"nix", false}}, "No external tool logs detected."},
		{
			name:      "only detected sources, sorted",
			providers: []stubProvider{{"npm", true}, {"nix", false}, {"devenv", true}},
			want:      "Auto-detected logs from devenv logs, npm logs (scrubbed for secrets).",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			reg := extlog.NewRegistry()
			for _, p := range tt.providers {
				reg.Register(p)
			}
			if got := extLogDescription(reg, "", ""); got != tt.want {
				t.Errorf("extLogDescription = %q, want %q", got, tt.want)
			}
		})
	}
}

// fakeDelivery records what a delivery did instead of touching gh, a browser
// or the filesystem.
type fakeDelivery struct {
	out       bytes.Buffer
	submitted []string
	saved     []string
	submitErr error
	checkErr  error
}

func (f *fakeDelivery) delivery() delivery {
	return delivery{
		out:     &f.out,
		errOut:  &f.out,
		checkGH: func() error { return f.checkErr },
		submitGH: func(_, body string) error {
			f.submitted = append(f.submitted, body)
			return f.submitErr
		},
		save: func(_, body string) (string, error) {
			f.saved = append(f.saved, body)
			return "/tmp/report.md", nil
		},
	}
}

func TestDeliver(t *testing.T) {
	t.Parallel()

	ghErr := errors.New("body is too long (maximum is 65536 characters)")
	tests := []struct {
		name          string
		method        string
		confirmed     bool
		submitErr     error
		checkErr      error
		wantErr       bool
		wantSubmitted int
		wantSaved     int
	}{
		{name: "gh confirmed submits", method: methodGH, confirmed: true, wantSubmitted: 1},
		{name: "gh not confirmed never submits", method: methodGH, wantSaved: 1},
		{name: "browser not confirmed never prints url", method: methodBrowser, wantSaved: 1},
		{name: "gh failure saves the report", method: methodGH, confirmed: true, submitErr: ghErr, wantErr: true, wantSubmitted: 1, wantSaved: 1},
		{name: "gh unavailable falls back to file", method: methodGH, confirmed: true, checkErr: errors.New("no gh"), wantSaved: 1},
		{name: "file saves", method: methodFile, wantSaved: 1},
		{name: "cancel does nothing", method: methodCancel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := &fakeDelivery{submitErr: tt.submitErr, checkErr: tt.checkErr}
			err := f.delivery().deliver(tt.method, tt.confirmed, "A title long enough", "body")
			if (err != nil) != tt.wantErr {
				t.Fatalf("deliver error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.submitErr != nil && !errors.Is(err, tt.submitErr) {
				t.Errorf("deliver error %v does not wrap %v", err, tt.submitErr)
			}
			if len(f.submitted) != tt.wantSubmitted {
				t.Errorf("submitted %d time(s), want %d", len(f.submitted), tt.wantSubmitted)
			}
			if len(f.saved) != tt.wantSaved {
				t.Errorf("saved %d time(s), want %d", len(f.saved), tt.wantSaved)
			}
			if strings.Contains(f.out.String(), "https://github.com/") {
				t.Errorf("no URL should be printed for %s:\n%s", tt.method, f.out.String())
			}
		})
	}
}

func TestDeliverBrowser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		body        string
		browserBody func() string
		wantSaved   bool
	}{
		{name: "short body is not saved", body: "short body"},
		{name: "long body is saved before truncating", body: strings.Repeat("{\"k\":\"v\"}\n", 2000), wantSaved: true},
		{name: "logs left out of url are saved", body: "full body with logs", browserBody: func() string { return "body without logs" }, wantSaved: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := &fakeDelivery{}
			d := f.delivery()
			d.browserBody = tt.browserBody
			if err := d.deliver(methodBrowser, true, "A title long enough", tt.body); err != nil {
				t.Fatalf("deliver: %v", err)
			}
			if got := len(f.saved) == 1; got != tt.wantSaved {
				t.Errorf("saved = %v, want %v", got, tt.wantSaved)
			}
			if tt.wantSaved && f.saved[0] != tt.body {
				t.Errorf("saved body is not the full report")
			}
			if !strings.Contains(f.out.String(), "https://github.com/") {
				t.Errorf("URL not printed:\n%s", f.out.String())
			}
		})
	}
}

func TestBrowserURL_FitsEncodedLimitOnRuneBoundary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
	}{
		{name: "json heavy", body: strings.Repeat(`{"level":"INFO","msg":"a b"}`+"\n", 400)},
		{name: "multibyte", body: strings.Repeat("\u00e9\u4e16\U0001F600", 3000)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			u, truncated := BrowserURL("title", tt.body)
			if !truncated {
				t.Fatal("expected truncation")
			}
			if len(u) > browserMaxURLLen {
				t.Errorf("URL length %d exceeds %d", len(u), browserMaxURLLen)
			}
			parsed, err := url.Parse(u)
			if err != nil {
				t.Fatalf("parsing URL: %v", err)
			}
			body := parsed.Query().Get("body")
			if !utf8.ValidString(body) {
				t.Error("truncated body is not valid UTF-8")
			}
			if !strings.HasSuffix(body, browserTruncationNote) {
				t.Error("truncated body lacks the truncation note")
			}
		})
	}
}

func TestFitIssueBody(t *testing.T) {
	t.Parallel()

	r := BugReport{
		Description:   "something broke",
		LogExcerpt:    strings.Repeat(`{"level":"INFO","msg":"line"}`+"\n", 3000),
		SessionInfo:   "1 session(s), 3000 lines",
		ExtLogExcerpt: strings.Repeat("[INFO] npm line\n", 3000),
	}
	body := fitIssueBody(&r, issueBodyMaxBytes)
	if len(body) > issueBodyMaxBytes {
		t.Errorf("body is %d bytes, want <= %d", len(body), issueBodyMaxBytes)
	}
	if !strings.Contains(body, "something broke") {
		t.Error("description was dropped")
	}
	if r.LogExcerpt == "" || !strings.HasSuffix(r.LogExcerpt, "\n") {
		t.Error("log excerpt should be kept, cut on a line boundary")
	}
}

func TestCollectLogs_SkipsMetaSessionsAndMergesTiers(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	global := t.TempDir()
	now := time.Now()

	writeLog(t, global, "aaa111", "qsdev self-update", now.Add(-10*time.Minute), `"msg":"update failed"`)
	writeLog(t, project, "bbb222", "qsdev enable semgrep", now.Add(-20*time.Minute), `"msg":"enable failed"`)
	writeLog(t, global, "ccc333", "qsdev report bug", now.Add(-1*time.Second), `"msg":"report"`)
	writeLog(t, project, "ddd444", "qsdev logs show x", now.Add(-2*time.Second), `"msg":"logs"`)
	writeLog(t, global, "eee555", "qsdev __complete en", now.Add(-3*time.Second), `"msg":"completion"`)

	dirs := []string{project, global}

	t.Run("last command is the newest real command in any tier", func(t *testing.T) {
		t.Parallel()
		excerpt, info := collectLogs("last-command", dirs, now)
		if !strings.Contains(excerpt, "update failed") {
			t.Errorf("expected the self-update session, got:\n%s", excerpt)
		}
		for _, unwanted := range []string{`"msg":"report"`, `"msg":"logs"`, `"msg":"completion"`, "enable failed"} {
			if strings.Contains(excerpt, unwanted) {
				t.Errorf("excerpt must not contain %s:\n%s", unwanted, excerpt)
			}
		}
		if !strings.HasPrefix(info, "1 session(s)") {
			t.Errorf("info = %q", info)
		}
	})

	t.Run("hour window includes both tiers", func(t *testing.T) {
		t.Parallel()
		excerpt, _ := collectLogs("1h", dirs, now)
		if !strings.Contains(excerpt, "update failed") || !strings.Contains(excerpt, "enable failed") {
			t.Errorf("expected sessions from both tiers:\n%s", excerpt)
		}
	})
}

func TestCollectLogs_ByteBudget(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	now := time.Now()
	long := strings.Repeat("x", 1000)
	extra := make([]string, 100)
	for i := range extra {
		extra[i] = fmt.Sprintf(`"msg":"%s-%d"`, long, i)
	}
	writeLog(t, dir, "big001", "qsdev init", now.Add(-time.Minute), extra...)

	excerpt, _ := collectLogs("last-command", []string{dir}, now)
	if len(excerpt) > logExcerptMaxBytes+200 {
		t.Errorf("excerpt is %d bytes, budget %d", len(excerpt), logExcerptMaxBytes)
	}
	if !strings.Contains(excerpt, "truncated") {
		t.Errorf("expected a truncation note")
	}
}

func TestFormatExtLogExcerpt_SortedProviders(t *testing.T) {
	t.Parallel()

	entries := map[string][]extlog.LogEntry{}
	for _, p := range []string{"npm", "devenv", "nix", "generic"} {
		entries[p] = []extlog.LogEntry{{Level: extlog.LevelInfo, Message: p + " line"}}
	}
	want := formatExtLogExcerpt(entries)
	for range 20 {
		if got := formatExtLogExcerpt(entries); got != want {
			t.Fatalf("output is not deterministic")
		}
	}
	if strings.Index(want, "--- devenv") > strings.Index(want, "--- npm") {
		t.Errorf("providers not sorted:\n%s", want)
	}
}

func TestCheckGH_InstallHint(t *testing.T) {
	// Not parallel: t.Setenv.
	t.Setenv("PATH", t.TempDir())

	err := CheckGH()
	if err == nil {
		t.Fatal("expected an error without gh on PATH")
	}
	if strings.Contains(err.Error(), "nix-env") {
		t.Errorf("hint must not recommend imperative nix-env installs: %v", err)
	}
	if !strings.Contains(err.Error(), "devenv add-package gh") {
		t.Errorf("hint should recommend devenv add-package: %v", err)
	}
}

func TestSubmitViaGH_BodyOnStdinAndLabelFallback(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake gh is a shell script")
	}

	tests := []struct {
		name          string
		failOnLabel   bool
		wantCalls     int
		wantLastLabel bool
	}{
		{name: "label exists", wantCalls: 1, wantLastLabel: true},
		{name: "missing label retries without it", failOnLabel: true, wantCalls: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			binDir := t.TempDir()
			record := t.TempDir()
			failFlag := "0"
			if tt.failOnLabel {
				failFlag = "1"
			}
			// $((...)) normalises the count: BSD wc (macOS) left-pads it
			// with spaces, which would otherwise name the files "args   0".
			script := `#!/bin/sh
n=$(($(ls "` + record + `" | wc -l)))
echo "$@" > "` + record + `/args$n"
cat > "` + record + `/stdin$n"
case "$*" in
  *--label*) if [ "` + failFlag + `" = 1 ]; then echo "could not add label: 'bug' not found" >&2; exit 1; fi ;;
esac
exit 0
`
			if err := os.WriteFile(filepath.Join(binDir, "gh"), []byte(script), 0o755); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

			body := strings.Repeat("large report body ", 20000)
			if err := SubmitViaGH("A title long enough", body); err != nil {
				t.Fatalf("SubmitViaGH: %v", err)
			}

			matches, _ := filepath.Glob(filepath.Join(record, "args*"))
			if len(matches) != tt.wantCalls {
				t.Fatalf("gh called %d time(s), want %d", len(matches), tt.wantCalls)
			}
			last := tt.wantCalls - 1
			args, _ := os.ReadFile(filepath.Join(record, fmt.Sprintf("args%d", last*2)))
			if !strings.Contains(string(args), "--body-file -") {
				t.Errorf("body must be passed on stdin, args: %s", args)
			}
			if strings.Contains(string(args), "large report body") {
				t.Errorf("body leaked into argv")
			}
			if got := strings.Contains(string(args), "--label"); got != tt.wantLastLabel {
				t.Errorf("last call has --label = %v, want %v", got, tt.wantLastLabel)
			}
			stdin, _ := os.ReadFile(filepath.Join(record, fmt.Sprintf("stdin%d", last*2)))
			if string(stdin) != body {
				t.Errorf("gh received %d body bytes on stdin, want %d", len(stdin), len(body))
			}
		})
	}
}

// writeLog writes a session log for command in dir with its mtime set to at,
// followed by one INFO record per extra attribute fragment.
func writeLog(t *testing.T, dir, id, command string, at time.Time, extra ...string) {
	t.Helper()
	ts := at.Format(time.RFC3339Nano)
	lines := []string{
		fmt.Sprintf(`{"time":%q,"level":"INFO","msg":"session started","session":%q}`, ts, id),
		fmt.Sprintf(`{"time":%q,"level":"INFO","msg":"command starting","session":%q,"command":%q}`, ts, id, command),
	}
	for _, e := range extra {
		lines = append(lines, fmt.Sprintf(`{"time":%q,"level":"INFO",%s}`, ts, e))
	}
	path := filepath.Join(dir, "qsdev-"+at.Format("2006-01-02T15-04-05")+"-"+id+".jsonl")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, at, at); err != nil {
		t.Fatal(err)
	}
}
