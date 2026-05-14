# Example CLAUDE.md: Claude Code Showcase (ChrisWiles/claude-code-showcase)
- **Source**: https://raw.githubusercontent.com/ChrisWiles/claude-code-showcase/main/CLAUDE.md
- **Retrieved**: 2026-03-27
- **Significance**: Reference implementation showing all Claude Code features together. ~70 lines.

---

# Project Name

> This is an example CLAUDE.md file showing how to configure Claude Code for your project.

## Quick Facts
- **Stack**: React, TypeScript, Node.js
- **Test Command**: `npm test`
- **Lint Command**: `npm run lint`
- **Build Command**: `npm run build`

## Key Directories
- `src/components/` - React components
- `src/hooks/` - Custom React hooks
- `src/utils/` - Utility functions
- `src/api/` - API client code
- `tests/` - Test files

## Code Style
- TypeScript strict mode enabled
- Prefer `interface` over `type` (except unions/intersections)
- No `any` - use `unknown` instead
- Use early returns, avoid nested conditionals
- Prefer composition over inheritance

## Git Conventions
- **Branch naming**: `{initials}/{description}` (e.g., `jd/fix-login`)
- **Commit format**: Conventional Commits (`feat:`, `fix:`, `docs:`, etc.)
- **PR titles**: Same as commit format

## Critical Rules (with emphasis)
### Error Handling
- NEVER swallow errors silently
- Always show user feedback for errors

### UI States
- Always handle: loading, error, empty, success states

### Mutations
- Disable buttons during async operations
- Always have onError handler

## Testing
- Write failing test first (TDD)
- Use factory pattern: `getMockX(overrides)`
- Test behavior, not implementation

## Skill Activation
Before implementing ANY task, check if relevant skills apply:
- Creating tests → `testing-patterns` skill
- Building forms → `formik-patterns` skill
- GraphQL operations → `graphql-schema` skill

## Notable Characteristics
- ~70 lines, well-organized
- Quick Facts section at top (stack + commands)
- Key Directories for navigation
- NEVER/Always emphasis for critical rules
- Skill activation routing pattern
- Git conventions (branch naming, commit format)
