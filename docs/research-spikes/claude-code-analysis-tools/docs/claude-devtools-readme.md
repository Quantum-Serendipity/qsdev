<!-- Source: https://github.com/matt1398/claude-devtools -->
<!-- Retrieved: 2026-03-26 -->

# claude-devtools: DevTools for Claude Code

## Repository Overview

**claude-devtools** is a desktop application that reconstructs and visualizes Claude Code sessions from raw logs stored locally at `~/.claude/`. It provides detailed inspection of every tool call, file operation, token usage, and agent interaction that the CLI deliberately hides.

The tagline captures its purpose: "Terminal tells you nothing. This shows you everything."

## Key Installation Methods

**Homebrew (macOS):**
```bash
brew install --cask claude-devtools
```

**Direct Downloads:** Available for macOS (Apple Silicon/Intel), Linux (AppImage/deb/rpm/pacman), and Windows (.exe)

**Docker/Standalone:**
```bash
docker compose up
# Open http://localhost:3456
```

## Core Features

### Context Reconstruction
The application reverse-engineers what actually occupies Claude's context window by analyzing session logs. It breaks down token attribution across seven categories: CLAUDE.md files, skill activations, @-mentions, tool I/O, extended thinking, team overhead, and user prompts. Results display via context badges, token popovers, and a dedicated session panel.

### Compaction Visualization
Detects when Claude silently compresses conversations due to context limits. Visualizes how context fills, compresses, and refills throughout a session, showing exact token deltas before and after each compaction event.

### Custom Notification Triggers
Define regex-based rules to receive system notifications for specific events:
- Built-in defaults: `.env` file access, tool errors, high token usage
- Custom matching against file paths, commands, prompts, and thinking
- Sensitive-file monitoring for project-specific patterns
- Configurable token thresholds and repository scoping

### Rich Tool Call Inspector
Each tool call pairs with its result in expandable cards with specialized viewers:
- **Read** calls show syntax-highlighted code with line numbers
- **Edit** calls display inline diffs with add/remove highlighting
- **Bash** calls render command output
- **Subagent** calls show full execution trees, expandable inline

### Team & Subagent Visualization
Fully untangles Claude Code's distributed execution model:
- Subagent sessions resolve from Task tool calls, rendered as expandable cards
- Teammate messages (via `SendMessage`) appear as color-coded cards
- Team lifecycle visibility includes `TeamCreate`, `TaskCreate`, `SendMessage`, and shutdown flow
- Session summary distinguishes teammate count from subagent count

### Command Palette & Search
Press Cmd+K for Spotlight-style searching across all sessions in a project. Results display context snippets with highlighted keywords, enabling direct navigation to specific messages.

### SSH Remote Sessions
Connect over SSH to inspect Claude Code sessions running on remote machines. The app parses `~/.ssh/config`, supports agent forwarding and private keys, and streams logs via SFTP. Each SSH host gets isolated service context with independent caches.

### Multi-Pane Layout
Open multiple sessions side-by-side with drag-and-drop tab management between panes, enabling parallel session comparison.

## CLI vs. claude-devtools Comparison

| CLI Output | claude-devtools Shows |
|---|---|
| "Read 3 files" | Exact paths, syntax-highlighted content with line numbers |
| "Searched for 1 pattern" | Regex pattern, matching files, matched lines |
| "Edited 2 files" | Inline diffs with add/remove highlighting per file |
| Three-segment context bar | Per-turn token attribution across 7 categories + compaction viz |
| Interleaved subagent output | Isolated trees per agent with own metrics |
| Buried teammate messages | Color-coded teammate cards with full lifecycle |
| Mixed critical events | Trigger-filtered inbox (.env access, errors, high tokens) |
| `--verbose` JSON dump | Structured, filterable, navigable interface |

## Why This Exists

Claude Code's recent updates replaced detailed tool output with opaque summaries ("Read 3 files"). The `--verbose` flag dumps raw JSON and internal prompts with thousands of lines of noise. There is no middle ground.

claude-devtools restores the missing information by reading session logs from `~/.claude/` (already on your machine) without modifying Claude Code itself. It works with every session ever executed, regardless of execution context.

## Security

IPC handlers validate all inputs with strict path containment checks. File reads are constrained to project root and `~/.claude/`. Credential paths are blocked.

## License

MIT
