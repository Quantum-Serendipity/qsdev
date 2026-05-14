# Claude Code: Skills System — Official Documentation
- **Source**: https://code.claude.com/docs/en/skills
- **Retrieved**: 2026-03-15
- **Type**: Official documentation

## Overview
Skills extend what Claude can do. Create a SKILL.md file with instructions, and Claude adds it to its toolkit. Claude uses skills when relevant, or you can invoke directly with /skill-name.

Skills follow the Agent Skills open standard (agentskills.io) which works across multiple AI tools.

## Bundled Skills
| Skill | Purpose |
|---|---|
| /batch <instruction> | Orchestrate large-scale changes across codebase in parallel via worktrees |
| /claude-api | Load Claude API reference for your project's language |
| /debug [description] | Troubleshoot current session by reading debug log |
| /loop [interval] <prompt> | Run a prompt repeatedly on an interval |
| /simplify [focus] | Review recently changed files for code reuse/quality issues |

## Skill Locations
| Location | Path | Applies to |
|---|---|---|
| Enterprise | Managed settings | All users in organization |
| Personal | ~/.claude/skills/<skill-name>/SKILL.md | All your projects |
| Project | .claude/skills/<skill-name>/SKILL.md | This project only |
| Plugin | <plugin>/skills/<skill-name>/SKILL.md | Where plugin is enabled |

## SKILL.md Format
YAML frontmatter + markdown content. Key frontmatter fields:
- name: Display name (becomes /slash-command)
- description: What it does and when to use it
- disable-model-invocation: true to prevent auto-loading
- user-invocable: false to hide from / menu
- allowed-tools: Restrict available tools
- model: Override model
- context: Set to "fork" for subagent execution
- agent: Which subagent type for context:fork
- hooks: Hooks scoped to skill lifecycle

## Key Features
- String substitutions: $ARGUMENTS, $ARGUMENTS[N], $N, ${CLAUDE_SESSION_ID}, ${CLAUDE_SKILL_DIR}
- Dynamic context injection: !`command` syntax runs shell commands before sending to Claude
- Supporting files: reference.md, examples.md, scripts/ alongside SKILL.md
- Run in subagent: context: fork for isolated execution
- Tool restriction: allowed-tools field limits what Claude can do
- Argument passing: /skill-name args passed via $ARGUMENTS
