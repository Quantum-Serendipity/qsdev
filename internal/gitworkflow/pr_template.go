package gitworkflow

import (
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/sliceutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// GeneratePRTemplate produces a GitHub pull request template tailored to
// the project's detected ecosystems and enabled security tools. It uses the
// Skip strategy: a repository that already has a PR template keeps it.
func GeneratePRTemplate(answers types.WizardAnswers) (*types.GeneratedFile, error) {
	var b strings.Builder

	b.WriteString("## Summary\n\n")
	b.WriteString("<!-- Describe what changed and why -->\n\n")

	b.WriteString("## Type of Change\n\n")
	b.WriteString("- [ ] Feature\n")
	b.WriteString("- [ ] Bug fix\n")
	b.WriteString("- [ ] Refactor\n")
	b.WriteString("- [ ] Documentation\n")
	b.WriteString("- [ ] Chore\n\n")

	// Security section when security hardening is enabled.
	if answers.ComplianceLevel != "" || hasSecurityTools(answers) {
		b.WriteString("## Security Checklist\n\n")
		b.WriteString("- [ ] No secrets or credentials in code\n")
		b.WriteString("- [ ] Dependency versions pinned\n")
		b.WriteString("- [ ] SAST scan passes\n")
		b.WriteString("- [ ] New endpoints require authentication\n\n")
	}

	b.WriteString("## Testing\n\n")
	b.WriteString("- [ ] Unit tests added/updated\n")
	b.WriteString("- [ ] Manual testing performed\n")

	// Per-ecosystem items: each selected language's own verification commands.
	for _, cmd := range verificationCommands(answers, ecosystem.DefaultRegistry()) {
		fmt.Fprintf(&b, "- [ ] `%s` passes\n", cmd)
	}

	// Docker section if Dockerfile detected.
	if answers.Detected.HasDockerfile {
		b.WriteString("- [ ] Container image builds\n")
		b.WriteString("- [ ] Image scanned for vulnerabilities\n")
	}

	b.WriteString("\n## Breaking Changes\n\n")
	b.WriteString("<!-- List breaking changes or write \"None\" -->\n\n")

	b.WriteString("## Reviewer Notes\n\n")
	b.WriteString("<!-- Any context the reviewer should know -->\n")

	return &types.GeneratedFile{
		Path:     ".github/pull_request_template.md",
		Content:  []byte(b.String()),
		Mode:     fileutil.ModeReadWrite,
		Strategy: types.Skip,
	}, nil
}

// verificationCommands returns the build, test, lint, type-check and format
// commands the ecosystem modules of the selected languages declare, in language
// order and without duplicates. Sourcing them from the modules keeps the
// checklist in step with every catalog language and its configured package
// manager, instead of a hand-maintained per-language table.
func verificationCommands(answers types.WizardAnswers, registry *ecosystem.Registry) []string {
	var cmds []string
	for _, lang := range answers.Languages {
		mod, ok := registry.ByName(lang.Name)
		if !ok {
			continue
		}
		cmds = append(cmds, mod.VerificationCommands(ecosystem.ToModuleConfig(lang)).All()...)
	}
	return sliceutil.Dedup(cmds)
}

func hasSecurityTools(answers types.WizardAnswers) bool {
	for _, tool := range []string{"semgrep", "gitleaks", "container-security"} {
		if answers.EnabledTools[tool] {
			return true
		}
	}
	return false
}
