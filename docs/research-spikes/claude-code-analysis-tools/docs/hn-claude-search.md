<!-- Source: https://news.ycombinator.com/item?id=47176556 -->
<!-- Retrieved: 2026-03-26 -->

# Claude-search – grep, resume your Claude Code session history from the CLI

**Submitter:** pi-netizen
**URL:** https://github.com/pi-netizen/claude-search
**Points:** 2 | **Comments:** 1

## Overview

The tool addresses a practical challenge: locating previous conversations within Claude Code's local session storage. "Claude Code stores every conversation as JSONL files under ~/.claude/projects/. I kept wanting to find old sessions, 'where did I debug that Redis issue?'"

## Key Features

- **Date filtering** using `--since` parameter (e.g., "2 weeks ago")
- **Code extraction** via `--code-only` flag
- **Project scoping** with `--project` option
- **Session reopening** through `--open` command
- **Extended thinking visibility** to review reasoning blocks

## Design Philosophy

Operates entirely offline: "No server, no API calls, no sync. It reads the files Claude Code already writes locally."
