package devenv

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/shellintegration"
)

const binaryName = "qsdev"

func completionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion",
		Short: "Generate or install shell completions for qsdev",
		Long: `Generate shell completion scripts for qsdev.

To load completions in your current session:

  bash:       source <(qsdev completion bash)
  zsh:        source <(qsdev completion zsh)
  fish:       qsdev completion fish | source
  powershell: qsdev completion powershell | Out-String | Invoke-Expression

To install completions permanently, use "qsdev completion install".`,
	}

	cmd.AddCommand(
		completionBashCmd(),
		completionZshCmd(),
		completionFishCmd(),
		completionPowershellCmd(),
		completionInstallCmd(),
	)

	return cmd
}

func completionBashCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bash",
		Short: "Generate bash completion script",
		Long:  "Output bash completion script to stdout.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Root().GenBashCompletionV2(cmd.OutOrStdout(), true)
		},
	}
}

func completionZshCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "zsh",
		Short: "Generate zsh completion script",
		Long:  "Output zsh completion script to stdout.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Root().GenZshCompletion(cmd.OutOrStdout())
		},
	}
}

func completionFishCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fish",
		Short: "Generate fish completion script",
		Long:  "Output fish completion script to stdout.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Root().GenFishCompletion(cmd.OutOrStdout(), true)
		},
	}
}

func completionPowershellCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "powershell",
		Short: "Generate PowerShell completion script",
		Long:  "Output PowerShell completion script to stdout.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Root().GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
		},
	}
}

func completionInstallCmd() *cobra.Command {
	var shellFlag string

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install shell completions permanently",
		Long: `Install completion scripts and update your shell RC file so completions
are loaded automatically in new shell sessions.

If --shell is not specified, the current shell is auto-detected from the
SHELL environment variable.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			shell := shellFlag
			if shell == "" {
				shell = detectShell()
				if shell == "" {
					return fmt.Errorf("could not auto-detect shell; use --shell to specify one")
				}
			}

			rcFile := defaultRCFile(shell)
			if rcFile == "" {
				return fmt.Errorf("could not determine RC file for shell %q", shell)
			}

			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("determining home directory: %w", err)
			}

			installer := &shellintegration.CompletionInstaller{
				BinaryName: binaryName,
				HomeDir:    home,
			}

			if err := installer.Install(cmd.Root(), shell, rcFile); err != nil {
				return fmt.Errorf("installing completions: %w", err)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Shell completions installed for %s.\n", shell)
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Restart your shell or source your RC file to activate.")
			return nil
		},
	}

	cmd.Flags().StringVar(&shellFlag, "shell", "", "Shell to install completions for (bash, zsh, fish, powershell)")

	return cmd
}

// detectShell returns the shell name from the SHELL environment variable.
func detectShell() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		return ""
	}
	// Extract basename: /usr/bin/zsh -> zsh
	for i := len(shell) - 1; i >= 0; i-- {
		if shell[i] == '/' {
			return shell[i+1:]
		}
	}
	return shell
}

// defaultRCFile returns the conventional RC file path for the given shell.
func defaultRCFile(shell string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	// Normalize: handle full paths and case.
	name := shell
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '/' {
			name = name[i+1:]
			break
		}
	}

	switch name {
	case "bash":
		return home + "/.bashrc"
	case "zsh":
		return home + "/.zshrc"
	case "fish":
		return home + "/.config/fish/config.fish"
	case "pwsh", "powershell":
		return home + "/.config/powershell/Microsoft.PowerShell_profile.ps1"
	default:
		return ""
	}
}
