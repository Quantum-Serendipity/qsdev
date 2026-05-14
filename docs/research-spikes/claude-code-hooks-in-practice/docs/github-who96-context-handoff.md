# Claude Code Context Handoff - who96/claude-code-context-handoff
- **Source**: https://github.com/who96/claude-code-context-handoff
- **Retrieved**: 2026-03-27

## Overview
Plugin that preserves critical context across Claude Code session transitions, preventing intelligence degradation after auto-compaction or `/clear` commands.

## Hooks Used

Three core hooks:
1. **PreCompact Hook** — captures context before auto-compact or `/compact`
2. **SessionEnd(clear) Hook** — captures context before `/clear` tears down session
3. **SessionStart(compact|clear) Hook** — restores context as `additionalContext`

## Event Flow

### Auto-Compact/Manual Compact Path
- PreCompact hook writes handoff data
- SessionStart(compact) restores the preserved handoff

### Clear Command Path
- SessionEnd(clear) writes handoff and updates latest pointer
- SessionStart(clear) restores handoff as additional context

## Context Preservation Details

**What Gets Preserved:**
- Last 15 user messages (deduplicated at 85% threshold)
- Last 10 assistant code snippets (filtered, truncated)
- File paths extracted from tool inputs (`file_path`/`path` fields)
- Command-like strings are filtered from path extraction

**Fallback Safety Guards:**
- Same working directory matching (when available)
- Maximum age window: 900 seconds (configurable)

## Configuration
```
HANDOFF_MAX_USER_MESSAGES=15 (default)
HANDOFF_MAX_ASSISTANT_CHARS=1000 (default)
HANDOFF_DEDUP_THRESHOLD=0.85 (default)
HANDOFF_LATEST_MAX_AGE_SEC=900 (default)
```

## Handoff Storage
- `~/.claude/handoff/<session_id>.md` — Session-specific data
- `~/.claude/handoff/latest-handoff.md` — Most recent handoff
- `~/.claude/handoff/latest-handoff.json` — Metadata

## Key Limitation
"Hooks cannot rewrite slash commands directly." External supervisor (`claude-handoff-supervisor.py`) handles `/compact` to `/clear` conversion.
