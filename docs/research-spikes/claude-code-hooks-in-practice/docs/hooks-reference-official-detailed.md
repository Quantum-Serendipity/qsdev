# Claude Code Hooks Reference — Detailed Execution Behavior
- **Source**: https://code.claude.com/docs/en/hooks
- **Retrieved**: 2026-03-27

## Execution Order
- Hooks within a matcher group run in PARALLEL
- Multiple matcher groups under one event run SEQUENTIALLY
- Identical handlers are deduplicated (by command string or URL)

## Timeout Defaults
| Handler Type | Default Timeout |
|---|---|
| Command | 600 seconds (10 min) |
| HTTP | 30 seconds |
| Prompt | 30 seconds |
| Agent | 60 seconds |

## Exit Code Behavior
- Exit 0: success, JSON parsed, execution proceeds
- Exit 2: blocking error — stderr fed back to Claude as error message
- Any other exit code (1, 3, 127, etc.): non-blocking, stderr shown in verbose mode only

## Exit Code 2 Per Event
- PreToolUse: BLOCKS tool call
- PermissionRequest: DENIES permission
- UserPromptSubmit: BLOCKS prompt, erases from context
- Stop: PREVENTS Claude from stopping, continues conversation
- SubagentStop: PREVENTS subagent from stopping
- PostToolUse/PostToolUseFailure: Cannot block (shows stderr to Claude)
- PreCompact/PostCompact: Cannot block (stderr to user only)
- SessionStart/SessionEnd: Cannot block

## Async Hooks
- Only command hooks support async: true
- Cannot affect decisions — side effects only
- Exit codes and JSON output ignored

## Subagent Interaction
- PreToolUse/PostToolUse fire for subagent tool calls with agent_id and agent_type fields
- SubagentStart cannot block — can only inject context
- SubagentStop can block (exit 2) to prevent subagent from finishing
- Hooks defined in skill/agent frontmatter are scoped to that component's lifecycle

## Prompt Handler Type
- Single-turn LLM evaluation
- $ARGUMENTS placeholder replaced with hook input JSON
- Default model: "a fast model" (unspecified)
- Can specify model aliases (sonnet, opus) or full IDs
- Uses tokens from API quota
- No tool access, no conversation history
- Returns binary allow/deny decision

## Agent Handler Type
- Spawns full subagent with tool access (Read, Grep, Glob, Bash, etc.)
- Multi-step verification with reasoning
- Higher token cost, scales with tool calls
- Cannot be used for PreToolUse (circular dependency)
- 60s default timeout

## PreToolUse Blocking Behavior
- Tool call prevented entirely
- Claude receives denial reason as error message
- Claude continues reasoning — can try different approach, modify command, ask for clarification
- Three outcomes: allow, deny, ask (user prompted)
- Can modify tool input with updatedInput field
- Permission rules still apply even when hook returns "allow"

## Performance Notes
- PreToolUse hooks run on every tool call (most frequent)
- Parallel execution within matcher groups
- Slowest hook determines total latency
- SessionStart hooks should be kept fast
- Deduplication reduces overhead for identical commands/URLs
- Hook output consumes context window space
