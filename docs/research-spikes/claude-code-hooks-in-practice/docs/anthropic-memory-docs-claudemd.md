# Anthropic Official Documentation: How Claude Remembers Your Project
- **Source**: https://code.claude.com/docs/en/memory
- **Retrieved**: 2026-03-27

## CLAUDE.md vs Auto Memory

Two complementary memory systems. Both loaded at start of every conversation. Claude treats them as context, not enforced configuration.

| | CLAUDE.md files | Auto memory |
|---|---|---|
| Who writes it | You | Claude |
| What it contains | Instructions and rules | Learnings and patterns |
| Scope | Project, user, or org | Per working tree |
| Loaded into | Every session | Every session (first 200 lines or 25KB) |
| Use for | Coding standards, workflows, project architecture | Build commands, debugging insights, preferences |

## CLAUDE.md File Locations

| Scope | Location | Purpose | Shared with |
|---|---|---|---|
| Managed policy | macOS: `/Library/Application Support/ClaudeCode/CLAUDE.md`, Linux: `/etc/claude-code/CLAUDE.md` | Organization-wide instructions | All users in org |
| Project instructions | `./CLAUDE.md` or `./.claude/CLAUDE.md` | Team-shared instructions | Team via source control |
| User instructions | `~/.claude/CLAUDE.md` | Personal preferences | Just you (all projects) |

## Loading Behavior

- Claude walks UP directory tree from working directory, loading all CLAUDE.md files found
- Subdirectory CLAUDE.md files load ON DEMAND when Claude reads files in those directories
- Ancestor loading at launch, descendant loading lazy
- HTML comments (`<!-- -->`) stripped before injection (free documentation for humans)
- CLAUDE.md files loaded in full regardless of length (unlike auto memory which caps at 200 lines)

## Writing Effective Instructions

- **Size**: target under 200 lines per file
- **Structure**: markdown headers and bullets
- **Specificity**: concrete enough to verify ("Use 2-space indentation" not "Format code properly")
- **Consistency**: contradictory rules → Claude picks arbitrarily

## Import Syntax

`@path/to/import` — relative to the file containing the import. Max depth: 5 hops.

```markdown
See @README for project overview and @package.json for npm commands.
@~/.claude/my-project-instructions.md
```

First time external imports encountered → approval dialog shown.

## .claude/rules/ Directory

For larger projects, organize instructions into topic-specific files:

```
.claude/rules/
├── code-style.md
├── testing.md
└── security.md
```

- All .md files discovered recursively
- Can use subdirectories (frontend/, backend/)
- Rules without `paths` frontmatter load at launch
- Rules with `paths` frontmatter load only when Claude works with matching files

### Path-Specific Rules

```yaml
---
paths:
  - "src/api/**/*.ts"
---
# API Development Rules
```

Glob patterns supported. Multiple patterns allowed. Brace expansion works.

### Symlinks Supported

```bash
ln -s ~/shared-claude-rules .claude/rules/shared
ln -s ~/company-standards/security.md .claude/rules/security.md
```

### User-Level Rules

`~/.claude/rules/` — personal rules for all projects. Loaded before project rules (project rules have higher priority).

## Large Team Management

### Organization-Wide CLAUDE.md

Managed policy location. Cannot be excluded by individual settings. Deploy via MDM, Group Policy, Ansible.

Use managed settings for technical enforcement, managed CLAUDE.md for behavioral guidance:
- Block tools/commands → managed settings (permissions.deny)
- Code style/quality → managed CLAUDE.md
- Compliance reminders → managed CLAUDE.md

### Excluding CLAUDE.md Files

`claudeMdExcludes` setting — glob patterns to skip irrelevant CLAUDE.md files in monorepos.

```json
{
  "claudeMdExcludes": [
    "**/monorepo/CLAUDE.md",
    "/home/user/monorepo/other-team/.claude/rules/**"
  ]
}
```

## Troubleshooting: Claude Not Following CLAUDE.md

"CLAUDE.md content is delivered as a user message after the system prompt, not as part of the system prompt itself. Claude reads it and tries to follow it, but there's no guarantee of strict compliance, especially for vague or conflicting instructions."

Debug steps:
1. Run `/memory` to verify files are loaded
2. Check file location is in loading path
3. Make instructions more specific
4. Look for conflicting instructions across files

For system-prompt-level instructions: use `--append-system-prompt` (scripts/automation only).

## Key Insight: Compaction Behavior

"CLAUDE.md fully survives compaction. After `/compact`, Claude re-reads your CLAUDE.md from disk and re-injects it fresh into the session."

If instruction disappeared after compaction, it was given only in conversation, not written to CLAUDE.md.

## InstructionsLoaded Hook

Use the `InstructionsLoaded` hook to log exactly which instruction files are loaded, when they load, and why. Useful for debugging path-specific rules or lazy-loaded files.
