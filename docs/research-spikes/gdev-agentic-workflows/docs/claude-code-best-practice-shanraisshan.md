<!-- Source: https://github.com/shanraisshan/claude-code-best-practice -->
<!-- Retrieved: 2026-05-12 -->

# Claude Code Best Practice - Workflow & Architecture Guide

## Three-Tier Orchestration Pattern
- Commands (.claude/commands/) trigger workflows via slash commands
- Subagents (.claude/agents/) provide isolated context and specialized focus
- Skills (.claude/skills/) bundle reusable capabilities with progressive disclosure

## Context Management
- Degradation begins ~300-400k tokens on 1M model
- "Dumb zone" at ~40% utilization; experienced users target 30%
- Use /rewind to backtrack vs accumulating failed attempts
- /compact with hints for focused compaction
- Subagents isolate exploratory work

## Multi-Agent Patterns
- Separation by specialization (feature-specific, not generic roles)
- Parallel dev with tmux + git worktrees
- Test-time compute: separate windows for generation and verification

## Workflow Pattern: Research → Plan → Execute → Review → Ship
- Vertical slice approach (DB → API → UI tracer bullets)
- End-to-end feedback before horizontal completion

## Skill Design
- Progressive disclosure via folders: references/, scripts/, examples/
- Descriptions as triggers (not summaries)
- Focus on non-default behavior
- Gotchas section for failure modes
- context: fork for isolated execution

## CLAUDE.md Structure
- Target under 200 lines per file (60 lines conservative)
- .claude/rules/*.md with paths: frontmatter for lazy-loading by file glob
- Deterministic requirements in settings.json, not CLAUDE.md prose

## Key Principles
1. Don't babysit Claude - set challenges instead
2. Rewarding specificity reduces ambiguity
3. Prototype over PRD - iteration is cheaper than specification
4. Clean codebases matter (confused by partially migrated frameworks)
5. Skills as progressive capability (include scripts, libraries, references)
