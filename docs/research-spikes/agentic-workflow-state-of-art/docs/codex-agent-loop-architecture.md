# OpenAI Codex Agent Loop Architecture
- **Sources**:
  - https://openai.com/index/unrolling-the-codex-agent-loop/
  - https://developers.openai.com/blog/run-long-horizon-tasks-with-codex/
  - https://developers.openai.com/codex/concepts/sandboxing/
- **Retrieved**: 2026-03-15
- **Note**: Synthesized from search results; full article WebFetch was denied

## Agent Loop Architecture

At the heart of Codex is an iterative agent loop. The orchestrator builds a structured prompt (system, developer, user roles). The model responds with either natural language or a tool call. When a tool call appears, the agent executes commands in an isolated container. Outputs (logs, diffs, test results) are appended to the conversation. Context grows after every turn. The loop repeats until the model signals completion or a maximum step limit triggers a stop.

## Prompt Structure

- **instructions field**: System-level directives from configuration files
- **tools field**: Functions the model can invoke (Codex-provided + MCP server tools)
- **input field**: Prioritized messages with assigned roles in descending order of weight
- Developer-role message describes sandbox permissions
- Optional developer instructions from user configuration
- Aggregated user instructions before the actual query

## Sandboxed Execution

Sandboxing lets Codex act autonomously without unrestricted machine access. Commands run inside a constrained environment defining what Codex can do (which files to modify, whether to use network). Platform-native enforcement: seccomp + landlock on Linux, native sandbox on Windows.

## Test Verification

Codex was trained using RL on real-world coding tasks to:
- Generate code that mirrors human style and PR preferences
- Adhere precisely to instructions
- Iteratively run tests until receiving a passing result
- Run verification steps (tests, lint, typecheck) for every milestone

## Technical Details

- Stateless request handling for Zero Data Retention compliance
- Strategic prompt caching for linear rather than quadratic performance
- Automatic context window management through intelligent compaction
- Handles multi-turn conversations across hundreds of model-tool iterations
