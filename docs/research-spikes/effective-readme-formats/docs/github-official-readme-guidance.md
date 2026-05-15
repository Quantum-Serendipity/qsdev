# GitHub's Official README Guidance
- **Source**: https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/about-readmes
- **Retrieved**: 2026-05-15

## Recommended Content

GitHub recommends READMEs typically include:

- "What the project does"
- "Why the project is useful"
- "How users can get started with the project"
- "Where users can get help with your project"
- "Who maintains and contributes to the project"

## File Placement

GitHub automatically recognizes READMEs in three locations, prioritized in this order: the `.github` directory, the repository root, or the `docs` directory. The platform will surface the first one found.

## Supported Features

**Auto-generated Table of Contents**: GitHub creates a table of contents based on section headings, accessible via an outline menu button on rendered markdown files.

**Section Links**: Users can hover over headings to reveal link icons, enabling direct navigation to specific sections within a document.

**Relative Links and Image Paths**: GitHub supports relative links (like `docs/CONTRIBUTING.md`) and automatically transforms them based on the current branch, ensuring consistency across clones.

## Important Constraints

- Content exceeding 500 KiB will be truncated
- Profile READMEs appear automatically when added to a public repository matching your username

## Structure Guidance

GitHub advises against multi-line link text and recommends using relative rather than absolute links for repository navigation.

## Documentation Strategy

GitHub suggests relegating longer documentation to wikis, keeping READMEs focused on essential developer onboarding information.
