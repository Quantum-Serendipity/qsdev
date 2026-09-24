package versionsentinel

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/answers"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// handlers returns the module's handlers by tool name.
func handlers(projectRoot string) map[string]spi.ToolHandler {
	out := make(map[string]spi.ToolHandler)
	for _, r := range Tools(projectRoot) {
		out[r.Name] = r.Handler
	}
	return out
}

func call(t *testing.T, h spi.ToolHandler, args map[string]any) *spi.ToolResult {
	t.Helper()
	res, err := h(context.Background(), &spi.ToolCallContext{}, &spi.ToolRequest{Arguments: args})
	if err != nil {
		t.Fatalf("handler returned a Go error: %v", err)
	}
	if res == nil {
		t.Fatal("handler returned a nil result")
	}
	return res
}

// TestToolsConfinePaths proves the version-sentinel tools only read inside the
// project root: project_root and log_path arguments that escape it (by "..",
// an absolute path elsewhere, or a symlink) are denied, while paths inside it
// still work.
func TestToolsConfinePaths(t *testing.T) {
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
	writeFiles(t, outside, map[string]string{"go.mod": goMod, "events.jsonl": "{}\n"})
	writeFiles(t, filepath.Join(root, "sub"), map[string]string{"go.mod": goMod})
	writeFiles(t, filepath.Join(root, ".version-sentinel"), map[string]string{"events.jsonl": ""})
	// Symlink creation needs elevated privileges on Windows, so the symlink
	// escape case only runs elsewhere.
	canSymlink := runtime.GOOS != "windows"
	if canSymlink {
		if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
			t.Fatal(err)
		}
	}

	hs := handlers(root)
	tests := []struct {
		name         string
		tool         string
		args         map[string]any
		wantDenied   bool
		wantLeak     bool
		needsSymlink bool
	}{
		{name: "check default root", tool: "check_versions", args: map[string]any{}},
		{name: "check subdirectory", tool: "check_versions", args: map[string]any{"project_root": "sub"}, wantLeak: true},
		{name: "check absolute outside", tool: "check_versions", args: map[string]any{"project_root": outside}, wantDenied: true},
		{name: "check dot-dot escape", tool: "check_versions", args: map[string]any{"project_root": "../other-private-repo"}, wantDenied: true},
		{name: "check symlink escape", tool: "check_versions", args: map[string]any{"project_root": "escape"}, wantDenied: true, needsSymlink: true},
		{name: "drift absolute outside", tool: "detect_drift", args: map[string]any{"project_root": outside}, wantDenied: true},
		{name: "drift subdirectory", tool: "detect_drift", args: map[string]any{"project_root": filepath.Join(root, "sub")}},
		{name: "history default", tool: "version_history", args: map[string]any{}},
		{name: "history outside log", tool: "version_history", args: map[string]any{"log_path": filepath.Join(outside, "events.jsonl")}, wantDenied: true},
		{name: "history dot-dot log", tool: "version_history", args: map[string]any{"log_path": "../other-private-repo/events.jsonl"}, wantDenied: true},
		{name: "history in-root log", tool: "version_history", args: map[string]any{"log_path": ".version-sentinel/events.jsonl"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.needsSymlink && !canSymlink {
				t.Skip("symlink creation requires privilege on Windows")
			}
			res := call(t, hs[tt.tool], tt.args)
			if tt.wantDenied {
				if !res.IsError || !strings.HasPrefix(res.Text, "denied:") || !strings.Contains(res.Text, "outside the project root") {
					t.Fatalf("result = %q, want an outside-the-project-root denial", res.Text)
				}
				return
			}
			if res.IsError {
				t.Fatalf("unexpected error result: %s", res.Text)
			}
			if strings.Contains(res.Text, "secret-dep") != tt.wantLeak {
				t.Errorf("output mentions secret-dep = %v, want %v: %s", !tt.wantLeak, tt.wantLeak, res.Text)
			}
		})
	}
}

// TestManifestCoverage proves coverage is computed from the project's primary
// answers file, and that a project without one is reported as not configured
// instead of as having no manifests.
func TestManifestCoverage(t *testing.T) {
	t.Parallel()

	uninitialized := t.TempDir()
	res := call(t, handlers(uninitialized)["manifest_coverage"], nil)
	if !res.IsError || !strings.HasPrefix(res.Text, "not_configured:") {
		t.Errorf("uninitialized project: result = %q, want not_configured", res.Text)
	}

	project := t.TempDir()
	if err := answers.SavePrimary(project, types.WizardAnswers{Languages: []types.LanguageChoice{{Name: "unregistered-language"}}}); err != nil {
		t.Fatal(err)
	}
	res = call(t, handlers(project)["manifest_coverage"], nil)
	if res.IsError {
		t.Fatalf("initialized project: unexpected error %q", res.Text)
	}
	if _, ok := res.Structured.(ecosystem.ManifestCoverageReport); !ok {
		t.Errorf("structured result = %T, want ecosystem.ManifestCoverageReport", res.Structured)
	}
}
