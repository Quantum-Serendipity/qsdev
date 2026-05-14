<!-- Source: https://github.com/philipp-spiess/claude-code-viewer -->
<!-- Retrieved: 2026-03-26 -->

# Claude Code Viewer (Philipp Spiess)

Upload your Claude Code transcripts to a web-based viewer for sharing and accessing session histories online.

## Primary Command

```bash
npx -y claude-code-uploader
```

## Core Functionality

1. **Discovery** -- Automatically locates transcript files in `~/.claude/projects/`
2. **Selection** -- Presents interactive menu for choosing which transcripts to share
3. **Upload** -- Transfers selected transcripts to the web viewer
4. **Access** -- Generates shareable URLs like `https://claude-code-viewer.pages.dev/abcd1234`

## Technical Stack

- TypeScript (88.5%), JavaScript (9.0%), CSS (2.5%)
- pnpm workspace-based monorepo
- Biome for linting/formatting

**Repository Stats:** 44 commits, 18 stars, 1 fork
