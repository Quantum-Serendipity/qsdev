package state

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

func TestNeedsJoin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		withConfig bool
		withState  bool
		want       bool
	}{
		{name: "no config", want: false},
		{name: "no config with init state", withState: true, want: false},
		{name: "config without init state", withConfig: true, want: true},
		{name: "config with init state", withConfig: true, withState: true, want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			if tc.withConfig {
				writeJoinFixture(t, filepath.Join(root, branding.Get().ConfigFile))
			}
			if tc.withState {
				writeJoinFixture(t, filepath.Join(root, filepath.FromSlash(InitStateFile())))
			}

			got, err := NeedsJoin(root)
			if err != nil {
				t.Fatalf("NeedsJoin: %v", err)
			}
			if got != tc.want {
				t.Errorf("NeedsJoin = %v, want %v", got, tc.want)
			}
		})
	}

	t.Run("unreadable parent", func(t *testing.T) {
		t.Parallel()
		if runtime.GOOS == "windows" {
			t.Skip("directory permission bits do not block stat on windows")
		}
		if os.Geteuid() == 0 {
			t.Skip("root bypasses directory permissions")
		}
		root := filepath.Join(t.TempDir(), "locked")
		writeJoinFixture(t, filepath.Join(root, branding.Get().ConfigFile))
		if err := os.Chmod(root, 0o000); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(root, 0o755) })

		got, err := NeedsJoin(root)
		if err == nil {
			t.Fatalf("NeedsJoin = %v, nil; want a stat error", got)
		}
		if !errors.Is(err, fs.ErrPermission) {
			t.Errorf("error %v does not wrap fs.ErrPermission", err)
		}
	})
}

func writeJoinFixture(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("x: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}
