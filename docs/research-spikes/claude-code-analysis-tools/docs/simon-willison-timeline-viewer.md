<!-- Source: https://tools.simonwillison.net/claude-code-timeline -->
<!-- Retrieved: 2026-03-26 -->

# Claude Code Timeline Viewer (by Simon Willison)

Interactive visualization tool for exploring Claude Code session files. Transforms .jsonl session logs into an explorable timeline interface.

## File Loading

Accepts session files via:
- File picker upload
- Drag-and-drop interface
- Direct JSONL paste
- URL fetch (CORS-enabled)

Session files typically found in `~/.claude/projects/`.

## Key Features

**Timeline Visualization:**
- Chronological event display with timestamps (local or UTC)
- Event line numbers and time deltas between events
- Color-coded badges for event type, content type, and role

**Filtering & Search:**
- Full-text search over event content
- Filter by event type (user, assistant, file-history-snapshot)
- Filter by content type (text, thinking, tool_use, image, etc.)
- Filter by role
- Toggle thinking block visibility

**Detail View:**
- Formatted JSON output
- "Pretty" markdown-rendered view
- Inline images with modal viewer
- Extracted image gallery from tool results
- Tool usage details with parameters

**Additional:**
- Extract and view all user prompts collectively
- Copy JSON or raw line data
- URL hash-based state persistence (shareable links)
- Timezone switching
- Responsive design
- Keyboard navigation (arrow keys)
