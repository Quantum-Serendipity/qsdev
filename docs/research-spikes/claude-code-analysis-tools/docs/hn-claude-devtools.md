<!-- Source: https://news.ycombinator.com/item?id=47004712 -->
<!-- Retrieved: 2026-03-26 -->

# Show HN: I built a tool to un-dumb Claude Code's CLI output (Local Log Viewer)

**Submitter:** matt1398
**Repository:** https://github.com/matt1398/claude-devtools
**Score:** 69 points | 44 comments

## Core Concept

The creator developed `claude-devtools`, a local Electron application that parses Claude Code session logs from `~/.claude/` to provide detailed visibility into agent execution. Recent CLI updates replaced detailed output with summaries like "Read 3 files," making debugging difficult.

The tool addresses three main observability gaps:

1. **Real-time diffs** showing file modifications with inline color-coding
2. **Context forensics** breaking down token consumption by file, tool output, and reasoning phases
3. **Agent tree visualization** clarifying sub-agent execution flows

## Key Discussion Points

### Tool Design Philosophy
Matt emphasizes this isn't a wrapper but a "passive viewer" that preserves native terminal workflows. Particularly useful for post-mortem debugging of completed sessions.

### Alternative Solutions Proposed
Commenters suggested competing tools including OpenCode and Pi Code Agent, though azuanrb raised concerns about Anthropic's Terms of Service potentially prohibiting external harnesses with Claude subscriptions.

### API Stability Concerns
The tool relies on undocumented `.jsonl` log formats. Matt notes confidence because "Claude Code's official VS Code extension is built to read these exact same local files," reducing breakage risk.

### Notable Comments
- **kzahel** shared a similar project (yepanywhere) using Zod schema validation to track format changes
- **igravious** suggested "Anthropic should hire this person"
- Multiple users praised the observability gap this tool addresses
