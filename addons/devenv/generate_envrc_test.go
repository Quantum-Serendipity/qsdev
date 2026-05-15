package devenv_test

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/devenv"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestGenerateEnvrc_DirenvEnabled(t *testing.T) {
	answers := types.WizardAnswers{Direnv: true}
	got := devenv.GenerateEnvrc(answers)

	if got == nil {
		t.Fatal("expected non-nil GeneratedFile when Direnv is enabled")
	}
	if got.Path != ".envrc" {
		t.Errorf("Path = %q, want %q", got.Path, ".envrc")
	}
	if got.Mode != 0o644 {
		t.Errorf("Mode = %o, want %o", got.Mode, 0o644)
	}
	if got.Strategy != types.Skip {
		t.Errorf("Strategy = %v, want Skip", got.Strategy)
	}
	if len(got.Content) == 0 {
		t.Error("Content is empty")
	}
}

func TestGenerateEnvrc_DirenvDisabled(t *testing.T) {
	answers := types.WizardAnswers{Direnv: false}
	got := devenv.GenerateEnvrc(answers)

	if got != nil {
		t.Errorf("expected nil when Direnv is disabled, got %+v", got)
	}
}

func TestGenerateEnvrc_ZeroValueAnswers(t *testing.T) {
	var answers types.WizardAnswers
	got := devenv.GenerateEnvrc(answers)

	if got != nil {
		t.Errorf("expected nil for zero-value WizardAnswers, got %+v", got)
	}
}

func TestGenerateEnvrc_ContentValid(t *testing.T) {
	answers := types.WizardAnswers{Direnv: true}
	got := devenv.GenerateEnvrc(answers)

	if got == nil {
		t.Fatal("expected non-nil GeneratedFile")
	}

	content := string(got.Content)
	if !strings.Contains(content, "use devenv") {
		t.Error("content does not contain 'use devenv'")
	}
	if !strings.Contains(content, "devenv direnvrc") {
		t.Error("content does not contain 'devenv direnvrc'")
	}
}

func TestNativeActivationInstructions_Bash(t *testing.T) {
	got := devenv.NativeActivationInstructions("bash")

	if !strings.Contains(got, `devenv hook bash`) {
		t.Error("bash instructions do not contain 'devenv hook bash'")
	}
	if !strings.Contains(got, "~/.bashrc") {
		t.Error("bash instructions do not contain '~/.bashrc'")
	}
}

func TestNativeActivationInstructions_Zsh(t *testing.T) {
	got := devenv.NativeActivationInstructions("zsh")

	if !strings.Contains(got, `devenv hook zsh`) {
		t.Error("zsh instructions do not contain 'devenv hook zsh'")
	}
	if !strings.Contains(got, "~/.zshrc") {
		t.Error("zsh instructions do not contain '~/.zshrc'")
	}
}

func TestNativeActivationInstructions_Fish(t *testing.T) {
	got := devenv.NativeActivationInstructions("fish")

	if !strings.Contains(got, `devenv hook fish`) {
		t.Error("fish instructions do not contain 'devenv hook fish'")
	}
	if !strings.Contains(got, "config.fish") {
		t.Error("fish instructions do not contain 'config.fish'")
	}
}

func TestNativeActivationInstructions_Unknown(t *testing.T) {
	got := devenv.NativeActivationInstructions("")

	if got == "" {
		t.Fatal("expected non-empty instructions for unknown shell")
	}
	// Should cover all three shells in the generic output.
	if !strings.Contains(got, "bash") {
		t.Error("generic instructions do not mention bash")
	}
	if !strings.Contains(got, "zsh") {
		t.Error("generic instructions do not mention zsh")
	}
	if !strings.Contains(got, "fish") {
		t.Error("generic instructions do not mention fish")
	}
}

func TestPostGenerationMessage_Direnv(t *testing.T) {
	got := devenv.PostGenerationMessage(true, "")

	if !strings.Contains(got, "direnv allow") {
		t.Error("direnv message does not contain 'direnv allow'")
	}
}

func TestPostGenerationMessage_Native(t *testing.T) {
	got := devenv.PostGenerationMessage(false, "bash")

	// Should delegate to NativeActivationInstructions, so it must include
	// the bash-specific hook instruction.
	if !strings.Contains(got, `devenv hook bash`) {
		t.Error("native message does not contain bash hook instructions")
	}
	if !strings.Contains(got, "~/.bashrc") {
		t.Error("native message does not contain '~/.bashrc'")
	}
}
