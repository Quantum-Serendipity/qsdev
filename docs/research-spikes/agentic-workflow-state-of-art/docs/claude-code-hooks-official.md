# Claude Code: Hooks System — Official Documentation
- **Source**: https://code.claude.com/docs/en/hooks-guide
- **Retrieved**: 2026-03-15
- **Type**: Official documentation

## Overview
Hooks are user-defined shell commands that execute at specific points in Claude Code's lifecycle. They provide deterministic control over behavior, ensuring certain actions always happen rather than relying on the LLM to choose to run them.

## Hook Events (21 total)
| Event | When it fires |
|---|---|
| SessionStart | Session begins or resumes |
| UserPromptSubmit | You submit a prompt, before Claude processes it |
| PreToolUse | Before a tool call executes. Can block it |
| PermissionRequest | When a permission dialog appears |
| PostToolUse | After a tool call succeeds |
| PostToolUseFailure | After a tool call fails |
| Notification | When Claude Code sends a notification |
| SubagentStart | When a subagent is spawned |
| SubagentStop | When a subagent finishes |
| Stop | When Claude finishes responding |
| TeammateIdle | When an agent team teammate is about to go idle |
| TaskCompleted | When a task is being marked as completed |
| InstructionsLoaded | When a CLAUDE.md or rules file is loaded |
| ConfigChange | When a configuration file changes |
| WorktreeCreate | When a worktree is being created |
| WorktreeRemove | When a worktree is being removed |
| PreCompact | Before context compaction |
| PostCompact | After context compaction completes |
| Elicitation | When an MCP server requests user input |
| ElicitationResult | After user responds to MCP elicitation |
| SessionEnd | When a session terminates |

## Handler Types
1. **Command** (type: "command") — runs shell commands, receives JSON via stdin
2. **HTTP** (type: "http") — POSTs event data to a URL
3. **Prompt** (type: "prompt") — single-turn LLM evaluation using model
4. **Agent** (type: "agent") — multi-turn verification with tool access (spawns subagent)

## Exit Code Behavior
- Exit 0: action proceeds (stdout added to context for some events)
- Exit 2: action is blocked (stderr becomes Claude's feedback)
- Other exit codes: action proceeds, stderr logged but not shown to Claude

## Hook Locations
| Location | Scope | Shareable |
|---|---|---|
| ~/.claude/settings.json | All your projects | No |
| .claude/settings.json | Single project | Yes |
| .claude/settings.local.json | Single project | No |
| Managed policy settings | Organization-wide | Yes |
| Plugin hooks/hooks.json | When plugin is enabled | Yes |
| Skill or agent frontmatter | While skill/agent is active | Yes |

## Key Patterns
- **Auto-format after edits**: PostToolUse with Edit|Write matcher
- **Block protected files**: PreToolUse with script checking file paths
- **Re-inject context after compaction**: SessionStart with compact matcher
- **Desktop notifications**: Notification event
- **Audit logging**: ConfigChange event
- **Quality gates**: Stop hook with prompt or agent type to verify completeness
