<!-- Source: https://github.com/FlorianBruniaux/claude-code-ultimate-guide/blob/main/guide/workflows/agent-teams.md -->
<!-- Retrieved: 2026-05-12 -->

# Agent Teams Comprehensive Guide

## Architecture
- Team Lead + Teammates structure
- Git-based coordination (lock files in .claude/tasks/)
- Mailbox system for peer-to-peer messaging
- Independent 1M token context per agent
- Experimental (v2.1.32+, Opus 4.6 required)

## When to Use
- Read-heavy tasks (code review, analysis)
- Complex coordination needed
- Multi-service debugging
- Large-scale refactoring with clear boundaries

## When NOT to Use
- Write-heavy tasks with shared files
- <5 files changed
- Sequential workflows
- Budget-constrained
- Tight interdependencies

## Best Practices
- Interface-first approach (define contracts before parallel work)
- Non-overlapping file assignments
- Fan-out/fan-in coordination pattern
- Dedicated read-only reviewer (1 per 3-4 builders)
- Iterative retrieval for sub-agents (3 cycles max)
- AGENTS.md for compound learning (humans write, not agents)
- Loop guardrails: max 8 iterations, mandatory reflection

## Cost Reality
- 2-2.5x token multiplier vs single agent
- Justified when time saved exceeds cost increase
