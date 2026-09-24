package outdated

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestSelectCommand proves the package manager is chosen from the project, not
// from PATH order: the configured manager wins, then a present lockfile/build
// file, and only then the fallback order. A manager the project uses but that
// is missing from PATH is reported, never silently replaced by another one.
func TestSelectCommand(t *testing.T) {
	original := lookPathFunc
	t.Cleanup(func() { lookPathFunc = original })

	tests := []struct {
		name       string
		eco        string
		files      []string
		configured string
		onPath     []string
		wantBinary string
		wantSkip   string
	}{
		{"pnpm lockfile beats npm on PATH", "javascript", []string{"pnpm-lock.yaml"}, "", []string{"npm", "pnpm"}, "pnpm", ""},
		{"configured yarn beats npm lockfile", "javascript", []string{"package-lock.json"}, "yarn", []string{"npm", "yarn"}, "yarn", ""},
		{"no markers falls back to PATH order", "javascript", nil, "", []string{"npm", "pnpm"}, "npm", ""},
		{"pnpm project without pnpm is skipped", "javascript", []string{"pnpm-lock.yaml"}, "", []string{"npm"}, "", "pnpm not found on PATH"},
		{"uv lockfile beats pip", "python", []string{"uv.lock"}, "", []string{"pip", "uv"}, "uv", ""},
		{"configured poetry", "python", nil, "poetry", []string{"pip", "poetry"}, "poetry", ""},
		{"gradle build beats mvn on PATH", "java", []string{"build.gradle.kts"}, "", []string{"mvn", "gradle"}, "gradle", ""},
		{"unknown configured manager falls back", "dotnet", nil, "nuget", []string{"dotnet"}, "dotnet", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, f := range tt.files {
				if err := os.WriteFile(filepath.Join(dir, f), nil, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			avail := make(map[string]bool, len(tt.onPath))
			for _, b := range tt.onPath {
				avail[b] = true
			}
			lookPathFunc = mockLookPath(avail)

			cmd, skip := selectCommand(dir, CommandsForEcosystem(tt.eco), tt.configured)
			if skip != tt.wantSkip {
				t.Errorf("skip reason = %q, want %q", skip, tt.wantSkip)
			}
			got := ""
			if cmd != nil {
				got = cmd.Binary
			}
			if got != tt.wantBinary {
				t.Errorf("selected %q, want %q", got, tt.wantBinary)
			}
		})
	}
}

// writeFakeTool puts an executable shell script named name on a fresh PATH that
// exits with the given status, and makes lookPathFunc resolve it.
func writeFakeTool(t *testing.T, name, exitStatus string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake tool is a POSIX shell script")
	}
	bin := t.TempDir()
	script := "#!/bin/sh\necho fake " + name + "\nexit " + exitStatus + "\n"
	if err := os.WriteFile(filepath.Join(bin, name), []byte(script), 0o755); err != nil { //nolint:gosec // test executable
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	original := lookPathFunc
	t.Cleanup(func() { lookPathFunc = original })
	lookPathFunc = mockLookPath(map[string]bool{name: true})
}

// TestRunOutdated_FailureIsReported proves an outdated command that fails (an
// exit status other than the tool's "outdated found" signal) is recorded as a
// failed check, not as success, while the documented outdated exit still sets
// HasOutdated.
func TestRunOutdated_FailureIsReported(t *testing.T) {
	tests := []struct {
		name         string
		eco, binary  string
		exitStatus   string
		wantFailed   bool
		wantOutdated bool
	}{
		{"cargo-outdated missing exits 101", "rust", "cargo", "101", true, false},
		{"npm error exits 2", "javascript", "npm", "2", true, false},
		{"cargo outdated found", "rust", "cargo", "1", false, true},
		{"go list exit 1 is an error", "go", "go", "1", true, false},
		{"clean run", "go", "go", "0", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writeFakeTool(t, tt.binary, tt.exitStatus)
			var buf bytes.Buffer
			result, err := RunOutdated(context.Background(), &buf, t.TempDir(), []string{tt.eco}, OutdatedOptions{})
			if err != nil {
				t.Fatalf("RunOutdated: %v", err)
			}
			failed := result.FailedEcosystems()
			if (len(failed) == 1 && failed[0] == tt.eco) != tt.wantFailed {
				t.Errorf("FailedEcosystems() = %v, wantFailed %t", failed, tt.wantFailed)
			}
			if result.HasAnyOutdated() != tt.wantOutdated {
				t.Errorf("HasAnyOutdated() = %t, want %t", result.HasAnyOutdated(), tt.wantOutdated)
			}
			if !strings.Contains(buf.String(), "fake "+tt.binary) {
				t.Errorf("tool output not streamed: %q", buf.String())
			}
		})
	}
}

// TestRunOutdated_YarnBerrySkipped proves a Yarn 2+ project is reported as
// skipped rather than running `yarn outdated`, whose "command not found" exit 1
// would read as outdated packages.
func TestRunOutdated_YarnBerrySkipped(t *testing.T) {
	writeFakeTool(t, "yarn", "1")
	dir := t.TempDir()
	for name, body := range map[string]string{"yarn.lock": "", ".yarnrc.yml": "nodeLinker: pnp\n"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var buf bytes.Buffer
	result, err := RunOutdated(context.Background(), &buf, dir, []string{"javascript"}, OutdatedOptions{})
	if err != nil {
		t.Fatalf("RunOutdated: %v", err)
	}
	if len(result.Ecosystems) != 1 || !result.Ecosystems[0].Skipped || result.HasAnyOutdated() {
		t.Errorf("result = %+v, want javascript skipped and nothing outdated", result.Ecosystems)
	}
}
