package selfupdate

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// clearUpdateCache removes the cached update check file so tests don't
// interfere with each other.
func clearUpdateCache(t *testing.T) {
	t.Helper()
	cfg := DefaultConfig()
	_ = os.Remove(cacheFile(cfg))
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
	clearUpdateCache(t)

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
	clearUpdateCache(t)

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

func TestPrintNotice_NilChannel(t *testing.T) {
	// Should not panic.
	PrintNotice(nil)
}

func TestPrintNotice_EmptyChannel(t *testing.T) {
	ch := make(chan string, 1)
	// Don't send anything — PrintNotice should return immediately.
	PrintNotice(ch)
}

func TestPrintNotice_WithNotice(t *testing.T) {
	ch := make(chan string, 1)
	ch <- "Update available!"

	// Should not panic or block.
	PrintNotice(ch)
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
