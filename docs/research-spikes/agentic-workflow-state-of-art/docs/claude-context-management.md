# Claude Code Context Management

- **Source URLs**:
  - https://platform.claude.com/docs/en/build-with-claude/compaction
  - https://platform.claude.com/docs/en/build-with-claude/context-editing
  - https://platform.claude.com/docs/en/build-with-claude/context-windows
  - https://code.claude.com/docs/en/how-claude-code-works
  - https://code.claude.com/docs/en/best-practices
  - https://claudefa.st/blog/guide/mechanics/context-buffer-management
  - https://code.claude.com/docs/en/memory
- **Retrieved**: 2026-03-15
- **Note**: Content synthesized from web search results.

## Context Window Architecture

Claude Code operates within a fixed ~200K token context window (195,072 tokens on Sonnet 4.0). The system consists of:
- System prompt (modular, ~269 base tokens + dynamic components from 110+ prompt strings)
- CLAUDE.md files (loaded from directory hierarchy)
- Auto-memory (MEMORY.md, first 200 lines loaded)
- Conversation history (user messages, assistant responses, tool calls/results)

## Auto-Compaction

When total token usage crosses ~75-92% of window capacity (as of early 2026, ~83.5%), Claude Code automatically triggers a compaction cycle:
1. Summarizes conversation history into key decisions and context
2. Discards verbose tool outputs (keeping only summaries)
3. Preserves critical information (file paths, function names, error messages)

## Server-Side Compaction API (Beta)

Available on Opus 4.6 and Sonnet 4.6 via beta header `compact-2026-01-12`:
- Automatically summarizes conversation when approaching configured token threshold
- Generates a summary, creates a compaction block, continues with compacted context
- Recommended over SDK compaction for less integration complexity

## Context Editing API (Beta)

Via beta header `context-management-2025-06-27`:
- **Tool result clearing** (`clear_tool_uses_20250919`): Clears tool results when context grows beyond threshold. Especially useful for agentic workflows with heavy tool use.
- **Thinking block clearing** (`clear_thinking_20251015`): Manages thinking blocks automatically. Cache invalidation occurs at clearing point.

## Subagent Context Isolation

Subagents get their own fresh context, completely separate from the main conversation:
- Their intermediate tool calls and results stay inside the subagent
- Only the final message returns to the parent
- This prevents context bloat from accumulating

## /clear and /compact Commands

- `/clear`: Wipes conversation history; REPL session continues but messages removed from context. CLAUDE.md and project files remain accessible.
- `/compact`: Summarizes current context to free space. Recommended at 70% capacity.
- Best practice: Clear after commits (logical checkpoints). If <50% of prior context is relevant, clear.

## Auto-Memory System

Claude saves notes for itself as it works: build commands, debugging insights, architecture notes, code style preferences:
- Stored in local MEMORY.md file within each project
- First 200 lines loaded into every new session
- Machine-local; all worktrees within same git repo share one auto memory directory
- On by default; plain markdown the user can read/edit/delete
