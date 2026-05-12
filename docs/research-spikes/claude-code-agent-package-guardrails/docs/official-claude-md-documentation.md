<!-- Source: https://code.claude.com/docs/en/claude-md -->
<!-- Retrieved: 2026-05-12 -->

# How Claude remembers your project — Official CLAUDE.md Documentation

## Key Architecture Facts

- CLAUDE.md content is delivered as a **user message after the system prompt**, not as part of the system prompt itself.
- Claude reads it and tries to follow it, but there's **no guarantee of strict compliance**, especially for vague or conflicting instructions.
- Both CLAUDE.md and auto memory are **loaded at the start of every conversation**. Claude treats them as **context, not enforced configuration**.
- The more specific and concise your instructions, the more consistently Claude follows them.

## CLAUDE.md vs Settings

| Concern | Configure in |
|---------|-------------|
| Block specific tools, commands, or file paths | Managed settings: `permissions.deny` |
| Enforce sandbox isolation | Managed settings: `sandbox.enabled` |
| Code style and quality guidelines | Managed CLAUDE.md |
| Data handling and compliance reminders | Managed CLAUDE.md |
| Behavioral instructions for Claude | Managed CLAUDE.md |

**"Settings rules are enforced by the client regardless of what Claude decides to do. CLAUDE.md instructions shape Claude's behavior but are not a hard enforcement layer."**

## File Hierarchy and Loading

| Scope | Location | Shared with |
|-------|----------|-------------|
| Managed policy | `/etc/claude-code/CLAUDE.md` (Linux) | All users on machine |
| Project instructions | `./CLAUDE.md` or `./.claude/CLAUDE.md` | Team via source control |
| User instructions | `~/.claude/CLAUDE.md` | Just you (all projects) |
| Local instructions | `./CLAUDE.local.md` | Just you (current project) |

- Files walk UP the directory tree from CWD
- All discovered files concatenated (not overriding)
- Ordered root-down (closer to CWD = read last = higher effective priority)
- CLAUDE.local.md appended after CLAUDE.md at each level

## Writing Effective Instructions

- **Size**: target under 200 lines per file. Longer files consume more context and reduce adherence.
- **Specificity**: concrete enough to verify. "Use 2-space indentation" not "Format code properly".
- **Consistency**: contradicting rules may be followed arbitrarily.
- **Structure**: markdown headers and bullets.

## Import Syntax

`@path/to/import` anywhere in CLAUDE.md. Relative paths resolve relative to the containing file. Max depth: 5 hops.

## Subagent and Compaction Behavior

- Project-root CLAUDE.md **survives compaction**: re-read from disk and re-injected.
- Nested subdirectory CLAUDE.md files are NOT re-injected automatically after compaction.
- Subdirectory CLAUDE.md files load on demand when Claude reads files in those directories.

## Troubleshooting: "Claude isn't following my CLAUDE.md"

Official guidance:
1. Run `/memory` to verify files are loaded
2. Make instructions more specific
3. Look for conflicting instructions across files
4. If instruction must run at a specific point → write it as a **hook** instead
5. For system-prompt-level instructions → use `--append-system-prompt`

## `.claude/rules/` Directory

- Markdown files in `.claude/rules/` loaded at launch (unless path-scoped)
- Path-specific rules use YAML frontmatter with `paths` field
- User-level rules in `~/.claude/rules/` apply to every project
- User rules loaded before project rules (project = higher priority)

## Managed CLAUDE.md

- Can be deployed via `/etc/claude-code/CLAUDE.md` or `claudeMd` key in managed-settings.json
- Cannot be excluded by individual settings
- Loads before user and project CLAUDE.md
