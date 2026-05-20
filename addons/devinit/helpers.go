package devinit

import (
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/tier"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// ExitError is a sentinel error that carries a process exit code. Command
// handlers return this instead of calling os.Exit directly so that deferred
// cleanup runs and tests can inspect the code without terminating the process.
type ExitError struct {
	Code int
}

func (e *ExitError) Error() string {
	return ""
}

func (e *ExitError) ExitCode() int {
	return e.Code
}

// postGenerationMessage returns a human-readable summary of next steps after
// qsdev init has generated files. It adapts the message based on what was
// generated (devenv, Claude Code, or both).
func postGenerationMessage(answers types.WizardAnswers, devenvGenerated, claudeGenerated bool) string {
	var steps []string

	if devenvGenerated {
		if answers.Direnv {
			steps = append(steps, "Run 'direnv allow' to activate the development environment.")
		} else {
			steps = append(steps, "Run 'devenv shell' to activate the development environment.")
		}
	}

	if claudeGenerated {
		steps = append(steps, fmt.Sprintf("Run '%s list' to see available tools and '%s enable <tool>' to add more.", branding.Get().AppName, branding.Get().AppName))
	}

	if devenvGenerated && claudeGenerated {
		steps = append(steps, "Both devenv and Claude Code configurations have been generated.")
	}

	if len(steps) == 0 {
		return "No files were generated."
	}

	var b strings.Builder
	fmt.Fprintln(&b, "Next steps:")
	for _, step := range steps {
		fmt.Fprintf(&b, "  - %s\n", step)
	}

	if answers.Tier != "" && answers.Tier != "full" {
		pos := tier.Position(answers.Tier)
		next, ok := tier.NextTier(answers.Tier)
		if ok && pos > 0 {
			fmt.Fprintf(&b, "\nSecurity tier: %s (%d/%d)\n", answers.Tier, pos, tier.Total())
			fmt.Fprintf(&b, "  Tip: Run '%s init --tier %s --dry-run' to preview the next tier.\n", branding.Get().AppName, next)
		}
	}

	return b.String()
}
