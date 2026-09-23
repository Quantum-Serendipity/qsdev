package capture

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestNew_TeesBothStreamsToOneCapture(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "logs", "capture")
	var stdout, stderr bytes.Buffer
	w, err := New(&stdout, dir, "devenv")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := w.Write([]byte("out line\n")); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Tee(&stderr).Write([]byte("err line\n")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	if stdout.String() != "out line\n" || stderr.String() != "err line\n" {
		t.Errorf("original writers got %q / %q", stdout.String(), stderr.String())
	}
	data, err := os.ReadFile(w.Path())
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "out line\nerr line\n" {
		t.Errorf("capture file = %q, want both streams", data)
	}
	if !strings.HasPrefix(filepath.Base(w.Path()), "devenv-") {
		t.Errorf("capture file %q lacks the provider prefix its log provider discovers", w.Path())
	}

	if runtime.GOOS != "windows" {
		for path, want := range map[string]os.FileMode{dir: modeDirPrivate, w.Path(): 0o600} {
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			if got := info.Mode().Perm(); got != want {
				t.Errorf("%s mode = %o, want %o", path, got, want)
			}
		}
	}
}

func TestNew_RejectsMissingDir(t *testing.T) {
	t.Parallel()
	if _, err := New(&bytes.Buffer{}, "", "nix"); err == nil {
		t.Fatal("New with no capture dir succeeded")
	}
}

func TestCaptureDir_NoTempFallback(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	t.Setenv("home", "")
	if got := CaptureDir(""); got != "" {
		t.Errorf("CaptureDir without project or home = %q, want empty", got)
	}
	if got := CaptureDir("/proj"); got != filepath.Join("/proj", ".qsdev", "logs", "capture") {
		t.Errorf("CaptureDir(project) = %q", got)
	}
}

func TestNew_PrunesOldCapturesPerProvider(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	old := time.Now().Add(-maxCaptureAge - time.Hour)
	for i := 0; i < maxCapturesPerProvider+5; i++ {
		path := filepath.Join(dir, fmt.Sprintf("nix-old-%02d.log", i))
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		// Recent, staggered captures: all within the age limit.
		mt := time.Now().Add(-time.Duration(i+1) * time.Minute)
		if err := os.Chtimes(path, mt, mt); err != nil {
			t.Fatal(err)
		}
	}
	stale := filepath.Join(dir, "devenv-stale.log")
	other := filepath.Join(dir, "devenv-fresh.log")
	for _, p := range []string{stale, other} {
		if err := os.WriteFile(p, nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chtimes(stale, old, old); err != nil {
		t.Fatal(err)
	}

	w, err := New(&bytes.Buffer{}, dir, "nix")
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	nix, _ := filepath.Glob(filepath.Join(dir, "nix-*.log"))
	if len(nix) != maxCapturesPerProvider {
		t.Errorf("kept %d nix captures, want %d", len(nix), maxCapturesPerProvider)
	}
	if _, err := os.Stat(w.Path()); err != nil {
		t.Errorf("new capture was pruned: %v", err)
	}
	if _, err := os.Stat(other); err != nil {
		t.Errorf("another provider's capture was pruned: %v", err)
	}

	// The devenv provider is pruned by age on its own next capture.
	w2, err := New(&bytes.Buffer{}, dir, "devenv")
	if err != nil {
		t.Fatal(err)
	}
	defer w2.Close()
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("stale devenv capture survived: %v", err)
	}
}

// TestWrite_CaptureFailureDoesNotFailCommand proves a failing capture file
// never turns into a write error of the captured command's own output.
func TestWrite_CaptureFailureDoesNotFailCommand(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	w, err := New(&stdout, filepath.Join(t.TempDir(), "capture"), "devenv")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// Closing the file underneath the writer makes every capture write fail.
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	if n, err := w.Write([]byte("out\n")); err != nil || n != 4 {
		t.Errorf("Write = %d, %v; want 4, nil", n, err)
	}
	if n, err := w.Tee(&stderr).Write([]byte("err\n")); err != nil || n != 4 {
		t.Errorf("Tee Write = %d, %v; want 4, nil", n, err)
	}
	if stdout.String() != "out\n" || stderr.String() != "err\n" {
		t.Errorf("original writers got %q / %q", stdout.String(), stderr.String())
	}
}
