# Claude Code Patterns and Ecosystem Research

## Overview

This report covers Claude Code's internal architecture, extension mechanisms, community best practices, and competitive landscape as of March 2026. Claude Code has evolved from a terminal-based coding assistant into a full agentic platform with hooks, skills, sub-agents, agent teams (swarms), MCP integrations, and a plugin marketplace with 9,000+ extensions. Understanding its architecture and ecosystem is essential for maximizing the quality and effectiveness of agentic workflows.

---

## 1. Claude Code Architecture

### The Agentic Loop

Claude Code implements a classic while-loop agent pattern. When given a task, it cycles through three blended phases: **gather context**, **take action**, and **verify results**. The loop continues as long as the model's responses include tool calls; when Claude produces plain text without tool invocations, the loop terminates and returns control to the user.

The loop is powered by two components:
- **Models** that reason about what to do next
- **Tools** that execute actions in the environment

Claude Code serves as the **agentic harness** — it provides the tools, context management, and execution environment that turn a language model into a capable coding agent. This is the same architecture that powers the Claude Agent SDK.

### System Prompt Architecture

Reverse-engineering by the community (documented in the Piebald-AI/claude-code-system-prompts repository) reveals that Claude Code does **not** use a single monolithic system prompt. Instead, it employs:

- **Conditional components** added based on environment, configuration, and session state
- **Tool descriptions** for 18+ built-in tools (Write, Edit, Read, Bash, Glob, Grep, WebFetch, WebSearch, TodoWrite, etc.)
- **Separate agent prompts** for Explore (517 tokens) and Plan (685 tokens) sub-agents
- **Utility prompts** for conversation compaction, CLAUDE.md generation, security review, session titling
- **System reminders** (~40) injected at specific lifecycle points to counter instruction fade-out
- **Over 110 distinct prompt strings** total, updated with each release

The system prompt is composed from priority-ordered independent sections including identity, safety, tool guidance, workflow rules, and dynamic context. Provider-specific variants and caching enable optimization.

### Tool System

Built-in tools fall into five categories:

| Category | Tools |
|---|---|
| File operations | Read, Write, Edit, Glob (file search) |
| Search | Grep (content search), Glob (file pattern) |
| Execution | Bash (shell commands) |
| Web | WebFetch, WebSearch |
| Orchestration | Agent (sub-agent spawning), Skill (skill invocation), TodoWrite (task tracking) |

Additional tools include code intelligence (LSP-based jump-to-definition, find-references, type errors) via plugins, and MCP tools from configured servers.

### Context Window Management

Context management is the **dominant design constraint** in Claude Code's architecture. The context window holds the entire conversation, file contents, command outputs, CLAUDE.md, loaded skills, and system instructions.

Key mechanisms:
- **Auto-compaction**: Triggered automatically as context approaches limits (~95% by default, configurable via CLAUDE_AUTOCOMPACT_PCT_OVERRIDE). Clears older tool outputs first, then summarizes conversation.
- **Manual compaction**: `/compact` with optional focus directive (e.g., `/compact focus on the API changes`)
- **Compact Instructions**: A section in CLAUDE.md that controls what is preserved during compaction
- **CLAUDE.md re-injection**: After compaction, CLAUDE.md is re-read from disk and re-injected fresh
- **System reminders**: Event-driven injection of targeted guidance at decision points to counter instruction fade-out
- **Subagent isolation**: Sub-agents run in completely separate context windows
- **Skill lazy loading**: Skill descriptions loaded at start, full content only when invoked
- **MCP Tool Search**: When MCP tool descriptions exceed 10% of context, tools are loaded on-demand instead of upfront

Community-observed performance degradation thresholds:
- 0-50%: Work freely
- 50-70%: Pay attention
- 70-90%: Use /compact
- 90%+: Use /clear (mandatory)

### TodoWrite and Task Management

When facing multi-step tasks, Claude Code internally creates structured JSON task lists via TodoWrite with IDs, content, status, and priority levels. The UI renders these as interactive checklists. System reminders inject the current TODO list state after tool uses to keep the model on track.

### Extended Thinking

Extended thinking is now enabled by default with maximum budget. The `/effort` command provides granular control over thinking depth. The keyword "ultrathink" (now deprecated as a trigger) activated extended thinking mode in earlier versions. Extended thinking is critical for complex architectural reasoning and multi-step problem solving.

