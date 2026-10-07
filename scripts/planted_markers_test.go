package scripts_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// These tests run scripts/ci/planted-markers.sh against a stub go that
// records how it was invoked and what the planted directories held at that
// moment, so no real test suite runs.

// fakePlantedGo writes its arguments, the test environment and listings of
// $TMPDIR and $PLANT_ROOT to $FAKE_REC, then exits with $FAKE_GO_EXIT.
const fakePlantedGo = `#!/bin/sh
{
  echo "args=$*"
  echo "gowork=$GOWORK"
  echo "tmpdir=$TMPDIR"
  echo "tmp=$TMP"
  echo "temp=$TEMP"
  echo "tmpdir_mode=$(ls -ld "$TMPDIR" | cut -c1-10)"
  echo "tmpdir_ls=$(ls -A "$TMPDIR" | tr '\n' ' ')"
  echo "tmpdir_claude=$(ls -A "$TMPDIR/.claude" | tr '\n' ' ')"
  echo "plant_ls=$(ls -A "$PLANT_ROOT" | tr '\n' ' ')"
  echo "pwd_vendor=$(test -d vendor && echo yes)"
} >"$FAKE_REC"
exit "${FAKE_GO_EXIT:-0}"
`

// plantedTools are the real binaries planted-markers.sh and the stub use.
var plantedTools = []string{"bash", "sh", "mkdir", "mktemp", "chmod", "rm", "ls", "cut", "tr"}

// plantedMarkers is the marker set every planted root must hold.
const plantedMarkers = ".claude .devinit .qsdev .qsdev.yaml CLAUDE.md devenv.nix "

type plantedRun struct {
	code      int
	out       string
	rec       map[string]string
	plantRoot string
	tmpParent string
}

// runPlanted runs planted-markers.sh with args, planting into a fresh
// PLANT_ROOT and creating its trusted directory under a fresh TMPDIR. pre
// runs before the script, against the plant root.
func runPlanted(t *testing.T, goExit string, pre func(plantRoot string), args ...string) plantedRun {
	t.Helper()
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("planted-markers.sh runs on the Linux CI runner; it needs bash")
	}
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	r := plantedRun{plantRoot: filepath.Join(root, "plant"), tmpParent: filepath.Join(root, "tmp")}
	for _, d := range []string{binDir, r.plantRoot, r.tmpParent} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, tool := range plantedTools {
		p, err := exec.LookPath(tool)
		if err != nil {
			t.Skipf("%s is not installed; the CI runner provides it", tool)
		}
		if err := os.Symlink(p, filepath.Join(binDir, tool)); err != nil {
			t.Fatal(err)
		}
	}
	writeExec(t, filepath.Join(binDir, "go"), fakePlantedGo)
	if pre != nil {
		pre(r.plantRoot)
	}
	recPath := filepath.Join(root, "rec")

	cmd := exec.Command(filepath.Join(binDir, "bash"), append([]string{"ci/planted-markers.sh"}, args...)...)
	cmd.Env = []string{
		"PATH=" + binDir, "HOME=" + root, "TMPDIR=" + r.tmpParent,
		"PLANT_ROOT=" + r.plantRoot, "FAKE_REC=" + recPath, "FAKE_GO_EXIT=" + goExit,
	}
	out, err := cmd.CombinedOutput()
	r.out = string(out)
	var exitErr *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &exitErr):
		r.code = exitErr.ExitCode()
	default:
		t.Fatalf("running planted-markers.sh: %v", err)
	}
	r.rec = map[string]string{}
	if data, err := os.ReadFile(recPath); err == nil {
		for line := range strings.Lines(string(data)) {
			k, v, _ := strings.Cut(strings.TrimSuffix(line, "\n"), "=")
			r.rec[k] = v
		}
	}
	return r
}

func lsA(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

// TestPlantedMarkers covers the XA-WS12 hermetic CI job: the script plants
// the project marker set into the shared plant root and into a fresh trusted
// (0755) directory it exports as TMPDIR, runs the whole suite in vendor mode
// with -shuffle=on, passes its exit code through and removes everything it
// planted.
func TestPlantedMarkers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		goExit   string
		args     []string
		wantCode int
		wantArgs string
	}{
		{
			name:     "default full suite",
			goExit:   "0",
			wantArgs: "test -mod=vendor -count=1 -shuffle=on ./...",
		},
		{
			name:     "extra args replace the package list",
			goExit:   "0",
			args:     []string{"-shuffle=42", "./addons/..."},
			wantArgs: "test -mod=vendor -count=1 -shuffle=on -shuffle=42 ./addons/...",
		},
		{
			name:     "test failure is passed through",
			goExit:   "3",
			wantCode: 3,
			wantArgs: "test -mod=vendor -count=1 -shuffle=on ./...",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := runPlanted(t, tt.goExit, nil, tt.args...)
			if r.code != tt.wantCode {
				t.Fatalf("exit code = %d, want %d; output:\n%s", r.code, tt.wantCode, r.out)
			}
			checks := map[string]string{
				"args":          tt.wantArgs,
				"gowork":        "off",
				"tmpdir_mode":   "drwxr-xr-x",
				"tmpdir_ls":     plantedMarkers,
				"tmpdir_claude": "settings.json ",
				"plant_ls":      plantedMarkers,
				"pwd_vendor":    "yes",
			}
			for k, want := range checks {
				if got := r.rec[k]; got != want {
					t.Errorf("%s = %q, want %q; output:\n%s", k, got, want, r.out)
				}
			}
			trusted := r.rec["tmpdir"]
			if filepath.Dir(trusted) != r.tmpParent {
				t.Errorf("TMPDIR = %q, want a fresh directory under %q", trusted, r.tmpParent)
			}
			if r.rec["tmp"] != trusted || r.rec["temp"] != trusted {
				t.Errorf("TMP = %q, TEMP = %q, want both %q", r.rec["tmp"], r.rec["temp"], trusted)
			}
			if got := lsA(t, r.plantRoot); len(got) != 0 {
				t.Errorf("plant root still holds %v after the run", got)
			}
			if _, err := os.Stat(trusted); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("trusted TMPDIR %q survived the run: %v", trusted, err)
			}
		})
	}
}

// TestPlantedMarkers_RefusesExistingMarker: a marker already in the plant
// root is somebody else's file; the script must neither run the suite nor
// overwrite or delete it.
func TestPlantedMarkers_RefusesExistingMarker(t *testing.T) {
	t.Parallel()
	r := runPlanted(t, "0", func(plantRoot string) {
		writeFile(t, filepath.Join(plantRoot, "CLAUDE.md"), "theirs\n")
	})
	if r.code == 0 {
		t.Fatalf("exit code = 0, want a refusal; output:\n%s", r.out)
	}
	if !strings.Contains(r.out, "CLAUDE.md") {
		t.Errorf("output does not name the existing marker:\n%s", r.out)
	}
	if len(r.rec) != 0 {
		t.Errorf("go ran despite the refusal: %v", r.rec)
	}
	got, err := os.ReadFile(filepath.Join(r.plantRoot, "CLAUDE.md"))
	if err != nil || string(got) != "theirs\n" {
		t.Errorf("existing marker changed: %q, %v", got, err)
	}
	if left := lsA(t, r.plantRoot); len(left) != 1 {
		t.Errorf("plant root holds %v, want only the existing CLAUDE.md", left)
	}
	if left := lsA(t, r.tmpParent); len(left) != 0 {
		t.Errorf("TMPDIR parent holds %v, want nothing left behind", left)
	}
}
