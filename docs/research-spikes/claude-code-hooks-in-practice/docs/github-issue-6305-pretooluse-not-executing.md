# Bug: Pre/PostToolUse Hooks Not Executing Despite Correct Configuration
- **Source**: https://github.com/anthropics/claude-code/issues/6305
- **Retrieved**: 2026-03-27

## Summary
PreToolUse and PostToolUse hooks are configured correctly but never execute, while Stop, SubagentStop, and UserPromptSubmit hooks work. Multiple users affected (13+ upvotes, 22+ comments).

## Key Detail: Security Feature
Sessions take a snapshot of hooks at startup. Config is locked during session to prevent injection attacks. Changes require /hooks command or session restart.

## Key Detail: Still Broken After Workarounds
Users report hooks never execute even after:
- Using /hooks command multiple times
- Restarting sessions dozens of times
- Following documentation exactly
- Manual hook testing confirms scripts work independently

## Anthropic Response
Team member explained the security snapshot mechanism but this did not resolve the issue for affected users.

## Related Issues
- #6403: PostToolUse not executing despite correct stdin JSON
- #6409: Grep tool causes "hook execution cancelled"
- #6522: Subagents not executing configured hooks correctly
- #6876: SED commands damage files despite PreToolUse blocking

## Status: OPEN (as of retrieval date)
