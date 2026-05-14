# Bug: PreToolUse Hook Exit Code 2 Causes Claude to Stop Instead of Acting on Feedback
- **Source**: https://github.com/anthropics/claude-code/issues/24327
- **Retrieved**: 2026-03-27

## Summary
When a PreToolUse hook blocks a tool call (exit code 2), Claude sometimes stops and waits for user input instead of acting on the error feedback. Behavior is intermittent — sometimes Claude fixes and retries, other times it goes idle.

## Root Cause
Model-level interpretation issue. The model's trained behavior treats blocked actions conservatively, similar to a user clicking "deny". The distinction between "user denial" and "automated hook block with fixable feedback" isn't apparent to the model.

## Investigation Finding
Closed as COMPLETED. Could NOT reproduce consistently. In controlled runs, Claude continued and acted on feedback. Original session likely experienced context/turn budget exhaustion during long hook execution (20.5s latency), not dropped block feedback.

## Workarounds
1. Combine multiple PreToolUse hooks into single Python dispatcher to avoid parallel race conditions
2. Add explicit directives via UserPromptSubmit hooks telling Claude to treat hook blocks as quality gates, not user denials
3. Keep hook execution time short to avoid turn budget issues

## Key Insight
Exit code 2 feedback IS delivered to Claude, but the model's interpretation is non-deterministic. This is a model behavior issue, not a client bug.

## Status: CLOSED (COMPLETED)
