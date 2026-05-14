# Claude Code Metadata - LobeHub Skills Market

- **Source**: https://lobehub.com/skills/marcus-marcus-skills-claude-code-metadata
- **Retrieved**: 2026-03-26
- **Note**: Content was AI-summarized by WebFetch; high-level only.

## JSONL Session Format

Per-session or per-message JSON entries with fields:
- `session_id`
- `messages` array
- `role`
- `content`
- `model`
- Timestamps

## Stats-Cache Schema

Stats tracking captures:
- Per-session token counts
- Aggregate token counts
- Prompt vs. completion token separation
- Timestamps
- Model usage information

## Settings Hierarchy

Three-tier configuration:
- User-level settings
- Workspace-level settings
- Default settings

## File Organization

Key directories within `~/.claude`:
- `sessions/` — conversation storage
- `attachments/` — file references
- `cache/` — performance optimization
- `file_history/` — backup and naming conventions
- `settings.json` — configuration
- `stats-cache.json` — usage metrics

## Key Capabilities

Supports "memory-efficient JSONL parsing, precise token-tracking fields for cost analysis," enabling dashboards, analytics pipelines, migration scripts, and compliance reporting.
