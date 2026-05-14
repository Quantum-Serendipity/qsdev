# Claude Code Session Storage - ClaudeWorld Tutorial

- **Source**: https://claude-world.com/tutorials/s16-session-storage/
- **Retrieved**: 2026-03-26
- **Note**: Content was AI-summarized by WebFetch; may not contain full page detail.

## Directory Structure

Sessions are stored hierarchically:
```
~/.claude/projects/
  <project-hash>/          # Hash of project's absolute path
    sessions/
      <session-id>.jsonl   # One transcript per session
    settings.json
```

The project directory uses a hash to keep sessions from different projects separate.

## Session Identification

**UUID-based naming**: "Every session gets a UUID (Universally Unique Identifier) when it starts" and appears in both the filename and every transcript line.

**Parent-child chains**: Sessions form hierarchical relationships through `parentSessionId` fields, creating traceable trees of work across resumed sessions and subagent spawns.

## JSONL Format

The system uses "JSON Lines — files. Each line is a self-contained JSON object representing one event in the conversation."

**Why JSONL over JSON array:**
- Append-only writes (no file rewriting)
- Line-by-line parsing
- Bad lines are skippable; one corruption doesn't affect entire file

## Event Types and Schema

| Event Type | Key Fields | Purpose |
|---|---|---|
| `session_start` | sessionId, parentSessionId, timestamp, project path | Marks session beginning |
| `message` | role, content blocks, timestamp | User/assistant messages |
| `tool_use` | toolName, input parameters, toolUseId, timestamp | AI tool requests |
| `tool_result` | toolUseId, content, durationMs, timestamp | Tool execution results |
| `compaction` | summary text, tokens saved | Context optimization |
| `session_end` | duration, token count, cost | Session statistics |

## Message Structure Example

Tool use entries contain toolName, structured inputs, and unique identifiers. Results link back via `toolUseId` for request-response pairing.

## Session Resumption Process

`claude --resume` performs: session listing → JSONL parsing → message array reconstruction → context compaction (if needed) → new child session creation.
