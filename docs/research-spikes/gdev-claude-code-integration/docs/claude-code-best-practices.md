---
source: https://code.claude.com/docs/en/best-practices
retrieved: 2026-05-12
---

# Best practices for Claude Code

Key patterns relevant to CLI tool integration:

## CLAUDE.md Best Practices
- Run /init to generate starter file, refine over time
- Keep concise and human-readable
- Include: Bash commands Claude can't guess, code style rules, testing instructions, repo etiquette, architectural decisions, common gotchas
- Exclude: Anything Claude can figure out by reading code, standard conventions, detailed API docs, frequently changing info
- Import additional files with @path/to/import syntax
- CLAUDE.md files loaded at session start; skills loaded on demand

## Skills Best Practices
- Create skill when same instructions are pasted repeatedly
- Skills load only when used, so long reference material costs almost nothing until needed
- Use disable-model-invocation: true for workflows with side effects
- Use allowed-tools to pre-approve specific Bash patterns

## CLI Tool Integration Pattern
- "Tell Claude Code to use CLI tools like gh, aws, gcloud, and sentry-cli when interacting with external services"
- "CLI tools are the most context-efficient way to interact with external services"
- "Claude is also effective at learning CLI tools it doesn't already know. Try: 'Use foo-cli-tool --help to learn about foo tool, then use it to solve A, B, C'"

## Hooks for Deterministic Behavior
- Hooks run scripts automatically at specific points in Claude's workflow
- Unlike CLAUDE.md instructions (advisory), hooks are deterministic
- Use for actions that must happen every time with zero exceptions

## Safety Patterns
- Auto mode: classifier reviews commands, blocks risky ones
- Permission allowlists: permit specific safe tools
- Sandboxing: OS-level isolation for filesystem and network
- For headless: minimum necessary permissions, prefer reversible actions, clear stopping conditions, human review checkpoints
