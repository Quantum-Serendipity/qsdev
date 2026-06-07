package claudecode

import (
	"fmt"
	"slices"

	"github.com/Quantum-Serendipity/qsdev/internal/tmpl"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

type lookupDocsTemplateData struct {
	DocSources []docSource
}

type docSource struct {
	Name       string
	Tag        string
	Priority   int
	ServerName string
	Available  bool
	UseCase    string
}

var docSourceDefs = []docSource{
	{Name: "DevDocs", Tag: "[DevDocs]", Priority: 1, ServerName: "local-docs-devdocs", UseCase: "API/library reference documentation"},
	{Name: "Stack Exchange", Tag: "[Stack Exchange]", Priority: 2, ServerName: "local-docs-zim", UseCase: "Q&A, troubleshooting, community solutions"},
	{Name: "man pages", Tag: "[man page]", Priority: 3, ServerName: "man-pages", UseCase: "System tools, CLI flags, POSIX utilities"},
	{Name: "NixOS options", Tag: "[NixOS]", Priority: 4, ServerName: "mcp-nixos", UseCase: "Nix/NixOS configuration options and packages"},
	{Name: "Web (Context7)", Tag: "[Web]", Priority: 5, ServerName: "context7", UseCase: "Library docs when local sources are insufficient"},
}

func generateLookupDocsSkill(answers types.WizardAnswers) (*types.GeneratedFile, error) {
	sources := make([]docSource, len(docSourceDefs))
	for i, def := range docSourceDefs {
		sources[i] = def
		sources[i].Available = slices.Contains(answers.MCPServers, def.ServerName)
	}

	data := lookupDocsTemplateData{
		DocSources: sources,
	}

	renderer, err := tmpl.NewMarkdownRenderer(templateFS, "templates")
	if err != nil {
		return nil, fmt.Errorf("creating renderer: %w", err)
	}

	content, err := renderer.Render("skills/lookup-docs/SKILL.md", data)
	if err != nil {
		return nil, fmt.Errorf("rendering lookup-docs skill: %w", err)
	}

	return &types.GeneratedFile{
		Path:    ".claude/skills/lookup-docs/SKILL.md",
		Content: content,
		Mode:    0o644,
		Owner:   "lookup-docs",
	}, nil
}
