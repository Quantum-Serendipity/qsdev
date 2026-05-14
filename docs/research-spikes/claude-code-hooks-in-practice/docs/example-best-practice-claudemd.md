# Example CLAUDE.md: Claude Code Best Practice (shanraisshan)
- **Source**: https://raw.githubusercontent.com/shanraisshan/claude-code-best-practice/main/CLAUDE.md
- **Retrieved**: 2026-03-27
- **Significance**: Comprehensive reference implementation (~200 lines) documenting all Claude Code features.

---

## Key Sections

### Repository Overview
"This is a best practices repository for Claude Code configuration, demonstrating patterns for skills, subagents, hooks, and commands."

### Key Components
- **Weather System**: Demonstrates Command → Agent → Skill architecture
- Two skill patterns: agent skills (preloaded via `skills:` field) vs skills (invoked via Skill tool)

### Skill Definition Structure
Full YAML frontmatter specification for `.claude/skills/<name>/SKILL.md`:
- name, description, argument-hint, disable-model-invocation
- user-invocable, allowed-tools, model, context, agent
- hooks (lifecycle hooks scoped to skill)

### Configuration Hierarchy
1. Managed (MDM plist / Registry): Organization-enforced, cannot be overridden
2. Command line arguments
3. `.claude/settings.local.json`: Personal project settings (git-ignored)
4. `.claude/settings.json`: Team-shared settings
5. `~/.claude/settings.json`: Global personal defaults
6. `hooks-config.local.json` overrides `hooks-config.json`

### Hooks System
Cross-platform sound notification system:
- `scripts/hooks.py`: Main handler
- `config/hooks-config.json`: Shared team configuration
- `config/hooks-config.local.json`: Personal overrides (git-ignored)
- All hook events configured: PreToolUse, PostToolUse, UserPromptSubmit, Notification, Stop, SubagentStart, SubagentStop, PreCompact, SessionStart, SessionEnd, Setup, PermissionRequest, TeammateIdle, TaskCompleted, ConfigChange

### Workflow Best Practices
- Keep CLAUDE.md under 200 lines per file for reliable adherence
- Use commands for workflows instead of standalone agents
- Create feature-specific subagents with skills (progressive disclosure)
- Perform manual `/compact` at ~50% context usage
- Start with plan mode for complex tasks
- Break subtasks small enough to complete in under 50% context

## Notable Characteristics
- ~200 lines
- Meta-documentation: documents how to document
- Configuration hierarchy explicitly spelled out
- Specific line count guidance (200 lines max)
- Self-referencing: "always search this repo first"
