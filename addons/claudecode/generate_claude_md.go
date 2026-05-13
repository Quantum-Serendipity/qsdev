package claudecode

import (
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/tmpl"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

// ClaudeMdTemplateData holds all data required to render the CLAUDE.md template.
type ClaudeMdTemplateData struct {
	ProjectName        string
	ProjectDescription string
	ArchitectureNotes  string
	Languages          []string // Display names: "Go", "JavaScript/TypeScript", etc.
	BuildCommands      []string
	TestCommands       []string
	LintCommands       []string
	HasSecurityHooks   bool
	PackageManagers    []string // "npm", "pip", "cargo" for security section specifics
}

// languageCommands maps ecosystem canonical names to their default build, test,
// and lint commands.
var languageCommands = map[string]struct {
	build []string
	test  []string
	lint  []string
}{
	"go":         {build: []string{"go build ./..."}, test: []string{"go test ./..."}, lint: []string{"golangci-lint run"}},
	"javascript": {build: []string{"npm run build"}, test: []string{"npm test"}, lint: []string{"npm run lint"}},
	"python":     {build: nil, test: []string{"python -m pytest"}, lint: []string{"ruff check ."}},
	"rust":       {build: []string{"cargo build"}, test: []string{"cargo test"}, lint: []string{"cargo clippy"}},
	"java":       {build: []string{"mvn compile"}, test: []string{"mvn test"}, lint: nil},
	"dotnet":     {build: []string{"dotnet build"}, test: []string{"dotnet test"}, lint: nil},
	"docker":     {build: []string{"docker build ."}, test: nil, lint: []string{"hadolint Dockerfile"}},
	"terraform":  {build: nil, test: []string{"terraform validate"}, lint: []string{"terraform plan", "tflint"}},
}

// BuildClaudeMdData assembles all template data from wizard answers and ecosystem
// modules. It maps language choices to display names, derives default commands,
// and collects package manager metadata.
func BuildClaudeMdData(answers types.WizardAnswers, registry *ecosystem.Registry) *ClaudeMdTemplateData {
	data := &ClaudeMdTemplateData{
		ProjectName:      answers.ProjectName,
		HasSecurityHooks: answers.Hooks.SafetyBlock,
	}

	var buildCmds, testCmds, lintCmds []string

	for _, lang := range answers.Languages {
		mod, ok := registry.ByName(lang.Name)
		if ok {
			data.Languages = append(data.Languages, mod.DisplayName())
			for _, pm := range mod.PackageManagers() {
				data.PackageManagers = append(data.PackageManagers, pm.Name)
			}
		}

		if cmds, exists := languageCommands[lang.Name]; exists {
			buildCmds = append(buildCmds, cmds.build...)
			testCmds = append(testCmds, cmds.test...)
			lintCmds = append(lintCmds, cmds.lint...)
		}
	}

	data.BuildCommands = dedup(buildCmds)
	data.TestCommands = dedup(testCmds)
	data.LintCommands = dedup(lintCmds)
	data.PackageManagers = dedup(data.PackageManagers)

	// Generate a default description if none provided.
	if data.ProjectName != "" && len(data.Languages) > 0 {
		data.ProjectDescription = fmt.Sprintf("%s — a %s project.", data.ProjectName, strings.Join(data.Languages, ", "))
	} else if data.ProjectName != "" {
		data.ProjectDescription = fmt.Sprintf("%s development environment.", data.ProjectName)
	} else {
		data.ProjectDescription = "Development environment managed by gdev."
	}

	return data
}

// GenerateClaudeMd produces a GeneratedFile containing the rendered CLAUDE.md
// from wizard answers and ecosystem module registry.
func GenerateClaudeMd(answers types.WizardAnswers, registry *ecosystem.Registry) (*types.GeneratedFile, error) {
	data := BuildClaudeMdData(answers, registry)

	renderer, err := tmpl.NewMarkdownRenderer(templateFS, "templates")
	if err != nil {
		return nil, fmt.Errorf("creating Markdown renderer: %w", err)
	}

	content, err := renderer.Render("claude-md", data)
	if err != nil {
		return nil, fmt.Errorf("rendering CLAUDE.md template: %w", err)
	}

	return &types.GeneratedFile{
		Path:     "CLAUDE.md",
		Content:  content,
		Mode:     0o644,
		Strategy: types.SectionMarker,
	}, nil
}