---

## 2. CLAUDE.md Patterns

### Hierarchy and Loading

CLAUDE.md files form a hierarchical instruction system:

| Scope | Location | Loading |
|---|---|---|
| Managed policy | System-wide directories | Always loaded, cannot be excluded |
| Project | ./CLAUDE.md or ./.claude/CLAUDE.md | Loaded at launch |
| Parent directories | Walking up from working directory | Loaded at launch |
| Child directories | Below working directory | Lazy-loaded when files accessed |
| User | ~/.claude/CLAUDE.md | Loaded at launch |

In monorepos, Claude walks up the directory tree and loads every CLAUDE.md it finds. Subdirectory CLAUDE.md files load on demand.

### Import Syntax

`@path/to/file` imports expand at launch. Paths resolve relative to the containing file. Maximum 5 hops of recursion. Example:

```markdown
See @README.md for project overview and @package.json for available npm commands.
# Personal overrides: @~/.claude/my-project-instructions.md
```

### .claude/rules/ Directory

For larger projects, modular instruction files in `.claude/rules/`:
- Path-specific rules via YAML frontmatter (`paths: ["src/api/**/*.ts"]`)
- Glob pattern matching for file types
- Symlinks supported for sharing across projects
- User-level rules in `~/.claude/rules/`

### Best Practices (Community Consensus)

**What works:**
1. **Keep it under 200 lines per file** — longer files cause Claude to ignore rules
2. **Be specific and verifiable** — "Use 2-space indentation" not "format code properly"
3. **Include only what Claude cannot infer** — build commands, non-obvious conventions, gotchas
4. **Use @imports for modularity** — keep main CLAUDE.md focused, reference details in other files
5. **Use emphasis for critical rules** — "IMPORTANT" or "YOU MUST" improves adherence
6. **Run /init to bootstrap** — analyzes codebase for build systems, test frameworks, patterns
7. **Check into git** — CLAUDE.md compounds in value over time as team contributes
8. **Treat like code** — review when things go wrong, prune regularly, test changes

**What to include:**
- Bash commands Claude cannot guess
- Code style rules that differ from defaults
- Testing instructions and preferred test runners
- Repository etiquette (branch naming, PR conventions)
- Architectural decisions specific to the project
- Developer environment quirks (required env vars)
- Common gotchas or non-obvious behaviors

**What to exclude:**
- Anything Claude can figure out from reading code
- Standard language conventions Claude already knows
- Detailed API documentation (link to docs instead)
- Information that changes frequently
- Long explanations or tutorials
- File-by-file descriptions of codebase

### Auto Memory

Claude Code has a dual memory system:
1. **CLAUDE.md** (user-written): Explicit instructions
2. **Auto Memory** (Claude-written): Notes Claude writes based on corrections and preferences

Auto memory is stored in `~/.claude/projects/<project>/memory/` with a MEMORY.md entrypoint (first 200 lines loaded at session start) and optional topic files loaded on demand. Claude decides what is worth remembering based on whether information would be useful in future conversations.

---

## 3. Hooks System

### Overview

Hooks provide **deterministic** control over Claude Code's behavior — they guarantee actions happen rather than relying on the LLM to choose to run them. This is a fundamental distinction: CLAUDE.md instructions are advisory; hooks are enforced.

### 21 Hook Events

The system exposes 21 lifecycle events including SessionStart, UserPromptSubmit, PreToolUse (the only one that can block actions), PostToolUse, Stop, SubagentStart/Stop, TeammateIdle, TaskCompleted, ConfigChange, PreCompact/PostCompact, WorktreeCreate/Remove, Elicitation, and SessionEnd.

### Four Handler Types

1. **Command** (type: "command"): Shell commands receiving JSON via stdin, controlling via exit codes
2. **HTTP** (type: "http"): POST event data to a URL endpoint
3. **Prompt** (type: "prompt"): Single-turn LLM evaluation returning ok/reason JSON
4. **Agent** (type: "agent"): Multi-turn verification with tool access, spawns a subagent

### High-Value Hook Patterns

**Quality enforcement:**
- **Stop hooks** with prompt/agent type to verify task completeness before Claude stops
- **PreToolUse hooks** to validate commands (block dangerous SQL, protect files)
- **PostToolUse hooks** to auto-format code after edits, run linters

