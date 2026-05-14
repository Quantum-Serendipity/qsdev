# Coding Agent Error Recovery Patterns

- **Sources**:
  - https://www.gocodeo.com/post/error-recovery-and-fallback-strategies-in-ai-agent-development
  - https://medium.com/@zaiinn440/building-a-coding-agent-to-solve-swe-bench-08939711a65a
  - https://apxml.com/courses/langchain-production-llm/chapter-2-sophisticated-agents-tools/agent-error-handling
  - https://github.com/SWE-agent/SWE-agent/issues/1194
  - https://callsphere.tech/blog/claude-agent-sdk-autonomous-agents
  - https://arxiv.org/pdf/2601.06112 (ReliabilityBench)
- **Retrieved**: 2026-03-15

## Core Error Recovery Strategies

### 1. Structured Retry Logic
Wrap tool invocations with retry logic including:
- Input validation using JSON schema or type constraints
- Exponential backoff retry strategies
- Particularly effective for transient issues (network, rate limiting)

### 2. Test-Execute Loop Recovery
- Identify failed test files from SWE dataset
- Re-run tests to check if errors resolved after fix
- If issues found, loop back to Suggestion Agent for revised fix
- Reduces collisions between functions, comments, surrounding lines

### 3. Linter-Based Edit Rejection (SWE-agent)
- Linter runs when edit command issued
- Invalid edits (syntactically incorrect) are rejected entirely
- Agent shown errors with before/after snippets
- Must retry until edit passes linting
- Prevents cascading errors from bad edits

### 4. Tool-Call Failure Reporting (Warp)
- Report tool-call failures back to the LLM
- Fall back to alternate models
- Put reasonable restrictions on tool use
- All increase odds agent accomplishes task

### 5. Stuck Detection (OpenHands)
- Automatic detection of pathological agent states
- Infinite loops: repeated same action without progress
- Redundant tool calls: querying same information repeatedly
- System auto-terminates to prevent resource waste

## Known Failure Modes

### Retry Loops
Agents can get stuck in loops with windowed edit during retry attempts. When edit cannot be applied, agent informed why and may retry. Common failure reasons:
- Agent misunderstanding current directory
- Duplicating path components
- Repeatedly attempting same failed edit

### Context Window Overflow
Long error recovery loops consume context window, reducing quality of subsequent reasoning.

### Cascading Errors
One bad edit can cause multiple downstream failures. SWE-agent's approach of rejecting invalid edits prevents this.

## Error Categories and Handling

| Error Type | Strategy | Example |
|---|---|---|
| Syntax error in edit | Reject and retry | SWE-agent linter |
| Test failure | Analyze output, modify fix, retry | Standard loop |
| Tool execution failure | Report to LLM, retry with different approach | Warp pattern |
| Network/transient | Exponential backoff retry | Standard retry |
| Stuck in loop | Detect and terminate | OpenHands stuck detection |
| Wrong directory | Show current state, let agent correct | Path awareness |

## Reliability Research

ReliabilityBench specifically evaluates LLM agent reliability — how consistently agents recover from errors and maintain quality across varied tasks.

## Best Practices

1. Always report tool outputs (including errors) back to the LLM
2. Set maximum retry counts to prevent infinite loops
3. Use linting/validation before applying changes
4. Detect stuck states and terminate early
5. Fall back to alternate models when primary fails
6. Restrict tool use to prevent dangerous operations during recovery
