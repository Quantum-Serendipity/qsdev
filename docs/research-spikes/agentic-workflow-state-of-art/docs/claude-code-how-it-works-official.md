# Claude Code: How It Works — Official Documentation
- **Source**: https://code.claude.com/docs/en/how-claude-code-works
- **Retrieved**: 2026-03-15
- **Type**: Official documentation

## The Agentic Loop

When you give Claude a task, it works through three phases: **gather context**, **take action**, and **verify results**. These phases blend together. Claude uses tools throughout.

The loop adapts to what you ask. You can interrupt at any point to steer Claude in a different direction.

The agentic loop is powered by two components: **models** that reason and **tools** that act. Claude Code serves as the **agentic harness** around Claude: it provides the tools, context management, and execution environment that turn a language model into a capable coding agent.

### Models
Multiple models available with different tradeoffs. Sonnet handles most coding tasks well. Opus provides stronger reasoning for complex architectural decisions. Switch with `/model` during a session or start with `claude --model <name>`.

### Tools
Tools are what make Claude Code agentic. Built-in tools in five categories:

| Category | What Claude can do |
|---|---|
| File operations | Read files, edit code, create new files, rename and reorganize |
| Search | Find files by pattern, search content with regex, explore codebases |
| Execution | Run shell commands, start servers, run tests, use git |
| Web | Search the web, fetch documentation, look up error messages |
| Code intelligence | See type errors and warnings after edits, jump to definitions, find references |

## What Claude Can Access
- Your project files
- Your terminal (any command you could run)
- Your git state
- Your CLAUDE.md
- Auto memory (learnings Claude saves automatically)
- Extensions you configure (MCP servers, skills, subagents, Chrome)

## Environments and Interfaces

### Execution Environments
| Environment | Where code runs | Use case |
|---|---|---|
| Local | Your machine | Default. Full access to your files, tools, and environment |
| Cloud | Anthropic-managed VMs | Offload tasks, work on repos you don't have locally |
| Remote Control | Your machine, controlled from browser | Use the web UI while keeping everything local |

## Sessions
Sessions are independent. Each new session starts with a fresh context window. Claude persists learnings across sessions using auto memory and CLAUDE.md.

### Context Window
Claude's context window holds conversation history, file contents, command outputs, CLAUDE.md, loaded skills, and system instructions. Claude compacts automatically. Instructions from early in conversation can get lost. Put persistent rules in CLAUDE.md.

To control what's preserved during compaction, add a "Compact Instructions" section to CLAUDE.md or run `/compact` with a focus.

Skills load on demand. Subagents get their own fresh context, completely separate from main conversation.

## Safety: Checkpoints and Permissions
- Every file edit creates a checkpoint (reversible with Esc+Esc or /rewind)
- Permission modes: Default, Auto-accept edits, Plan mode
- Settings can be scoped from organization-wide down to personal

## Working Effectively Tips
1. Ask Claude Code for help (it can teach you how to use it)
2. It's a conversation — iterate, don't need perfect prompts
3. Be specific upfront — reference specific files, mention constraints
4. Give Claude something to verify against — tests, screenshots, expected output
5. Explore before implementing — use plan mode for complex problems
6. Delegate, don't dictate — give context and direction, trust Claude for details
