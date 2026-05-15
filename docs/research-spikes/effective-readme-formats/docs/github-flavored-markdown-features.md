# GitHub Flavored Markdown: Complete Feature Guide
- **Source**: https://markdownftw.com/blog/github-flavored-markdown
- **Retrieved**: 2026-05-15

## Extensions Beyond Standard Markdown

GFM extends CommonMark with eight key additions: "Tables with alignment support, Task lists (checkboxes), Strikethrough text, Autolinked URLs and references, Fenced code blocks with syntax highlighting, Emoji shortcodes, Footnotes, Alerts/admonitions."

## Features in README Files

All GFM capabilities function in README files, though certain features prove particularly valuable there. Documentation writers should prioritize alerts for highlighting crucial information and tables for presenting structured data comparisons.

## Tables

Tables employ pipes and dashes with required separator rows. Column alignment uses colons: `:---` for left-aligned, `:---:` for center, and `---:` for right-aligned columns.

## Task Lists

Interactive checkboxes render directly in issues and pull requests. "GitHub shows a progress bar at the top of the issue when task lists are present (e.g., '3/5 tasks complete')." They function without manual Markdown editing when toggled on GitHub.

## Strikethrough

Double tildes create struck-through text: `~~example~~`. This GFM-specific extension lacks CommonMark support.

## Syntax-Highlighted Code Blocks

Language identifiers after opening backticks enable highlighting across hundreds of languages, from TypeScript to Dockerfile.

## Alerts/Admonitions

Blockquote syntax creates colored, icon-labeled alerts: `> [!NOTE]`, `> [!TIP]`, `> [!WARNING]`, `> [!IMPORTANT]`, and `> [!CAUTION]`.

## Mermaid Diagrams

GitHub renders flowcharts and sequence diagrams directly from code blocks marked with the `mermaid` identifier, enabling version-controlled diagrams.

## Additional GitHub Features

Beyond the formal GFM specification, GitHub supports LaTeX math expressions, GeoJSON/TopoJSON interactive maps, and STL 3D model rendering.

## Autolinks

Automatic linkification converts URLs, issue references (`#123`), cross-repository references, user mentions, and commit SHAs into clickable elements.