**Context preservation:**
- **SessionStart with compact matcher** to re-inject critical context after compaction
- **PostCompact hooks** to restore key information

**Workflow automation:**
- **Notification hooks** for desktop alerts when Claude needs input
- **SubagentStart/Stop hooks** for setup/cleanup
- **TeammateIdle/TaskCompleted hooks** for quality gates in agent teams

**Security:**
- **PreToolUse** to block edits to protected files (.env, .git/, package-lock.json)
- **ConfigChange** hooks for audit logging

### Configuration Scopes

Hooks can be defined at user level (~/.claude/settings.json), project level (.claude/settings.json), local level (.claude/settings.local.json), in managed policies, in plugins, or in skill/agent frontmatter.

---

## 4. MCP (Model Context Protocol) Servers

### Ecosystem State

As of early 2026, the MCP ecosystem has 200+ servers. Claude Code supports four transport types: HTTP (recommended), SSE (deprecated), stdio (local processes), and WebSocket.

### Essential Servers

The community identifies GitHub, Brave Search, and Playwright as the "essential trio" covering 90% of development needs. Other notable servers:
- **Sentry**: Error monitoring and debugging
- **Notion**: Documentation and knowledge base
- **PostgreSQL/SQLite**: Database querying
- **Figma**: Design integration
- **Slack**: Team communication
- **Rube**: Connects 500+ apps (Gmail, GitHub, Notion, etc.)

### Key Architecture Features

- **MCP Tool Search**: Automatically defers tool loading when descriptions exceed 10% of context. Tools discovered on-demand via search, preventing context bloat.
- **Dynamic tool updates**: `list_changed` notifications let servers update tools without reconnection
- **MCP resources**: Reference via `@server:protocol://resource/path` mentions
- **MCP prompts**: Available as `/mcp__servername__promptname` commands
- **Elicitation**: Servers can request structured user input mid-task
- **Claude Code as MCP server**: `claude mcp serve` exposes Claude Code's tools to other applications

### Scoping and Management

Three scopes (local, project, user) with environment variable expansion in .mcp.json. Managed MCP configuration for organizations via allowlists/denylists. Plugin-provided MCP servers start automatically.

### Context Cost Considerations

MCP servers add tool definitions to every request. Running many servers can consume significant context before work begins. `/mcp` shows per-server context costs. MAX_MCP_OUTPUT_TOKENS (default 25,000) controls maximum tool output size.

---

## 5. Sub-Agent Patterns

### Built-in Sub-Agents

| Agent | Model | Purpose |
|---|---|---|
| Explore | Haiku (fast) | Read-only codebase search and exploration |
| Plan | Inherits | Research agent for plan mode |
| General-purpose | Inherits | Complex multi-step tasks with all tools |
| Bash | Inherits | Terminal commands in separate context |
| Claude Code Guide | Haiku | Questions about Claude Code features |

### Custom Sub-Agent Configuration

Sub-agents are defined as Markdown files with YAML frontmatter (AGENT.md) supporting:
- **Tool restrictions**: Allowlist and denylist
- **Model selection**: sonnet, opus, haiku, or full model ID
- **Permission modes**: default, acceptEdits, dontAsk, bypassPermissions, plan
- **Scoped MCP servers**: Inline server definitions kept out of main context
- **Lifecycle hooks**: PreToolUse, PostToolUse, Stop events
- **Persistent memory**: user, project, or local scope for cross-session learning
- **Skill preloading**: Full skill content injected at startup
- **Git worktree isolation**: `isolation: "worktree"` for independent code copies
- **Background execution**: Non-blocking concurrent work

### Effective Patterns

1. **Context isolation**: The primary value — subagents keep verbose output (tests, builds, logs) out of main conversation
2. **Parallel research**: Spawn multiple subagents for independent investigations, each exploring different areas
3. **Chained workflows**: Sequential subagent use — reviewer then optimizer then tester
4. **Cost routing**: Use Haiku for exploration, Sonnet/Opus for complex reasoning
5. **Persistent memory**: Subagents accumulate knowledge (debugging insights, codebase patterns) across sessions
6. **Foreground vs background**: Foreground blocks with pass-through prompts; background runs concurrent with pre-approved permissions (Ctrl+B to background)

### When to Use Subagents vs Main Conversation

