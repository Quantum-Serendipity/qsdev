# Claude Code: Best Practices — Official Documentation
- **Source**: https://code.claude.com/docs/en/best-practices
- **Retrieved**: 2026-03-15
- **Type**: Official documentation

## Central Constraint
Claude's context window fills up fast, and performance degrades as it fills. Context holds entire conversation, every file read, every command output.

## Key Practices

### 1. Give Claude a Way to Verify Its Work
"This is the single highest-leverage thing you can do."
- Provide verification criteria (tests, expected outputs)
- Verify UI changes visually (screenshots)
- Address root causes, not symptoms

### 2. Explore First, Then Plan, Then Code
Four phases: Explore (Plan Mode) → Plan → Implement → Commit
- Use Plan Mode (Shift+Tab twice) for read-only exploration
- Ctrl+G to open plan in text editor for editing
- Skip planning when scope is clear and fix is small

### 3. Provide Specific Context
- Scope the task (which file, what scenario, testing preferences)
- Point to sources (git history, documentation)
- Reference existing patterns in codebase
- Describe symptoms with likely location and what "fixed" looks like
- Rich content: @file references, paste images, give URLs, pipe data

### 4. Configure Environment
- CLAUDE.md: Run /init to generate starter, keep under 200 lines
- Permissions: /permissions to allowlist safe commands, /sandbox for isolation
- CLI tools: gh, aws, gcloud, sentry-cli etc.
- MCP servers: claude mcp add for external tools
- Hooks: Deterministic actions that must happen every time
- Skills: Domain knowledge and reusable workflows
- Subagents: Specialized assistants for isolated tasks
- Plugins: /plugin to browse marketplace (9,000+ available)

### 5. Manage Sessions
- Course-correct early: Esc to stop, Esc+Esc to rewind, /clear to reset
- Manage context aggressively: /clear between tasks
- Context thresholds: 0-50% (work freely), 50-70% (attention), 70-90% (/compact), 90%+ (/clear mandatory)
- Use subagents for investigation (separate context)
- Rewind with checkpoints (every file edit reversible)
- Resume conversations: --continue, --resume

### 6. Automate and Scale
- Non-interactive mode: claude -p "prompt" for CI/scripts
- Multiple parallel sessions via desktop app, web, or agent teams
- Fan out across files with loops calling claude -p
- Writer/Reviewer pattern across sessions

### 7. Common Failure Patterns to Avoid
- Kitchen sink session (unrelated tasks in one session)
- Correcting over and over (after 2 failures: /clear and write better prompt)
- Over-specified CLAUDE.md (too long = Claude ignores rules)
- Trust-then-verify gap (always provide verification)
- Infinite exploration (scope narrowly or use subagents)
