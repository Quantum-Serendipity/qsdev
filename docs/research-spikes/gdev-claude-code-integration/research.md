# Research Summary: gdev Claude Code Integration

## Overview
Research how to build Claude Code skills and commands that let Claude Code operate gdev, a Go CLI tool that bootstraps secure development environments. Covers skill/command file formats, recommended gdev operations to expose, CLI wrapper patterns from the ecosystem, CLAUDE.md integration, and safety considerations for autonomous vs human-confirmed operations.

## Topics

### 1. Claude Code Skill File Format — status: complete
SKILL.md format with YAML frontmatter (name, description, allowed-tools, disable-model-invocation, context, arguments, etc.) and markdown body. Dynamic context injection via `!`command`` preprocessor. Supporting files for reference material. Skills replace legacy .claude/commands/ format.
- See: [claude-code-integration-research.md](claude-code-integration-research.md) Sections 1-2

### 2. gdev Operation Mapping — status: complete
10 gdev operations mapped to skills with concrete SKILL.md implementations. 6 user-only operations (init, onboard, setup, enable, disable, compliance, update) and 4 Claude-invocable operations (doctor, status, list, detect). Each skill uses dynamic context injection for live state and `--json` output for structured data.
- See: [claude-code-integration-research.md](claude-code-integration-research.md) Section 3

### 3. CLI Wrapper Patterns — status: complete
5 patterns from the ecosystem: knowledge+CLI (Terraform), generator/validator pairs (Docker/K8s), dynamic state injection (GitHub PR skills), multi-step workflow (fix-issue), script bundling (codebase visualizer). Dynamic state injection is the strongest pattern for gdev.
- See: [claude-code-integration-research.md](claude-code-integration-research.md) Section 4

### 4. CLAUDE.md Integration — status: complete
CLAUDE.md for always-loaded quick reference and security policy. Skills for detailed workflows loaded on demand. Section markers for safe gdev-managed updates. @-import for larger reference docs.
- See: [claude-code-integration-research.md](claude-code-integration-research.md) Section 5

### 5. Safety Considerations — status: complete
5-layer safety architecture: skill-level (disable-model-invocation), tool-level (allowed-tools scoping), gdev-level (--dry-run, --non-interactive), permission-level (Claude Code deny rules), hooks-level (enterprise managed settings). Read-only = autonomous, side-effects = user-confirmed.
- See: [claude-code-integration-research.md](claude-code-integration-research.md) Section 6

### 6. Skills vs Alternatives — status: complete
Compared skills to MCP servers, hooks, CLAUDE.md, and deny rules. Skills are the right primary interface for gdev. MCP is overkill for CLI wrapping. Hooks for enterprise enforcement. CLAUDE.md for quick reference.
- See: [claude-code-integration-research.md](claude-code-integration-research.md) Section 8

## Open Questions
- Should gdev ship as a Claude Code plugin (installable via `/plugin marketplace add`) in addition to embedding skills during `gdev init`?
- How should gdev skills interact with the existing claudecode addon's hooks and deny rules?
- Should there be a `/gdev-wizard` skill that provides an interactive chat-based alternative to the huh form wizard?

## Conclusions

Skills are the correct primary integration mechanism for exposing gdev to Claude Code. They provide user-visible workflows with pre-approved tool permissions, dynamic state injection for context-efficient reasoning, and clear safety boundaries via `disable-model-invocation`.

The key design decisions:
1. **Skills, not commands**: Use `.claude/skills/` directory structure for supporting files, frontmatter control, and auto-invocation
2. **Dynamic context injection**: Every skill pre-captures `gdev doctor --json` / `gdev status --json` so Claude reasons from actual state
3. **User-only for side effects**: Init, enable, disable, update require explicit `/gdev-*` invocation
4. **Claude-invocable for diagnostics**: Doctor, status, list are safe for autonomous use
5. **JSON output contract**: All gdev commands must support `--json` for reliable machine parsing
6. **Embed in binary**: Skills embedded via Go's embed.FS, deployed by the claudecode addon during `gdev init`
7. **Section markers in CLAUDE.md**: Safe updates without clobbering user content
8. **5-layer safety**: From skill invocation control to enterprise managed settings