**Use subagents when:**
- Task produces verbose output not needed in main context
- You want to enforce specific tool restrictions
- Work is self-contained and returns a summary
- You need parallel investigation

**Use main conversation when:**
- Frequent back-and-forth needed
- Multiple phases share significant context
- Quick, targeted changes
- Latency matters (subagents start fresh)

### Community Advice

- Limit to 3-4 active subagents maximum to avoid coordination overhead
- "Make your agents critical and honest" — override default agreeable behavior
- Use definition-of-done checklists per agent
- Subagents cannot spawn other subagents (no nesting)

---

## 6. Agent Teams (Swarms)

### Architecture

Agent teams are an experimental feature (since January 2026) enabling coordinated multi-agent workflows:

| Component | Role |
|---|---|
| Team lead | Creates team, spawns teammates, coordinates work |
| Teammates | Separate Claude Code instances with independent contexts |
| Task list | Shared, with claiming, dependencies, and status tracking |
| Mailbox | Direct inter-agent messaging system |

Each teammate works in an independent Git Worktree, preventing code conflicts. The lead creates a plan, workers create worktrees for isolated code copies, and multiple agents write code simultaneously.

### vs Subagents

Subagents report results back to main agent only — no inter-agent communication. Agent teams enable direct teammate messaging, shared task lists, self-coordination, and debate/challenge between agents.

### Best Use Cases

- **Research and review**: Parallel investigation from different perspectives
- **New modules/features**: Independent ownership of separate pieces
- **Debugging with competing hypotheses**: Agents test different theories and challenge each other
- **Cross-layer coordination**: Frontend, backend, tests each owned by different teammate

### Quality Mechanisms

- **Plan approval**: Require teammates to plan before implementing; lead approves/rejects
- **TeammateIdle hooks**: Run when teammate is about to go idle; exit code 2 sends feedback to keep working
- **TaskCompleted hooks**: Validate before task marked complete
- **Direct messaging**: Message teammates directly (Shift+Down to cycle in-process mode)

### Practical Considerations

- Start with 3-5 teammates
- 5-6 tasks per teammate keeps everyone productive
- Token costs scale linearly with each teammate
- Coordination overhead increases with team size
- Always have lead manage cleanup
- Enable: `{"env": {"CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS": "1"}}`

---

## 7. Slash Commands and Skills

### Built-in Commands vs Skills

**Slash commands** are hardcoded fixed-logic operations: /clear, /compact, /help, /model, /cost, /init, /doctor, /hooks, /mcp, /memory, /agents, /permissions, /context, /rewind, /rename, /plugin, /effort, /btw, /statusline.

**Skills** are prompt-based capabilities loaded from SKILL.md files. The two systems have been unified — `.claude/commands/` files and `.claude/skills/` directories both work.

### Bundled Skills

| Skill | Purpose |
|---|---|
| /batch | Parallel large-scale changes via worktrees (5-30 independent units) |
| /claude-api | Load API reference for current language |
| /debug | Troubleshoot session by reading debug log |
| /loop | Run prompt repeatedly on interval (polling, monitoring) |
| /simplify | Review changed files, spawn 3 parallel review agents |

### Skill System Features

- **SKILL.md format**: YAML frontmatter + markdown instructions
- **Invocation control**: `disable-model-invocation: true` for manual-only, `user-invocable: false` for Claude-only
- **Tool restrictions**: `allowed-tools` field limits available tools
- **Model override**: Use different model when skill is active
- **Forked execution**: `context: fork` for subagent isolation
- **Dynamic context injection**: `!`command`` syntax runs shell commands before sending to Claude
- **String substitutions**: $ARGUMENTS, $N, ${CLAUDE_SESSION_ID}, ${CLAUDE_SKILL_DIR}
- **Supporting files**: Templates, examples, scripts alongside SKILL.md
- **Argument passing**: /skill-name args accessible via $ARGUMENTS

### Agent Skills Open Standard

Skills follow the Agent Skills standard (agentskills.io) which works across multiple AI tools, not just Claude Code.

### Community Skills Ecosystem

- awesome-claude-code repositories curate community skills
- awesome-claude-code-toolkit: 135 agents, 35 curated skills (+15,000 via SkillKit), 42 commands, 120 plugins, 19 hooks
- Specialized skill packs for various domains
- Agnix: Linter for AI agent configurations (validates SKILL.md, CLAUDE.md, hooks, MCP configs with 156 rules)

