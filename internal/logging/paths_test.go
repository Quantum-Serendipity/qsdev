package logging

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/projectctx"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// TestGlobalLogDir_UsesXDGState pins that the global log tier lives in the
// per-user state directory: $XDG_STATE_HOME/<app>/logs when that is set, the
// OS default state directory otherwise, and never the legacy ~/.<app>.
func TestGlobalLogDir_UsesXDGState(t *testing.T) {
	b := branding.Get()
	base := t.TempDir()
	home := filepath.Join(base, "home")
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("LocalAppData", filepath.Join(base, "LocalAppData"))
	t.Setenv(b.EnvLogDirVar, "")

	t.Run("xdg-set", func(t *testing.T) {
		state := filepath.Join(base, "xdg-state")
		t.Setenv("XDG_STATE_HOME", state)
		if got, want := GlobalLogDir(), filepath.Join(state, b.AppName, "logs"); got != want {
			t.Errorf("GlobalLogDir() = %q, want %q", got, want)
		}
	})
	t.Run("xdg-unset", func(t *testing.T) {
		t.Setenv("XDG_STATE_HOME", "")
		dirs, err := projectctx.UserDirs()
		if err != nil {
			t.Fatalf("UserDirs: %v", err)
		}
		got := GlobalLogDir()
		if want := filepath.Join(dirs.State, "logs"); got != want {
			t.Errorf("GlobalLogDir() = %q, want %q", got, want)
		}
		if strings.HasPrefix(got, dirs.Legacy+string(filepath.Separator)) {
			t.Errorf("GlobalLogDir() = %q lies under the legacy %s", got, dirs.Legacy)
		}
	})
	t.Run("override-wins", func(t *testing.T) {
		override := filepath.Join(base, "override")
		t.Setenv(b.EnvLogDirVar, override)
		if got := GlobalLogDir(); got != override {
			t.Errorf("GlobalLogDir() = %q, want the override %q", got, override)
		}
	})
}
