---
source: https://code.claude.com/docs/en/skills
retrieved: 2026-05-12
---

# Extend Claude with skills - Claude Code Docs

Skills extend what Claude can do. Create a SKILL.md file with instructions, and Claude adds it to its toolkit. Claude uses skills when relevant, or you can invoke one directly with /skill-name.

## Key Points

- Custom commands have been merged into skills. A file at .claude/commands/deploy.md and a skill at .claude/skills/deploy/SKILL.md both create /deploy and work the same way.
- Skills follow the Agent Skills open standard (agentskills.io), which works across multiple AI tools.
- Claude Code extends the standard with invocation control, subagent execution, and dynamic context injection.

## Where Skills Live

| Location   | Path                                                | Applies to                     |
|:-----------|:----------------------------------------------------|:-------------------------------|
| Enterprise | See managed settings                                | All users in your organization |
| Personal   | ~/.claude/skills/<skill-name>/SKILL.md              | All your projects              |
| Project    | .claude/skills/<skill-name>/SKILL.md                | This project only              |
| Plugin     | <plugin>/skills/<skill-name>/SKILL.md               | Where plugin is enabled        |

## SKILL.md Structure

Every skill needs a SKILL.md file with two parts: YAML frontmatter between --- markers and markdown content with instructions.

## Frontmatter Reference

| Field                      | Required    | Description |
|:---------------------------|:------------|:------------|
| name                       | No          | Display name. Lowercase letters, numbers, hyphens (max 64 chars). |
| description                | Recommended | What the skill does and when to use it. Combined with when_to_use, truncated at 1,536 chars. |
| when_to_use                | No          | Additional trigger context. Appended to description. |
| argument-hint              | No          | Hint for autocomplete (e.g., [issue-number]). |
| arguments                  | No          | Named positional arguments for $name substitution. Space-separated string or YAML list. |
| disable-model-invocation   | No          | Set true to prevent Claude auto-loading. Default: false. |
| user-invocable             | No          | Set false to hide from / menu. Default: true. |
| allowed-tools              | No          | Tools Claude can use without asking permission. Space-separated or YAML list. |
| model                      | No          | Model override for this skill. |
| effort                     | No          | Effort level override (low, medium, high, xhigh, max). |
| context                    | No          | Set to "fork" to run in a forked subagent context. |
| agent                      | No          | Subagent type when context: fork is set. |
| hooks                      | No          | Hooks scoped to this skill's lifecycle. |
| paths                      | No          | Glob patterns limiting when skill activates. |
| shell                      | No          | Shell for !`command` and ```! blocks (bash or powershell). |

## String Substitutions

| Variable               | Description |
|:------------------------|:------------|
| $ARGUMENTS              | All arguments passed when invoking the skill. |
| $ARGUMENTS[N]           | Specific argument by 0-based index. |
| $N                      | Shorthand for $ARGUMENTS[N]. |
| $name                   | Named argument from arguments frontmatter. |
| ${CLAUDE_SESSION_ID}    | Current session ID. |
| ${CLAUDE_EFFORT}        | Current effort level. |
| ${CLAUDE_SKILL_DIR}     | Directory containing this SKILL.md. |

## Dynamic Context Injection

!`<command>` syntax runs shell commands before content is sent to Claude. Output replaces the placeholder.

For multi-line: use fenced code block opened with ```!

## Invocation Control

| Frontmatter                      | You can invoke | Claude can invoke | When loaded |
|:---------------------------------|:---------------|:------------------|:------------|
| (default)                        | Yes            | Yes               | Description always, full on invoke |
| disable-model-invocation: true   | Yes            | No                | Description not in context |
| user-invocable: false            | No             | Yes               | Description always, full on invoke |

## Skill Content Lifecycle

When invoked, SKILL.md content enters conversation as a single message and stays for rest of session. Auto-compaction re-attaches most recent invocations within 25,000 token budget (first 5,000 tokens each).

## allowed-tools

Grants permission for listed tools while skill is active. Does NOT restrict other tools. Permission settings still govern unlisted tools.

Example: allowed-tools: Bash(git add *) Bash(git commit *) Bash(git status *)

## Supporting Files

```
my-skill/
├── SKILL.md           # Main instructions (required)
├── template.md        # Template for Claude to fill in
├── examples/
│   └── sample.md      # Example output showing expected format
└── scripts/
    └── validate.sh    # Script Claude can execute
```

## context: fork

Runs skill in isolated subagent. Content becomes the prompt. No access to conversation history.

## skillOverrides

Settings control: "on", "name-only", "user-invocable-only", "off"
