<!-- Source: https://news.ycombinator.com/item?id=47182387 -->
<!-- Retrieved: 2026-03-26 -->

# Show HN: Claude-File-Recovery, recover files from your ~/.claude sessions

**Submitter:** rikk3rt
**URL:** https://github.com/hjtenklooster/claude-file-recovery
**Points:** 99 | **Comments:** 41

## Original Problem
The creator accidentally executed `rm -rf` through a symlink in their Obsidian vault via Claude Code, deleting research and planning files. With backups not running for a month, they developed this recovery tool to extract files from session histories.

## Tool Capabilities
The tool functions as both a CLI and TUI application, extracting files that Claude Code previously read, edited, or wrote. It can recover earlier versions of files from specific points in time using session history data.

## Important Limitations
**Retention Window Issue:** Claude Code by default auto-deletes local chat logs after 30 days, limiting recovery scope. The creator adjusted this setting to 9999 days after discovering the issue.

## Installation
```
pip install claude-file-recovery
```

## Related Solutions Discussed
- Alternative approach using Claude itself to search across multiple conversations
- Similar tool: unfucked.ai (tracking all file writes across agents)
- claude-devtools (for session documentation and history)
- Aider's git commit approach for file protection

## Key Concerns Raised
- Undocumented local session infrastructure reliance
- Need for proper session persistence standards from Anthropic
- File sandboxing considerations for AI coding tools
- Recovery tool necessity indicating broader system design gaps
