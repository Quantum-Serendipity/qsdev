<!-- Source: https://news.ycombinator.com/item?id=46805870 -->
<!-- Retrieved: 2026-03-26 -->

# Show HN: Cc-sessions – Fast CLI to list and resume Claude Code sessions

**Submitter:** Chronologos
**Tool URL:** https://github.com/chronologos/cc-sessions

## Key Features

- Scans all project directories in parallel (Rust + rayon)
- Session listings sorted by modification date with relative timestamps
- Interactive fzf picker with transcript preview capability
- Project filtering options
- Fork mode for branching existing sessions

## Technical Details

~350 lines of Rust with custom ISO 8601 parser. Reads from Claude's sessions-index.json files rather than parsing transcripts directly.

## Limitations

Reads local session files only; works on a single local machine and breaks if repositories are moved.
