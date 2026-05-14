# HN Discussion: Claude Code now supports hooks
- **Source**: https://news.ycombinator.com/item?id=44429225
- **Retrieved**: 2026-03-27

## Primary Use Cases Discussed

### Linting and Code Quality
Multiple commenters discuss auto-formatting challenges. One developer uses hooks to "run formatters on C files and shell scripts, and just fixes missing returns on other files." Another wants live linting instead of waiting for Claude to format code.

### Workflow Enforcement
Users describe using hooks to restrict dangerous commands. Example: allowing `docker compose exec django python manage.py test` while preventing `makemigrations`. A commenter proposes sophisticated workflows where "one agent writes code another reviews it another deploys it. each step gated by verification hooks."

### Monorepo Management
Concern raised about directory-specific linting in monorepos, suggesting hooks need conditional logic based on changed files.

## Key Limitations and Concerns

### Context Window Issues
User frustration about needing to restart conversations frequently, noting "larger context window...allows full session without /clear."

### Instruction Compliance
Multiple developers note Claude frequently ignores CLAUDE.md guidelines. One states: "Claude Code loses focus quickly" despite explicit documentation. Hooks aim to solve this by making enforcement automatic rather than reliant on model attention.

### Complexity Trade-offs
Some question whether hooks simply shift problems: "you could add support yourself" through custom scripting, but requires additional engineering effort.

## Shared Configurations

Users propose practical implementations:
- Setting `MAKEFLAGS=-j8` environment variables as alternatives to hooks
- Using pre-commit frameworks integrated with hooks
- Running git-aware linters that check changed lines only
- Custom shell scripts parsing JSON input to determine conditional logic

One developer shares that hook-based file-ending validation and reformatting took "ten minutes" of iteration versus extended AI prompting.
