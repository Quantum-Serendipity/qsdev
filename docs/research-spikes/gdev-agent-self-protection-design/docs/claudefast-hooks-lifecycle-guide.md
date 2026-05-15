# Claude Code Hooks: Complete Guide to All 12 Lifecycle Events

- **Source URL**: https://claudefa.st/blog/tools/hooks/hooks-guide
- **Retrieved**: 2026-05-15

## The 12 Hook Lifecycle Events

| Event | Timing | Blocking Capability |
|-------|--------|-------------------|
| SessionStart | Session begins/resumes | NO |
| UserPromptSubmit | User hits enter | YES |
| PreToolUse | Before tool execution | YES |
| PermissionRequest | Permission dialog appears | YES |
| PostToolUse | After tool succeeds | NO* |
| PostToolUseFailure | After tool fails | NO |
| SubagentStart | Spawning subagent | NO |
| SubagentStop | Subagent finishes | YES |
| Stop | Claude finishes responding | YES |
| PreCompact | Before compaction | NO |
| Setup | With --init/--maintenance | NO |
| SessionEnd | Session terminates | NO |
| Notification | Claude sends notification | NO |

## PermissionRequest "ask" Behavior

The documentation provides limited detail on the "ask" verdict. For PreToolUse, it states: "'ask': Prompts user for confirmation" -- but this appears in the JSON output examples, not explicitly for PermissionRequest hooks. The guide doesn't clarify whether PermissionRequest hooks can return "ask" or what the user interaction flow entails.

## Multiple Hooks & Interaction Patterns

The guide states that "multiple hooks run in parallel" in certain contexts. However, the documentation doesn't specify:
- Priority ordering when hooks conflict
- How parallel execution resolves competing decisions
- Precedence rules across matchers

## Decision Precedence Rules

**Not explicitly documented in this guide.** The guide covers individual hook decision types (allow/deny/ask for PreToolUse; behavior options for PermissionRequest) but provides no formal precedence hierarchy when multiple hooks return conflicting decisions on the same event.

This represents a significant gap -- organizations using multiple hooks need clarity on which decision wins.
