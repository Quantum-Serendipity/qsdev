<!-- Source: https://github.com/phuryn/claude-usage -->
<!-- Retrieved: 2026-05-12 -->

# Claude Usage Dashboard (claude-usage)

## What This Tool Does

The claude-usage project is a local dashboard that monitors Claude Code token consumption and associated costs. It "reads those logs and turns them into charts and cost estimates" for users across API, Pro, and Max subscription plans.

## Data Source & Collection Method

Claude Code automatically generates local JSONL transcripts in `~/.claude/projects/` regardless of subscription type. Each session file contains JSON records with token metrics. The scanner component parses these files and populates a SQLite database (`~/.claude/usage.db`), tracking "token counts, models, sessions, projects."

Captured data includes:
- Claude Code CLI usage
- VS Code extension activity
- Dispatched Code sessions

**Excluded:** Cowork sessions, which "run server-side and do not write local JSONL transcripts."

## Metrics & Dashboard Features

The dashboard displays:
- **Token breakdown:** Input, output, cache creation, and cache read tokens
- **Model identification:** Extracts model names from transcripts
- **Daily and weekly summaries:** Per-model usage totals
- **All-time statistics:** Cumulative consumption data
- **Interactive filtering:** Bookmarkable URL-based model filters
- **Auto-refresh:** Dashboard updates every 30 seconds

Pro and Max subscribers receive progress bars visualizing subscription limits.

## Cost Calculations

Pricing uses "Anthropic API pricing as of April 2026." Only models containing "opus," "sonnet," or "haiku" are included; unknown or local models show costs as "n/a." The tool notes that "If you use Claude Code via a Max or Pro subscription, your actual cost structure is different (subscription-based, not per-token)."

## Technical Details

- **No external dependencies:** Uses only Python standard library
- **Incremental scanning:** Tracks file modification times to avoid reprocessing
- **Local-first:** All data stored locally; no cloud sync
- **Customizable:** Supports custom project directories and server configuration via environment variables

## Limitations

- Cannot track Cowork sessions
- Only reflects usage from local Claude Code installations
- Cost estimates are approximations for subscription users
