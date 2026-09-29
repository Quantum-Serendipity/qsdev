package selfupdate

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestTruncateChangelog(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		maxLines int
		wantFull bool // if true, output should equal input
	}{
		{
			name:     "short changelog",
			body:     "Line 1\nLine 2\nLine 3",
			maxLines: 5,
			wantFull: true,
		},
		{
			name:     "exact limit",
			body:     "Line 1\nLine 2\nLine 3",
			maxLines: 3,
			wantFull: true,
		},
		{
			name:     "truncated",
			body:     "Line 1\nLine 2\nLine 3\nLine 4\nLine 5",
			maxLines: 2,
			wantFull: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateChangelog(tt.body, tt.maxLines)
			if tt.wantFull && got != tt.body {
				t.Errorf("truncateChangelog() = %q, want %q", got, tt.body)
			}
			if !tt.wantFull && got == tt.body {
				t.Error("expected truncated output, got full body")
			}
		})
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"single line", 1},
		{"line1\nline2\nline3", 3},
		{"line1\nline2\n", 2},
	}

	for _, tt := range tests {
		got := splitLines(tt.input)
		if len(got) != tt.want {
			t.Errorf("splitLines(%q) = %d lines, want %d", tt.input, len(got), tt.want)
		}
	}
}

func TestVerifyBinary_InvalidBinary(t *testing.T) {
	tmpDir := t.TempDir()
	badBinary := filepath.Join(tmpDir, "bad")
	if err := os.WriteFile(badBinary, []byte("not executable"), 0o755); err != nil {
		t.Fatal(err)
	}

	err := verifyBinary(context.Background(), badBinary)
	if err == nil {
		t.Error("expected error for invalid binary")
	}
}

// skipIfNoShell skips tests whose fake binaries are /bin/sh scripts.
func skipIfNoShell(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake binaries are shell scripts")
	}
}

// writeScript writes an executable shell script and returns its path.
func writeScript(t *testing.T, path, body string) string {
	t.Helper()
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// assertFileContent fails unless path holds want.
func assertFileContent(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	if string(got) != want {
		t.Errorf("%s content = %q, want %q", path, got, want)
	}
}

// assertOnlyEntries fails unless dir contains exactly the named entries, so a
// leaked staging temp file or .bak is caught.
func assertOnlyEntries(t *testing.T, dir string, want ...string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, e := range entries {
		got = append(got, e.Name())
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("dir entries = %v, want %v", got, want)
	}
}

func TestReplaceBinary(t *testing.T) {
	skipIfNoShell(t)

	const oldBody = "#!/bin/sh\necho old\n"
	const newBody = "#!/bin/sh\necho new\n"
	errVerify := errors.New("binary does not run")

	tests := []struct {
		name        string
		missingNew  bool
		verifyErr   error
		wantErr     bool
		wantContent string
	}{
		{name: "success replaces atomically", wantContent: newBody},
		{name: "verify failure leaves current untouched", verifyErr: errVerify, wantErr: true, wantContent: oldBody},
		{name: "staging failure leaves current untouched", missingNew: true, wantErr: true, wantContent: oldBody},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			installDir := t.TempDir()
			current := filepath.Join(installDir, "qsdev")
			if err := os.WriteFile(current, []byte(oldBody), 0o750); err != nil {
				t.Fatal(err)
			}
			newBin := filepath.Join(t.TempDir(), "qsdev")
			if !tt.missingNew {
				if err := os.WriteFile(newBin, []byte(newBody), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			var verified string
			verify := func(_ context.Context, path string) error {
				verified = path
				// The live binary must not have been touched yet.
				assertFileContent(t, current, oldBody)
				return tt.verifyErr
			}

			err := replaceBinary(context.Background(), current, newBin, 0o750, verify)
			if (err != nil) != tt.wantErr {
				t.Fatalf("replaceBinary() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.verifyErr != nil && !errors.Is(err, tt.verifyErr) {
				t.Errorf("error %v does not wrap the verify error", err)
			}
			if !tt.missingNew && (verified == "" || verified == current) {
				t.Errorf("verify ran on %q, want the staged file (not the live path)", verified)
			}
			assertFileContent(t, current, tt.wantContent)
			assertOnlyEntries(t, installDir, "qsdev")

			info, err := os.Stat(current)
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm() != 0o750 {
				t.Errorf("mode = %v, want 0750", info.Mode().Perm())
			}
		})
	}
}

func TestSwapBinary_ViaBackup(t *testing.T) {
	t.Parallel()

	t.Run("success removes backup", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		current := filepath.Join(dir, "qsdev")
		staged := filepath.Join(dir, ".qsdev.new-1")
		for p, body := range map[string]string{current: "old", staged: "new", current + ".bak": "stale"} {
			if err := os.WriteFile(p, []byte(body), 0o755); err != nil {
				t.Fatal(err)
			}
		}
		if err := swapBinary(staged, current, true); err != nil {
			t.Fatalf("swapBinary() error: %v", err)
		}
		assertFileContent(t, current, "new")
		assertOnlyEntries(t, dir, "qsdev")
	})

	t.Run("failed install restores backup", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		current := filepath.Join(dir, "qsdev")
		if err := os.WriteFile(current, []byte("old"), 0o755); err != nil {
			t.Fatal(err)
		}
		missingStaged := filepath.Join(dir, ".qsdev.new-missing")
		if err := swapBinary(missingStaged, current, true); err == nil {
			t.Fatal("expected error when the staged file cannot be renamed")
		}
		assertFileContent(t, current, "old")
		assertOnlyEntries(t, dir, "qsdev")
	})
}

