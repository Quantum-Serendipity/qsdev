<!-- Source: https://code.claude.com/docs/en/best-practices -->
<!-- Retrieved: 2026-05-12 -->

# Claude Code Best Practices - Official Docs

## Core Constraint
Context window fills up fast, performance degrades as it fills. This is the most important resource to manage.

## Key Practices

### 1. Give Claude Verification
- Include tests, screenshots, expected outputs
- Single highest-leverage thing you can do
- UI changes via Claude in Chrome extension

### 2. Explore First, Then Plan, Then Code
- Plan mode separates exploration from execution
- Four phases: Explore → Plan → Implement → Commit
- Skip planning for small, clear-scope tasks

### 3. Provide Specific Context
- Reference specific files, mention constraints
- Use @ to reference files, paste images, pipe data
- Let Claude fetch what it needs

### 4. Configure Environment
- CLAUDE.md: persistent context, keep short and human-readable
- Run /init for starter CLAUDE.md
- Include: bash commands Claude can't guess, non-default code style, testing instructions, repo etiquette, architecture decisions, environment quirks, gotchas
- Exclude: things Claude can figure out from code, standard conventions, detailed API docs, frequently-changing info, file-by-file descriptions
- CLAUDE.md locations: ~/.claude/CLAUDE.md (global), ./CLAUDE.md (project), ./CLAUDE.local.md (personal)
- Skills: domain knowledge loaded on demand, not every session
- Hooks: deterministic, guaranteed actions

### 5. Communicate Effectively
- Ask codebase questions directly (like asking a senior engineer)
- Let Claude interview you for larger features

### 6. Manage Sessions
- Esc to stop, Esc+Esc to rewind, /clear between tasks
- After 2 failed corrections, /clear and start fresh
- Use subagents for investigation (separate context)
- Checkpoints persist across sessions
- /rename sessions for resumability

### 7. Automate and Scale
- claude -p for non-interactive/CI mode
- Multiple parallel sessions (worktrees, desktop, web, agent teams)
- Writer/Reviewer pattern across sessions
- Fan out across files with --allowedTools

### Common Failure Patterns
- Kitchen sink session (unrelated tasks in one context)
- Correcting over and over (context pollution)
- Over-specified CLAUDE.md (instructions get lost)
- Trust-then-verify gap (no verification criteria)
- Infinite exploration (unscoped investigation)
