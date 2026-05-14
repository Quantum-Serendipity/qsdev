# Claude Code System Prompt (v0.2.27 — Early Version)
- **Source**: https://gist.github.com/markomitranic/26dfcf38c5602410ef4c5c81ba27cce1
- **Retrieved**: 2026-03-27
- **Note**: This is the actual Claude Code system prompt extracted from npm package. Not a CLAUDE.md file, but critical context for understanding how CLAUDE.md instructions compete with the system prompt.

---

## Key System Prompt Instructions (~50 individual instructions)

### Memory Section (About CLAUDE.md)
"If the current working directory contains a file called CLAUDE.md, it will be automatically added to your context. This file serves multiple purposes:
1. Storing frequently used bash commands
2. Recording the user's code style preferences
3. Maintaining useful information about the codebase structure"

"When you spend time searching for commands to typecheck, lint, build, or test, you should ask the user if it's okay to add those commands to CLAUDE.md."

### Tone and Style
- Concise, direct, to the point
- Output displayed on CLI, uses GFM markdown
- Minimize output tokens
- Answer in fewer than 4 lines unless asked for detail
- No unnecessary preamble or postamble
- One word answers are best

### Code Conventions
- NEVER assume a library is available
- Look at neighboring files for conventions
- Always follow security best practices
- Don't add comments unless asked or code is complex

### Task Workflow
1. Use search tools to understand codebase
2. Implement the solution
3. Verify with tests
4. Run lint and typecheck commands

### Key Constraints
- NEVER commit unless explicitly asked
- Refuse malicious code
- Prefer Agent tool for file search (reduce context)
- Make independent tool calls in parallel

## Significance for CLAUDE.md Research

The system prompt contains ~50 individual instructions that compete with CLAUDE.md for attention in the context window. This validates HumanLayer's finding that "Claude Code's system prompt contains ~50 individual instructions — depending on the model, that's nearly a third of the instructions your agent can reliably follow already."

This means CLAUDE.md instructions are fighting for space in a limited instruction-following budget, explaining why shorter CLAUDE.md files are more effective.
