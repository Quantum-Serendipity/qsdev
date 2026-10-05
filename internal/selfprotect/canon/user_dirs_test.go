package canon

import (
	"path/filepath"
	"sync"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/projectctx"
)

// TestIsProtected_UserDirs pins that the per-user state and cache directories
// (projectctx.UserDirs) are protected wherever they resolve, with the XDG
// variables set and unset: the global session logs, including the automated
// sub-tier that records hook invocations, as audit, and the rest as config.
// They moved out of ~/.<app>, which the segment tables guard, so without these
// entries an agent could forge or delete the hook logs.
//
// It is sequential because it re-runs ensureInit with a different
// environment; the parallel tests resume only after its cleanup.
func TestIsProtected_UserDirs(t *testing.T) {
	resetInit := func() {
		initOnce = sync.Once{}
		initErr = nil
		protectedPrefixes = nil
		protectedSuffixes = nil
	}
	t.Cleanup(resetInit)

	for _, xdg := range []bool{true, false} {
		name := "xdg-unset"
		if xdg {
			name = "xdg-set"
		}
		t.Run(name, func(t *testing.T) {
			base := t.TempDir()
			home := filepath.Join(base, "home")
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			t.Setenv("LocalAppData", filepath.Join(base, "localappdata"))
			if xdg {
				t.Setenv("XDG_STATE_HOME", filepath.Join(base, "xdg-state"))
				t.Setenv("XDG_CACHE_HOME", filepath.Join(base, "xdg-cache"))
			} else {
				t.Setenv("XDG_STATE_HOME", "")
				t.Setenv("XDG_CACHE_HOME", "")
			}
			resetInit()
			dirs, err := projectctx.UserDirs()
			if err != nil {
				t.Fatalf("UserDirs: %v", err)
			}

			tests := []struct {
				path, wantCategory string
			}{
				{filepath.Join(dirs.Logs(), "qsdev-x.jsonl"), "audit"},
				{filepath.Join(dirs.Logs(), "automated", "hook.jsonl"), "audit"},
				{filepath.Join(dirs.State, "bugreports", "draft.md"), "config"},
				{filepath.Join(dirs.Cache, "update-check.json"), "config"},
				{filepath.Join(filepath.Dir(dirs.State), "other-app", "x"), ""},
			}
			for _, tt := range tests {
				canonical, err := Canonicalize(tt.path)
				if err != nil {
					t.Fatalf("Canonicalize(%q): %v", tt.path, err)
				}
				got, category := IsProtected(canonical)
				if got != (tt.wantCategory != "") || category != tt.wantCategory {
					t.Errorf("IsProtected(%q) = (%v, %q), want category %q", canonical, got, category, tt.wantCategory)
				}
			}
		})
	}
}
