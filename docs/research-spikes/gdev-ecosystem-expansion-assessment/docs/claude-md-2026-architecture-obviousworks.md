# CLAUDE.md 2026 Architecture - ObviousWorks
- **Source**: https://www.obviousworks.ch/en/designing-claude-md-right-the-2026-architecture-that-finally-makes-claude-code-work/
- **Retrieved**: 2026-05-14

## WHAT/WHY/HOW Structure

- **WHAT**: Project context (name, tech stack with versions, repository structure, dependencies)
- **WHY**: Principles (architectural decisions, code style, naming conventions, anti-patterns, security constraints)
- **HOW**: Workflows (build commands, testing procedures, commit strategies, deployment steps)

## Project Context Structure

Essential elements include:
- Specific technology versions (e.g., "React 18.3 + TypeScript 5.4 + Vite 5" rather than generic names)
- Repository layout mapping
- Critical dependencies identification
- Monorepo component descriptions

## Templates and Frameworks

The guide references a GitHub repository providing:
- CLAUDE.md template (under 200 lines)
- AGENTS.md for multi-agent workflows
- Hooks configuration in `.claude/settings.json`
- SKILL.md templates and examples

Available at: `github.com/obviousworks/agentic-coding-meta-prompt`

## @import System

The modular approach uses `@imports` for organization:
```
@docs/architecture.md
@.claude/git-conventions.md
@.claude/security-rules.md
```

Splits configuration across files while maintaining clarity and reducing bloat.

## Five Scope Cascade (last wins on conflicts)

- Global (`~/.claude/CLAUDE.md`)
- Project (`./CLAUDE.md`)
- Local secret (`./CLAUDE.local.md`)
- Folder (`./src/CLAUDE.md`)

## Project-Specific Context Advice

Replace vague instructions with precision. Instead of "write clean code," specify "Use camelCase for variables, PascalCase for React components." This articulates implicit team standards.
