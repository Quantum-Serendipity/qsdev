<!-- Source: https://news.ycombinator.com/item?id=47128630 -->
<!-- Retrieved: 2026-03-26 -->

# Show HN: Search-sessions – Search all your Claude Code session history in <300ms

**Submitter:** sinzin91
**Tool URL:** https://github.com/sinzin91/search-sessions
**Points:** 4 | **Comments:** 4

## Full Discussion

The creator addresses a practical problem: Claude Code doesn't retain context across sessions, leaving developers unable to recover past solutions from accumulated session files. Rather than implementing complex solutions like vector databases, the developer built a lightweight Rust tool that performs "text search over structured files" directly on JSONL session data stored locally.

The tool offers two search modes: quick index searching (~18ms) and comprehensive deep searching using ripgrep (~280ms). Users can resume conversations using `claude --resume` with the session UUID. The solution emphasizes simplicity — "No database, no indexing step, no dependencies" — making it transparent and portable.

Installation options include Homebrew or Cargo for macOS and Linux users.

## Comments & Replies

**SteveVeilStream:** Suggested adding security detection, noting that "Claude sometimes pulls an API key out of a .env file and drops it into that folder."

**sinzin91's Response:** Acknowledged the suggestion, mentioning an existing separate security-focused project that could potentially be integrated.

**kirilligum:** Praised the database-free approach, asking how Claude Code discovers the tool's functionality.

**sinzin91's Response:** Shared their CLAUDE.md configuration, instructing Claude to use `/search-sessions` for historical recalls.
