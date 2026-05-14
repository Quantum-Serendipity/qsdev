# Test-Driven Development with AI Coding Agents
- **Sources**:
  - https://tweag.github.io/agentic-coding-handbook/WORKFLOW_TDD/
  - https://simonwillison.net/guides/agentic-engineering-patterns/first-run-the-tests/
  - https://simonwillison.net/guides/agentic-engineering-patterns/
  - https://medium.com/effortless-programming/better-ai-driven-development-with-test-driven-development
  - https://developertoolkit.ai/en/shared-workflows/core-methodology/test-driven-development/
- **Retrieved**: 2026-03-15
- **Note**: Synthesized from search results; full articles WebFetch was denied for some

## Core Pattern

The agentic loop is where TDD with AI truly shines. Instead of prompting the AI to generate everything at once, describe one behavior at a time through tests — let the AI build up logic incrementally, safely, and cleanly.

## Workflow

1. Write tests based on expected input/output
2. Confirm they fail (red)
3. Commit tests
4. Write code to pass them without modifying tests (green)
5. Unit tests are perfect for AI agents — fast feedback enables rapid iteration cycles

## Why TDD + AI Agents Works

- Tests become natural language specs guiding the AI toward exactly the expected behavior
- The implementation thread focuses only on making tests green, without test-generation clutter
- Fast feedback loops enable the agent to iterate rapidly
- Verification is objective — tests pass or they don't

## Simon Willison's Agentic Engineering Patterns (March 2026)

Key principles:
1. "Code is now inexpensive" — agents generate code quickly
2. "Preserve domain expertise" — developers retain knowledge work
3. First run the tests — apply red/green TDD methodology adapted for agent workflows
4. Test-first development with agents, running existing tests before generating new code

## Codex and Test Verification

GitHub Codex agent mode can iterate on its own code, recognize errors, and fix mistakes in real time. The agent explores the repository, writes code, passes tests, and opens PRs.

Codex trained via RL to iteratively run tests until passing. The model runs verification steps (tests, lint, typecheck) for every milestone.

## Architecture Best Practices

It's advantageous to instruct AI to do the docs and tests first before the implementation, which leads to better code. Hierarchical agent architectures, git-based memory systems, context engineering, and rigorous verification loops are key technical mechanisms.
