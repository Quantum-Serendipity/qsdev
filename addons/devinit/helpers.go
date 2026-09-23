package devinit

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/exitcode"
	"github.com/Quantum-Serendipity/qsdev/internal/tier"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// ExitError is the shared exit-code error (internal/exitcode.Error). Command
// handlers return it instead of calling os.Exit directly so that deferred
// cleanup runs and tests can inspect the code without terminating the process.
// Handlers that have already printed their report leave Message empty and only
// set Code.
type ExitError = exitcode.Error

// claudeMDPath is the project-relative path of the generated CLAUDE.md.
const claudeMDPath = "CLAUDE.md"

// postGenerationMessage returns a human-readable summary of next steps after
// qsdev init has generated files. It adapts the message to what acc actually
// generated: devenv, Claude Code, or both, and CLAUDE.md only when it is
// among the files (supply-chain-only produces settings and hooks only).
func postGenerationMessage(answers types.WizardAnswers, acc accumulatorResult) string {
	var steps []string
	devenvGenerated, claudeGenerated := acc.devenvGenerated, acc.claudeGenerated

	if devenvGenerated {
		if answers.Direnv {
			steps = append(steps, "Run 'direnv allow' to activate the development environment.")
		} else {
			steps = append(steps, "Run 'devenv shell' to activate the development environment.")
		}
	}

	if claudeGenerated {
		if slices.ContainsFunc(acc.allFiles, func(f types.GeneratedFile) bool { return f.Path == claudeMDPath }) {
			steps = append(steps, "Review "+claudeMDPath+" to customize project documentation.")
		}
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
			fmt.Fprintf(&b, "  Tip: Run '%s' to preview the next tier.\n", tier.PreviewCommand(branding.Get().AppName, next))
		}
	}

	return b.String()
}

// missingAnswerFields lists the required answer fields that are unset, using
// the same rule as types.WizardAnswers.IsComplete (minus its Confirmed gate):
// at least one language, and a permission level or tier when Claude Code is
// enabled (a tier implies its permission preset). Both the --yes and the
// --answers-file completeness checks report through it so they cannot drift.
func missingAnswerFields(answers types.WizardAnswers) []string {
	var missing []string
	if len(answers.Languages) == 0 {
		missing = append(missing, "languages (at least one language is required)")
	}
	if answers.ClaudeCode && answers.PermissionLevel == "" && answers.Tier == "" {
		missing = append(missing, "permission_level or tier (required when claude_code is true)")
	}
	return missing
}

func incompleteAnswersMessage(answers types.WizardAnswers) string {
	var b strings.Builder
	for _, field := range missingAnswerFields(answers) {
		fmt.Fprintf(&b, "  - %s\n", field)
	}
	return b.String()
}
