package selfupdate

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// useTestConfig sets the default config override to use a temp directory,
// preventing tests from polluting the real ~/.qsdev/ cache.
func useTestConfig(t *testing.T) {
	t.Helper()
	cfg := DefaultConfig()
	cfg.CacheDir = t.TempDir()
	testConfigOverride = &cfg
	t.Cleanup(func() { testConfigOverride = nil })
}

func TestBackgroundCheck_Suppressed(t *testing.T) {
	t.Setenv("QSDEV_NO_UPDATE_CHECK", "1")

	ch := BackgroundCheck("1.0.0")
	if ch != nil {
		t.Error("expected nil channel when QSDEV_NO_UPDATE_CHECK=1")
	}
}

func TestBackgroundCheck_NotSuppressed(t *testing.T) {
	t.Setenv("QSDEV_NO_UPDATE_CHECK", "")
	useTestConfig(t)

	gh := githubRelease{
		TagName: "v2.0.0",
		HTMLURL: "https://github.com/test/releases/v2.0.0",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(gh)
	}))
	defer srv.Close()

	oldBase := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = oldBase }()

	ch := BackgroundCheck("1.0.0")
	if ch == nil {
		t.Fatal("expected non-nil channel")
	}

	// Wait for result with timeout.
	select {
	case notice := <-ch:
		if notice == "" {
			t.Error("expected non-empty notice for available update")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for background check")
	}
}

func TestBackgroundCheck_NoUpdate(t *testing.T) {
	t.Setenv("QSDEV_NO_UPDATE_CHECK", "")
	useTestConfig(t)

	gh := githubRelease{
		TagName: "v1.0.0",
		HTMLURL: "https://github.com/test/releases/v1.0.0",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(gh)
	}))
	defer srv.Close()

	oldBase := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = oldBase }()

	ch := BackgroundCheck("1.0.0")
	if ch == nil {
		t.Fatal("expected non-nil channel")
	}

	// Channel should close without sending a notice.
	select {
	case notice, ok := <-ch:
		if ok && notice != "" {
			t.Errorf("expected no notice when up to date, got %q", notice)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for background check to complete")
	}
}

// capturePrintNotice runs PrintNotice(ch) with os.Stderr redirected to a
// pipe and returns what it wrote.
func capturePrintNotice(t *testing.T, ch <-chan string) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	orig := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = orig }()

	done := make(chan struct{})
	go func() {
		defer close(done)
		PrintNotice(ch)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("PrintNotice blocked")
	}

	if err := w.Close(); err != nil {
		t.Fatalf("closing pipe writer: %v", err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading captured stderr: %v", err)
	}
	return string(out)
}

func TestPrintNotice(t *testing.T) {
	withNotice := make(chan string, 1)
	withNotice <- "Update available!"
	closed := make(chan string)
	close(closed)

	tests := []struct {
		name string
		ch   <-chan string
		want string
	}{
		{"nil channel", nil, ""},
		{"empty channel", make(chan string, 1), ""},
		{"closed channel", closed, ""},
		{"with notice", withNotice, "\nUpdate available!\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := capturePrintNotice(t, tt.ch); got != tt.want {
				t.Errorf("PrintNotice wrote %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBackgroundCheck_DevVersion(t *testing.T) {
	t.Setenv("QSDEV_NO_UPDATE_CHECK", "")

	ch := BackgroundCheck("dev")
	if ch == nil {
		t.Fatal("expected non-nil channel")
	}

	// Should close without a notice because dev versions skip the check.
	select {
	case notice, ok := <-ch:
		if ok && notice != "" {
			t.Errorf("expected no notice for dev version, got %q", notice)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timed out")
	}
}
