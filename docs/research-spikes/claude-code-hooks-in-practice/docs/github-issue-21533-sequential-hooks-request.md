# Feature Request: Sequential Hook Execution Option
- **Source**: https://github.com/anthropics/claude-code/issues/21533
- **Retrieved**: 2026-03-27

## Problem
All matching hooks run in parallel. This prevents: dependent transformations, ordered validation, pipeline processing, priority-based execution, and resolution of resource conflicts (e.g., both hooks modifying same file).

## Proposed Solution
Add "sequential": true option to matcher groups. Hooks execute in array order. Each must complete before next starts. Exit code 2 skips subsequent hooks.

## Anthropic Response
Closed as "not planned" (2026-02-28).

## Key Insight
Gemini CLI already supports sequential hook execution — this is a proven pattern other AI tools have adopted.

## Status: Closed as NOT_PLANNED
