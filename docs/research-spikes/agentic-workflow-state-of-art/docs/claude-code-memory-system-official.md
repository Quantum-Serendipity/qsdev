# Claude Code: Memory System (CLAUDE.md + Auto Memory) — Official Documentation
- **Source**: https://code.claude.com/docs/en/memory
- **Retrieved**: 2026-03-15
- **Type**: Official documentation

## Two Memory Systems

### CLAUDE.md Files (User-Written)
- Instructions you write for persistent context
- Loaded at start of every session
- Scopes: Managed policy, Project, User

### Auto Memory (Claude-Written)
- Notes Claude writes itself based on corrections and preferences
- Stored in ~/.claude/projects/<project>/memory/
- First 200 lines of MEMORY.md loaded at session start
- Topic files (debugging.md, patterns.md) read on demand

## CLAUDE.md Hierarchy
| Scope | Location | Shared with |
|---|---|---|
| Managed policy | /Library/Application Support/ClaudeCode/CLAUDE.md (macOS) | All users in org |
| Project | ./CLAUDE.md or ./.claude/CLAUDE.md | Team via source control |
| User | ~/.claude/CLAUDE.md | Just you (all projects) |

## Import Syntax
@path/to/file imports expand at launch. Relative paths resolve relative to containing file. Max 5 hops of recursion.

## .claude/rules/ Directory
- Modular instruction files (testing.md, api-design.md)
- Path-specific rules via YAML frontmatter with paths field
- Glob patterns: **/*.ts, src/**/*.tsx, etc.
- User-level rules: ~/.claude/rules/
- Symlinks supported for sharing across projects

## Writing Effective Instructions
- Target under 200 lines per file
- Use markdown headers and bullets
- Make instructions specific and verifiable
- Check for contradictions across files
- Use emphasis (IMPORTANT, YOU MUST) for critical rules

## Key Behaviors
- CLAUDE.md files above working directory: loaded in full at launch
- CLAUDE.md files in subdirectories: lazy-loaded when files accessed
- CLAUDE.md survives compaction (re-read from disk after /compact)
- Auto memory is machine-local, scoped to git repo
- claudeMdExcludes setting for monorepos
- Managed policy CLAUDE.md cannot be excluded
