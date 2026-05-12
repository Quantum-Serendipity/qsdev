---
source: https://www.mindstudio.ai/blog/claude-code-skills-vs-slash-commands
retrieved: 2026-05-12
---

# Claude Code Skills vs Slash Commands: Key Differences

## Invocation Mechanisms

**Claude Code Skills:**
- Auto-invoke based on context detection
- Agent-triggered, not user-triggered
- Claude automatically recognizes patterns and executes without manual prompting
- "When Claude detects that the current context matches the conditions described in that skill file, it runs the skill automatically."

**Slash Commands:**
- Require explicit user invocation
- User types the command to trigger execution
- Nothing executes until manually requested

## Autonomous Execution

Skills enable true autonomous operation — agents invoke them without any user typing a slash command. The detection mechanism reads context (open files, recent code, task descriptions) and pattern-matches against available skills to determine execution.

Slash commands demand human judgment to decide *when* activation is appropriate, making them intentionally non-autonomous.

## Configuration Differences

**Skills:**
- Stored as `SKILL.md` files containing process steps
- Include detection conditions triggering auto-invocation
- Should remain lean, focusing on actions rather than background context

**Slash Commands:**
- Built-in utilities (like `/compact`, `/simplify`, `/batch`)
- Custom commands stored in `.claude/commands/` directory
- Act as "personal macros" expanding typed shorthand into full instructions

## When to Use Each

**Choose Skills When:**
- Tasks recur regularly (multiple times weekly)
- Processes are well-defined with consistent steps
- Consistency matters without requiring manual intervention
- Examples: automated code reviews on every PR, test generation for new functions

**Choose Slash Commands When:**
- Tasks are situational or one-off
- Human judgment determines appropriate timing
- Context management is needed
- Examples: debugging specific outputs, occasional audits
