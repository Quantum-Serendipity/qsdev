<!-- Source: https://code.claude.com/docs/en/skills -->
<!-- Retrieved: 2026-05-14 -->

# Extend Claude with Skills - Claude Code Documentation

Skills extend what Claude can do. Create a SKILL.md file with instructions, and Claude adds it to its toolkit. Claude uses skills when relevant, or you can invoke one directly with /skill-name.

Create a skill when you keep pasting the same instructions, checklist, or multi-step procedure into chat, or when a section of CLAUDE.md has grown into a procedure rather than a fact. Unlike CLAUDE.md content, a skill's body loads only when it's used, so long reference material costs almost nothing until you need it.

Custom commands have been merged into skills. A file at .claude/commands/deploy.md and a skill at .claude/skills/deploy/SKILL.md both create /deploy and work the same way.

Claude Code skills follow the Agent Skills open standard (agentskills.io), which works across multiple AI tools. Claude Code extends the standard with additional features like invocation control, subagent execution, and dynamic context injection.

## Where Skills Live

| Location   | Path                                          | Applies to                     |
|:-----------|:----------------------------------------------|:-------------------------------|
| Enterprise | See managed settings                          | All users in your organization |
| Personal   | ~/.claude/skills/<skill-name>/SKILL.md        | All your projects              |
| Project    | .claude/skills/<skill-name>/SKILL.md          | This project only              |
| Plugin     | <plugin>/skills/<skill-name>/SKILL.md         | Where plugin is enabled        |

When skills share the same name across levels, enterprise overrides personal, and personal overrides project.

## Frontmatter Reference

All fields are optional. Only description is recommended.

| Field                      | Required    | Description |
|:---------------------------|:------------|:------------|
| name                       | No          | Display name. Lowercase letters, numbers, hyphens (max 64 chars) |
| description                | Recommended | What the skill does and when to use it. Claude uses this to decide when to apply the skill. Truncated at 1,536 chars in listing. |
| when_to_use                | No          | Additional trigger context. Appended to description. |
| argument-hint              | No          | Hint shown during autocomplete |
| arguments                  | No          | Named positional arguments for $name substitution |
| disable-model-invocation   | No          | true = only user can invoke. Default: false |
| user-invocable             | No          | false = hidden from / menu. Default: true |
| allowed-tools              | No          | Tools Claude can use without permission when skill is active |
| model                      | No          | Model override for this skill |
| effort                     | No          | Effort level override |
| context                    | No          | Set to 'fork' to run in subagent |
| agent                      | No          | Which subagent type when context: fork |
| hooks                      | No          | Hooks scoped to skill lifecycle |
| paths                      | No          | Glob patterns limiting when skill activates |
| shell                      | No          | bash (default) or powershell |

## String Substitutions

| Variable               | Description |
|:-----------------------|:------------|
| $ARGUMENTS             | All arguments passed when invoking |
| $ARGUMENTS[N]          | Specific argument by 0-based index |
| $N                     | Shorthand for $ARGUMENTS[N] |
| $name                  | Named argument from arguments frontmatter |
| ${CLAUDE_SESSION_ID}   | Current session ID |
| ${CLAUDE_EFFORT}       | Current effort level |
| ${CLAUDE_SKILL_DIR}    | Directory containing SKILL.md |

## Dynamic Context Injection

The !`<command>` syntax runs shell commands before the skill content is sent to Claude. The command output replaces the placeholder.

Example:
```yaml
---
name: pr-summary
description: Summarize changes in a pull request
context: fork
agent: Explore
allowed-tools: Bash(gh *)
---

## Pull request context
- PR diff: !`gh pr diff`
- PR comments: !`gh pr view --comments`
- Changed files: !`gh pr diff --name-only`
```

For multi-line commands, use a fenced code block opened with ```!

## Skill Content Lifecycle

When invoked, rendered SKILL.md content enters the conversation as a single message and stays for the rest of the session. Auto-compaction carries invoked skills forward within a token budget. Re-attached skills share a combined budget of 25,000 tokens. Budget fills starting from most recently invoked skill.

## Control Who Invokes

- disable-model-invocation: true — Only user can invoke
- user-invocable: false — Only Claude can invoke (background knowledge)

| Frontmatter                      | User can invoke | Claude can invoke | When loaded |
|:---------------------------------|:----------------|:------------------|:------------|
| (default)                        | Yes             | Yes               | Description always in context, full skill loads when invoked |
| disable-model-invocation: true   | Yes             | No                | Description not in context, full skill loads when you invoke |
| user-invocable: false            | No              | Yes               | Description always in context, full skill loads when invoked |

## MCP Server Handling

MCP servers override by name with priority: local > project > user.

Tool search is on by default, so idle MCP tools consume minimal context. MCP connections can fail silently mid-session.

## Context Costs

| Feature         | When it loads             | What loads                                    | Context cost |
|:----------------|:--------------------------|:----------------------------------------------|:-------------|
| CLAUDE.md       | Session start             | Full content                                  | Every request |
| Skills          | Session start + when used | Descriptions at start, full content when used | Low (descriptions every request) |
| MCP servers     | Session start             | Tool names; full schemas on demand            | Low until tool is used |
| Subagents       | When spawned              | Fresh context with specified skills           | Isolated from main session |
| Hooks           | On trigger                | Nothing (runs externally)                     | Zero unless hook returns output |
