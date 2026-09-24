package devenv

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// TestScriptHookEntry_EvaluatesToScript checks that a module hook's Script
// reaches bash byte for byte: shell antiquotes ("${var}") and quote pairs
// must survive the Nix indented string, and the hook's package is put first
// on PATH.
func TestScriptHookEntry_EvaluatesToScript(t *testing.T) {
	t.Parallel()

	nixInstantiate, err := exec.LookPath("nix-instantiate")
	if err != nil {
		t.Skip("nix-instantiate not available")
	}

	script := "v=${HOME##*/}\necho '' \"$@\"\n  indented ${1:-x}"
	entry := scriptHookEntry(ecosystem.HookConfig{ID: "demo", Script: script, NixPackage: "tool"})

	expr := `let pkgs = { tool = "/nix/tool"; writeShellScript = name: text: { inherit name text; }; }; in ` + entry
	path := filepath.Join(t.TempDir(), "entry.nix")
	if err := os.WriteFile(path, []byte(expr), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command(nixInstantiate, "--eval", "--strict", "--json", path).Output()
	if err != nil {
		t.Fatalf("evaluating entry: %v\n%s", err, entry)
	}
	var got struct{ Name, Text string }
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("decoding %s: %v", out, err)
	}
	if got.Name != "demo" {
		t.Errorf("script name = %q, want demo", got.Name)
	}
	if want := "export PATH=/nix/tool/bin:$PATH\n" + script + "\n"; got.Text != want {
		t.Errorf("script text = %q, want %q", got.Text, want)
	}
}
