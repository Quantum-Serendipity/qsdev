# Claude Code Agent Files: Format, Multi-Agent Workflows, and Sub-Agent Delegation

## Executive Summary

Claude Code agent files (`.claude/agents/*.md`) are markdown files with YAML frontmatter that define specialized subagents with their own context windows, tool access, permission modes, hooks, MCP servers, skills, and persistent memory. They are distinct from skills (which add instructions to the main context) and commands (which are now merged into skills). Subagents cannot spawn other subagents, but the main agent can chain them sequentially or run them in parallel. Agent teams (experimental) enable fully independent sessions with peer-to-peer messaging. For gdev, the key design decision is which consulting workflows should be agents (isolated, specialized workers) vs skills (shared context, orchestrated procedures).

## 1. Agent File Format

### File Location and Priority

| Location | Scope | Priority |
|---|---|---|
| Managed settings | Organization-wide | 1 (highest) |
| `--agents` CLI flag | Current session | 2 |
| `.claude/agents/` | Current project | 3 |
| `~/.claude/agents/` | All projects | 4 |
| Plugin `agents/` | Where plugin enabled | 5 (lowest) |

Same-name agents at higher priority override lower priority. Project agents are recommended for team-shared, version-controlled definitions.

### Frontmatter Fields

```yaml
---
name: code-reviewer          # Required: lowercase + hyphens
description: Reviews code...  # Required: when Claude should delegate
tools: Read, Grep, Glob, Bash # Optional: tool allowlist (inherits all if omitted)
disallowedTools: Write, Edit   # Optional: tool denylist
model: sonnet                  # Optional: sonnet, opus, haiku, full ID, or inherit
permissionMode: default        # Optional: default, acceptEdits, auto, dontAsk, bypassPermissions, plan
maxTurns: 50                   # Optional: maximum agentic turns
skills:                        # Optional: preload skills into agent context
  - api-conventions
  - error-handling-patterns
mcpServers:                    # Optional: scope MCP servers to agent
  - playwright:
      type: stdio
      command: npx
      args: ["-y", "@playwright/mcp@latest"]
  - github                     # Reference existing server by name
hooks:                         # Optional: lifecycle hooks scoped to agent
  PreToolUse:
    - matcher: "Bash"
      hooks:
        - type: command
          command: "./scripts/validate.sh"
memory: project                # Optional: user, project, or local
background: false              # Optional: always run as background task
effort: high                   # Optional: low, medium, high, xhigh, max
isolation: worktree            # Optional: git worktree isolation
color: blue                    # Optional: display color
initialPrompt: "..."           # Optional: auto-submitted first user turn
---

You are a senior security engineer. Review code for:
- Injection vulnerabilities (SQL, XSS, command injection)
- Authentication and authorization flaws
...
```

The markdown body becomes the subagent's system prompt. Subagents receive only this system prompt (plus environment details like working directory), NOT the full Claude Code system prompt.

### Key Constraints

1. **Subagents cannot spawn other subagents** -- this is a hard architectural constraint. If nested delegation is needed, use skills or chain from the main conversation.
2. **Subagents work within a single session** -- for cross-session parallelism, use agent teams or background agents.
3. **Each subagent gets its own context window** -- results return as summaries to the main conversation.
4. **Agent files loaded at session start** -- edits on disk require session restart (unlike skills which hot-reload via the `/agents` interface).

## 2. How Agents Differ from Skills and Commands

| Aspect | Agent | Skill | Command (legacy) |
|---|---|---|---|
| **What it is** | Isolated worker with own context | Instructions added to main context | Merged into skills |
| **Context** | Separate context window | Shared with main conversation | Shared with main conversation |
| **Invocation** | Auto-delegated or explicit `@agent-name` | Auto-invoked or `/skill-name` | `/command-name` |
| **Tools** | Configurable per-agent (allowlist/denylist) | `allowed-tools` pre-approves, doesn't restrict | Same as skills |
| **System prompt** | Agent body replaces system prompt | Skill content injected as message | Same as skills |
| **Model** | Configurable per-agent | Configurable per-skill (for current turn only) | Inherits |
| **Hooks** | Can define scoped hooks | Can define scoped hooks | N/A |
| **Memory** | Persistent memory directory | No persistent memory | N/A |
| **MCP servers** | Can scope MCP servers | No MCP scoping | N/A |
| **Compaction** | Independent (own context) | Re-attached after compaction (5000 token budget per skill) | Same as skills |

### Trail of Bits Principle
"Encode expertise in agents, procedures in commands [skills]."

