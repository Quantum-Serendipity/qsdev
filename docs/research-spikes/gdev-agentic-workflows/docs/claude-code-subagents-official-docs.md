<!-- Source: https://code.claude.com/docs/en/sub-agents -->
<!-- Retrieved: 2026-05-12 -->

# Create custom subagents - Claude Code Official Docs

## Key Format Details

Subagents are `.md` files in `.claude/agents/` (project) or `~/.claude/agents/` (user) with YAML frontmatter.

### Frontmatter Fields
- `name` (required): Unique identifier, lowercase letters and hyphens
- `description` (required): When Claude should delegate to this subagent
- `tools`: Tools the subagent can use (inherits all if omitted)
- `disallowedTools`: Tools to deny
- `model`: sonnet, opus, haiku, full model ID, or inherit (default: inherit)
- `permissionMode`: default, acceptEdits, auto, dontAsk, bypassPermissions, plan
- `maxTurns`: Maximum agentic turns
- `skills`: Skills to preload into context at startup
- `mcpServers`: MCP servers available to this subagent
- `hooks`: Lifecycle hooks scoped to this subagent
- `memory`: Persistent memory scope (user, project, local)
- `background`: true to always run as background task
- `effort`: low, medium, high, xhigh, max
- `isolation`: Set to 'worktree' for git worktree isolation
- `color`: Display color (red, blue, green, yellow, purple, orange, pink, cyan)
- `initialPrompt`: Auto-submitted as first user turn when running as main session agent

### Priority Order
1. Managed settings (highest)
2. --agents CLI flag
3. .claude/agents/ (project)
4. ~/.claude/agents/ (user)
5. Plugin agents/ directory (lowest)

### Built-in Subagents
- Explore: Haiku, read-only, for codebase search/analysis
- Plan: Inherits model, read-only, for plan mode research
- General-purpose: Inherits model, all tools, for complex multi-step tasks

### Key Constraints
- Subagents cannot spawn other subagents
- Subagents work within a single session
- Each subagent runs in its own context window
- Forked subagents inherit full conversation history

### Hook Events for Subagents
- PreToolUse/PostToolUse in frontmatter (scoped to subagent lifetime)
- SubagentStart/SubagentStop in settings.json (main session events)

### Memory System
- user scope: ~/.claude/agent-memory/<name>/
- project scope: .claude/agent-memory/<name>/
- local scope: .claude/agent-memory-local/<name>/
- First 200 lines or 25KB of MEMORY.md auto-loaded

### Fork Mode (experimental, v2.1.117+)
- CLAUDE_CODE_FORK_SUBAGENT=1 to enable
- Fork inherits full conversation history, system prompt, tools, model
- Named subagents still spawn as before
- Forks share prompt cache with main session (cheaper)
