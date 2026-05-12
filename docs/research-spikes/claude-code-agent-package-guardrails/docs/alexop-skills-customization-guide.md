---
source: https://alexop.dev/posts/claude-code-customization-guide-claudemd-skills-subagents/
retrieved: 2026-05-12
---

# Claude Code Skills: Customization & Internal Mechanics

## How Skills Work Internally

Skills are auto-discovered capabilities with optional supporting files that Claude applies within main conversations. Claude decides whether to invoke a skill "largely based on its description." When Claude evaluates available tools, it receives structured blocks showing:

```
<available_skills>
  <skill>
    <name>skill-name</name>
    <description>...</description>
  </skill>
</available_skills>
```

Skills differ fundamentally from slash commands: "the difference is mostly UX + packaging. Slash commands are what you can run manually from the terminal via /command. Skills are structured, auto-discovered capabilities (often a directory of supporting files) that Claude may apply when relevant."

## Can Agents Invoke Skills Autonomously?

Yes, but with nuance. Subagents can access skills if "configured via skills:" in their setup. Skill invocation is primarily Claude's decision — "Claude may apply skills" when the task description matches. Skills are auto-discovered and typically get applied when Claude decides they match the current task. They run in your main conversation, so you can iterate live.

## Relationship to Other Mechanisms

Skills vs. MCP/Hooks: Hooks handle "automation responses to Claude Code events" while skills represent structured workflow packaging. These are complementary, not competing mechanisms.

Skills vs. Subagents: Subagents keep main context clean — in plan mode, Claude Code will typically delegate repo scanning to an Explore-style subagent. Skills share main context space, whereas subagents use separate context windows.

## Practical Patterns

Directory structure:
```
.claude/skills/
  dexie-expert/
    SKILL.md (main definition)
    PATTERNS.md (supporting reference)
    MIGRATIONS.md (migration guide)
    scripts/validate-schema.ts
```

Example skill with tool restrictions:
```yaml
---
name: dexie-expert
description: Dexie.js database guidance. Use when working with IndexedDB, schemas, queries, liveQuery...
allowed-tools: Read, Grep, Glob, WebFetch
---
```

## Security Patterns

Tool restriction as a control mechanism: "Can restrict tools for security" under subagent trade-offs. The skill definition includes allowed-tools, providing granular capability boundaries. Subagents can be configured with specific tool permissions to prevent unauthorized operations.
