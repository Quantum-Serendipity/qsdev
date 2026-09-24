package devinit

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestUpdate_AppliesCloudIsolateCLIConfig guards W135: once
// cloud.isolate_cli_config is committed, `init --update` points the Azure and
// Google Cloud CLIs at per-project configuration directories under the
// gitignored .qsdev/, and keeps the setting in .qsdev.yaml. Without it
// devenv.nix sets neither variable. .claude/settings.json masks both
// directories from the agent either way.
func TestUpdate_AppliesCloudIsolateCLIConfig(t *testing.T) {
	dir := t.TempDir()
	for name, content := range map[string]string{
		"azure-pipelines.yml": "trigger: none\n",
		"cloudbuild.yaml":     "steps: []\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// devenv.nix gathers every env.NAME binding into one env block.
	lines := []string{
		`AZURE_CONFIG_DIR = lib.mkDefault "${config.devenv.root}/.qsdev/cloud/azure";`,
		`CLOUDSDK_CONFIG = lib.mkDefault "${config.devenv.root}/.qsdev/cloud/gcp";`,
	}
	assertIsolation := func(t *testing.T, want bool) {
		t.Helper()
		nix := readProjectFile(t, dir, "devenv.nix")
		for _, line := range lines {
			if got := strings.Contains(nix, line); got != want {
				t.Errorf("devenv.nix contains %q = %v, want %v", line, got, want)
			}
		}
		settings := readProjectFile(t, dir, filepath.Join(".claude", "settings.json"))
		for _, rule := range []string{`"Read(./.qsdev/cloud/azure/**)"`, `"Read(./.qsdev/cloud/gcp/**)"`} {
			if !strings.Contains(settings, rule) {
				t.Errorf(".claude/settings.json lacks %s", rule)
			}
		}
		if !strings.Contains(readProjectFile(t, dir, ".gitignore"), "\n.qsdev/\n") {
			t.Error(".gitignore does not ignore .qsdev/, where the isolated CLI configuration lives")
		}
	}

	if out, err := executeInitCmd(t, dir, "--yes"); err != nil {
		t.Fatalf("init failed: %v\n%s", err, out)
	}
	cfgPath := filepath.Join(dir, ".qsdev.yaml")
	cfg := readProjectFile(t, dir, ".qsdev.yaml")
	for _, lang := range []string{"- name: azure", "- name: gcp"} {
		if !strings.Contains(cfg, lang) {
			t.Fatalf("init did not detect %q:\n%s", lang, cfg)
		}
	}
	assertIsolation(t, false)

	if err := os.WriteFile(cfgPath, []byte(cfg+"cloud:\n  isolate_cli_config: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := executeInitCmd(t, dir, "--update"); err != nil {
		t.Fatalf("update failed: %v\n%s", err, out)
	}
	assertIsolation(t, true)
	assertNixParses(t, dir)
	if !strings.Contains(readProjectFile(t, dir, ".qsdev.yaml"), "isolate_cli_config: true") {
		t.Error("update dropped cloud.isolate_cli_config from .qsdev.yaml")
	}
}

// assertNixParses checks dir's devenv.nix with nix-instantiate --parse when
// it is available.
func assertNixParses(t *testing.T, dir string) {
	t.Helper()
	nixInstantiate, err := exec.LookPath("nix-instantiate")
	if err != nil {
		t.Log("nix-instantiate not available, skipping syntax validation")
		return
	}
	if out, err := exec.Command(nixInstantiate, "--parse", filepath.Join(dir, "devenv.nix")).CombinedOutput(); err != nil {
		t.Fatalf("nix-instantiate --parse rejected devenv.nix: %v\n%s", err, out)
	}
}
