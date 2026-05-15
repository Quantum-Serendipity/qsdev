package devinit

import (
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// ExitError is a sentinel error that carries a process exit code. Command
// handlers return this instead of calling os.Exit directly so that deferred
// cleanup runs and tests can inspect the code without terminating the process.
type ExitError struct {
	Code int
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("exit code %d", e.Code)
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
		steps = append(steps, "Review .claude/settings.json and CLAUDE.md for Claude Code configuration.")
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

	return b.String()
}
