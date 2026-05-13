<!-- Source: https://code.claude.com/docs/en/common-workflows -->
<!-- Retrieved: 2026-05-12 -->

# Claude Code Common Workflows

## Workflow Categories
1. Understand new codebases: overview, find relevant code, trace execution flows
2. Fix bugs: share error, get recommendations, apply fix
3. Refactor code: find deprecated APIs, get recommendations, apply safely, verify tests
4. Work with tests: identify untested code, generate scaffolding, add edge cases, run/verify
5. Create pull requests: summarize changes, generate PR, review/refine
6. Handle documentation: find undocumented code, generate docs, review/enhance, verify standards
7. Work with images: drag/drop, paste, analyze
8. Reference files: @ syntax for files, directories, MCP resources
9. Run on schedule: Routines (cloud), Desktop tasks (local), GitHub Actions (CI), /loop (session)

## Session Management
- claude --continue to resume most recent
- claude --resume for picker
- claude --worktree for isolated parallel sessions

## Key Patterns
- Plan mode (Shift+Tab or --permission-mode plan)
- Delegate research to subagents for context isolation
- Pipe Claude into scripts (claude -p for non-interactive)
