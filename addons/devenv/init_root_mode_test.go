package devenv

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/testutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// TestDevenvInitTargetsCwd is the U12-05/U13-01 regression: `devenv init`
// run from <proj>/services/api plans its files in services/api (it is marked
// MarkRootHere), so neither the enclosing project's existing devenv.nix nor
// its join state (a committed config without local state) applies.
func TestDevenvInitTargetsCwd(t *testing.T) {
	proj := testutil.MarkerFreeTempDir(t)
	for _, f := range []string{branding.Get().ConfigFile, "devenv.nix"} {
		if err := os.WriteFile(filepath.Join(proj, f), []byte("version: 1\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	api := filepath.Join(proj, "services", "api")
	if err := os.MkdirAll(api, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(api)

	cmd := initCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--dry-run", "--lang", "go"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("devenv init --dry-run from services/api: %v\n%s", err, out.String())
	}
	if !regexp.MustCompile(`(?m)^devenv\.nix\s+create\b`).MatchString(out.String()) {
		t.Errorf("devenv init did not plan to create devenv.nix in services/api:\n%s", out.String())
	}
	if _, err := os.Stat(filepath.Join(api, "devenv.nix")); !os.IsNotExist(err) {
		t.Errorf("dry run wrote devenv.nix (stat err = %v)", err)
	}
}

// TestInitCmd_IgnoresAncestorMarker locks in U13-01 for a real write: `devenv
// init` run from a subdirectory of a project writes devenv.nix into the
// working directory, never into the enclosing project's root.
func TestInitCmd_IgnoresAncestorMarker(t *testing.T) {
	isolateHome(t)
	proj := testutil.MarkerFreeTempDir(t)
	if err := os.WriteFile(filepath.Join(proj, branding.Get().ConfigFile), []byte("version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(proj, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub)

	cmd := initCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--lang", "go", "--yes"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("devenv init from sub: %v\n%s", err, out.String())
	}
	if _, err := os.Stat(filepath.Join(sub, "devenv.nix")); err != nil {
		t.Errorf("devenv init did not write sub/devenv.nix: %v\n%s", err, out.String())
	}
	if _, err := os.Stat(filepath.Join(proj, "devenv.nix")); !os.IsNotExist(err) {
		t.Errorf("devenv init wrote the enclosing project's devenv.nix (stat err = %v)", err)
	}
}

// isolateHome points HOME, USERPROFILE and the XDG base directories at fresh
// temporary directories, so a command under test neither reads nor writes the
// developer's real per-user state.
func isolateHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	for _, v := range []string{"XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_STATE_HOME", "XDG_CACHE_HOME"} {
		t.Setenv(v, filepath.Join(home, strings.ToLower(v)))
	}
}
