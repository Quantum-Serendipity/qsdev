<!-- Source: https://github.com/kanopi/claude-dev-insights -->
<!-- Retrieved: 2026-03-26 -->

# Claude Dev Insights

Plugin for Claude Code that enables automatic session tracking with comprehensive analytics. Captures 29 data points per session with optional Google Sheets synchronization.

## Core Features

**Session Analytics Tracking**: Monitors SessionStart, SessionEnd, and UserPromptSubmit events.

**Data Storage Options:**
- Local CSV storage at `~/.claude/session-logs/sessions.csv`
- Optional Google Sheets integration for team collaboration
- Environment detection including CMS type, dependency counts, git status
- Issue/ticket number auto-detection and logging

**Work Documentation via tags:**
- `#ticket: JIRA-1234` for issue tracking
- `#topic: feat: Description` for work summaries

## Data Capture (29 fields)

- Temporal data (timestamps, session duration)
- Message metrics (user and assistant message counts)
- Token usage (input, output, cache read/write, totals)
- Cost calculations in USD
- Tool invocation counts and top-5 tool usage
- API performance metrics (call times, averages)
- Environment context (git branch, CMS type, dependencies, uncommitted changes)
- System information (Claude version, model used, permission mode)

## Installation

**Via Marketplace:**
```
/plugin marketplace add kanopi/claude-toolbox
/plugin install claude-dev-insights@claude-toolbox
```

Hooks activate automatically upon plugin enablement.

## Privacy & Security

Does NOT log actual code, conversation content, or sensitive data. Only captures session metadata and usage statistics.

**License:** GPL-2.0-or-later
**Maintained by:** Kanopi Studios
