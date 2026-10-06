package mcpserve

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	qsdevcatalog "github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools/devenv"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"
)

// TestServeBuildsNixRunPolicyOnlyWhenMounted proves the policy for
// qsdev_nix_run is built only when the tool is mounted, so a server without
// it (and every caller of tools.Names, which builds the registrations
// without serving them) never reads the catalog rules or settings files for
// it (W0N-19).
func TestServeBuildsNixRunPolicyOnlyWhenMounted(t *testing.T) {
	t.Parallel()
	want := []string{"Bash(curl * | sh)"}
	for _, mounted := range []bool{false, true} {
		calls := 0
		load := func() (devenv.NixRunPolicy, error) {
			calls++
			return devenv.NixRunPolicy{DenyRules: want}, nil
		}
		got, err := withNixRunPolicy(tools.Options{NixRun: mounted}, load)
		if err != nil {
			t.Fatalf("mounted=%t: %v", mounted, err)
		}
		wantCalls := 0
		if mounted {
			wantCalls = 1
		}
		if calls != wantCalls {
			t.Errorf("mounted=%t: policy built %d times, want %d", mounted, calls, wantCalls)
		}
		if mounted && !slices.Equal(got.NixRunPolicy.DenyRules, want) {
			t.Errorf("mounted: NixRunPolicy.DenyRules = %q, want %q", got.NixRunPolicy.DenyRules, want)
		}
		if !mounted && got.NixRunPolicy.DenyRules != nil {
			t.Errorf("not mounted: NixRunPolicy.DenyRules = %q, want nil", got.NixRunPolicy.DenyRules)
		}
	}

	// A mounted tool whose policy cannot be built fails startup rather than
	// running unchecked.
	boom := errors.New("boom")
	if _, err := withNixRunPolicy(tools.Options{NixRun: true}, func() (devenv.NixRunPolicy, error) { return devenv.NixRunPolicy{}, boom }); !errors.Is(err, boom) {
		t.Errorf("load error = %v, want it to propagate", err)
	}
}

// TestNixRunJudge proves the serve command's judge holds a Bash command
// line to self-protection, SP-014 included, in the project root.
func TestNixRunJudge(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	j := nixRunJudge(root, []cmdscan.CommandSpec{{Path: [][]string{{"teardown"}}}})
	for line, want := range map[string]bool{
		"rm .claude/settings.json": true,
		"qsdev teardown --force":   true,
		"jq . data.json":           false,
	} {
		if _, _, denied := j(line); denied != want {
			t.Errorf("judge(%q) denied = %v, want %v", line, denied, want)
		}
	}
}

// TestNixRunDenyRules proves the deny set is the user-scope catalog's deny
// rules plus the deny rules of the user, project and local Claude Code
// settings.
func TestNixRunDenyRules(t *testing.T) {
	t.Parallel()
	root, userDir := t.TempDir(), t.TempDir()
	writeSettings(t, filepath.Join(root, ".claude", "settings.json"), "Bash(jq *secret*)")
	writeSettings(t, filepath.Join(root, ".claude", "settings.local.json"), "Bash(local-deny *)")
	writeSettings(t, filepath.Join(userDir, "settings.json"), "Bash(user-deny *)")
	cat, err := qsdevcatalog.LoadEmbeddedOnly()
	if err != nil {
		t.Fatal(err)
	}

	rules, err := nixRunDenyRules(root, cat, func() (string, error) { return userDir, nil })
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range append(cat.PermissionDenyRules("pipe_to_shell"),
		"Bash(jq *secret*)", "Bash(local-deny *)", "Bash(user-deny *)") {
		if !slices.Contains(rules, want) {
			t.Errorf("deny set lacks %q", want)
		}
	}

	if _, err := nixRunDenyRules(root, cat, func() (string, error) { return "", errors.New("no home") }); err == nil {
		t.Error("an unresolvable Claude Code user directory must fail closed")
	}
}

func writeSettings(t *testing.T, path, deny string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"permissions":{"deny":["` + deny + `"]}}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}
