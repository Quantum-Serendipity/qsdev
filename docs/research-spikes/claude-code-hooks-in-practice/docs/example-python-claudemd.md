# Example CLAUDE.md: Python Project (ArthurClune/claude-md-examples)
- **Source**: https://raw.githubusercontent.com/ArthurClune/claude-md-examples/main/python-CLAUDE.md
- **Retrieved**: 2026-03-27
- **Note**: Edited/merged from modelcontextprotocol/python-sdk, p33m5t3r/vibecoding, saaspegasus/pegasus-docs

---

# Development Guidelines

This document contains critical information about working with this codebase. Follow these guidelines precisely.

## Core Development Rules

1. Package Management
   - ONLY use uv, NEVER pip
   - Installation: `uv add package`
   - Running tools: `uv run tool`
   - Upgrading: `uv add --dev package --upgrade-package package`
   - FORBIDDEN: `uv pip install`, `@latest` syntax

2. Code Quality
   - Type hints required for all code
   - Public APIs must have docstrings
   - Functions must be focused and small
   - Follow existing patterns exactly
   - Line length: 88 chars maximum

3. Testing Requirements
   - Framework: `uv run pytest`
   - Async testing: use anyio, not asyncio
   - Coverage: test edge cases and errors
   - New features require tests
   - Bug fixes require regression tests

4. Code Style
    - PEP 8 naming (snake_case for functions/variables)
    - Class names in PascalCase
    - Constants in UPPER_SNAKE_CASE
    - Document with docstrings
    - Use f-strings for formatting

- For commits fixing bugs or adding features based on user reports add:
  `git commit --trailer "Reported-by:<name>"`

- For commits related to a Github issue, add
  `git commit --trailer "Github-Issue:#<number>"`

- NEVER ever mention a `co-authored-by` or similar aspects. In particular, never mention the tool used to create the commit message or PR.

## Development Philosophy

- Simplicity, Readability, Performance, Maintainability, Testability, Reusability
- Less Code = Less Debt: Minimize code footprint

## Coding Best Practices

- Early Returns, Descriptive Names, Constants Over Functions, DRY Code
- Functional Style, Minimal Changes, TODO Comments, Simplicity
- Build Iteratively, Run Tests, Build Test Environments
- Clean logic: Keep core logic clean and push implementation details to the edges

## System Architecture

[fill in here] — placeholder sections for customization

## Pull Requests

- Create a detailed message focusing on high level description
- Always add `ArthurClune` as reviewer
- NEVER ever mention a `co-authored-by` or similar aspects

## Code Formatting

1. Ruff: Format (`uv run ruff format .`), Check (`uv run ruff check .`), Fix (`uv run ruff check . --fix`)
2. Type Checking: `uv run pyright`
3. Pre-commit: `.pre-commit-config.yaml`, runs Prettier (YAML/JSON) and Ruff (Python)

## Error Resolution

1. CI Failures — fix order: Formatting → Type errors → Linting
2. Common Issues — line length, types
3. Best Practices — check git status, run formatters before type checks, keep changes minimal