// newReleaseServer serves a GitHub-like API (/repos/o/r/releases/latest) whose
// release assets (archive + checksums.txt for the running platform) are
// served by the same server. The binary inside the archive is binaryBody.
func newReleaseServer(t *testing.T, version, binaryBody string) *httptest.Server {
	t.Helper()
	archiveName := ArchiveFilename("qsdev", version, runtime.GOOS, runtime.GOARCH)
	archivePath := createTestTarGz(t, t.TempDir(), "qsdev", binaryBody)
	archiveData, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	checksums := fmt.Sprintf("%s  %s\n", sha256sum(t, archivePath), archiveName)

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/test-owner/test-repo/releases/latest":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"tag_name":"v%s","html_url":"https://example.com/r","body":"notes","assets":[`+
				`{"name":%q,"browser_download_url":%q},{"name":"checksums.txt","browser_download_url":%q}]}`,
				version, archiveName, srv.URL+"/archive", srv.URL+"/checksums")
		case "/archive":
			_, _ = w.Write(archiveData)
		case "/checksums":
			_, _ = w.Write([]byte(checksums))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// useScratchExecutable points DoUpdate at a scratch "current" binary.
func useScratchExecutable(t *testing.T, path string) {
	t.Helper()
	old := executablePath
	executablePath = func() (string, error) { return path, nil }
	t.Cleanup(func() { executablePath = old })
}

// TestDoUpdate_AfterCachedCheck is the regression for the cached-check bug:
// once a check has been cached, CheckForUpdate must still hand DoUpdate a
// release it can install (assets, tag), not a Version/URL-only stub.
func TestDoUpdate_AfterCachedCheck(t *testing.T) {
	skipIfNoShell(t)

	const newBody = "#!/bin/sh\nexit 0\n"
	srv := newReleaseServer(t, "2.0.0", newBody)
	oldBase := apiBaseURL
	apiBaseURL = srv.URL
	t.Cleanup(func() { apiBaseURL = oldBase })

	installDir := t.TempDir()
	current := writeScript(t, filepath.Join(installDir, "qsdev"), "echo old")
	useScratchExecutable(t, current)

	cfg := testConfig(t)
	for i := range 2 { // second call is served from the cache
		release, err := CheckForUpdate(context.Background(), cfg, "1.0.0")
		if err != nil || release == nil {
			t.Fatalf("CheckForUpdate #%d = %v, %v; want a release", i+1, release, err)
		}
		if i == 0 {
			continue
		}
		if release.TagName != "v2.0.0" || len(release.Assets) != 2 || release.Body == "" {
			t.Fatalf("cached CheckForUpdate returned an incomplete release: %+v", release)
		}
		if err := DoUpdate(context.Background(), cfg, release); err != nil {
			t.Fatalf("DoUpdate() after cached check: %v", err)
		}
	}
	assertFileContent(t, current, newBody)
	assertOnlyEntries(t, installDir, "qsdev")
}

func TestDoUpdate_VerifyFailureKeepsCurrent(t *testing.T) {
	skipIfNoShell(t)

	// The new "binary" exits non-zero, so the pre-swap test run fails.
	srv := newReleaseServer(t, "2.0.0", "#!/bin/sh\nexit 3\n")
	oldBase := apiBaseURL
	apiBaseURL = srv.URL
	t.Cleanup(func() { apiBaseURL = oldBase })

	installDir := t.TempDir()
	current := writeScript(t, filepath.Join(installDir, "qsdev"), "echo old")
	useScratchExecutable(t, current)

	cfg := testConfig(t)
	release, err := FetchLatestRelease(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	err = DoUpdate(context.Background(), cfg, release)
	if err == nil || !strings.Contains(err.Error(), "verification failed") {
		t.Fatalf("DoUpdate() error = %v, want a verification failure", err)
	}
	assertFileContent(t, current, "#!/bin/sh\necho old\n")
	assertOnlyEntries(t, installDir, "qsdev")
}
