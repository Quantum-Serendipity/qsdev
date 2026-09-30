package claudesettings

import (
	"errors"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func writeSettings(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRead_LocalOverridesProject(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		project string
		local   string
		check   func(t *testing.T, e Effective)
	}{
		{
			name:    "local disableAllHooks",
			project: generated,
			local:   `{"disableAllHooks": true}`,
			check: func(t *testing.T, e Effective) {
				t.Helper()
				if !e.DisableAllHooks {
					t.Error("DisableAllHooks = false, want true")
				}
				if got := e.Sources[KeyDisableAllHooks]; !slices.Equal(got, []string{LocalRelPath}) {
					t.Errorf("Sources[%s] = %v, want [%s]", KeyDisableAllHooks, got, LocalRelPath)
				}
			},
		},
		{
			name:    "disableAllHooks is fail-safe: either file sets it",
			project: `{"disableAllHooks": true}`,
			local:   `{"disableAllHooks": false}`,
			check: func(t *testing.T, e Effective) {
				t.Helper()
				if !e.DisableAllHooks {
					t.Error("local false re-enabled hooks the project file disabled")
				}
				if got := e.Sources[KeyDisableAllHooks]; !slices.Equal(got, []string{ProjectRelPath}) {
					t.Errorf("Sources[%s] = %v, want [%s]", KeyDisableAllHooks, got, ProjectRelPath)
				}
			},
		},
		{
			name:    "both files disable hooks",
			project: `{"disableAllHooks": true}`,
			local:   `{"disableAllHooks": true}`,
			check: func(t *testing.T, e Effective) {
				t.Helper()
				if got := e.Sources[KeyDisableAllHooks]; !slices.Equal(got, []string{ProjectRelPath, LocalRelPath}) {
					t.Errorf("Sources[%s] = %v, want both files", KeyDisableAllHooks, got)
				}
			},
		},
		{
			name:    "local defaultMode overrides project",
			project: generated,
			local:   `{"permissions": {"defaultMode": "bypassPermissions"}}`,
			check: func(t *testing.T, e Effective) {
				t.Helper()
				if e.DefaultMode != ModeBypassPermissions {
					t.Errorf("DefaultMode = %q, want %q", e.DefaultMode, ModeBypassPermissions)
				}
				if got := e.Sources[KeyDefaultMode]; !slices.Equal(got, []string{LocalRelPath}) {
					t.Errorf("Sources[%s] = %v, want [%s]", KeyDefaultMode, got, LocalRelPath)
				}
				if e.DisableBypassPermissionsMode != "disable" {
					t.Errorf("DisableBypassPermissionsMode = %q, want the project value", e.DisableBypassPermissionsMode)
				}
				if got := e.Sources[KeyDisableBypassPermissionsMode]; !slices.Equal(got, []string{ProjectRelPath}) {
					t.Errorf("Sources[%s] = %v, want [%s]", KeyDisableBypassPermissionsMode, got, ProjectRelPath)
				}
			},
		},
		{
			name:    "hooks and deny are a union",
			project: generated,
			local: `{"permissions": {"deny": ["Read(./.env)"]}, "hooks": {"PreToolUse": [{"matcher": "Edit", "hooks": [{"type": "command", "command": "local-hook"}]}],
  "Stop": [{"hooks": [{"type": "command", "command": "stop-hook"}]}]}}`,
			check: func(t *testing.T, e Effective) {
				t.Helper()
				if !e.Registered(EventPreToolUse, "Bash", Hook{Type: "command", Command: guardCommand, If: "Bash(npm *)"}) {
					t.Error("project PreToolUse hook lost")
				}
				if !e.Registered(EventPreToolUse, "Edit", Hook{Type: "command", Command: "local-hook"}) {
					t.Error("local PreToolUse hook missing")
				}
				if !e.Registered("Stop", "", Hook{Type: "command", Command: "stop-hook"}) {
					t.Error("local-only event missing")
				}
				if want := []string{"Bash(curl *)", "Read(./.env)"}; !slices.Equal(e.Deny, want) {
					t.Errorf("Deny = %v, want %v", e.Deny, want)
				}
			},
		},
		{
			name:    "env per-key override",
			project: `{"env": {"TOOL_GATES_DENIED": "WebFetch", "KEEP": "p"}}`,
			local:   `{"env": {"TOOL_GATES_DENIED": "", "EXTRA": "l"}}`,
			check: func(t *testing.T, e Effective) {
				t.Helper()
				want := map[string]string{"TOOL_GATES_DENIED": "", "KEEP": "p", "EXTRA": "l"}
				if !maps.Equal(e.Env, want) {
					t.Errorf("Env = %v, want %v", e.Env, want)
				}
				if e.Project == nil || e.Project.Env["TOOL_GATES_DENIED"] != "WebFetch" {
					t.Error("Project view does not keep the committed value")
				}
			},
		},
		{
			name:    "no local file keeps the project view",
			project: generated,
			check: func(t *testing.T, e Effective) {
				t.Helper()
				if e.Local != nil {
					t.Errorf("Local = %+v, want nil", e.Local)
				}
				if e.DefaultMode != "default" || e.DisableAllHooks {
					t.Errorf("effective = %+v, want the project settings", e.Settings)
				}
				if got := e.Sources[KeyDefaultMode]; !slices.Equal(got, []string{ProjectRelPath}) {
					t.Errorf("Sources[%s] = %v, want [%s]", KeyDefaultMode, got, ProjectRelPath)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeSettings(t, root, ProjectRelPath, tt.project)
			if tt.local != "" {
				writeSettings(t, root, LocalRelPath, tt.local)
			}
			e, err := Read(root)
			if err != nil {
				t.Fatal(err)
			}
			tt.check(t, e)
		})
	}
}

// TestRead_DecoyKeysIgnored checks that the exact-key parse applies to the
// local file too: a decoy key there cannot switch hooks off in the view.
func TestRead_DecoyKeysIgnored(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeSettings(t, root, ProjectRelPath, generated)
	writeSettings(t, root, LocalRelPath, `{"DisableAllHooks": true, "Permissions": {"defaultMode": "bypassPermissions"}}`)
	e, err := Read(root)
	if err != nil {
		t.Fatal(err)
	}
	if e.DisableAllHooks || e.DefaultMode != "default" {
		t.Errorf("decoy keys in %s were read: %+v", LocalRelPath, e.Settings)
	}
}

func TestRead_MissingFilesAreEmpty(t *testing.T) {
	t.Parallel()
	for _, root := range []string{t.TempDir(), ""} {
		e, err := Read(root)
		if err != nil {
			t.Fatalf("Read(%q): %v", root, err)
		}
		if e.Project != nil || e.Local != nil || e.DisableAllHooks || len(e.Hooks) != 0 || len(e.Sources) != 0 {
			t.Errorf("Read(%q) = %+v, want an empty view", root, e)
		}
	}
}

func TestRead_ParseErrorNamesFile(t *testing.T) {
	t.Parallel()
	for _, rel := range []string{ProjectRelPath, LocalRelPath} {
		t.Run(rel, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeSettings(t, root, ProjectRelPath, generated)
			writeSettings(t, root, rel, `{"permissions": `)
			_, err := Read(root)
			if err == nil {
				t.Fatal("Read accepted truncated JSON")
			}
			if !strings.Contains(err.Error(), rel) {
				t.Errorf("error %q does not name %s", err, rel)
			}
			if errors.Unwrap(err) == nil {
				t.Errorf("error %q does not wrap the parse error", err)
			}
		})
	}
}
