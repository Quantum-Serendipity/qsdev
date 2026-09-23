package vsentinel

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestMCPProviderConfinesPaths proves the version-sentinel tools only read
// inside the project root: project_root and log_path arguments that escape it
// (by "..", an absolute path elsewhere, or a symlink) are rejected, while paths
// inside it still work.
func TestMCPProviderConfinesPaths(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	root := filepath.Join(base, "project")
	outside := filepath.Join(base, "other-private-repo")
	for _, dir := range []string{filepath.Join(root, "sub"), filepath.Join(root, ".version-sentinel"), outside} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	const goMod = "module example.com/x\n\ngo 1.22\n\nrequire golang.org/x/secret-dep v1.0.0\n"
	writeFixtures(t, outside, map[string]string{"go.mod": goMod, "events.jsonl": "{}\n"})
	writeFixtures(t, filepath.Join(root, "sub"), map[string]string{"go.mod": goMod})
	writeFixtures(t, filepath.Join(root, ".version-sentinel"), map[string]string{"events.jsonl": ""})
	// Symlink creation needs elevated privileges on Windows, so the symlink
	// escape case only runs elsewhere.
	canSymlink := runtime.GOOS != "windows"
	if canSymlink {
		if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
			t.Fatal(err)
		}
	}

	p := &MCPProvider{ProjectRootFunc: func() (string, error) { return root, nil }}
	handlers := map[string]func(context.Context, map[string]any) (string, error){
		"check_versions":  p.handleCheckVersions,
		"detect_drift":    p.handleDetectDrift,
		"version_history": p.handleVersionHistory,
	}

	tests := []struct {
		name         string
		tool         string
		args         map[string]any
		wantErr      bool
		needsSymlink bool
	}{
		{"check default root", "check_versions", map[string]any{}, false, false},
		{"check subdirectory", "check_versions", map[string]any{"project_root": "sub"}, false, false},
		{"check absolute outside", "check_versions", map[string]any{"project_root": outside}, true, false},
		{"check dot-dot escape", "check_versions", map[string]any{"project_root": "../other-private-repo"}, true, false},
		{"check symlink escape", "check_versions", map[string]any{"project_root": "escape"}, true, true},
		{"drift absolute outside", "detect_drift", map[string]any{"project_root": outside}, true, false},
		{"drift subdirectory", "detect_drift", map[string]any{"project_root": filepath.Join(root, "sub")}, false, false},
		{"history default", "version_history", map[string]any{}, false, false},
		{"history outside log", "version_history", map[string]any{"log_path": filepath.Join(outside, "events.jsonl")}, true, false},
		{"history dot-dot log", "version_history", map[string]any{"log_path": "../other-private-repo/events.jsonl"}, true, false},
		{"history in-root log", "version_history", map[string]any{"log_path": ".version-sentinel/events.jsonl"}, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.needsSymlink && !canSymlink {
				t.Skip("symlink creation requires privilege on Windows")
			}
			out, err := handlers[tt.tool](context.Background(), tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got output %s", out)
				}
				if !strings.Contains(err.Error(), "outside the project root") {
					t.Errorf("error = %v, want an outside-the-project-root error", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if strings.Contains(out, "secret-dep") && !strings.Contains(tt.name, "subdirectory") {
				t.Errorf("output leaked the outside manifest: %s", out)
			}
		})
	}
}
