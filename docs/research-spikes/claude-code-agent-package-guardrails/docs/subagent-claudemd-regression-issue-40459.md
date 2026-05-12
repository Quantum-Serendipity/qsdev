<!-- Source: https://github.com/anthropics/claude-code/issues/40459 -->
<!-- Retrieved: 2026-05-12 -->

# Issue #40459: Subagents Lose CLAUDE.md Context (omitClaudeMd:true)

## Summary
Since v2.1.84, Claude Code subagents (Explore, Plan, built-in agents) no longer receive the user's CLAUDE.md instructions, causing them to ignore project-specific rules, language preferences, and environment configurations.

## Root Causes

### 1. omitClaudeMd: true on Built-in Subagents (NEW in v2.1.84)
v2.1.83: omitClaudeMd did NOT exist
v2.1.84+: Two built-in agents now have omitClaudeMd:true

Combined with feature flag `tengu_slim_subagent_claudemd` (default: true), subagents are stripped of CLAUDE.md context.

### 2. System Prompt Global Cache Enabled with ToolSearch
Changed logic for when to skip global caching:
- v2.1.83: ANY MCP tool -> skip global cache
- v2.1.84: only NON-deferred MCP tools -> skip global cache

For users with ToolSearch (all tools deferred), system prompt (including CLAUDE.md) is now globally cached, meaning cached tokens receive less model attention.

### 3. deferLoading Changed from Global to Per-Tool Flag

## Observed Impact
- Subagent no longer follows CLAUDE.md language rules
- Subagent doesn't know project environments
- Makes categorical claims from partial data (frequent)
- User corrections needed per session: 5+

## Affected Versions
- v2.1.84 through v2.1.87+ (regression introduced)
- v2.1.83 and earlier: No issues

## Resolution Status
Open — labeled as regression with "has repro" label. No fix noted.
