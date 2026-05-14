# CLAUDE.md Best Practices and Instruction File Patterns

- **Source URLs**:
  - https://code.claude.com/docs/en/best-practices
  - https://uxplanet.org/claude-md-best-practices-1ef4f861ce7c
  - https://www.builder.io/blog/claude-md-guide
  - https://dev.to/docat0209/5-patterns-that-make-claude-code-actually-follow-your-rules-44dh
  - https://claudefa.st/blog/guide/mechanics/rules-directory
  - https://code.claude.com/docs/en/memory
- **Retrieved**: 2026-03-15
- **Note**: Content synthesized from web search results.

## File Hierarchy

### CLAUDE.md Discovery
Claude Code reads CLAUDE.md files by walking up the directory tree from the current working directory:
- Parent directory files: loaded in full at launch
- Subdirectory files: loaded on demand when Claude reads files in those directories
- Useful for monorepos with per-package instructions

### Memory Hierarchy Levels
1. **User level** (`~/.claude/CLAUDE.md`): Global preferences across all projects
2. **Project level** (`./CLAUDE.md`): Most common, project-specific instructions
3. **Subdirectory level**: Loaded on demand per subtree
4. **Auto-memory** (`~/.claude/projects/.../MEMORY.md`): Claude's self-written notes

### .claude/rules/ Directory
All markdown files in `.claude/rules/` automatically loaded with same priority as CLAUDE.md:
- No imports needed — just drop files in
- Useful for modular instruction organization
- Can scope rules to specific paths or concerns

### Import System
`@path/to/file.md` syntax to import other markdown files:
- Resolved recursively up to 5 levels deep
- Keeps root CLAUDE.md clean while organizing detailed instructions

## What Makes a Good CLAUDE.md

### Content to Include
- Common bash commands (build, test, lint)
- Code style guidelines ("Use ES modules, not CommonJS")
- Key files and architectural patterns
- Testing instructions and conventions
- Project-specific terminology

### Size Guidelines
- Target under 200 lines per file
- Use .claude/rules/ for modular organization
- Import detailed docs rather than inlining everything

### Instruction Compliance Patterns

1. **Positional priority**: Put most-violated rules at very top (first 5 lines) and very bottom (last 5 lines). Less critical rules in middle. (Exploits "lost in the middle" phenomenon.)

2. **Positive framing**: Flip negative rules to positive equivalents. "Use ES modules" instead of "Don't use CommonJS". Cuts rule violations by roughly half.

3. **Specificity over vagueness**: "Run `npm test` before committing" not "Make sure tests pass."

4. **Context management focus**: Most successful users obsess over context management through CLAUDE.md files, aggressive /clear usage, documentation systems, and token-efficient tool design.

## Community Patterns

- Keep CLAUDE.md as a living document that evolves
- Use /init to generate a starter based on project structure
- Pair with auto-memory for self-improving instructions
- Create per-environment rules (dev vs. CI vs. production)
