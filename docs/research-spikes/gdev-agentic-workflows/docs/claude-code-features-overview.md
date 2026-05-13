<!-- Source: https://code.claude.com/docs/en/features-overview -->
<!-- Retrieved: 2026-05-12 -->

# Claude Code Features Overview - Extension Taxonomy

## Feature → Goal Mapping
| Feature | What | When | Example |
|---------|------|------|---------|
| CLAUDE.md | Persistent context every session | "Always do X" rules | "Use pnpm, not npm" |
| Skill | Instructions, knowledge, workflows | Reusable content, repeatable tasks | /deploy checklist |
| Subagent | Isolated execution context | Context isolation, parallel tasks | Research reading many files |
| Agent teams | Multiple independent sessions | Parallel research, competing hypotheses | Security + perf + tests review |
| MCP | External service connections | External data or actions | Query DB, post Slack |
| Hook | Script/HTTP/prompt on events | Must happen every time | ESLint after every edit |
| Plugin | Bundle of above features | Reuse across repos | Team toolkit |

## Build Over Time Triggers
- Claude gets convention wrong twice → CLAUDE.md
- Keep typing same prompt → skill
- Paste same procedure third time → skill
- Keep copying data from external → MCP
- Side task floods context → subagent
- Something must always happen → hook
- Second repo needs same setup → plugin

## Key Comparisons
- Skill vs Subagent: skill adds to main context, subagent isolated
- CLAUDE.md vs Skill: CLAUDE.md every session, skill on demand
- CLAUDE.md vs Rules vs Skills: CLAUDE.md always, rules path-scoped, skills task-specific
- Subagent vs Agent team: subagent reports to parent, team members message each other
- MCP vs Skill: MCP provides connection, skill teaches how to use it
- Hook vs Skill: hook deterministic/guaranteed, skill interpreted by Claude

## Context Costs
- CLAUDE.md: loads every request (keep under 200 lines)
- Skills: descriptions at start, full content when used
- MCP: tool names at start, schemas on demand
- Subagents: isolated from main
- Hooks: zero unless returns output

## Layering Rules
- CLAUDE.md: additive (all levels contribute)
- Skills/Subagents: override by name (priority-based)
- MCP: override by name (local > project > user)
- Hooks: merge (all fire for matching events)