---

## 8. IDE Integration

### VS Code Extension

The most polished IDE integration, providing:
- Native graphical chat panel
- Checkpoint-based undo
- @-mention file references
- Parallel conversations
- Diff viewing
- Selection context sharing

### JetBrains Plugin

Available via JetBrains Marketplace for IntelliJ IDEA, WebStorm, PyCharm:
- Quick launch: Cmd+Esc (Mac) / Ctrl+Esc (Windows/Linux)
- Diff viewing in IDE diff viewer
- Automatic selection context sharing
- Diagnostic error sharing

### Terminal Integration

Running `claude` from IDE's integrated terminal activates all integration features. The agentic loop, tools, and capabilities are identical across all interfaces.

### Desktop App

Manages multiple local sessions visually, each with its own isolated worktree.

### Web Interface

claude.ai/code runs on Anthropic's secure cloud infrastructure in isolated VMs.

### Comparison: Terminal vs IDE

- **IDE extensions**: Best for review and adjustments — opening modified files, reviewing diffs visually, quick fixes
- **Terminal**: Best for complex tasks, finalization, automation, CI/CD integration
- **Recommendation**: Use both — IDE for visual review, terminal for complex work

---

## 9. Community Workflows and Best Practices

### Planning-First Approach

The strongest community consensus: separate research and planning from implementation.

1. **Explore** (Plan Mode): Read files, understand codebase
2. **Plan**: Create detailed implementation plan (Ctrl+G to edit in editor)
3. **Implement**: Switch to Normal Mode, execute plan
4. **Commit**: Descriptive message and PR

### Verification-Driven Development

"The single highest-leverage thing you can do" — give Claude a way to verify its own work:
- Run tests after implementation
- Compare screenshots for UI work
- Validate outputs against expected results
- Use Chrome extension for browser-based UI testing

### Context Management Strategies

- `/clear` between unrelated tasks (most impactful practice)
- After 2 failed corrections: `/clear` and write better initial prompt
- Use subagents for exploration (keeps main context clean)
- `/compact <focus>` for controlled summarization
- `/btw` for quick questions that don't enter context history
- Track context with custom status line

### Writer/Reviewer Pattern

Use multiple sessions for quality:
- Session A writes code
- Session B reviews (fresh context, no bias toward own code)
- Session A incorporates feedback
- Variation: one session writes tests, another writes code to pass them

### Interview Pattern

For larger features:
```
I want to build [brief description]. Interview me in detail using the AskUserQuestion tool.
Ask about technical implementation, UI/UX, edge cases, concerns, and tradeoffs.
```
Claude asks about things you might not have considered. Once spec is complete, start fresh session for implementation.

### Non-Interactive Mode for Automation

`claude -p "prompt"` enables:
- CI/CD integration
- Pre-commit hooks
- Batch processing across files
- Data pipeline integration
- Structured output (--output-format json or stream-json)

### Fan-Out Pattern

For large migrations:
1. Generate task list with Claude
2. Loop through with `claude -p` per file
3. Use `--allowedTools` to scope permissions
4. Refine prompt on first 2-3 files, then run at scale

### Plugin Ecosystem

- 9,000+ plugins available via /plugin marketplace
- Code intelligence plugins give Claude LSP-based symbol navigation
- Plugins bundle skills, hooks, subagents, and MCP servers into installable units
- Official Anthropic marketplace pre-configured

---

## 10. Comparison with Competitors

### Landscape Overview

| Tool | Type | Price | Strength |
|---|---|---|---|
| Claude Code | Terminal agent | $20/mo | Deepest reasoning, richest extensibility |
| Cursor | IDE agent | $20/mo | Best IDE experience, visual feedback |
| GitHub Copilot | IDE extension | $10/mo | Ubiquitous, enterprise-safe |
| Aider | Terminal agent | Free + API | Model flexibility, cost efficiency |
| Cline | VS Code extension | Free + API | BYOM, VS Code native |
| Windsurf | IDE agent | $15/mo | Good IDE experience |
| Devin | Autonomous agent | Variable | Multi-hour autonomous tasks |

### What Each Does Best

**Claude Code**: Complex multi-file reasoning, terminal automation, extensibility platform (hooks/MCP/skills/agents). "The tool developers reach for when other tools fail."

