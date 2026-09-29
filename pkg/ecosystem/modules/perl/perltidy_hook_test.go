package perl_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/perl"
)

func perltidyHook(t *testing.T) ecosystem.HookConfig {
	t.Helper()
	for _, h := range (&perl.Module{}).PreCommitHooks(ecosystem.ModuleConfig{}) {
		if h.ID == "perltidy" {
			return h
		}
	}
	t.Fatal("perltidy hook not found")
	return ecosystem.HookConfig{}
}

// TestPerltidyHook_Entry pins the entry to perltidy's real check option.
// `perltidy --check` is not a formatting check: it exits 0 on untidy code and
// writes <file>.tdy next to each input.
func TestPerltidyHook_Entry(t *testing.T) {
	t.Parallel()
	h := perltidyHook(t)
	fields := strings.Fields(h.Entry)
	if len(fields) == 0 || fields[0] != "perltidy" {
		t.Fatalf("Entry = %q, want a perltidy invocation", h.Entry)
	}
	for _, want := range []string{"--assert-tidy", "--standard-error-output"} {
		if !strings.Contains(h.Entry, want) {
			t.Errorf("Entry = %q, missing %s", h.Entry, want)
		}
	}
	if strings.Contains(h.Entry, "--check") {
		t.Errorf("Entry = %q uses --check, which never fails on untidy code", h.Entry)
	}
	if !h.PassFilenames {
		t.Error("perltidy hook must receive the staged filenames")
	}
}

// TestPerltidyHook_FailsOnUntidyCode runs the hook entry against real files
// when perltidy is installed: it must fail when any file is untidy, pass on
// tidy files, and leave no .tdy/.bak/.ERR artifacts behind.
func TestPerltidyHook_FailsOnUntidyCode(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("perltidy"); err != nil {
		t.Skip("perltidy not available")
	}
	fields := strings.Fields(perltidyHook(t).Entry)

	run := func(t *testing.T, files map[string]string) error {
		t.Helper()
		dir := t.TempDir()
		args := append([]string(nil), fields[1:]...)
		for name, content := range files {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			args = append(args, name)
		}
		cmd := exec.CommandContext(t.Context(), fields[0], args...)
		cmd.Dir = dir
		err := cmd.Run()
		entries, readErr := os.ReadDir(dir)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if len(entries) != len(files) {
			var names []string
			for _, e := range entries {
				names = append(names, e.Name())
			}
			t.Errorf("hook left extra files behind: %v", names)
		}
		return err
	}

	const tidy = "sub f {\n    my $x = 1;\n    return $x;\n}\n"
	const untidy = "sub f{my $x=1;return $x;}\n"

	if err := run(t, map[string]string{"good.pm": tidy}); err != nil {
		t.Errorf("hook failed on tidy code: %v", err)
	}
	if err := run(t, map[string]string{"good.pm": tidy, "bad.pm": untidy}); err == nil {
		t.Error("hook passed although bad.pm is untidy")
	}
}
