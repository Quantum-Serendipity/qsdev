# Claude Code: Sub-Agents — Official Documentation
- **Source**: https://code.claude.com/docs/en/sub-agents
- **Retrieved**: 2026-03-15
- **Type**: Official documentation

## Overview
Subagents are specialized AI assistants that handle specific types of tasks. Each runs in its own context window with custom system prompt, specific tool access, and independent permissions.

## Built-in Subagents
- **Explore**: Fast, read-only (Haiku model). File discovery, code search, codebase exploration.
- **Plan**: Research agent for plan mode (inherits model). Read-only tools.
- **General-purpose**: Complex multi-step tasks (inherits model). All tools.
- **Bash**: Running terminal commands in separate context.
- **statusline-setup**: Configure status line (Sonnet).
- **Claude Code Guide**: Questions about Claude Code features (Haiku).

## Subagent Scope/Priority
1. --agents CLI flag (highest, session only)
2. .claude/agents/ (project)
3. ~/.claude/agents/ (user, all projects)
4. Plugin agents/ directory (lowest)

## AGENT.md Frontmatter Fields
- name (required): Unique identifier
- description (required): When Claude should delegate
- tools: Tool allowlist
- disallowedTools: Tool denylist
- model: sonnet, opus, haiku, full model ID, or inherit
- permissionMode: default, acceptEdits, dontAsk, bypassPermissions, plan
- maxTurns: Maximum agentic turns
- skills: Skills to preload at startup
- mcpServers: MCP servers (inline or reference)
- hooks: Lifecycle hooks scoped to subagent
- memory: Persistent memory scope (user, project, local)
- background: true for background execution
- isolation: "worktree" for git worktree isolation

## Key Patterns
1. **Isolate high-volume operations**: Keep verbose test/build output in subagent context
2. **Run parallel research**: Spawn multiple subagents for independent investigations
3. **Chain subagents**: Sequential workflow — reviewer then optimizer
4. **Persistent memory**: Subagents accumulate knowledge across sessions

## Foreground vs Background
- Foreground: blocks main conversation, permission prompts pass through
- Background: concurrent, pre-approved permissions, auto-denies unapproved tools
- Press Ctrl+B to background a running task
