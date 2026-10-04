package devenv_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/devenv"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/canon"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestGeneratedEnvLoadsOnlyListedSources ties the self-protection hook's list
// of environment source files (canon.EnvSourceFiles, SP-015) to what the
// generated environment loads: devenv's own configuration files are listed,
// and the dotenv file and .envrc.local, which the hook does not guard, stay
// unloaded: devenv.nix disables dotenv and .envrc runs only devenv's
// direnvrc. A generator change that starts loading another file must list it
// in canon.envSourceFiles (U18-WS1).
func TestGeneratedEnvLoadsOnlyListedSources(t *testing.T) {
	t.Parallel()
	listed := canon.EnvSourceFiles()
	for _, name := range []string{"devenv.yaml", "devenv.local.yaml", "devenv.nix", devenv.DevenvLocalNixFile} {
		if !slices.Contains(listed, name) {
			t.Errorf("canon.EnvSourceFiles() = %q, missing %s, which devenv loads", listed, name)
		}
	}
	for _, name := range []string{".env", ".envrc.local"} {
		if slices.Contains(listed, name) {
			t.Errorf("canon.EnvSourceFiles() lists %s, which the generated environment does not load", name)
		}
	}

	nix, err := devenv.GenerateDevenvNix(types.WizardAnswers{}, ecosystem.DefaultRegistry())
	if err != nil {
		t.Fatalf("GenerateDevenvNix: %v", err)
	}
	if !strings.Contains(string(nix.Content), "dotenv.enable = false;") {
		t.Errorf("generated devenv.nix no longer disables dotenv; list .env in canon.envSourceFiles:\n%s", nix.Content)
	}
	envrc := devenv.GenerateEnvrc(types.WizardAnswers{Direnv: true})
	if envrc == nil {
		t.Fatal("GenerateEnvrc returned nil with direnv enabled")
	}
	for _, line := range strings.Split(string(envrc.Content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if line != `eval "$(devenv direnvrc)"` && line != "use devenv" {
			t.Errorf("generated .envrc runs %q; list any file it loads in canon.envSourceFiles", line)
		}
	}
}
