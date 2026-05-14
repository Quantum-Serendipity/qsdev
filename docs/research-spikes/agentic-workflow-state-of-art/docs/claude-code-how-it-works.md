# How Claude Code Works

- **Source**: https://code.claude.com/docs/en/how-claude-code-works
- **Retrieved**: 2026-03-15

## The Agentic Loop

Three phases: gather context, take action, verify results. Phases blend together. Claude uses tools throughout.

The loop adapts to the task. A question might only need context gathering. A bug fix cycles through all three phases repeatedly. Claude decides what each step requires based on previous results.

User is part of the loop — can interrupt at any point to steer.

## Core Tools (Five Categories)

| Category | Capabilities |
|---|---|
| File operations | Read files, edit code, create new files, rename and reorganize |
| Search | Find files by pattern, search content with regex, explore codebases |
| Execution | Run shell commands, start servers, run tests, use git |
| Web | Search the web, fetch documentation, look up error messages |
| Code intelligence | Type errors and warnings after edits, jump to definitions, find references |

Claude chooses tools based on prompt and what it learns along the way. Example for "fix failing tests":
1. Run the test suite to see what's failing
2. Read the error output
3. Search for the relevant source files
4. Read those files to understand the code
5. Edit the files to fix the issue
6. Run the tests again to verify

## Access Model

When run in a directory, Claude Code accesses:
- Project files in directory and subdirectories
- Terminal (any command user could run)
- Git state (current branch, uncommitted changes, recent history)
- CLAUDE.md for project-specific instructions
- Auto memory (learnings saved automatically)
- Extensions (MCP servers, skills, subagents, Chrome)

## Context Window Management

Context holds conversation history, file contents, command outputs, CLAUDE.md, loaded skills, system instructions. Claude compacts automatically — clears older tool outputs first, then summarizes conversation.

Skills load on demand. Subagents get their own fresh context, completely separate from main conversation.

## Safety Mechanisms

- **Checkpoints**: Every file edit is reversible. Before editing, Claude snapshots current contents.
- **Permission modes**: Default (asks before edits/commands), Auto-accept edits, Plan mode (read-only tools only)
- Allowed commands configurable in .claude/settings.json

## Best Practices

- Give Claude something to verify against (test cases, expected output)
- Explore before implementing (use plan mode for analysis first)
- Be specific upfront (reference specific files, mention constraints)
- Delegate, don't dictate (give context and direction, trust agent for details)
- It's conversational — iterate, don't start over

## Subagents

Specialized AI assistants with own context window, custom system prompt, specific tool access, independent permissions. Built-in: Explore, Plan, general-purpose. Custom subagents supported.

## Git Worktrees

--worktree flag creates isolated working directories for parallel development. Each sub-agent can get its own worktree. Agent teams coordinate through git-based system.

## Sessions

Independent, saved locally. Resume with --continue, fork with --fork-session. Git worktrees enable parallel sessions without file conflicts.
