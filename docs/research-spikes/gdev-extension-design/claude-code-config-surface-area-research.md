# Claude Code Configuration Surface Area Research

## Overview

Claude Code's configuration spans 7 major domains across 3 scope layers (managed/project/user). A gdev addon for Claude Code setup needs to generate and manage files across these domains, with team-shared project config committed to git and per-user config left to individual developers.

## Configuration Domains

### 1. CLAUDE.md — Project Instructions

**Files:** `CLAUDE.md` or `.claude/CLAUDE.md` (project), `~/.claude/CLAUDE.md` (user), `CLAUDE.local.md` (local/gitignored)

**What it controls:** Instructions loaded into every Claude Code session — coding conventions, architecture notes, commands to run, patterns to follow or avoid.

**What a gdev addon would generate:**
- Project-specific conventions (language, framework, testing patterns)
- Build/run commands
- Architecture overview
- Links to relevant docs
- Team coding standards

**Team vs personal:** CLAUDE.md is team-shared (committed). CLAUDE.local.md is personal overrides.

### 2. settings.json — Enforcement Layer

**Files:** `.claude/settings.json` (project), `~/.claude/settings.json` (user), `.claude/settings.local.json` (local)

**Key configuration knobs:**
- `permissions.allow` — Allowlisted tool patterns (e.g., `Bash(npm run *)`, `Read(~/.zshrc)`)
- `permissions.deny` — Denylisted patterns (e.g., `Bash(curl *)`, `Read(./.env)`)
- `permissions.ask` — Patterns requiring confirmation
- `permissions.defaultMode` — Default permission stance
- `permissions.additionalDirectories` — Extra directories Claude can access
- `sandbox` — Filesystem and network sandboxing (write/read allow/deny, allowed domains)
- `model` — Default model override
- `availableModels` — Restrict which models users can select
- `effortLevel` — Default thinking effort
- `attribution` — Commit/PR attribution strings
- `hooks` — Event-triggered automation (see below)

**What a gdev addon would generate:**
- Team-standard permission allowlists (build tools, test runners, linters)
- Sandbox config for the project's security profile
- Standard attribution format

### 3. Skills — Reusable Workflows

**Location:** `.claude/skills/` directory

**Structure:** Each skill is a markdown file with YAML frontmatter:
```yaml
---
name: skill-name
description: What it does
---
Instructions for Claude when this skill is invoked...
```

**What a gdev addon would generate:**
- Team-standard skills (deploy, review, test, security-review)
- Project-type-specific skills (database migrations, API scaffolding)
- Custom workflow skills matching team processes

### 4. Hooks — Event-Triggered Automation

**Location:** Inside `settings.json` under `hooks` key

**Hook lifecycle events:** 25+ events including pre/post tool execution for Bash, Edit, Write, etc.

**What a gdev addon would generate:**
- Auto-format hooks (run prettier/eslint after file edits)
- Safety hooks (block dangerous commands like `rm -rf`)
- Logging/audit hooks
- Pre-commit validation hooks

### 5. MCP Servers — External Tool Integration

**Files:** `.mcp.json` (project), `~/.claude.json` (user), `managed-mcp.json` (managed)

**What it controls:** Registration of Model Context Protocol servers that give Claude access to external tools (GitHub, Slack, databases, custom APIs).

**What a gdev addon would generate:**
- Team-standard MCP server configurations
- Project-specific tool integrations
- Approved external service connections

### 6. Permissions Model

**Pattern format:** `ToolName(glob pattern)` — e.g., `Bash(npm test *)`, `Read(./.env)`

**Precedence:** Managed (highest) → Local → Project → User (lowest)

**What a gdev addon would manage:**
- Team-wide security policies via managed settings
- Project-specific tool allowlists
- Sandbox boundaries for the project type

### 7. Directory Structure

**Standard `.claude/` layout:**
```
.claude/
├── settings.json        # Project settings (committed)
├── settings.local.json  # Personal overrides (gitignored)
├── skills/              # Reusable workflow skills
├── rules/               # Path-scoped conditional instructions
├── agents/              # Named subagent configurations
└── output-styles/       # Custom output formatting
```

**`.claude/rules/` subdirectory:** Rules files loaded conditionally based on which files are being edited. Supports path-scoped instructions (e.g., rules that only apply when editing frontend code).

## Scope Layers for Team Standardization

| What | Project (committed) | User (personal) | Managed (IT-enforced) |
|------|-------------------|-----------------|----------------------|
| Instructions | CLAUDE.md | ~/.claude/CLAUDE.md | N/A |
| Permissions | .claude/settings.json | ~/.claude/settings.json | managed-settings.json |
| Skills | .claude/skills/ | N/A | N/A |
| MCP servers | .mcp.json | ~/.claude.json | managed-mcp.json |
| Rules | .claude/rules/ | N/A | N/A |

**Managed settings deployment:** Supports MDM (macOS plist, Windows registry), file-based (`/etc/claude-code/` on Linux), and drop-in directory (`managed-settings.d/` with numeric-prefix merge order).

## gdev Addon Design Implications

### What the addon should generate (project-level, committed):
- `.claude/settings.json` with team permission defaults
- `CLAUDE.md` with project conventions, build commands, architecture
- `.claude/skills/` with team-standard workflows
- `.claude/rules/` with path-scoped instructions
- `.mcp.json` with approved tool integrations
- `.gitignore` entries for `settings.local.json` and `CLAUDE.local.md`

### What the addon should template but not commit:
- `.claude/settings.local.json` (personal overrides)
- `CLAUDE.local.md` (personal instructions)
- `~/.claude.json` MCP servers (user-level integrations)

### Wizard questions for Claude Code setup:
1. What language/framework? (drives CLAUDE.md conventions and skills)
2. What build/test commands? (drives permission allowlists)
3. Enable sandbox? What network domains to allow? (drives sandbox config)
4. What MCP servers to integrate? (drives .mcp.json)
5. What team attribution format? (drives attribution config)
6. Install standard skills? Which ones? (drives .claude/skills/)

### Migration/update considerations:
- settings.json is additive (new permissions can be merged)
- CLAUDE.md is free-form text (harder to merge automatically)
- Skills are individual files (easy to add/update independently)
- Rules are individual files (same)
- Need a strategy for updating team config without clobbering local customizations
