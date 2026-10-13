package devenv

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
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
