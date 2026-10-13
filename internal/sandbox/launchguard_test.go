package sandbox

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func newTestGuard(t *testing.T) *LaunchGuard {
	t.Helper()
	g, err := NewLaunchGuard()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = g.Close() })
	return g
}

func TestLaunchGuard_Started(t *testing.T) {
	t.Parallel()
	g := newTestGuard(t)
	if _, err := g.ChildFile().Write([]byte{'R'}); err != nil {
		t.Fatal(err)
	}
	if !g.Started() {
		t.Error("Started = false after the ready byte was written")
	}
}

func TestLaunchGuard_NotStarted(t *testing.T) {
	t.Parallel()
	g := newTestGuard(t)
	if g.Started() {
		t.Error("Started = true with nothing written")
	}
	if g.Started() {
		t.Error("second Started = true with nothing written")
	}
}

func TestLaunchGuard_CloseWithoutStarted(t *testing.T) {
	t.Parallel()
	g, err := NewLaunchGuard()
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

// TestLaunchGuard_AppendChildFile: the ready fd's number follows from the
// files before it (cmd.ExtraFiles start at fd 3), so it is never hard-coded.
func TestLaunchGuard_AppendChildFile(t *testing.T) {
	t.Parallel()
	other, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = other.Close() })

	tests := []struct {
		name   string
		before []*os.File
		wantFD int
	}{
		{name: "no other files", before: nil, wantFD: 3},
		{name: "after one file", before: []*os.File{other}, wantFD: 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := newTestGuard(t)
			files, fd := g.AppendChildFile(tt.before)
			if fd != tt.wantFD {
				t.Errorf("fd = %d, want %d", fd, tt.wantFD)
			}
			if len(files) != len(tt.before)+1 || files[len(files)-1] != g.ChildFile() {
				t.Errorf("files = %v, want %v followed by the child file", files, tt.before)
			}
		})
	}
}

// TestLaunchGuard_Err: a started guard yields no error and never computes the
// hint; a guard that did not start yields ErrSetupFailed naming the exit
// code, the first stderr line and the hint.
func TestLaunchGuard_Err(t *testing.T) {
	t.Parallel()

	t.Run("started", func(t *testing.T) {
		t.Parallel()
		g := newTestGuard(t)
		if _, err := g.ChildFile().Write([]byte{'R'}); err != nil {
			t.Fatal(err)
		}
		hint := func() string { t.Error("hint computed for a started guard"); return "" }
		if err := g.Err(1, hint); err != nil {
			t.Errorf("Err = %v, want nil", err)
		}
	})

	t.Run("not started", func(t *testing.T) {
		t.Parallel()
		g := newTestGuard(t)
		_, _ = g.StderrTap().Write([]byte("bwrap: Can't find source path /nope\nsecond line\n"))
		err := g.Err(1, func() string { return "rebuild it" })
		if !errors.Is(err, ErrSetupFailed) {
			t.Fatalf("Err = %v, want ErrSetupFailed", err)
		}
		msg := err.Error()
		for _, want := range []string{"exit 1", "bwrap: Can't find source path /nope", "rebuild it"} {
			if !strings.Contains(msg, want) {
				t.Errorf("Err = %q, missing %q", msg, want)
			}
		}
		if strings.Contains(msg, "second line") {
			t.Errorf("Err = %q, want only the first stderr line", msg)
		}
	})

	t.Run("not started without hint", func(t *testing.T) {
		t.Parallel()
		g := newTestGuard(t)
		err := g.Err(5, func() string { return "" })
		if !errors.Is(err, ErrSetupFailed) || strings.HasSuffix(err.Error(), "; ") {
			t.Errorf("Err = %v, want ErrSetupFailed without an empty hint", err)
		}
	})
}

func TestHeadWriter(t *testing.T) {
	t.Parallel()

	h := &headWriter{limit: 8}
	for _, chunk := range []string{"abc", "defgh", "ijk"} {
		n, err := h.Write([]byte(chunk))
		if err != nil || n != len(chunk) {
			t.Fatalf("Write(%q) = (%d, %v), want (%d, nil)", chunk, n, err, len(chunk))
		}
	}
	if got := string(h.buf); got != "abcdefgh" {
		t.Errorf("retained %q, want the first 8 bytes %q", got, "abcdefgh")
	}
}

func TestFirstLine(t *testing.T) {
	t.Parallel()

	if got := FirstLine([]byte("ll-restrict: boom\nmore\n")); got != "ll-restrict: boom" {
		t.Errorf("FirstLine = %q", got)
	}
	if got := FirstLine([]byte(strings.Repeat("x", 3))); got != "xxx" {
		t.Errorf("FirstLine without newline = %q", got)
	}
}
