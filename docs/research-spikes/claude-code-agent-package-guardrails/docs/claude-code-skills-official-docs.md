---
source: https://code.claude.com/docs/en/skills
retrieved: 2026-05-12
---

# Extend Claude with skills

Skills extend what Claude can do. Create a `SKILL.md` file with instructions, and Claude adds it to its toolkit. Claude uses skills when relevant, or you can invoke one directly with `/skill-name`.

Create a skill when you keep pasting the same instructions, checklist, or multi-step procedure into chat, or when a section of CLAUDE.md has grown into a procedure rather than a fact. Unlike CLAUDE.md content, a skill's body loads only when it's used, so long reference material costs almost nothing until you need it.

> **Custom commands have been merged into skills.** A file at `.claude/commands/deploy.md` and a skill at `.claude/skills/deploy/SKILL.md` both create `/deploy` and work the same way. Your existing `.claude/commands/` files keep working. Skills add optional features: a directory for supporting files, frontmatter to control whether you or Claude invokes them, and the ability for Claude to load them automatically when relevant.

Claude Code skills follow the Agent Skills open standard, which works across multiple AI tools. Claude Code extends the standard with additional features like invocation control, subagent execution, and dynamic context injection.

## Where skills live

| Location   | Path                                                | Applies to                     |
| :--------- | :-------------------------------------------------- | :----------------------------- |
| Enterprise | See managed settings                                | All users in your organization |
| Personal   | `~/.claude/skills/<skill-name>/SKILL.md`            | All your projects              |
| Project    | `.claude/skills/<skill-name>/SKILL.md`              | This project only              |
| Plugin     | `<plugin>/skills/<skill-name>/SKILL.md`             | Where plugin is enabled        |

When skills share the same name across levels, enterprise overrides personal, and personal overrides project.

## Skill Directory Structure

```
my-skill/
├── SKILL.md           # Main instructions (required)
├── template.md        # Template for Claude to fill in
├── examples/
│   └── sample.md      # Example output showing expected format
└── scripts/
    └── validate.sh    # Script Claude can execute
```

## Frontmatter Reference

All fields are optional. Only `description` is recommended.

| Field                      | Required    | Description                                                                                                |
| :------------------------- | :---------- | :--------------------------------------------------------------------------------------------------------- |
| `name`                     | No          | Display name. Lowercase letters, numbers, hyphens (max 64 chars).                                          |
| `description`              | Recommended | What the skill does and when to use it. Truncated at 1,536 chars.                                          |
| `when_to_use`              | No          | Additional context for when Claude should invoke the skill.                                                |
| `argument-hint`            | No          | Hint shown during autocomplete. Example: `[issue-number]`                                                  |
| `arguments`                | No          | Named positional arguments for `$name` substitution.                                                       |
| `disable-model-invocation` | No          | Set to `true` to prevent Claude from auto-loading. Default: `false`.                                       |
| `user-invocable`           | No          | Set to `false` to hide from `/` menu. Default: `true`.                                                     |
| `allowed-tools`            | No          | Tools Claude can use without asking permission when skill is active.                                       |
| `model`                    | No          | Model to use when this skill is active.                                                                    |
| `effort`                   | No          | Effort level when this skill is active.                                                                    |
| `context`                  | No          | Set to `fork` to run in a forked subagent context.                                                         |
| `agent`                    | No          | Which subagent type to use when `context: fork` is set.                                                    |
| `hooks`                    | No          | Hooks scoped to this skill's lifecycle.                                                                    |
| `paths`                    | No          | Glob patterns that limit when this skill is activated.                                                     |
| `shell`                    | No          | Shell to use for inline commands. `bash` (default) or `powershell`.                                        |

## Control who invokes a skill

| Frontmatter                      | You can invoke | Claude can invoke | When loaded into context                                     |
| :------------------------------- | :------------- | :---------------- | :----------------------------------------------------------- |
| (default)                        | Yes            | Yes               | Description always in context, full skill loads when invoked |
| `disable-model-invocation: true` | Yes            | No                | Description not in context, full skill loads when you invoke |
| `user-invocable: false`          | No             | Yes               | Description always in context, full skill loads when invoked |

## Skill Content Lifecycle

When invoked, SKILL.md content enters the conversation as a single message and stays for the rest of the session. Auto-compaction carries invoked skills forward within a token budget — re-attaches the most recent invocation of each skill after the summary, keeping the first 5,000 tokens of each. Re-attached skills share a combined budget of 25,000 tokens.

## Pre-approve tools for a skill

The `allowed-tools` field grants permission for listed tools while the skill is active. Does not restrict which tools are available. Permission settings still govern unlisted tools.

```yaml
---
name: commit
description: Stage and commit the current changes
disable-model-invocation: true
allowed-tools: Bash(git add *) Bash(git commit *) Bash(git status *)
---
```

## Restrict Claude's skill access

Three ways to control which skills Claude can invoke:

1. **Disable all skills** by denying the Skill tool in `/permissions`: `Skill`
2. **Allow or deny specific skills** using permission rules: `Skill(commit)`, `Skill(deploy *)`
3. **Hide individual skills** with `disable-model-invocation: true`

## Dynamic Context Injection

The `!`command`` syntax runs shell commands before skill content is sent to Claude. Output replaces the placeholder.

## Run skills in a subagent

`context: fork` runs skill in isolation. The skill content becomes the prompt driving the subagent. Won't have access to conversation history.

| Approach                     | System prompt                             | Task                        | Also loads                   |
| :--------------------------- | :---------------------------------------- | :-------------------------- | :--------------------------- |
| Skill with `context: fork`   | From agent type (`Explore`, `Plan`, etc.) | SKILL.md content            | CLAUDE.md                    |
| Subagent with `skills` field | Subagent's markdown body                  | Claude's delegation message | Preloaded skills + CLAUDE.md |