**Cursor**: Visual editing feedback, tab completion, fast inline suggestions, largest IDE agent user base. Best for daily feature work.

**Aider**: Model-agnostic (any LLM including local models), diff-first approach, git-native auto-commits, surgical/reviewable changes. Best for bounded projects where cost matters.

**Cline**: VS Code-native "serious agent" without vendor lock-in. BYOM flexibility.

**Copilot**: Path of least resistance for enterprise. Works everywhere with existing GitHub/Azure relationships.

**Devin**: Fully autonomous multi-hour tasks with no human supervision needed.

### What Claude Code Can Learn from Competitors

1. **Aider's model flexibility**: Supporting multiple LLM providers and local models would increase accessibility
2. **Aider's cost efficiency**: ~3x fewer tokens for comparable quality on bounded tasks
3. **Cursor's visual feedback**: Inline suggestions and visual diffs reduce cognitive load
4. **Copilot's enterprise distribution**: Lower barrier to entry, corporate-friendly
5. **Cline's BYOM approach**: No vendor lock-in
6. **Combined IDE + Terminal**: The emerging consensus is to use both — IDE for visual review, terminal for complex work

### Benchmark Performance

- Claude Code: 80.9% on SWE-bench
- Aider: 81-88% on polyglot benchmarks
- Claude Code uses ~3x more tokens than Aider for ~2.8% accuracy gain
- Cost-quality tradeoff is use-case dependent

---

## 11. Claude Code for Research

### Deep Research Patterns

Claude Code has emerged as a capable research tool, not just a coding assistant. Anthropic internally uses Claude Code for deep research, video creation, and note-taking — the agent harness now powers most of their major agent loops.

Key patterns:
1. **Two-phase research**: Outline generation followed by deep investigation
2. **Multi-agent research system**: Orchestrator-worker architecture with specialized subagents in parallel
3. **Human-in-the-loop control**: Approval gates at each phase
4. **Subagent isolation**: Research exploration in separate contexts keeps main conversation clean

### Effective Research Workflows

- **Use subagents for investigation**: "use subagents to investigate X" explores in separate context
- **Parallel research branches**: Spawn multiple subagents for independent topic exploration
- **Agent teams for competing hypotheses**: Multiple investigators actively trying to disprove each other
- **Skills for structured research**: Custom SKILL.md files defining research methodology, output format
- **Progressive disclosure**: Start with subagent exploration, bring key findings into main context

### Community Tools

- **Deep-Research-skills**: Structured deep research skill for Claude Code with human-in-the-loop control
- **Claude-Deep-Research**: MCP server enabling comprehensive research capabilities
- **academic-research-skills**: Research, write, review, revise, finalize workflow

### Research-Specific Best Practices

