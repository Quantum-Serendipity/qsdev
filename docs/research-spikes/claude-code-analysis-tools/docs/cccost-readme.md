<!-- Source: https://github.com/badlogic/cccost -->
<!-- Retrieved: 2026-03-26 -->

# cccost - Claude Code Cost Tracker

Tool for tracking token usage and costs during Claude Code sessions in real-time by intercepting API requests to Anthropic's servers.

## Problem Statement

Claude Code doesn't display costs for Pro/Max plan users. The `/cost` command has documented bugs, and session transcripts don't capture all API requests Claude Code makes.

## How It Works

Operates as a minimally invasive wrapper:
1. Spawns Claude Code with all arguments forwarded unchanged
2. Injects monitoring code that hooks Node.js's `fetch()` function
3. Intercepts all API requests to Anthropic's servers
4. Writes usage data to `~/.claude/projects/[project]/sessionid.usage.json`
5. Updates statistics continuously with each new request

## Output Format

JSON file containing: total requests count, cumulative cost across all models, per-model statistics (input/output/cache tokens, cost), last request details with timestamps.

## Installation

```
npm install -g @mariozechner/cccost
```

## Usage

Replace `claude` with `cccost`:
```
cccost --dangerously-skip-permissions --model sonnet
```

All Claude Code arguments pass through unchanged.

**License**: MIT
**Author**: Mario Zechner
