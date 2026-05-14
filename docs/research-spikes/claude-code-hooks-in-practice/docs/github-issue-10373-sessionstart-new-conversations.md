# Bug: SessionStart Hooks Not Working for New Conversations
- **Source**: https://github.com/anthropics/claude-code/issues/10373
- **Retrieved**: 2026-03-27

## Summary
SessionStart hooks execute but their output is never processed or injected into Claude's context when starting brand new conversations. Works for /clear, /compact, and URL resume — fails silently for new sessions only.

## Root Cause
The qz() function that processes hook output is only called in 3 places:
1. During /compact command
2. During URL resume
3. During /clear command

For brand new interactive conversations, the wm6() function only replays old hook responses from message history and never calls qz("startup").

## Impact
17+ upvotes. Affects all users configuring SessionStart hooks for context injection.

## Workaround
Run /clear at the start of each new session to trigger SessionStart hooks.

## Status: Related to #10287, #9591, #12634, #14433, #15726
