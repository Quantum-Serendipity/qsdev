package gitworkflow

import (
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// GeneratePRTemplate produces a GitHub pull request template tailored to
// the project's detected ecosystems and enabled security tools.
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

	// Per-ecosystem items.
	for _, lang := range answers.Languages {
		switch lang.Name {
		case "go":
			b.WriteString("- [ ] `go vet ./...` passes\n")
			b.WriteString("- [ ] `go test ./...` passes\n")
		case "javascript", "typescript":
			b.WriteString("- [ ] TypeScript types exported correctly\n")
			b.WriteString("- [ ] `npm audit` clean\n")
		case "python":
			b.WriteString("- [ ] Type hints added\n")
			b.WriteString("- [ ] Linter passes\n")
		case "rust":
			b.WriteString("- [ ] `cargo clippy` clean\n")
			b.WriteString("- [ ] `cargo test` passes\n")
		case "java":
			b.WriteString("- [ ] Build passes\n")
			b.WriteString("- [ ] Static analysis clean\n")
		case "dotnet":
			b.WriteString("- [ ] `dotnet build` passes\n")
			b.WriteString("- [ ] `dotnet test` passes\n")
		}
	}

	// Docker section if Dockerfile detected.
	if answers.Detected.HasDockerfile {
		b.WriteString("- [ ] Docker image builds\n")
		b.WriteString("- [ ] Image scanned for vulnerabilities\n")
	}

	b.WriteString("\n## Breaking Changes\n\n")
	b.WriteString("<!-- List breaking changes or write \"None\" -->\n\n")

	b.WriteString("## Reviewer Notes\n\n")
	b.WriteString("<!-- Any context the reviewer should know -->\n")

	return &types.GeneratedFile{
		Path:     ".github/pull_request_template.md",
		Content:  []byte(b.String()),
		Mode:     0o644,
		Strategy: types.Overwrite,
	}, nil
}

func hasSecurityTools(answers types.WizardAnswers) bool {
	for _, tool := range []string{"semgrep", "gitleaks", "container-security"} {
		if answers.EnabledTools[tool] {
			return true
		}
	}
	return false
}
