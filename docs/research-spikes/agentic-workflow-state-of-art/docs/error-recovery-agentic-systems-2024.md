# Error Recovery and Retry Patterns in Agentic AI Systems

- **Sources**: Multiple industry reports and research papers from 2024-2025
- **Retrieved**: 2026-03-15

## Core Error Recovery Patterns

### 1. Exponential Backoff with Jitter
- Increases delays between retries
- Parses rate-limit headers (Retry-After)
- Maximum attempt limits prevent infinite loops
- Standard for transient API/infrastructure failures

### 2. Semantic Fallback
- When LLM output doesn't meet requirements, try alternative prompt formulations
- Multiple prompt templates varying tone and constraints
- Validation-first retries (check output before accepting)

### 3. State-Based Orchestration
- Explicit states, transitions, retries, timeouts
- Human-in-the-loop pauses for critical decisions
- Orchestrator invokes specialized fallback modules by error type
- Not hardcoded fallback — dynamic routing based on error classification

### 4. Progressive Refinement
- Reflexion pattern: critique-and-revision loop with memory
- Plan-and-execute: generate plan, execute steps, replan on failure
- Each retry is informed by previous failure analysis

## Coding Agent Specific Patterns

### SWE-Agent / OpenHands
- Maximum 100 iterations per instance
- Run code → parse errors → add debug statements → iterate
- Full inner loop: make changes → test → construct PRs
- Error messages as direct feedback for next attempt

### TDFlow
- Per-test debugging with restricted debugger
- Diagnostic reports identifying root causes (not symptoms)
- Failed patches + test outputs accumulate in context
- Separate agents for different error recovery tasks

### LLMLOOP
- Five iterative loops: compilation errors → static analysis → test failures → test quality → mutation analysis
- Compiler feedback at multiple granularities (binary success vs detailed error)
- Integration of static analysis tools (pylint) for proactive error prevention

## Key Insight

The most effective error recovery is not generic retry but **error-specific recovery**:
1. Parse the error to understand its type
2. Route to appropriate recovery strategy
3. Provide targeted context for the fix attempt
4. Accumulate error history to prevent repeating failures

## Production Considerations (12 Failure Patterns - Concentrix)

Agentic systems don't fail suddenly — they drift over time. Agent state (learned behaviors, conversation context, implicit knowledge) cannot be easily externalized or reconstructed, making traditional stateless recovery strategies insufficient.

## Iteration Limits

Practical systems set hard limits:
- SWE-Agent: 100 LLM calls per instance
- OpenHands: 100 iterations
- TDFlow: diminishing returns after 5-10 iterations
- These prevent infinite loops and manage cost
