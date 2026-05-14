# GitButler Claude Code Hooks Integration
- **Source**: https://docs.gitbutler.com/features/ai-integration/claude-code-hooks + search results
- **Retrieved**: 2026-03-27

## Overview
GitButler integrates with Claude Code through hooks to automatically manage commits and branches during AI-assisted coding sessions.

## Hook Configuration

Three hooks are used:
- **PreToolUse**: Command `but claude pre-tool` — Runs before code generation or editing
- **PostToolUse**: Command `but claude post-tool` — Runs after code editing
- **Stop**: Command `but claude stop` — Ensures all changes are committed and branches updated when agent finishes

## How It Works
Claude tells GitButler when code has been generated or edited and in which session. GitButler isolates changes into a single branch per session. If three sessions of Claude Code are running simultaneously, each communicates with GitButler at each step and changes are assigned to the correct branch automatically.

## Key Pattern
GitButler uses hooks to automatically stage file changes and create new commits, and can isolate work from different Claude sessions into separate branches automatically.

## Recommendation
Users should add a memory to ask Claude not to try committing using Git, as commits are handled by GitButler.
