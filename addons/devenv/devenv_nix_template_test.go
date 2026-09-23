package devenv_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/addons/devenv"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"

	// Register every ecosystem module with the DefaultRegistry via init().
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules"
)

// renderFullDevenvNix renders devenv.nix for answers that exercise overlays,
// services, tool packages and development tasks.
func renderFullDevenvNix(t *testing.T) string {
	t.Helper()
	answers := types.WizardAnswers{
		ProjectName:  "demo",
		Languages:    []types.LanguageChoice{{Name: "go"}},
		Overlays:     []string{"nix/go-overlay.nix", "nix/my overlay.nix"},
		EnvVars:      map[string]string{"EDITOR": "vim"},
		EnabledTools: map[string]bool{"gitleaks": true, "semgrep": true},
		Services: []types.ServiceChoice{
			{Name: "kafka"}, {Name: "nats"}, {Name: "minio"},
			{Name: "keycloak", Settings: map[string]string{"admin_password": `a${x}"b\c`}},
		},
	}
	got, err := devenv.GenerateDevenvNix(answers, ecosystem.DefaultRegistry())
	if err != nil {
		t.Fatalf("GenerateDevenvNix: %v", err)
	}
	return string(got.Content)
}

func TestGenerateDevenvNix_Structure(t *testing.T) {
	t.Parallel()
	content := renderFullDevenvNix(t)

	tests := []struct {
		name    string
		sub     string
		present bool
	}{
		// devenv already loads devenv.local.nix; a second import applies it twice.
		{"no duplicate devenv.local.nix import", "./devenv.local.nix", false},
		{"no imports line", "imports =", false},
		// Extends devenv's default unset list instead of replacing it.
		{"unset list keeps devenv defaults", "unsetEnvVars = options.unsetEnvVars.default ++ [", true},
		{"module takes options", "{ pkgs, lib, config, options, ... }:", true},
		// Tasks are PATH scripts (usable under direnv), not shell functions.
		{"task as script", `scripts."qsdev-lint" = {`, true},
		{"task errexit", "set -euo pipefail", true},
		{"no task shell function", "qsdev-lint() {", false},
		{"overlay bare path", "(import ./nix/go-overlay.nix)", true},
		{"overlay with space", `(import (./. + "/nix/my overlay.nix"))`, true},
		// Worktree-aware hook check instead of [ -d .git ].
		{"hooks via git rev-parse", "git rev-parse --git-path hooks", true},
		{"no .git dir test", "[ -d .git ]", false},
		// mcp-secrets.env is parsed, never sourced; gh token is gated.
		{"secrets file not sourced", `. "$PWD/.qsdev/mcp-secrets.env"`, false},
		{"gh token gated on .mcp.json", "if mcp_refs GITHUB_TOKEN &&", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := strings.Contains(content, tt.sub); got != tt.present {
				t.Errorf("strings.Contains(devenv.nix, %q) = %v, want %v", tt.sub, got, tt.present)
			}
		})
	}
}

// TestGenerateDevenvNix_ParsesAndEvaluates checks the rendered file is valid
// Nix and that escaped values and task scripts evaluate to what was intended.
func TestGenerateDevenvNix_ParsesAndEvaluates(t *testing.T) {
	t.Parallel()
	nixInstantiate, err := exec.LookPath("nix-instantiate")
	if err != nil {
		t.Skip("nix-instantiate not available")
	}
	path := filepath.Join(t.TempDir(), "devenv.nix")
	if err := os.WriteFile(path, []byte(renderFullDevenvNix(t)), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if out, err := exec.CommandContext(ctx, nixInstantiate, "--parse", path).CombinedOutput(); err != nil {
		t.Fatalf("generated devenv.nix does not parse: %v\n%s", err, out)
	}

	// Stub module arguments: only lazily-unused or plain attributes are read.
	expr := `let m = import "` + path + `" { pkgs = {}; lib = {}; config = {}; options = { unsetEnvVars.default = [ "HOST_PATH" ]; }; };
in [ m.services.keycloak.initialAdminPassword (builtins.head m.unsetEnvVars) (builtins.head (builtins.split "\n" m.scripts."qsdev-lint".exec)) ]`
	out, err := exec.CommandContext(ctx, nixInstantiate, "--eval", "--strict", "--json", "--expr", expr).CombinedOutput()
	if err != nil {
		t.Fatalf("evaluating generated devenv.nix: %v\n%s", err, out)
	}
	want := `["a${x}\"b\\c","HOST_PATH","set -euo pipefail"]`
	if got := strings.TrimSpace(string(out)); got != want {
		t.Errorf("evaluated = %s\nwant       %s", got, want)
	}
}
