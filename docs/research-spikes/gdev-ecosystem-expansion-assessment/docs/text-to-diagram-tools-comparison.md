# Text-to-Diagram Tools Comparison
- **Source**: https://text-to-diagram.com/?example=text
- **Retrieved**: 2026-05-14

## Tools Covered

Comparison of D2, Mermaid, PlantUML, and Graphviz.

## Key Features Compared

**Licensing & Availability:**
- D2 uses MPL 2.0 licensing and was released in 2022
- PlantUML operates under GPL 3.0 and launched in 2009
- Mermaid uses MIT license
- Graphviz uses EPL 1.0

**Technical Implementation:**
- D2 is compiled in Go — single binary, no runtime deps
- PlantUML uses Java as its foundation — requires JRE
- Mermaid is JavaScript/Node.js — requires Node for CLI (mmdc)
- Graphviz is C — lightweight native binary

**Editor Integration:**
- D2 provides creator-made extensions for VSCode and Vim
- PlantUML relies on community-developed extensions across multiple platforms including VSCode, Vim, and Atom
- Mermaid has native rendering in GitHub, GitLab, Notion, and dozens of other platforms
- Graphviz has wide IDE support through established ecosystem

**Diagram Capabilities:**
Both D2 and PlantUML support sequence diagrams, SQL tables (ERDs), markdown text, syntax-highlighted code snippets, and LaTeX rendering.

**User Experience Features:**
Supported features include friendly error messages, configurable themes, accessibility options, autoformatting, and responsive dark mode rendering.

**Export Options:**
PlantUML requires additional installs for PDF exports. Both tools support various output formats (SVG, PNG, PDF).
