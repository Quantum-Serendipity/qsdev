<!-- Source: https://code.claude.com/docs/en/sub-agents -->
<!-- Retrieved: 2026-05-12 -->
<!-- Note: Extracted sections relevant to CLAUDE.md inheritance and subagent context -->

# Subagent CLAUDE.md Inheritance

## Key Facts

- "Subagents receive only this system prompt (plus basic environment details like working directory), not the full Claude Code system prompt."
- Custom subagents do NOT automatically receive CLAUDE.md content — they get their own system prompt from the agent definition's markdown body.
- Built-in subagents (Explore, Plan, general-purpose) have their own system prompts.

## Agents Defined in User-Level Directories

User-level agents (~/.claude/agents/) are available across all projects. They inherit the parent conversation's permissions but get their own system prompt.

## Skills Inheritance

- Subagents do NOT inherit skills from the parent conversation.
- Skills must be explicitly listed in the `skills` frontmatter field.
- Without explicit listing, subagents can still discover and invoke skills through the Skill tool during execution.

## Permissions Inheritance

- Subagents inherit the permission context from the main conversation.
- If the parent uses bypassPermissions or acceptEdits, this takes precedence and cannot be overridden.
- If the parent uses auto mode, the subagent inherits auto mode and any permissionMode in its frontmatter is ignored.

## Hooks for Subagents

- Subagents can have their own hooks defined in frontmatter.
- Plugin subagents do NOT support hooks, mcpServers, or permissionMode fields (ignored).

## The omitClaudeMd Regression (v2.1.84+)

Issue #40459: Since v2.1.84, built-in subagents have omitClaudeMd:true, stripping CLAUDE.md context from Explore, Plan, and general-purpose subagents. Combined with tengu_slim_subagent_claudemd feature flag. Resulted in subagents ignoring project-specific rules, language preferences, and environment configurations. Status: open regression.
