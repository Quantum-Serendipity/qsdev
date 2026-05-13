<!-- Source: https://code.claude.com/docs/en/skills -->
<!-- Retrieved: 2026-05-12 -->

# Claude Code Skills - Official Docs

## Key Format
- SKILL.md with YAML frontmatter + markdown body
- Directory structure: my-skill/SKILL.md + optional supporting files
- Locations: Enterprise > Personal (~/.claude/skills/) > Project (.claude/skills/) > Plugin

## Frontmatter Fields
- name: Display name (optional, defaults to directory name)
- description: What skill does + when to use (recommended, 1536 char cap)
- when_to_use: Additional trigger context (appended to description)
- argument-hint: Shown during autocomplete
- arguments: Named positional arguments for $name substitution
- disable-model-invocation: true prevents auto-invocation by Claude
- user-invocable: false hides from / menu
- allowed-tools: Pre-approve tools without per-use prompts
- model: Override model when skill active
- effort: Override effort level when active
- context: fork to run in isolated subagent
- agent: Which subagent type for context: fork
- hooks: Lifecycle hooks scoped to skill
- paths: Glob patterns limiting activation
- shell: bash (default) or powershell

## String Substitutions
- $ARGUMENTS, $ARGUMENTS[N], $N, $name
- ${CLAUDE_SESSION_ID}, ${CLAUDE_EFFORT}, ${CLAUDE_SKILL_DIR}

## Dynamic Context Injection
- !`command` syntax runs shell commands before skill sent to Claude
- ```! for multi-line commands
- Output replaces placeholder in skill content

## Key Behaviors
- Commands merged into skills (both .claude/commands/ and .claude/skills/ work)
- Live change detection without restart
- Auto-discovery from parent and nested directories (monorepo support)
- Content stays in context across turns after invocation
- Auto-compaction: first 5000 tokens per skill, 25000 combined budget
- skillListingBudgetFraction: 1% of model context window for descriptions
