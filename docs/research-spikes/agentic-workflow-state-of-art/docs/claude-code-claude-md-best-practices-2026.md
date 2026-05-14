# CLAUDE.md Best Practices and Claude Code Community Patterns (2025-2026)

- **Source URLs**:
  - https://code.claude.com/docs/en/best-practices
  - https://www.humanlayer.dev/blog/writing-a-good-claude-md
  - https://arize.com/blog/claude-md-best-practices-learned-from-optimizing-claude-code-with-prompt-learning/
  - https://rosmur.github.io/claudecode-best-practices/
  - https://www.eesel.ai/blog/claude-code-best-practices
- **Retrieved**: 2026-03-15
- **Note**: Compiled from web search results on Claude Code usage patterns and CLAUDE.md optimization.

---

## CLAUDE.md Best Practices

### Content to Include
- Common bash commands
- Core files and utility functions
- Code style guidelines
- Testing instructions
- Repository etiquette
- Developer environment setup

### Size and Structure
- **Target under 200 lines** per file
- If too long, Claude ignores half — important rules get lost in noise
- Use `/init` to generate starter CLAUDE.md, then refine

### Monorepo Pattern
Use multiple CLAUDE.md files:
- General root CLAUDE.md
- Specific subfolder files (e.g., /frontend, /backend)
- Gives Claude focused context where needed

## Key Community Patterns (2025-2026)

### Context Management (Most Impactful)
Successful users obsessively manage context through:
- CLAUDE.md files
- Aggressive `/clear` usage
- Documentation systems (dev docs, living plans)
- Token-efficient tool design

### Three Most Impactful Practices
1. Configure CLAUDE.md with project conventions
2. Structure prompts with precise context
3. Use plan mode before complex tasks

### Productivity Impact
Developers applying ten key practices (configuration, communication, workflow) reduce iterations needed for satisfactory results by 35%.

## Advanced Patterns

### Skills and Subagents
Best practices repositories demonstrate patterns for:
- Skills (reusable capability modules)
- Subagents (delegated parallel work)
- Hooks (pre/post execution callbacks)
- Commands (custom slash commands)

### Working Memory Pattern
Use external files as working memory that survives context compaction:
- tasks.md for task tracking
- progress.txt for session state
- tests.json for structured verification data
- Git for state tracking across sessions