1. Scope investigations narrowly to prevent infinite exploration consuming context
2. Save findings to files immediately (external memory survives context compression)
3. Use subagents for high-volume reading (file reads consume main context)
4. Leverage agent teams for adversarial verification (agents challenge each other's findings)
5. Use persistent memory for cross-session knowledge accumulation
6. Auto-memory captures research patterns and debugging insights

---

## 12. What Works Well and What's Missing

### Works Well
- Rich extensibility model (hooks, skills, MCP, subagents, agent teams)
- Context management (compaction, subagent isolation, lazy loading)
- Deterministic enforcement via hooks (vs advisory CLAUDE.md instructions)
- Plugin ecosystem (9,000+ extensions)
- Terminal-native composability with unix tools
- Deep multi-file codebase understanding
- Extended thinking for complex reasoning
- Git-aware operations with checkpoints
- /batch skill for parallel codebase-wide changes

### Missing or Needs Improvement
- **Context pressure remains the central constraint** — performance degrades predictably as context fills
- **No multi-model routing** — cannot use different models for different phases (thinking, critique, fast decisions) within one session (competitors like OpenDev and Aider support this)
- **Model lock-in** — only Claude models supported (vs Aider's any-LLM approach)
- **Visual feedback gap** — terminal-native means less visual editing feedback than IDE agents
- **Agent teams are experimental** — known limitations around session resumption, task coordination, shutdown
- **Token cost** — ~3x more than competitors for comparable bounded tasks
- **Instruction fade-out** — long sessions lose early instructions despite system reminders
- **No formal verification of completeness** — Stop hooks help but are not standard practice
- **Subagent nesting not supported** — cannot spawn subagents from subagents
- **MCP tool context bloat** — many MCP servers consume significant context before work begins

---

## 13. Recommendations for Our Research Workflow

### High-Priority Adoptions

1. **Implement Stop hooks for quality verification**: Add prompt-based or agent-based Stop hooks that check depth checklists before completing research tasks. This converts our advisory CLAUDE.md depth checklist into deterministic enforcement.

2. **Use subagents for all web research**: Every WebFetch and research exploration should happen in a subagent to preserve main context for synthesis and writing.

3. **Create research-specific skills**: Custom SKILL.md files encoding our research methodology, report templates, depth checklist verification, and log file formats. These can be invoked via /research-topic or loaded automatically.

4. **Add SessionStart compact hooks**: Re-inject critical context (current task status, spike state) after compaction to prevent loss of research orientation.

5. **Adopt auto-format hooks**: PostToolUse hooks for markdown formatting after file writes to maintain consistent document quality.

### Medium-Priority Adoptions

6. **Create custom subagents for specialized research roles**: Define research-reader, source-saver, and depth-checker subagents with appropriate tool restrictions and model selection.

7. **Leverage persistent memory for research agents**: Enable user-scoped memory on research subagents so they accumulate knowledge about common sources, patterns, and dead-ends across sessions.

8. **Use .claude/rules/ for research-specific instructions**: Path-specific rules for docs/ (source saving conventions), *-research.md (report quality standards), log.md (log entry format).

9. **Explore agent teams for multi-spike synthesis**: When synthesizing across multiple completed spikes, agent teams could investigate different spikes in parallel with adversarial verification.

10. **Add notification hooks**: Desktop notifications when Claude needs input during long research sessions, allowing multitasking.

### Lower-Priority Explorations

11. **MCP servers for research**: Brave Search MCP, database MCP for structured data queries, Notion MCP for knowledge management.

12. **/batch skill for codebase-wide research updates**: When updating research across multiple files, the built-in /batch skill could parallelize the work.

13. **Plugin exploration**: Code intelligence plugins for any codebases being studied, specialized research plugins from the marketplace.

---

## Sources

### Official Documentation
- [How Claude Code Works](https://code.claude.com/docs/en/how-claude-code-works) — Architecture and agentic loop
- [Hooks Guide](https://code.claude.com/docs/en/hooks-guide) — Hook system documentation
- [Skills](https://code.claude.com/docs/en/skills) — Skill system documentation
- [Sub-agents](https://code.claude.com/docs/en/sub-agents) — Subagent documentation
- [Agent Teams](https://code.claude.com/docs/en/agent-teams) — Agent teams (swarms) documentation
- [Best Practices](https://code.claude.com/docs/en/best-practices) — Official best practices
- [Memory](https://code.claude.com/docs/en/memory) — CLAUDE.md and auto memory system
- [MCP](https://code.claude.com/docs/en/mcp) — Model Context Protocol integration

### Community & Research
- [Piebald-AI/claude-code-system-prompts](https://github.com/Piebald-AI/claude-code-system-prompts) — Reverse-engineered system prompts (110+ prompt strings, 126+ versions tracked)
- [OpenDev Paper](https://arxiv.org/html/2603.05344) — "Building Effective AI Coding Agents for the Terminal" (arXiv, March 2026)
- [hesreallyhim/awesome-claude-code](https://github.com/hesreallyhim/awesome-claude-code) — Curated community extensions
- [rohitg00/awesome-claude-code-toolkit](https://github.com/rohitg00/awesome-claude-code-toolkit) — 135 agents, 35 skills, 120 plugins
- [Weizhena/Deep-Research-skills](https://github.com/Weizhena/Deep-Research-skills) — Structured deep research skill
- [FlorianBruniaux/claude-code-ultimate-guide](https://github.com/FlorianBruniaux/claude-code-ultimate-guide) — Comprehensive beginner-to-power-user guide

### Comparison & Analysis
- [Morph: "We Tested 15 AI Coding Agents"](https://www.morphllm.com/ai-coding-agent) — 2026 competitive analysis
- [Artificial Analysis: Coding Agents Comparison](https://artificialanalysis.ai/insights/coding-agents-comparison) — Quantitative benchmarks
- [Addy Osmani: Claude Code Swarms](https://addyosmani.com/blog/claude-code-agent-teams/) — Agent teams analysis
