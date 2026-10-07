package projectctx

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

func TestUserDirs(t *testing.T) {
	t.Parallel()
	app := branding.Get().AppName
	// Absolute on the host OS, so filepath.IsAbs accepts them whatever goos
	// the pure function is asked to model.
	base := t.TempDir()
	home := filepath.Join(base, "home")
	xdgState := filepath.Join(base, "xdg-state")
	xdgCache := filepath.Join(base, "xdg-cache")
	localAppData := filepath.Join(base, "LocalAppData")

	tests := []struct {
		name      string
		goos      string
		env       map[string]string
		wantState string
		wantCache string
		wantErr   bool
	}{
		{
			name:      "linux-defaults",
			goos:      "linux",
			wantState: filepath.Join(home, ".local", "state", app),
			wantCache: filepath.Join(home, ".cache", app),
		},
		{
			name:      "freebsd-defaults",
			goos:      "freebsd",
			wantState: filepath.Join(home, ".local", "state", app),
			wantCache: filepath.Join(home, ".cache", app),
		},
		{
			name:      "linux-xdg-set",
			goos:      "linux",
			env:       map[string]string{"XDG_STATE_HOME": xdgState, "XDG_CACHE_HOME": xdgCache},
			wantState: filepath.Join(xdgState, app),
			wantCache: filepath.Join(xdgCache, app),
		},
		{
			name:      "linux-xdg-relative-ignored",
			goos:      "linux",
			env:       map[string]string{"XDG_STATE_HOME": "rel/state", "XDG_CACHE_HOME": "rel/cache"},
			wantState: filepath.Join(home, ".local", "state", app),
			wantCache: filepath.Join(home, ".cache", app),
		},
		{
			name:      "darwin-defaults",
			goos:      "darwin",
			wantState: filepath.Join(home, "Library", "Application Support", app),
			wantCache: filepath.Join(home, "Library", "Caches", app),
		},
		{
			name:      "darwin-xdg-set",
			goos:      "darwin",
			env:       map[string]string{"XDG_STATE_HOME": xdgState, "XDG_CACHE_HOME": xdgCache},
			wantState: filepath.Join(xdgState, app),
			wantCache: filepath.Join(xdgCache, app),
		},
		{
			name:      "windows-defaults",
			goos:      "windows",
			env:       map[string]string{"LocalAppData": localAppData},
			wantState: filepath.Join(localAppData, app),
			wantCache: filepath.Join(localAppData, app),
		},
		{
			name:      "windows-xdg-set",
			goos:      "windows",
			env:       map[string]string{"LocalAppData": localAppData, "XDG_STATE_HOME": xdgState, "XDG_CACHE_HOME": xdgCache},
			wantState: filepath.Join(xdgState, app),
			wantCache: filepath.Join(xdgCache, app),
		},
		{
			name:    "windows-missing-localappdata",
			goos:    "windows",
			wantErr: true,
		},
		{
			name:    "windows-relative-localappdata",
			goos:    "windows",
			env:     map[string]string{"LocalAppData": "rel"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			getenv := func(k string) string { return tt.env[k] }
			got, err := userDirs(tt.goos, getenv, home)
			if tt.wantErr {
				if !errors.Is(err, ErrNoUserDir) {
					t.Fatalf("userDirs error = %v, want ErrNoUserDir", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("userDirs: %v", err)
			}
			want := Dirs{Home: home, State: tt.wantState, Cache: tt.wantCache, Legacy: filepath.Join(home, "."+app)}
			if got != want {
				t.Errorf("userDirs = %+v, want %+v", got, want)
			}
		})
	}
}

// TestUserDirsLive checks the process-wide entry point honours XDG overrides.
func TestUserDirsLive(t *testing.T) {
	app := branding.Get().AppName
	base := t.TempDir()
	home := filepath.Join(base, "home")
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_STATE_HOME", filepath.Join(base, "state"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(base, "cache"))

	got, err := UserDirs()
	if err != nil {
		t.Fatalf("UserDirs: %v", err)
	}
	want := Dirs{
		Home:   home,
		State:  filepath.Join(base, "state", app),
		Cache:  filepath.Join(base, "cache", app),
		Legacy: filepath.Join(home, "."+app),
	}
	if got != want {
		t.Errorf("UserDirs = %+v, want %+v", got, want)
	}
}

// TestLegacyDirNeedsOnlyHome: the legacy directory, which holds security
// state, resolves from the home directory alone, even where the state and
// cache directories cannot (Windows without %LocalAppData% or XDG_*).
func TestLegacyDirNeedsOnlyHome(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	for _, k := range []string{"LocalAppData", "XDG_STATE_HOME", "XDG_CACHE_HOME"} {
		t.Setenv(k, "")
	}
	got, err := LegacyDir()
	if err != nil {
		t.Fatalf("LegacyDir: %v", err)
	}
	if want := filepath.Join(home, "."+branding.Get().AppName); got != want {
		t.Errorf("LegacyDir = %q, want %q", got, want)
	}
	if _, err := userDirs("windows", func(string) string { return "" }, home); !errors.Is(err, ErrNoUserDir) {
		t.Fatalf("userDirs(windows, no LocalAppData) error = %v, want ErrNoUserDir", err)
	}
	if got := legacyDir(home); got != filepath.Join(home, "."+branding.Get().AppName) {
		t.Errorf("legacyDir = %q", got)
	}
}
