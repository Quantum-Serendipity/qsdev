package devinit

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/answers"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// freshCloneProject returns an initialised project with every local-only
// file removed, leaving what a `git clone` of the committed tree contains.
func freshCloneProject(t *testing.T) string {
	t.Helper()
	dir := initLifecycleProject(t)
	local := []string{branding.Get().StateDir, answers.DevenvCopyFile(), answers.LegacyClaudeCopyFile()}
	statePaths := state.StateFilePaths()
	local = append(local, statePaths[:]...)
	for _, rel := range local {
		if err := os.RemoveAll(filepath.Join(dir, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("removing %s: %v", rel, err)
		}
	}
	return dir
}

// treeDigest maps every regular file under dir (excluding .git) to the
// SHA-256 of its content, so any write, create or delete shows up as a diff.
func treeDigest(t *testing.T, dir string) map[string]string {
	t.Helper()
	digest := make(map[string]string)
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		digest[filepath.ToSlash(rel)] = hex.EncodeToString(sum[:])
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", dir, err)
	}
	return digest
}

func diffDigests(before, after map[string]string) []string {
	var diffs []string
	for rel, sum := range before {
		switch got, ok := after[rel]; {
		case !ok:
			diffs = append(diffs, "deleted "+rel)
		case got != sum:
			diffs = append(diffs, "modified "+rel)
		}
	}
	for rel := range after {
		if _, ok := before[rel]; !ok {
			diffs = append(diffs, "created "+rel)
		}
	}
	return diffs
}

// TestLifecycle_FreshClone_RefusesAndLeavesTreeUntouched is the H0-3
// regression (U03-01, U06-01): in a clone with a committed config but no local
// state, mutating commands used to regenerate from empty answers, rewriting
// or deleting the team's committed files. They must refuse with the join hint
// and exit 3 without touching the tree.
func TestLifecycle_FreshClone_RefusesAndLeavesTreeUntouched(t *testing.T) {
	tests := []struct {
		name string
		cmd  func() *cobra.Command
		args []string
		// staged commands report the refusal as a failed update stage, so
		// the hint is in the output and the returned error only carries
		// the stage's exit code.
		staged bool
	}{
		{name: "enable changelog", cmd: enableCmd, args: []string{"changelog"}},
		{name: "enable changelog --dry-run", cmd: enableCmd, args: []string{"changelog", "--dry-run"}},
		{name: "disable --force gitleaks", cmd: disableCmd, args: []string{"gitleaks", "--force"}},
		{name: "teardown --force", cmd: teardownCmd, args: []string{"--force"}},
		{name: "teardown --dry-run", cmd: teardownCmd, args: []string{"--dry-run"}},
		{name: "repair", cmd: repairCmd},
		{name: "repair --reset", cmd: repairCmd, args: []string{"--reset"}},
		{name: "init --update", cmd: initCmd, args: []string{"--update"}},
		{name: "update --configs-only", cmd: updateCmd, args: []string{"--configs-only"}, staged: true},
		{name: "update --deps-only", cmd: updateCmd, args: []string{"--deps-only"}, staged: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := freshCloneProject(t)
			stubDevenvOnPath(t, dir)
			before := treeDigest(t, dir)

			out, err := runLifecycleCmd(t, dir, tc.cmd(), tc.args...)

			if err == nil {
				t.Fatalf("command succeeded, want a join refusal\n%s", out)
			}
			hint := err.Error()
			if tc.staged {
				hint = out
			} else if !errors.Is(err, state.ErrNotJoined) {
				t.Fatalf("err = %v, want state.ErrNotJoined\n%s", err, out)
			}
			var coded interface{ ExitCode() int }
			if !errors.As(err, &coded) || coded.ExitCode() != exitNotJoined {
				t.Errorf("err %v does not carry exit code %d", err, exitNotJoined)
			}
			if !strings.Contains(hint, "init --yes") {
				t.Errorf("%q lacks the join hint", hint)
			}
			if diffs := diffDigests(before, treeDigest(t, dir)); len(diffs) > 0 {
				t.Errorf("command changed the tree: %v", diffs)
			}
			sidecars, _ := filepath.Glob(filepath.Join(dir, "*.new"))
			if len(sidecars) > 0 {
				t.Errorf("sidecars written: %v", sidecars)
			}
		})
	}
}

// stubDevenvOnPath puts a fake devenv first on PATH that records any run as
// a file in dir, so a `devenv update` that should have been refused shows up
// in the tree digest instead of touching the real toolchain. The stub is a
// shell script, so on Windows only the refusal itself is asserted.
func stubDevenvOnPath(t *testing.T, dir string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		return
	}
	bin := t.TempDir()
	marker := filepath.Join(dir, "devenv-was-run")
	script := "#!/bin/sh\necho \"$@\" > '" + marker + "'\n"
	if err := os.WriteFile(filepath.Join(bin, "devenv"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// TestLifecycle_NeverInitialized_EnableUnchanged keeps enable's behaviour in
// a directory that was never initialised: without a config it is not a clone
// and must not be told to join.
func TestLifecycle_NeverInitialized_EnableUnchanged(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/lc\n\ngo 1.24\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := enableTool(t, dir, "changelog")
	if errors.Is(err, state.ErrNotJoined) {
		t.Fatalf("enable in a never-initialised dir asked to join: %v\n%s", err, out)
	}
}

func TestRequireJoined(t *testing.T) {
	t.Parallel()
	cfg := branding.Get().ConfigFile
	tests := []struct {
		name     string
		files    []string
		wantJoin bool
	}{
		{name: "never initialised"},
		{name: "fresh clone", files: []string{cfg}, wantJoin: true},
		{name: "joined", files: []string{cfg, state.InitStateFile()}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			for _, rel := range tc.files {
				path := filepath.Join(dir, filepath.FromSlash(rel))
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte("x: 1\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			err := requireJoined(dir)
			if got := errors.Is(err, state.ErrNotJoined); got != tc.wantJoin {
				t.Fatalf("requireJoined = %v, want ErrNotJoined=%v", err, tc.wantJoin)
			}
			if tc.wantJoin && !strings.Contains(err.Error(), cfg) {
				t.Errorf("err %q does not name %s", err, cfg)
			}
		})
	}
}
