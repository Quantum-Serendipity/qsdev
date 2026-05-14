# Claude Code Showcase - ChrisWiles/claude-code-showcase
- **Source**: https://github.com/ChrisWiles/claude-code-showcase/blob/main/.claude/settings.json
- **Retrieved**: 2026-03-27

## Overview
Comprehensive Claude Code project configuration example with hooks, skills, agents, commands, and GitHub Actions workflows.

## Core Settings
- **includeCoAuthoredBy**: true
- **Environment variables**: INSIDE_CLAUDE_CODE, BASH timeouts (420000ms max)

## Hook Events and Handlers

### UserPromptSubmit Hook
Executes `.claude/hooks/skill-eval.sh` with 5-second timeout before processing user input.

### PreToolUse Hook
Prevents file modifications on the main branch, requiring developers to "Create a feature branch first" before edits are allowed.

### PostToolUse Hooks (4 sequential handlers)

1. **Prettier Formatting**: Auto-formats JavaScript/TypeScript files (`.js|jsx|ts|tsx`) using `npx prettier --write` with 30-second timeout

2. **Dependency Installation**: Triggers `npm install` automatically when `package.json` changes, 60-second timeout, suppresses output on success

3. **Test Automation**: Runs `npm test` with `--findRelatedTests` flag when test files (`.test.js|jsx|ts|tsx`) are modified, 90-second timeout, shows last 30 lines

4. **TypeScript Validation**: Executes `npx tsc --noEmit` for type-checking on `.ts|tsx` files, displays type errors, non-blocking with 30-second timeout

All hooks use conditional bash logic to target specific file types and provide JSON-formatted feedback messages.
