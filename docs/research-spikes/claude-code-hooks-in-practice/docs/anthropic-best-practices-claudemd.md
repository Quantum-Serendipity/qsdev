# Anthropic Official Best Practices for Claude Code
- **Source**: https://code.claude.com/docs/en/best-practices
- **Retrieved**: 2026-03-27

## Write an effective CLAUDE.md

CLAUDE.md is a special file that Claude reads at the start of every conversation. Include Bash commands, code style, and workflow rules. This gives Claude persistent context it can't infer from code alone.

The `/init` command analyzes your codebase to detect build systems, test frameworks, and code patterns, giving you a solid foundation to refine.

There's no required format for CLAUDE.md files, but keep it short and human-readable. For example:

```markdown
# Code style
- Use ES modules (import/export) syntax, not CommonJS (require)
- Destructure imports when possible (eg. import { foo } from 'bar')

# Workflow
- Be sure to typecheck when you're done making a series of code changes
- Prefer running single tests, and not the whole test suite, for performance
```

CLAUDE.md is loaded every session, so only include things that apply broadly. For domain knowledge or workflows that are only relevant sometimes, use skills instead. Claude loads them on demand without bloating every conversation.

Keep it concise. For each line, ask: "Would removing this cause Claude to make mistakes?" If not, cut it. Bloated CLAUDE.md files cause Claude to ignore your actual instructions!

### What to Include vs. Exclude

| Include | Exclude |
|---------|---------|
| Bash commands Claude can't guess | Anything Claude can figure out by reading code |
| Code style rules that differ from defaults | Standard language conventions Claude already knows |
| Testing instructions and preferred test runners | Detailed API documentation (link to docs instead) |
| Repository etiquette (branch naming, PR conventions) | Information that changes frequently |
| Architectural decisions specific to your project | Long explanations or tutorials |
| Developer environment quirks (required env vars) | File-by-file descriptions of the codebase |
| Common gotchas or non-obvious behaviors | Self-evident practices like "write clean code" |

### Key Guidance

- If Claude keeps doing something you don't want despite having a rule against it, the file is probably too long and the rule is getting lost.
- If Claude asks you questions that are answered in CLAUDE.md, the phrasing might be ambiguous.
- Treat CLAUDE.md like code: review it when things go wrong, prune it regularly, and test changes by observing whether Claude's behavior actually shifts.
- You can tune instructions by adding emphasis (e.g., "IMPORTANT" or "YOU MUST") to improve adherence.
- Check CLAUDE.md into git so your team can contribute. The file compounds in value over time.

### Import Syntax

CLAUDE.md files can import additional files using `@path/to/import` syntax:

```markdown
See @README.md for project overview and @package.json for available npm commands.

# Additional Instructions
- Git workflow: @docs/git-instructions.md
- Personal overrides: @~/.claude/my-project-instructions.md
```

### File Locations

- **Home folder (`~/.claude/CLAUDE.md`)**: applies to all Claude sessions
- **Project root (`./CLAUDE.md`)**: check into git to share with your team
- **Parent directories**: useful for monorepos where both `root/CLAUDE.md` and `root/foo/CLAUDE.md` are pulled in automatically
- **Child directories**: Claude pulls in child CLAUDE.md files on demand when working with files in those directories

## Hooks vs. CLAUDE.md

"Use hooks for actions that must happen every time with zero exceptions."

"Hooks run scripts automatically at specific points in Claude's workflow. Unlike CLAUDE.md instructions which are advisory, hooks are deterministic and guarantee the action happens."

## The Over-Specified CLAUDE.md Anti-Pattern

"If your CLAUDE.md is too long, Claude ignores half of it because important rules get lost in the noise."

Fix: Ruthlessly prune. If Claude already does something correctly without the instruction, delete it or convert it to a hook.

## Skills as CLAUDE.md Overflow

Skills extend Claude's knowledge with information specific to your project, team, or domain. Claude applies them automatically when relevant, or you can invoke them directly.

Create skills in `.claude/skills/` with `SKILL.md` files that have YAML frontmatter (name, description) and markdown body.

## Key Principle

"Most best practices are based on one constraint: Claude's context window fills up fast, and performance degrades as it fills."

CLAUDE.md is loaded into context every single session. Every line consumes context window budget.
