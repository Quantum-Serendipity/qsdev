package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/testutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// ancestorOverlay is a project defaults file adding a custom hook to the
// baseline tier, as scripts/e2e/root-hijack.sh plants it.
const ancestorOverlay = `custom_hooks:
  - id: evil-hook
    name: Evil
    description: planted by an enclosing project
    entry: ./evil.sh
    language: system
    stages: [pre-commit]
hook_tiers:
  baseline:
    - evil-hook
`

// TestInitIgnoresAncestorProjectDefaults is the U01-01 residual (a)
// regression on the full command tree: `init` creates its project where it
// runs (Here mode), so in a non-git child of a trusted project whose
// .<app>/defaults.yaml adds a hook, it must not take that project's policy.
// An Enclosing command (`defaults show`) from the same child keeps the
// enclosing project's policy, which also proves the overlay is applied at
// all. The ancestor project is left unchanged.
//
// Each command runs in a fresh process (runQsdev), as the binary does: the
// runtime takes the command from os.Args, and the tree keeps flag values and
// the stored project between executions in one process.
func TestInitIgnoresAncestorProjectDefaults(t *testing.T) {
	t.Parallel()
	b := branding.Get()
	env := guardrailEnv(t)

	proj := filepath.Join(testutil.MarkerFreeTempDir(t), "proj")
	sub := filepath.Join(proj, "sub")
	overlay := catalog.ProjectConfigPath(proj)
	for _, d := range []string{sub, filepath.Dir(overlay)} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeFile(t, filepath.Join(proj, b.ConfigFile), "version: 2\nsecurity:\n  level: standard\n")
	writeFile(t, overlay, ancestorOverlay)

	show, code := runQsdev(t, env, sub, nil, "defaults", "show")
	if code != 0 || !strings.Contains(show, "evil-hook") {
		t.Fatalf("defaults show from %s: exit %d, want the enclosing project's evil-hook:\n%s", sub, code, show)
	}

	// defaults show logs its session in the enclosing project; init, run in
	// sub, must leave the ancestor alone.
	before := treeSnapshot(t, proj, sub)
	if out, code := runQsdev(t, env, sub, nil, "init", "--yes", "--lang", "go"); code != 0 {
		t.Fatalf("init --yes --lang go: exit %d\n%s", code, out)
	}
	nix, err := os.ReadFile(filepath.Join(sub, "devenv.nix"))
	if err != nil {
		t.Fatalf("init wrote no devenv.nix in the working directory: %v", err)
	}
	if strings.Contains(string(nix), "evil-hook") {
		t.Errorf("devenv.nix in %s carries the ancestor's evil-hook", sub)
	}
	if after := treeSnapshot(t, proj, sub); after != before {
		t.Errorf("init changed the ancestor project outside %s:\nbefore:\n%s\nafter:\n%s", sub, before, after)
	}
}

// treeSnapshot lists every path under root outside skip with the content of
// each regular file, so a created or rewritten file shows up as a diff.
func treeSnapshot(t *testing.T, root, skip string) string {
	t.Helper()
	var sb strings.Builder
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == skip {
			return filepath.SkipDir
		}
		sb.WriteString(p + "\n")
		if d.Type().IsRegular() {
			data, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			sb.Write(data)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return sb.String()
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
