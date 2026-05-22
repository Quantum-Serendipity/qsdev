package shellintegration

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"fastcat.org/go/gdev/addons/bootstrap/textedit"
	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

func completionMarkerStart() string { return "# " + branding.Get().AppName + ": shell completions" }
func completionMarkerEnd() string   { return "# " + branding.Get().AppName + " end shell completions" }

// CompletionInstaller writes shell completion files and updates RC files
// so that completions are loaded automatically in new shell sessions.
type CompletionInstaller struct {
	// BinaryName is the name of the CLI binary (e.g. "qsdev").
	BinaryName string
	// HomeDir is the user's home directory, used to derive completion file
	// paths. If empty, os.UserHomeDir() is used.
	HomeDir string
}

// Install generates a completion script for the given shell from rootCmd and
// writes it to the appropriate location. For shells that require RC file
// modifications (bash, zsh, powershell), it also updates rcFile with the
// necessary source/fpath lines using idempotent marker-based editing.
func (c *CompletionInstaller) Install(rootCmd *cobra.Command, shell string, rcFile string) error {
	home := c.HomeDir
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("determining home directory: %w", err)
		}
	}

	normalized := normalizeShellName(shell)

	switch normalized {
	case "bash":
		return c.installBash(rootCmd, home, rcFile)
	case "zsh":
		return c.installZsh(rootCmd, home, rcFile)
	case "fish":
		return c.installFish(rootCmd, home)
	case "pwsh", "powershell":
		return c.installPowershell(rootCmd, home, rcFile)
	default:
		return fmt.Errorf("unsupported shell %q for completion installation", shell)
	}
}

func (c *CompletionInstaller) installBash(rootCmd *cobra.Command, home string, rcFile string) error {
	completionDir := filepath.Join(home, "."+c.BinaryName, "completions")
	completionFile := filepath.Join(completionDir, c.BinaryName+".bash")

	if err := writeCompletionFile(completionFile, func(buf *bytes.Buffer) error {
		return rootCmd.GenBashCompletionV2(buf, true)
	}); err != nil {
		return err
	}

	// Add source line to RC file.
	sourceLine := fmt.Sprintf(`[ -f "%s" ] && source "%s"`, completionFile, completionFile)
	return spliceRCFile(rcFile, sourceLine)
}

func (c *CompletionInstaller) installZsh(rootCmd *cobra.Command, home string, rcFile string) error {
	completionDir := filepath.Join(home, "."+c.BinaryName, "completions")
	completionFile := filepath.Join(completionDir, "_"+c.BinaryName)

	if err := writeCompletionFile(completionFile, func(buf *bytes.Buffer) error {
		return rootCmd.GenZshCompletion(buf)
	}); err != nil {
		return err
	}

	// Add fpath and compinit to RC file.
	fpathLine := fmt.Sprintf(`fpath=("%s" $fpath)`, completionDir)
	compinitLine := `autoload -Uz compinit && compinit -C`
	return spliceRCFileMulti(rcFile, fpathLine, compinitLine)
}

func (c *CompletionInstaller) installFish(rootCmd *cobra.Command, home string) error {
	// Fish auto-loads completions from ~/.config/fish/completions/
	completionDir := filepath.Join(home, ".config", "fish", "completions")
	completionFile := filepath.Join(completionDir, c.BinaryName+".fish")

	return writeCompletionFile(completionFile, func(buf *bytes.Buffer) error {
		return rootCmd.GenFishCompletion(buf, true)
	})
	// No RC file edit needed for fish.
}

func (c *CompletionInstaller) installPowershell(rootCmd *cobra.Command, home string, rcFile string) error {
	completionDir := filepath.Join(home, "."+c.BinaryName, "completions")
	completionFile := filepath.Join(completionDir, c.BinaryName+".ps1")

	if err := writeCompletionFile(completionFile, func(buf *bytes.Buffer) error {
		return rootCmd.GenPowerShellCompletionWithDesc(buf)
	}); err != nil {
		return err
	}

	// Source the completion file from the PowerShell profile.
	sourceLine := fmt.Sprintf(`. "%s"`, completionFile)
	return spliceRCFile(rcFile, sourceLine)
}

// writeCompletionFile generates completion output into a buffer using genFn,
// then writes it to the specified path, creating parent directories as needed.
func writeCompletionFile(path string, genFn func(*bytes.Buffer) error) error {
	var buf bytes.Buffer
	if err := genFn(&buf); err != nil {
		return fmt.Errorf("generating completion script: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating completion directory %s: %w", dir, err)
	}

	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("writing completion file %s: %w", path, err)
	}
	return nil
}

// spliceRCFile uses textedit.SpliceRange to idempotently insert a single
// content line into the RC file between marker comments.
func spliceRCFile(rcFile string, contentLine string) error {
	if rcFile == "" {
		return fmt.Errorf("rcFile must not be empty")
	}

	editor := textedit.SpliceRange(
		completionMarkerStart(),
		contentLine,
		completionMarkerEnd(),
	)

	if err := os.MkdirAll(filepath.Dir(rcFile), 0o755); err != nil {
		return fmt.Errorf("creating parent directory for %s: %w", rcFile, err)
	}

	_, err := textedit.EditFile(rcFile, editor)
	if err != nil {
		return fmt.Errorf("editing %s: %w", rcFile, err)
	}
	return nil
}

// spliceRCFileMulti inserts multiple content lines between markers. This is
// used for zsh where we need both an fpath line and a compinit line.
func spliceRCFileMulti(rcFile string, lines ...string) error {
	if rcFile == "" {
		return fmt.Errorf("rcFile must not be empty")
	}

	spliceArgs := make([]string, 0, len(lines)+2)
	spliceArgs = append(spliceArgs, completionMarkerStart())
	spliceArgs = append(spliceArgs, lines...)
	spliceArgs = append(spliceArgs, completionMarkerEnd())

	editor := textedit.SpliceRange(spliceArgs...)

	if err := os.MkdirAll(filepath.Dir(rcFile), 0o755); err != nil {
		return fmt.Errorf("creating parent directory for %s: %w", rcFile, err)
	}

	_, err := textedit.EditFile(rcFile, editor)
	if err != nil {
		return fmt.Errorf("editing %s: %w", rcFile, err)
	}
	return nil
}
