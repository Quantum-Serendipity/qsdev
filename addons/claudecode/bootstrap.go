package claudecode

import (
	"fmt"
	"os"
	"os/exec"

	"fastcat.org/go/gdev/addons/bootstrap"

	"fastcat.org/go/gdev-secure-devenv-bootstrap/internal/toolcheck"
)

// StepNameInstallClaude is the name of the bootstrap step that installs Claude Code.
const StepNameInstallClaude = "Install Claude Code"

// InstallClaudeStep returns a bootstrap step that ensures Claude Code is installed.
func InstallClaudeStep() *bootstrap.Step {
	return bootstrap.NewStep(
		StepNameInstallClaude,
		installClaude,
		bootstrap.SimFunc(simInstallClaude),
		bootstrap.SkipInContainer(),
	)
}

func installClaude(ctx *bootstrap.Context) error {
	info := toolcheck.Detect(ctx, "claude", "--version")
	if info.Found {
		fmt.Printf("Claude Code already installed: %s (%s)\n", info.Version, info.Path)
		return nil
	}

	if _, err := exec.LookPath("npm"); err == nil {
		fmt.Println("Installing Claude Code via npm...")
		cmd := exec.CommandContext(ctx, "npm", "install", "-g", "@anthropic-ai/claude-code")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("installing Claude Code: %w", err)
		}
		fmt.Println("Claude Code installed successfully.")
		return nil
	}

	fmt.Println("Claude Code is not installed and npm is not available.")
	fmt.Println("Install options:")
	fmt.Println("  1. Install Node.js: https://nodejs.org/")
	fmt.Println("     Then: npm install -g @anthropic-ai/claude-code")
	fmt.Println("  2. Direct: https://docs.anthropic.com/en/docs/claude-code/overview")
	return fmt.Errorf("Claude Code not installed; manual installation required")
}

func simInstallClaude(ctx *bootstrap.Context) error {
	info := toolcheck.Detect(ctx, "claude", "--version")
	if info.Found {
		fmt.Printf("Claude Code already installed: %s (%s)\n", info.Version, info.Path)
		return nil
	}
	if _, err := exec.LookPath("npm"); err == nil {
		fmt.Println("Would run: npm install -g @anthropic-ai/claude-code")
	} else {
		fmt.Println("Would need manual Claude Code installation (npm not available)")
	}
	return nil
}
