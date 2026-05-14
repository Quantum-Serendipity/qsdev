package gitworkflow

import (
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/cigeneration"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

// GenerateLabelerConfig produces the GitHub Actions labeler configuration
// (labeler.yml) and the workflow file that runs it. Labels are tailored
// to the project's detected ecosystems.
func GenerateLabelerConfig(answers types.WizardAnswers) ([]types.GeneratedFile, error) {
	// Build labeler.yml
	var lb strings.Builder

	// Standard labels.
	lb.WriteString("documentation:\n")
	lb.WriteString("  - changed-files:\n")
	lb.WriteString("      - any-glob-to-any-file: ['docs/**', '*.md', 'README*']\n")
	lb.WriteString("\n")

	lb.WriteString("infrastructure:\n")
	lb.WriteString("  - changed-files:\n")
	lb.WriteString("      - any-glob-to-any-file: ['devenv.nix', 'devenv.yaml', '.envrc', 'flake.*', 'Dockerfile*', '.github/**']\n")
	lb.WriteString("\n")

	lb.WriteString("security:\n")
	lb.WriteString("  - changed-files:\n")
	lb.WriteString("      - any-glob-to-any-file: ['.semgrep.yml', '.gitleaks.toml', '.scancode.yml']\n")
	lb.WriteString("\n")

	lb.WriteString("dependencies:\n")
	lb.WriteString("  - changed-files:\n")
	lb.WriteString("      - any-glob-to-any-file: ['**/package-lock.json', '**/yarn.lock', '**/pnpm-lock.yaml', '**/go.sum', '**/Cargo.lock', '**/requirements*.txt', '**/poetry.lock', '**/Gemfile.lock', '**/composer.lock']\n")
	lb.WriteString("\n")

	// Per-ecosystem labels.
	ecoGlobs := map[string]string{
		"go":         "'**/*.go'",
		"javascript": "'**/*.{js,jsx,ts,tsx}'",
		"python":     "'**/*.py'",
		"rust":       "'**/*.rs'",
		"java":       "'**/*.java'",
		"ruby":       "'**/*.rb'",
		"php":        "'**/*.php'",
		"dotnet":     "'**/*.cs'",
		"elixir":     "'**/*.{ex,exs}'",
		"swift":      "'**/*.swift'",
	}

	for _, lang := range answers.Languages {
		glob, ok := ecoGlobs[lang.Name]
		if !ok {
			continue
		}
		fmt.Fprintf(&lb, "%s:\n", lang.Name)
		lb.WriteString("  - changed-files:\n")
		fmt.Fprintf(&lb, "      - any-glob-to-any-file: [%s]\n", glob)
		lb.WriteString("\n")
	}

	// Build workflow file.
	var wf strings.Builder
	wf.WriteString("name: PR Labeler\n")
	wf.WriteString("on:\n")
	wf.WriteString("  pull_request_target:\n")
	wf.WriteString("    types: [opened, synchronize]\n")
	wf.WriteString("\n")
	wf.WriteString("permissions:\n")
	wf.WriteString("  contents: read\n")
	wf.WriteString("  pull-requests: write\n")
	wf.WriteString("\n")
	wf.WriteString("jobs:\n")
	wf.WriteString("  label:\n")
	wf.WriteString("    runs-on: ubuntu-latest\n")
	wf.WriteString("    steps:\n")
	fmt.Fprintf(&wf, "      - uses: %s %s\n", cigeneration.ActionLabeler.String(), cigeneration.ActionLabeler.Comment())
	wf.WriteString("        with:\n")
	wf.WriteString("          repo-token: ${{ secrets.GITHUB_TOKEN }}\n")

	return []types.GeneratedFile{
		{
			Path:     ".github/labeler.yml",
			Content:  []byte(lb.String()),
			Mode:     0o644,
			Strategy: types.Overwrite,
		},
		{
			Path:     ".github/workflows/labeler.yml",
			Content:  []byte(wf.String()),
			Mode:     0o644,
			Strategy: types.Overwrite,
		},
	}, nil
}
