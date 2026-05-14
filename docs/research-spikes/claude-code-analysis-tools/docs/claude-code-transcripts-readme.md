<!-- Source: https://github.com/simonw/claude-code-transcripts -->
<!-- Retrieved: 2026-03-26 -->

# claude-code-transcripts

Tools for converting Claude Code session files into clean, mobile-friendly HTML transcripts with pagination support. By Simon Willison.

## Key Features

**Core Functionality:**
- Converts JSON/JSONL session files to paginated HTML pages
- Generates index pages with timelines of prompts and commits
- Mobile-friendly, browsable transcript format
- Supports both local and web-based Claude Code sessions

**Command Options:**
The tool offers four primary commands:
- `local`: Browse sessions from `~/.claude/projects`
- `web`: Import sessions via Claude API
- `json`: Convert specific JSON/JSONL files
- `all`: Generate browsable archives of all local sessions

**Output Capabilities:**
- Multiple HTML pages with navigation
- GitHub Gist publishing with preview URLs
- Optional source file inclusion
- Auto-named output directories based on session IDs
- Commit link integration with GitHub repositories

## Installation

Install via `uv`:
```bash
uv tool install claude-code-transcripts
```

Or run without installing:
```bash
uvx claude-code-transcripts --help
```

## Usage Examples

**Quick start** (interactive session picker):
```bash
claude-code-transcripts
```

**Web session import with Gist publishing:**
```bash
claude-code-transcripts web SESSION_ID --gist
```

**Batch conversion** of all sessions:
```bash
claude-code-transcripts all --open
```

**JSON file conversion**:
```bash
claude-code-transcripts json session.json -o ./output
```

## Technical Details

- Written in Python with HTML/JavaScript components
- Requires GitHub CLI for Gist functionality
- macOS keychain integration for API credentials
- Supports manual token/org-UUID input on other platforms
- Includes dry-run capability for batch operations