- **Agents** = specialized personas (security reviewer, debugger, data scientist)
- **Skills** = repeatable procedures (deploy, review PR, fix issue)

This maps to consulting workflows:
- Agent for "security review specialist" (needs own context for deep analysis)
- Skill for "run our standard code review checklist" (procedure in main context)

## 3. Multi-Agent Workflows

### Pattern 1: Sequential Chaining
Main agent → Agent A → Main agent → Agent B → Main agent

```
Use the code-reviewer to find issues, then use the optimizer to fix them
```

Each agent completes, returns results to main, which passes context to next agent.

### Pattern 2: Parallel Subagents
Main agent → Agent A + Agent B + Agent C (parallel) → Main agent synthesizes

```
Research the auth, database, and API modules in parallel using separate subagents
```

Each agent explores independently. Results return to main conversation (caution: many detailed results can consume significant context).

### Pattern 3: Foreground vs Background
- **Foreground**: Blocks main conversation, permission prompts pass through
- **Background**: Runs concurrently, auto-denies tool calls that would prompt, fails clarifying questions

Press Ctrl+B to background a running task.

### Pattern 4: Agent Teams (Experimental)
- Requires CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS=1
- Requires Opus 4.6+
- Team Lead + Teammates architecture
- Git-based coordination with lock files in `.claude/tasks/`
- Mailbox system for peer-to-peer messaging
- Each teammate gets independent 1M token context

Best for read-heavy tasks (code review, analysis). Problematic for write-heavy tasks (shared file modifications cause merge conflicts).

### Pattern 5: Fork Mode (Experimental)
- CLAUDE_CODE_FORK_SUBAGENT=1
- Fork inherits full conversation history instead of starting fresh
- Shares prompt cache with main session (cheaper)
- Good for "try several approaches from same starting point"

## 4. Persistent Memory System

Agents can accumulate knowledge across sessions:

| Scope | Location | Use case |
|---|---|---|
| `user` | `~/.claude/agent-memory/<name>/` | Cross-project learnings |
| `project` | `.claude/agent-memory/<name>/` | Project-specific, shareable via VCS |
| `local` | `.claude/agent-memory-local/<name>/` | Project-specific, not in VCS |

First 200 lines or 25KB of `MEMORY.md` auto-loaded at agent startup. Agent can read/write to memory directory.

This is significant for consulting: a code-reviewer agent on a client project accumulates knowledge of codebase patterns, recurring issues, and architectural decisions across multiple sessions.

## 5. Implications for gdev

### What gdev Should Generate as Agents

Agents are appropriate when the workflow:
- Needs isolated context (won't pollute main conversation)
- Benefits from specialized tool restrictions
- Involves reading many files (exploration, review, analysis)
- Has a clear "return summary" end state
- Benefits from persistent memory across sessions

**Recommended agents for consulting**:
- `security-reviewer`: Read-only + Bash, scoped security review
- `performance-analyzer`: Read-only, focused performance analysis
- `codebase-explorer`: Haiku model (fast, cheap), read-only, onboarding exploration
- `test-gap-analyzer`: Read + Bash(test runners), find untested code

### What gdev Should Generate as Skills

Skills are appropriate when the workflow:
- Is a repeatable procedure with clear steps
- Benefits from sharing context with main conversation
- Needs user invocation control (`disable-model-invocation: true`)
- Is a reference document Claude applies broadly
- Has side effects the user should trigger manually

**Recommended skills for consulting**:
- `/review-pr`: Multi-step PR review checklist
- `/add-tests`: Test generation for uncovered code
- `/upgrade-dep`: Dependency upgrade with verification
- `/onboard`: Systematic codebase exploration guide
- `/write-adr`: Architecture Decision Record generator
- `/incident-debug`: Systematic production debugging
- `/migration-plan`: Framework/dependency migration planner

### What gdev Should Generate as Rules

Rules (`.claude/rules/*.md`) are appropriate for:
- Language-specific conventions with `paths:` frontmatter
- Security rules that should always be active
- Testing conventions for specific directories
- Client-specific compliance requirements

## Depth Checklist

- [x] Underlying mechanism explained (file format, loading, context isolation, lifecycle)
- [x] Key tradeoffs identified (agent vs skill, context isolation vs shared context, foreground vs background)
- [x] Compared to alternatives (agents vs skills vs commands vs hooks vs MCP)
- [x] Failure modes described (no nested delegation, context cost of returned results, merge conflicts in teams)
- [x] Concrete examples found (official docs examples, Trail of Bits patterns, agent team case studies)
- [x] Standalone-readable
