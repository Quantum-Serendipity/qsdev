<!-- Source: https://news.ycombinator.com/item?id=46512501 -->
<!-- Retrieved: 2026-03-26 -->

# Show HN: ccrider - Search and Resume Your Claude Code Sessions – TUI / MCP / CLI

**Submitter:** nberkman
**Source:** https://github.com/neilberkman/ccrider
**Points:** 19 | Posted ~January 7, 2026

## Tool Overview
Maintains a complete record of Claude Code conversations, enabling users to locate and restore previous sessions. Offers three interfaces: a text-based UI, command-line interface, and Model Context Protocol server.

Single compiled Go application with session data persisted in SQLite. The TUI provides session browsing, full-text search capabilities, session resumption, and markdown export functionality.

## Installation Methods
- **macOS:** Homebrew package available
- **Linux/Other:** Source code compilation via Git
- **MCP Integration:** Can be registered as an MCP server with Claude

## Community Discussion

A commenter asked whether Claude Code's built-in `/rename` and `/resume` commands already addressed this need. The creator responded that built-in commands don't solve "being able to search history from earlier in your current session or from your entire history of sessions. Or resuming sessions for which you've forgotten the name."
