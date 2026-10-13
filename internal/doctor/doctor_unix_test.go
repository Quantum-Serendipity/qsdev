//go:build unix

package doctor

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/sysinfo"
)

// TestRunSingleCheck_LookupOnlyNeverExecutes: a check with no version flag
// only looks the binary up; doctor never runs a program a hook needs.
func TestRunSingleCheck_LookupOnlyNeverExecutes(t *testing.T) {
	bin := t.TempDir()
	marker := filepath.Join(t.TempDir(), "ran")
	script := "#!/bin/sh\n: > '" + marker + "'\necho 9.9.9\n"
	if err := os.WriteFile(filepath.Join(bin, "hook-tool"), []byte(script), 0o755); err != nil { //nolint:gosec // test executable
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)

	status := runSingleCheck(context.Background(), ToolCheck{Name: "hook-tool", Binary: "hook-tool", Required: true}, &sysinfo.OSInfo{})
	if !status.Installed || status.Path != filepath.Join(bin, "hook-tool") || !status.VersionOK {
		t.Errorf("status = %+v, want installed at %s with no floor to meet", status, bin)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Error("the lookup-only check executed the binary")
	}
}

// writeExe writes an executable shell script named name into dir.
func writeExe(t *testing.T, dir, name, script string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script), 0o755); err != nil { //nolint:gosec // test executable
		t.Fatal(err)
	}
	return path
}

// TestRequireBinaries_AlternativeOnPathDoesNotSatisfy: with only one of a
// check's binaries on PATH, a hook needing the other is reported missing,
// whichever way round.
func TestRequireBinaries_AlternativeOnPathDoesNotSatisfy(t *testing.T) {
	for _, tt := range []struct{ onPath, needed string }{{"pre-commit", "prek"}, {"prek", "pre-commit"}} {
		t.Run(tt.needed, func(t *testing.T) {
			bin := t.TempDir()
			writeExe(t, bin, tt.onPath, "echo '"+tt.onPath+" 4.0.0'\n")
			t.Setenv("PATH", bin)
			checks := RequireBinaries(DefaultChecks(), []string{tt.needed}, "r")
			for _, st := range RunChecks(context.Background(), &sysinfo.OSInfo{}, checks) {
				if st.Name == tt.needed && st.Required && st.Installed {
					t.Errorf("%s reported installed at %s with only %s on PATH", tt.needed, st.Path, tt.onPath)
				}
			}
		})
	}
}

// TestRunSingleCheck_InProjectVersion: a binary inside the project is never
// run; its version comes from ProjectVersion, and without one it stays
// unknown and is marked in the project.
func TestRunSingleCheck_InProjectVersion(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", t.TempDir())
	t.Chdir(project)
	marker := filepath.Join(t.TempDir(), "ran")
	venvBin := filepath.Join(project, ".venv", "bin")
	writeExe(t, venvBin, "python3", ": > '"+marker+"'\necho 'Python 3.12.4'\n")
	t.Setenv("PATH", venvBin)
	tc, ok := CheckNamed("python3")
	if !ok {
		t.Fatal("no python3 check")
	}
	cfg := filepath.Join(project, ".venv", "pyvenv.cfg")

	for _, tt := range []struct {
		name, cfg, want string
		ok              bool
	}{
		{"venv", "version = 3.12.4\n", "3.12.4", true},
		{"below floor", "version = 3.8.10\n", "3.8.10", false},
		{"no version key", "home = /usr/bin\n", "", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := os.WriteFile(cfg, []byte(tt.cfg), 0o644); err != nil {
				t.Fatal(err)
			}
			st := runSingleCheck(context.Background(), tc, &sysinfo.OSInfo{})
			if !st.Installed || !st.InProject || st.Version != tt.want || st.VersionOK != tt.ok {
				t.Errorf("status = %+v; want in-project version %q, VersionOK %v", st, tt.want, tt.ok)
			}
		})
	}
	if _, err := os.Stat(marker); err == nil {
		t.Error("doctor ran the in-project python3")
	}
}
