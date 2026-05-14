# How to Resume, Search, and Manage Claude Code Conversations

- **Source**: https://kentgigger.com/posts/claude-code-conversation-history
- **Retrieved**: 2026-03-26
- **Note**: Content was AI-summarized by WebFetch.

## Directory Structure

Claude Code stores conversation data across two primary locations:

**Global history:**
- `~/.claude/history.jsonl` — A line-delimited JSON log containing "every input you've ever sent across every project"

**Project-specific data:**
- `~/.claude/projects/` — Contains subdirectories for each project (named with dashes replacing slashes)
- Individual session files stored as `.jsonl` format
- `sessions-index.json` with metadata including summaries, message counts, git branches, and timestamps
- `memory/` directory for auto-memory features like `MEMORY.md`

## File Formats & Data Structure

**history.jsonl format:**
Each line is a JSON object containing prompt text, timestamp, project path, and session ID. The document indicates the global index "grows forever," with example sizes reaching "over 600KB."

**sessions-index.json structure:**
Contains metadata per session:
- Auto-generated conversation summaries
- Message counts
- Git branch information
- Creation and modification timestamps

## Session Management Commands

| Command | Function |
|---------|----------|
| `claude --continue` / `claude -c` | Resume last session in current directory |
| `claude --resume` / `claude -r` | Browse interactive picker of all recent sessions |
| `claude --resume [ID]` | Jump directly to specific session by ID |
| `/resume` | Switch sessions from within active conversation |
| `/rename [name]` | Assign human-readable names to sessions |
| `/clear` | Wipe conversation context (preserves session file) |
| `/compact` | Summarize conversation to preserve tokens |
| `/context` | Visualize current context usage |

## Custom Commands & Search

Custom `/history` command installed in `~/.claude/commands/` queries `history.jsonl` for "git log"-style searching across projects.

## Conversation Deletion

- **Context only:** `/clear` (preserves session files)
- **Specific sessions:** `rm ~/.claude/projects/[project]/[SESSION_ID].jsonl`
- **All history:** `rm ~/.claude/history.jsonl && rm -rf ~/.claude/projects/*/`

## Auto-Compaction

Claude Code automatically summarizes conversations approaching context limits, replacing older messages while maintaining functionality.
