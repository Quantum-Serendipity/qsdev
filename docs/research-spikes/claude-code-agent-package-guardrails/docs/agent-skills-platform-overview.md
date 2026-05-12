---
source: https://platform.claude.com/docs/en/agents-and-tools/agent-skills/overview
retrieved: 2026-05-12
---

# Agent Skills Overview

Agent Skills are modular capabilities that extend Claude's functionality. Each Skill packages instructions, metadata, and optional resources (scripts, templates) that Claude uses automatically when relevant.

## Why use Skills

Skills are reusable, filesystem-based resources that provide Claude with domain-specific expertise. Unlike prompts (conversation-level instructions for one-off tasks), Skills load on-demand and eliminate the need to repeatedly provide the same guidance across multiple conversations.

## How Skills work

Skills leverage Claude's VM environment to provide capabilities beyond what's possible with prompts alone. Claude operates in a virtual machine with filesystem access, allowing Skills to exist as directories containing instructions, executable code, and reference materials.

### Three levels of loading

| Level | When Loaded | Token Cost | Content |
|-------|------------|------------|---------|
| **Level 1: Metadata** | Always (at startup) | ~100 tokens per Skill | `name` and `description` from YAML frontmatter |
| **Level 2: Instructions** | When Skill is triggered | Under 5k tokens | SKILL.md body with instructions and guidance |
| **Level 3+: Resources** | As needed | Effectively unlimited | Bundled files executed via bash without loading contents into context |

## Skill Structure

```yaml
---
name: your-skill-name
description: Brief description of what this Skill does and when to use it
---

# Your Skill Name

## Instructions
[Clear, step-by-step guidance for Claude to follow]

## Examples
[Concrete examples of using this Skill]
```

## Security Considerations

Use Skills only from trusted sources. Skills provide Claude with new capabilities through instructions and code, and a malicious Skill can direct Claude to invoke tools or execute code in ways that don't match the Skill's stated purpose.

Key security considerations:
- Audit thoroughly: Review all files bundled in the Skill
- External sources are risky: Skills that fetch data from external URLs pose particular risk
- Tool misuse: Malicious Skills can invoke tools in harmful ways
- Data exposure: Skills with access to sensitive data could leak information
- Treat like installing software

## Claude Code Specifics

- Full network access: Skills have the same network access as any other program on the user's computer
- Global package installation discouraged: Skills should only install packages locally
- Custom Skills are filesystem-based and don't require API uploads

## Limitations

- Custom Skills do not sync across surfaces
- Skills uploaded to one surface are not automatically available on others
- Claude Code Skills are filesystem-based and separate from both claude.ai and API
