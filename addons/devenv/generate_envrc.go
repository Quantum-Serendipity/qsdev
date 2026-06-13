package devenv

import (
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// envrcContent returns the standard .envrc file for direnv-based activation.
func envrcContent() string {
	return "#!/usr/bin/env bash\n" +
		"# " + branding.GeneratedBy() + ".\n" +
		"# Run 'direnv allow' to activate.\n" +
		"\n" +
		"eval \"$(devenv direnvrc)\"\n" +
		"use devenv\n"
}

// GenerateEnvrc returns a GeneratedFile for .envrc when direnv is enabled,
// or nil when native activation is selected.
func GenerateEnvrc(answers types.WizardAnswers) *types.GeneratedFile {
	if !answers.Direnv {
		return nil
	}
	return &types.GeneratedFile{
		Path:     ".envrc",
		Content:  []byte(envrcContent()),
		Mode:     fileutil.ModeReadWrite,
		Strategy: types.Skip,
	}
}

// NativeActivationInstructions returns shell-specific instructions for
// devenv 2.0+ native activation (bash, zsh, fish).
func NativeActivationInstructions(shell string) string {
	switch strings.ToLower(shell) {
	case "bash":
		return `Add the following line to ~/.bashrc:

  eval "$(devenv hook bash)"

Then restart your shell or run: source ~/.bashrc`

	case "zsh":
		return `Add the following line to ~/.zshrc:

  eval "$(devenv hook zsh)"

Then restart your shell or run: source ~/.zshrc`

	case "fish":
		return `Add the following line to ~/.config/fish/config.fish:

  devenv hook fish | source

Then restart your shell or run: source ~/.config/fish/config.fish`

	default:
		return `Add the devenv hook to your shell configuration:

  bash: add 'eval "$(devenv hook bash)"' to ~/.bashrc
  zsh:  add 'eval "$(devenv hook zsh)"' to ~/.zshrc
  fish: add 'devenv hook fish | source' to ~/.config/fish/config.fish

Then restart your shell to activate.`
	}
}

// PostGenerationMessage returns the appropriate next-step message based on
// whether direnv or native activation was chosen.
func PostGenerationMessage(direnvEnabled bool, shell string) string {
	if direnvEnabled {
		return "Generated .envrc for direnv activation.\nRun 'direnv allow' in your project directory to activate the environment."
	}
	return fmt.Sprintf(
		"Native activation selected (no .envrc generated).\n%s",
		NativeActivationInstructions(shell),
	)
}
