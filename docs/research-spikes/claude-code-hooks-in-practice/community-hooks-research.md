# Community Claude Code Hook Configurations: Empirical Research

## Executive Summary

Analysis of 15+ GitHub repositories, 6 Hacker News discussions, and multiple blog posts/tutorials reveals that the Claude Code hooks ecosystem is active and rapidly maturing. The dominant usage patterns cluster into four categories: **auto-formatting** (PostToolUse), **safety/security gates** (PreToolUse), **notifications** (Stop/Notification), and **observability/logging** (all events). The `command` handler type overwhelmingly dominates; `prompt` and `agent` handler types are used almost exclusively for Stop-event quality gates. Most hooks are individual developer configurations rather than team-enforced, though organizational managed policies exist for enterprise deployment.

---

## 1. Repository Survey: Who's Publishing Hook Configs?

### Dedicated Hook Collections/Frameworks

| Repository | Focus | Events Covered | Handler Type | Language |
|---|---|---|---|---|
| [disler/claude-code-hooks-mastery](https://github.com/disler/claude-code-hooks-mastery) | Comprehensive reference implementation | 13 events | command | Python (UV) |
| [karanb192/claude-code-hooks](https://github.com/karanb192/claude-code-hooks) | Safety + notifications collection | PreToolUse, PostToolUse, Notification | command | Node.js |
| [decider/claude-hooks](https://github.com/decider/claude-hooks) | Clean code enforcement | PreToolUse, PostToolUse, Stop | command | Python |
| [johnlindquist/claude-hooks](https://github.com/johnlindquist/claude-hooks) | TypeScript hook framework | PreToolUse, PostToolUse, Notification, Stop | command (ts-node) | TypeScript |
| [timoconnellaus/define-claude-code-hooks](https://github.com/timoconnellaus/define-claude-code-hooks) | Type-safe hook definition library | PreToolUse, PostToolUse, Notification, Stop, SubagentStop | command (ts-node) | TypeScript |
| [ryanlewis/claude-format-hook](https://github.com/ryanlewis/claude-format-hook) | Multi-language auto-formatter | PostToolUse | command | Shell |

### Real Project Configurations

| Repository | Notable Hooks | Pattern |
|---|---|---|
| [ChrisWiles/claude-code-showcase](https://github.com/ChrisWiles/claude-code-showcase) | Branch protection, Prettier, npm install, test runner, TypeScript validation | Full CI-in-editor pipeline |
| [ZacheryGlass/.claude](https://github.com/ZacheryGlass/.claude) | Commit guard, GitHub issue guard, CLAUDE.md protection, emoji remover | Personal quality enforcement |
| [feiskyer/claude-code-settings](https://github.com/feiskyer/claude-code-settings) | Hooks directory + extensive skills/agents | Skills-first ecosystem |

### Observability Tools Using Hooks

| Tool | Mechanism | Events Hooked |
|---|---|---|
| [disler/claude-code-hooks-multi-agent-observability](https://github.com/disler/claude-code-hooks-multi-agent-observability) | HTTP POST to Bun server → SQLite → WebSocket → Vue dashboard | 12 events (all major ones) |
| [obsessiondb/rudel](https://github.com/obsessiondb/rudel) | SessionEnd hook uploads transcript to ClickHouse | SessionEnd |
| [ColeMurray/claude-code-otel](https://github.com/ColeMurray/claude-code-otel) | Built-in OTEL export (not custom hooks) → Prometheus/Loki → Grafana | N/A (native telemetry) |

### Specialized/Niche Hook Tools

| Tool | What It Does | Hook Events |
|---|---|---|
| [who96/claude-code-context-handoff](https://github.com/who96/claude-code-context-handoff) | Preserves context across compaction/clear | PreCompact, SessionEnd, SessionStart |
| [Capnjbrown/c0ntextKeeper](https://github.com/Capnjbrown/c0ntextKeeper) | Context preservation (187 semantic patterns) | Multiple session lifecycle events |
| [Talieisin/britfix](https://github.com/Talieisin/britfix) | Converts American → British English in code | PostToolUse (Edit/Write) |
| [dazuiba/CCNotify](https://github.com/dazuiba/CCNotify) | Desktop notifications + VS Code integration | Notification, Stop |
| [wyattjoh/claude-code-notification](https://github.com/wyattjoh/claude-code-notification) | macOS native notifications with sounds | Notification, Stop |
| [vaporif/parry](https://github.com/vaporif/parry) | Prompt injection scanner | PreToolUse, PostToolUse |
| [aannoo/claude-hook-comms](https://github.com/aannoo/claude-hook-comms) | Inter-agent communication via hooks | Multiple |
| [GowayLee/cchooks](https://github.com/GowayLee/cchooks) | Python SDK for hook development | All |

### Product Integrations

| Product | Integration Pattern | Events |
|---|---|---|
| [GitButler](https://docs.gitbutler.com/features/ai-integration/claude-code-hooks) | Auto-branching per Claude session | PreToolUse, PostToolUse, Stop |

---

## 2. Most Commonly Hooked Events

Based on frequency across all surveyed sources:

### Tier 1: Nearly Universal
1. **PostToolUse** — Auto-formatting, test running, linting, git staging, logging. Present in virtually every hook configuration.
2. **PreToolUse** — Safety gates (dangerous commands, secret protection, file protection). Second most common.
3. **Stop** — Notifications, quality gates, commit finalization. Third most common.

### Tier 2: Common
4. **Notification** — Desktop/mobile alerts when Claude needs input.
5. **SessionStart** — Context injection, environment setup.
6. **UserPromptSubmit** — Prompt logging, validation, context enrichment.

### Tier 3: Observability/Advanced
7. **SessionEnd** — Session analytics upload, cleanup.
8. **PreCompact** — Context preservation before compaction.
9. **SubagentStart/SubagentStop** — Multi-agent monitoring.
10. **PostToolUseFailure** — Error logging and analysis.
11. **PermissionRequest** — Audit logging, auto-approval for safe operations.

### Tier 4: Rarely Seen in Community
- **ConfigChange**, **WorktreeCreate/Remove**, **TaskCompleted**, **TeammateIdle**, **Elicitation/ElicitationResult**, **InstructionsLoaded** — Almost no community usage found. These are newer events or have very specialized use cases.

---

## 3. Handler Type Distribution

### Overwhelmingly: `command` (shell)
Every surveyed repository uses `command` type handlers. This is the baseline. Shell scripts (bash, Python via uv, Node.js, TypeScript via ts-node/tsx) are the universal implementation choice.

### Emerging: `http`
Used primarily by observability tools (disler's multi-agent dashboard). HTTP hooks POST event data to a local or remote server for aggregation. Not yet common in individual configs.

### Rare in Community: `prompt` and `agent`
Despite being documented for quality gates, `prompt` and `agent` handler types are almost absent from community configurations. The JP Caparas Dev Genius article describes using Stop hooks with `prompt` type for end-of-turn quality verification, but this pattern hasn't been widely adopted. Possible reasons:
- Adds latency and token cost to every turn
- Requires trust that the evaluating model will catch issues the generating model missed
- `command` type with linters/tests provides deterministic verification that developers trust more

---

## 4. Functional Categories and Patterns

### Category A: Auto-Formatting (most common)
**Event:** PostToolUse | **Matcher:** `Edit|Write|MultiEdit`

The single most frequently implemented hook. Runs formatters after Claude edits files.

**Implementations found:**
- Prettier for JS/TS (ChrisWiles, multiple blog posts)
- Biome with Prettier fallback (ryanlewis/claude-format-hook)
- Ruff for Python (ryanlewis, disler)
- goimports + go fmt for Go (ryanlewis)
- ktlint for Kotlin (ryanlewis)
- ESLint --fix chained after Prettier (multiple blog posts)

**Key design pattern:** Graceful degradation — hooks use `|| true` or silent skip when formatter is not installed, so the hook never blocks Claude's workflow. The file path is extracted from tool_input JSON via `jq`.

### Category B: Safety/Security Gates (second most common)
**Event:** PreToolUse | **Matcher:** `Bash` or `Read|Edit|Write|Bash`

**What gets blocked (exit code 2):**
- Destructive commands: `rm -rf /`, `rm -rf ~`, fork bombs, `curl | sh`
- Force pushes: `git push --force main`, `git push -f main`
- Hard resets: `git reset --hard`
- Database destruction: `DROP TABLE`, `DROP DATABASE`
- Secret file access: `.env`, credentials files, SSH keys
- Protected file modification: `CLAUDE.md`, lock files

**Implementations found:**
- karanb192/claude-code-hooks: Node.js with configurable safety levels (critical/high/strict)
- ZacheryGlass: Python guards for commits and GitHub issues
- decider/claude-hooks: Package age checker (blocks npm packages older than 180 days)
- ChrisWiles: Branch protection (blocks edits on main, requires feature branch)
- parry: Prompt injection detection in tool inputs/outputs

**Tiered safety levels** (karanb192) is an interesting pattern:
| Level | Coverage |
|---|---|
| critical | Only catastrophic operations (rm -rf /) |
| high | Catastrophic + risky (force push, hard reset) |
| strict | All of the above + cautionary patterns |

### Category C: Notifications (third most common)
**Events:** Stop, Notification

The most universally desired hook. Multiple independent implementations exist:
- macOS: `osascript` for native notifications, `terminal-notifier`
- Linux: `notify-send`
- Mobile: Pushover (decider/claude-hooks), ntfy.sh (claude-remote-approver)
- Slack: karanb192's notify-permission hook
- TTS (text-to-speech): disler's hooks-mastery (ElevenLabs > OpenAI > pyttsx3)
- VS Code integration: CCNotify with one-click return to editor

**Pattern:** Stop hook fires when Claude finishes → script sends notification → developer returns to terminal. Solves the "tab away and forget" problem.

### Category D: Test/Lint Automation
**Event:** PostToolUse | **Matcher:** `Edit|Write`

**Implementations found:**
- ChrisWiles: `npm test --findRelatedTests` when test files change (90s timeout)
- ChrisWiles: `npx tsc --noEmit` for TypeScript validation (30s timeout, non-blocking)
- decider: Code quality validator (max function length 30 lines, max file 200 lines, max nesting 4)
- disler: Ruff linter + ty type checker as PostToolUse validators

**Key insight from HN discussions:** Developers want hooks to run only the *relevant* tests, not the full suite. `--findRelatedTests` pattern and git-aware linting (only changed lines) are emerging best practices.

### Category E: Observability/Logging
**Events:** All (comprehensive) or PostToolUse+Stop (lightweight)

**Three tiers of observability:**
1. **Lightweight logging** — JSON event logs to local files (disler/hooks-mastery, define-claude-code-hooks with `logPreToolUseEvents()`)
2. **Session analytics** — Upload transcripts to cloud (Rudel → ClickHouse)
3. **Real-time dashboards** — HTTP hooks → server → WebSocket → Vue dashboard (disler/multi-agent-observability)

**Native OTEL** (ColeMurray/claude-code-otel) uses Claude Code's built-in telemetry (`CLAUDE_CODE_ENABLE_TELEMETRY=1`) rather than custom hooks. Provides Grafana dashboards for token usage, costs, tool performance, and session metrics.

### Category F: Context Preservation
**Events:** PreCompact, SessionStart, SessionEnd

**Problem solved:** Context compaction loses important details. These hooks capture state before compaction and re-inject it when sessions resume.

**Implementations found:**
- who96/claude-code-context-handoff: Captures last 15 user messages + 10 code snippets + file paths, restores via `additionalContext`
- Capnjbrown/c0ntextKeeper: 187 semantic patterns, 7 hooks, 3 MCP tools
- disler/hooks-mastery: PreCompact transcript backup

**SessionStart context injection** is a widely recommended pattern: load git status, recent issues, project-specific context on every session start.

### Category G: Git/VCS Integration
**Events:** PreToolUse, PostToolUse, Stop

**Implementations found:**
- GitButler: Full session-aware branching (`but claude pre-tool`, `but claude post-tool`, `but claude stop`)
- karanb192: Auto-stage files after Edit/Write
- ZacheryGlass: Commit message quality guard
- ChrisWiles: Auto `npm install` when package.json changes
- decider: Pre-commit check hook

### Category H: Workflow Enforcement
**Events:** PreToolUse, UserPromptSubmit

**Implementations found:**
- ChrisWiles: Block edits on main branch (require feature branch first)
- ZacheryGlass: Protect CLAUDE.md from modification
- decider: Block outdated npm packages (>180 days)
- UserPromptSubmit: Context injection (project standards, API conventions)
- Bedtime hook: Time-based usage restriction
- WIP nudge: Alerts about accumulating uncommitted work

---

## 5. Individual vs. Team/Organization Level

### Individual (vast majority)
Almost all community hooks are configured at the individual level:
- `~/.claude/settings.json` (user-global)
- `.claude/settings.json` (project-level, committed to repo)
- `.claude/settings.local.json` (project-level, gitignored)

Individual developers iterate on hooks based on personal pain points ("Claude tried to run rm -rf ~/ during debugging").

### Team/Project Level
Some hooks are designed for team sharing via `.claude/settings.json` committed to the repo:
- ChrisWiles/claude-code-showcase: Full project config with formatting, testing, branch protection
- define-claude-code-hooks: Explicit support for project vs. local hook files
- GitButler integration: Designed for team-wide adoption

### Organization/Enterprise
Managed policy settings (`managed-settings.json`) allow administrators to:
- Deploy hooks that apply to every developer
- Prevent developers from overriding organizational hooks
- Lock configurations with `allowManagedHooksOnly`
- Enforce compliance and audit trails

**No public examples found** of enterprise managed hook configurations — these are proprietary by nature.

---

## 6. Hook-Related Tools and Frameworks

### Hook Definition Frameworks
| Tool | Language | Value Proposition |
|---|---|---|
| define-claude-code-hooks | TypeScript | Type safety, auto settings.json management, predefined utilities |
| claude-hooks (johnlindquist) | TypeScript | Type safety, CLI-generated scaffolding |
| cchooks | Python | Clean Python SDK abstracting JSON complexity |
| disler/hooks-mastery | Python (UV) | Self-contained UV scripts, no venv management |

### Plugin Marketplaces
- [jimmc414/claude-code-plugin-marketplace](https://github.com/jimmc414/claude-code-plugin-marketplace): Community plugin marketplace including hooks
- [claudemarketplaces.com](https://claudemarketplaces.com/): Web-based directory
- Plugins can bundle hooks in `hooks/hooks.json` within the plugin package

### The TypeScript vs. Python Split
Two dominant approaches for hook implementation:
1. **TypeScript** (johnlindquist, timoconnellaus): Type safety, autocomplete, npm ecosystem. Better for JS/TS-heavy teams.
2. **Python with UV** (disler, decider): Self-contained scripts with embedded deps, fast execution. Better for polyglot teams.

Shell scripts (bash) are used for simple hooks but don't scale well for complex logic or JSON parsing.

---

## 7. Reported Issues and Limitations

### From GitHub Issues
- **Hooks not loading**: Bug where `/hooks` shows "No hooks configured" despite valid settings.json ([#11544](https://github.com/anthropics/claude-code/issues/11544))
- **PreToolUse/PostToolUse not executing**: Reported in [#6305](https://github.com/anthropics/claude-code/issues/6305)

### From HN Discussions
- **CLAUDE.md compliance gap**: Multiple developers report Claude ignoring CLAUDE.md instructions, motivating the move to hooks for *deterministic* enforcement
- **Context window pressure**: Hook output consumes context window space. Verbose hooks accelerate compaction.
- **Monorepo challenges**: Directory-specific linting requires conditional logic in hook scripts
- **Latency**: Each hook adds execution time. Complex hooks (npm test, tsc) need generous timeouts (60-90s)
- **Silent failures**: Non-zero exit codes other than 2 are logged but not shown to Claude, making debugging difficult

### From Blog Posts/Tutorials
- **Hooks cannot rewrite slash commands**: Limitation noted by context-handoff plugin (requires external supervisor)
- **No matcher for some events**: UserPromptSubmit always fires (no matcher support)
- **Error in hook = unclear behavior**: Exit code 1 logs warning but doesn't block — developers must use exit code 2 for enforcement

---

## 8. Emerging Patterns and "Must-Have" Hooks

### The "Starter Pack" (most universally recommended)
1. **Auto-format on edit** — PostToolUse + Edit|Write matcher + your project's formatter
2. **Block dangerous commands** — PreToolUse + Bash matcher + pattern matching
3. **Desktop notification on completion** — Stop event + system notification command

### The "Power User" Addition
4. **Protect sensitive files** — PreToolUse + Read|Edit|Write|Bash matcher + .env/secrets check
5. **Context injection on session start** — SessionStart + git status / project context
6. **Auto-run related tests** — PostToolUse + test file matcher + `--findRelatedTests`
7. **Context preservation across compaction** — PreCompact + SessionStart

### The "Team Lead" Layer
8. **Branch protection** — PreToolUse blocks edits on main
9. **Commit message quality** — PreToolUse on git commit
10. **Standardized formatting** — Project-level `.claude/settings.json` with formatter hooks

### The "Observability-First" Setup
11. **Event logging** — All major events → JSON log files
12. **Session analytics** — SessionEnd → upload to analytics platform
13. **Real-time dashboard** — HTTP hooks → monitoring server

---

## 9. What's Missing from the Community

1. **Quality gate hooks using `prompt`/`agent` types** — Theoretically powerful but almost no real adoption. The "AI reviewing AI" pattern hasn't caught on yet.
2. **CI/CD integration** — Hooks run locally in the developer's terminal. No clear pattern for connecting hook enforcement to CI/CD pipelines or reporting hook violations to team dashboards.
3. **Cost/budget alerting** — Despite being mentioned as a use case, no published implementation of token tracking or cost alerting via hooks was found.
4. **Compliance/audit trail** — Enterprise-level compliance hooks are presumably proprietary. No open-source examples of SOC2/HIPAA-relevant hook configurations.
5. **Cross-agent coordination** — HCOM (claude-hook-comms) is the only tool attempting inter-agent communication via hooks, and it's early-stage.

---

## Sources

All raw source material is saved in `docs/` as full-content markdown files:

### GitHub Repositories
- `docs/github-disler-hooks-mastery.md`
- `docs/github-johnlindquist-claude-hooks.md`
- `docs/github-chriswiles-claude-code-showcase.md`
- `docs/github-karanb192-claude-code-hooks.md`
- `docs/github-decider-claude-hooks.md`
- `docs/github-feiskyer-claude-code-settings.md`
- `docs/github-disler-hooks-multi-agent-observability.md`
- `docs/github-timoconnellaus-define-claude-code-hooks.md`
- `docs/github-awesome-claude-code-hooks.md`
- `docs/github-obsessiondb-rudel.md`
- `docs/github-ryanlewis-claude-format-hook.md`
- `docs/github-colemurray-claude-code-otel.md`
- `docs/github-who96-context-handoff.md`
- `docs/github-zacheryglass-claude-settings.md`
- `docs/gitbutler-claude-code-hooks-integration.md`

### Hacker News Discussions
- `docs/hn-claude-code-hooks-announcement.md`
- `docs/hn-claude-hooks-6-hooks-discussion.md`
- `docs/hn-block-dangerous-commands.md`
- `docs/hn-wip-nudge-hook.md`
- `docs/hn-bedtime-hook.md`

### Search Result Insights (not separately saved — synthesized from multiple searches)
- Blake Crosley's "95 hooks" article and tutorial
- Anthropic's official hooks blog post
- Dev.to articles on workflow automation
- DataCamp and SmartScope tutorials
- Multiple notification implementation blog posts (d12frosted, khromov, nakamasato, aitmpl, wmedia, susomejias)
