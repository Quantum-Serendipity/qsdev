# Custom Claude Code Skills for Secure Package Installation

## Executive Summary

Claude Code skills are prompt-based workflow packages that can wrap package installation with security checks, but they are fundamentally a **convenience and guidance layer, not an enforcement mechanism**. Skills cannot prevent agents from bypassing them — an agent can always use raw `Bash(npm install ...)` instead of invoking a `/safe-install` skill. The critical finding is that skills gain enforcement value only when **combined with hooks and permission deny rules** that block raw install commands and force the agent to route through skill-defined workflows. However, skills bring unique capabilities that hooks and MCP servers lack: they can embed pre-flight validation scripts, carry domain-specific security knowledge into context, define lifecycle-scoped hooks in their own frontmatter, and provide a structured user-facing interface for security decisions. The optimal architecture uses skills as the **user-facing workflow orchestrator** while hooks and permissions provide the **deterministic enforcement backbone**.

---

## 1. How Claude Code Skills Work

### 1.1 Directory Structure and Discovery

Skills are directories containing a `SKILL.md` file (required) plus optional supporting files:

```
.claude/skills/safe-install/
├── SKILL.md              # Main instructions (required)
├── known-malicious.txt   # Blocklist data
├── scripts/
│   └── check-package.sh  # Validation script
└── examples/
    └── usage.md          # Example output format
```

Skills are stored at three levels with override priority: enterprise > personal (`~/.claude/skills/`) > project (`.claude/skills/`). Plugin skills use namespaced identifiers and cannot conflict with other levels.

Claude Code watches skill directories for file changes — edits take effect within the current session without restarting.

**Source**: `docs/claude-code-skills-official-docs.md`

### 1.2 SKILL.md Format

Every skill requires a `SKILL.md` with YAML frontmatter and markdown body:

```yaml
---
name: safe-install
description: Securely install a package with vulnerability and supply chain checks. Use when the user or agent needs to install any npm, pip, cargo, or go package.
disable-model-invocation: false   # Let Claude auto-invoke
allowed-tools: Bash(node *) Bash(python3 *) Read
arguments: [package-manager, package-spec]
---

# Safe Package Installation

Install $package-manager package $package-spec with pre-flight security validation.

## Steps
1. Run `${CLAUDE_SKILL_DIR}/scripts/check-package.sh $package-manager $package-spec`
2. If the check fails, report the findings and DO NOT install
3. If the check passes, proceed with installation using the standard command
4. Run post-install audit (`npm audit` / `pip-audit` / etc.)
```

Key frontmatter fields for the security use case:

| Field | Security Relevance |
|-------|-------------------|
| `description` | Primary trigger mechanism — Claude matches against this to decide when to auto-invoke |
| `disable-model-invocation` | `false` (default) allows Claude to auto-invoke when it detects a package install context |
| `user-invocable` | `true` (default) for user-triggered `/safe-install`, or `false` for background knowledge |
| `allowed-tools` | Pre-approves specific tools (e.g., scripts) without per-use permission prompts |
| `hooks` | Embeds PreToolUse hooks scoped to the skill's lifetime |
| `context` | `fork` runs in isolated subagent (prevents context pollution) |
| `paths` | Glob patterns to limit activation to specific project paths |

**Source**: `docs/claude-code-skills-official-docs.md`

### 1.3 Invocation Mechanisms

Skills can be invoked two ways:

1. **User-triggered**: User types `/safe-install lodash` in the chat. The skill content loads and Claude follows its instructions.
2. **Auto-triggered by Claude**: Claude reads the `description` field, matches it against the current task context, and autonomously decides to load and follow the skill. This is the key differentiator from slash commands.

The invocation matrix:

| Configuration | User can invoke | Claude can invoke | Description in context |
|---------------|----------------|-------------------|----------------------|
| Default | Yes | Yes | Always (description only, ~100 tokens) |
| `disable-model-invocation: true` | Yes | No | Not loaded |
| `user-invocable: false` | No | Yes | Always |

When Claude auto-invokes a skill, the full SKILL.md content enters the conversation as a single message and **stays for the rest of the session** (or until compacted). After compaction, the first 5,000 tokens of each invoked skill are re-attached, with a combined budget of 25,000 tokens across all invoked skills.

**Source**: `docs/claude-code-skills-official-docs.md`, `docs/skills-vs-slash-commands-mindstudio.md`

### 1.4 Tools Available Inside Skills

Skills have access to **all tools available in the Claude Code session**:

- **Bash**: Run arbitrary shell commands (subject to permission rules)
- **Read/Write/Edit**: File system operations
- **WebFetch/WebSearch**: Internet access (Claude Code has full network access)
- **MCP tools**: Any connected MCP server tools (e.g., `mcp__socket__check_package`)
- **Bundled scripts**: Scripts in the skill's directory via `${CLAUDE_SKILL_DIR}/scripts/`
- **Subagent spawning**: Via `context: fork` or Task tool

The `allowed-tools` frontmatter field **grants permissions** for listed tools without per-use prompts, but does **not restrict** which tools are available. All tools remain callable; permission settings still govern unlisted tools.

**Critical limitation**: `allowed-tools` only grants — it cannot deny. To block tools, use permission deny rules in settings.json.

### 1.5 Dynamic Context Injection

Skills support the `` !`command` `` syntax to run shell commands **before** the skill content is sent to Claude:

```yaml
---
name: safe-install
description: Install packages with security validation
---

## Current project dependencies
!`cat package.json | jq '.dependencies' 2>/dev/null || echo "No package.json"`

## Known vulnerable packages in this project
!`npm audit --json 2>/dev/null | jq '.vulnerabilities | keys' || echo "No npm audit available"`

## Instructions
...
```

The command output replaces the placeholder, so Claude receives actual data grounded in the current project state. This is preprocessing — Claude only sees the final rendered content.

This can be disabled organization-wide via `disableSkillShellExecution: true` in managed settings.

**Source**: `docs/claude-code-skills-official-docs.md`

---

## 2. Can Skills Enforce Secure Installation?

### 2.1 The Fundamental Limitation: Skills Are Advisory

Skills are **prompt-based instructions**, not deterministic enforcement. When Claude auto-invokes a `/safe-install` skill, it receives instructions to run security checks before installing. But nothing prevents Claude from:

1. Ignoring the skill entirely and running `Bash(npm install malicious-pkg)` directly
2. Partially following the skill (running the check but installing anyway despite failures)
3. Installing via indirect means (editing `package.json` directly, running a script that installs)

This is the same fundamental limitation as CLAUDE.md instructions: they shape what Claude *tries* to do, but do not control what Claude Code *allows*. Permission rules and hooks are the enforcement layer.

### 2.2 Skills + Hooks + Permissions = Enforcement

The enforcement architecture requires all three layers:

```
Layer 1: Permission Deny Rules (deterministic blocking)
  → Block: Bash(npm install *), Bash(pip install *), Bash(cargo add *), etc.
  → Block: Bash(yarn add *), Bash(pnpm add *), Bash(go get *), etc.
  → Allow: Bash(${SKILL_DIR}/scripts/safe-install.sh *)

Layer 2: PreToolUse Hooks (programmatic validation)
  → Intercept any Bash command containing install patterns
  → Parse package name and version
  → Query vulnerability databases
  → Block via exit code 2 or JSON { "decision": "block" }

Layer 3: Custom Skill (workflow orchestration)
  → Provide the structured installation workflow
  → Carry security context and knowledge
  → Guide the agent through check → decide → install → audit
  → Present findings to the user in a readable format
```

When all three layers are in place:
- The agent **cannot** run raw install commands (Layer 1 blocks them)
- Even if Layer 1 is bypassed, the hook **intercepts and validates** (Layer 2)
- The skill provides the **approved pathway** that satisfies both layers (Layer 3)

### 2.3 Hooks Inside Skills: Lifecycle-Scoped Enforcement

A powerful feature: skills can define their own hooks in SKILL.md frontmatter. These hooks are **scoped to the skill's lifetime** — they activate when the skill is invoked and deactivate when the skill finishes.

```yaml
---
name: safe-install
description: Install packages with security validation
hooks:
  PreToolUse:
    - matcher: "Bash"
      hooks:
        - type: command
          command: "./scripts/security-check.sh"
  PostToolUse:
    - matcher: "Bash"
      hooks:
        - type: command
          command: "./scripts/post-install-audit.sh"
---
```

This means a security skill can carry its own enforcement hooks. When the skill is active, every Bash command is validated by the skill's bundled script. When the skill is not active, these hooks don't fire.

**Key characteristics**:
- All hook events are supported (PreToolUse, PostToolUse, SessionStart, etc.)
- Hooks use the same configuration format as settings-based hooks
- Automatic cleanup when the skill finishes
- For subagents, `Stop` hooks are automatically converted to `SubagentStop`

**Limitation**: These lifecycle-scoped hooks only fire while the skill is active. If the agent never invokes the skill (or if the skill is compacted away), the hooks are not in effect. This is why **global hooks in settings.json are still necessary** as the always-on enforcement layer.

**Source**: `docs/hooks-in-skills-official-reference.md`

---

## 3. Practical Pattern: A `/safe-install` Skill

### 3.1 Complete Skill Implementation

```
.claude/skills/safe-install/
├── SKILL.md
├── scripts/
│   ├── check-package.sh       # Pre-flight validation
│   └── post-install-audit.sh  # Post-install verification
└── data/
    └── known-malicious.txt    # Local blocklist
```

**SKILL.md**:

```yaml
---
name: safe-install
description: >
  Securely install a package with pre-flight vulnerability and supply chain
  checks. Use whenever installing npm, pip, cargo, go, or system packages.
  Checks OSV.dev for known vulnerabilities, validates package age and
  popularity, and runs post-install audit.
disable-model-invocation: false
allowed-tools: Bash(${CLAUDE_SKILL_DIR}/scripts/*) Read
hooks:
  PreToolUse:
    - matcher: "Bash"
      hooks:
        - type: command
          command: "${CLAUDE_SKILL_DIR}/scripts/check-package.sh"
arguments: [pm, package]
---

# Secure Package Installation

Install the package `$package` using `$pm` with security validation.

## Pre-flight Checks

1. Run the security check script:
   ```bash
   ${CLAUDE_SKILL_DIR}/scripts/check-package.sh $pm $package
   ```

2. Interpret the results:
   - **PASS**: Proceed to installation
   - **WARN**: Report warnings to the user and ask for confirmation
   - **FAIL**: Do NOT install. Report the security findings.

## Installation

3. If checks pass, install with version pinning:
   - npm: `npm install --save-exact $package`
   - pip: `pip install $package` then pin in requirements.txt
   - cargo: `cargo add $package`
   - go: `go get $package`

## Post-Install Audit

4. Run the post-install audit:
   ```bash
   ${CLAUDE_SKILL_DIR}/scripts/post-install-audit.sh $pm
   ```

5. Report any new vulnerabilities introduced by the installation.

## Important

- NEVER install a package that fails the pre-flight check
- ALWAYS pin exact versions
- If the check script is unavailable, refuse to install and explain why
```

### 3.2 Companion Permission Rules

For the skill to have enforcement value, raw install commands must be blocked:

```json
{
  "permissions": {
    "deny": [
      "Bash(npm install *)",
      "Bash(npm add *)",
      "Bash(yarn add *)",
      "Bash(pnpm add *)",
      "Bash(pip install *)",
      "Bash(pip3 install *)",
      "Bash(uv pip install *)",
      "Bash(uv add *)",
      "Bash(poetry add *)",
      "Bash(cargo add *)",
      "Bash(cargo install *)",
      "Bash(go get *)",
      "Bash(go install *)",
      "Bash(gem install *)",
      "Bash(bundle add *)",
      "Bash(composer require *)",
      "Bash(dotnet add package *)",
      "Bash(nix-env -i *)",
      "Bash(nix profile install *)"
    ],
    "allow": [
      "Bash(.claude/skills/safe-install/scripts/*)"
    ]
  }
}
```

**Critical gap**: These deny rules can be bypassed via shell wrappers (`bash -c "npm install ..."`, variable expansion, etc.). A global PreToolUse hook in settings.json must catch what deny rules miss. See `hooks-research.md` for the full bypass vector analysis.

---

## 4. Skills vs. MCP Servers vs. Hooks: Comparative Analysis

### 4.1 Comparison Matrix

| Dimension | Skills | MCP Servers | Hooks |
|-----------|--------|-------------|-------|
| **Enforcement** | Advisory (prompt-based) | Advisory (agent can bypass) | Deterministic (fires before execution) |
| **Auto-trigger** | Yes (description matching) | No (agent must call tool) | Yes (event-based, always fires) |
| **Bypass resistance** | Low (agent can ignore) | Low (agent can use raw Bash) | High (fires before permission check) |
| **User interface** | `/slash-command` invocation | Tool call in conversation | Invisible (background) |
| **Portability** | Project-local or personal | Reusable across projects | Settings-level, configurable per scope |
| **Context cost** | ~100 tokens (description) to ~5K (full) | ~200 tokens per tool listing | Zero (runs outside model context) |
| **External API access** | Yes (via Bash/WebFetch) | Yes (native, server-side) | Yes (via command handler) |
| **Complexity** | Low (markdown + optional scripts) | Medium (server implementation) | Low-Medium (shell scripts or HTTP) |
| **Bundled scripts** | Yes (skill directory) | Yes (server code) | Limited (command string) |
| **Lifecycle scoping** | Skill can define its own hooks | Always-on while connected | Always-on (settings) or skill-scoped |
| **Enterprise distribution** | Managed settings, plugins | MCP config in managed settings | Managed hooks |
| **User visibility** | High (slash command, readable instructions) | Low (tool calls in conversation) | None (invisible enforcement) |

### 4.2 What Skills Add Beyond Hooks + MCP

1. **Structured workflow orchestration**: Skills provide step-by-step instructions that guide the agent through a multi-step security check → decision → install → audit pipeline. Hooks can block or allow, but cannot orchestrate a multi-step workflow.

2. **User-facing interface**: `/safe-install lodash` is a clear, discoverable action. Users can see exactly what the skill does by reading SKILL.md. Hooks and MCP are invisible infrastructure.

3. **Contextual knowledge injection**: Skills can carry domain-specific security knowledge (known-malicious lists, organizational policies, approved package registries) that loads into context only when relevant. Hooks cannot inject knowledge into the model's context.

4. **Dynamic pre-flight context**: The `!`command`` syntax can inject current project state (existing vulnerabilities, dependency tree, lockfile hash) into the skill's instructions before Claude sees them. This grounds security decisions in project reality.

5. **Lifecycle-scoped hooks**: Skills can define their own hooks that activate only when the skill is in use. This enables security validation that is contextually appropriate — different checks for npm vs. pip vs. cargo, activated by the skill variant invoked.

6. **Progressive disclosure**: Skill metadata (~100 tokens) is always in context; full instructions (~5K tokens) load only when invoked. This is more context-efficient than embedding all security instructions in CLAUDE.md.

### 4.3 What Skills Cannot Do That Hooks/MCP Can

1. **Deterministic enforcement**: Skills cannot block execution. Only hooks (via exit code 2) and permission deny rules can deterministically prevent a command from running.

2. **Always-on protection**: Skills only activate when invoked (by user or auto-triggered by Claude). Hooks fire on every tool call regardless. A global PreToolUse hook protects even when the skill is not in context.

3. **Bypass resistance**: An agent can choose to ignore a skill or work around it. Hooks and deny rules cannot be ignored by the agent (barring the documented bypass vectors).

4. **Subcommand interception**: Hooks receive the raw command JSON and can parse individual subcommands. Skills operate at the instruction level and cannot intercept commands they didn't initiate.

---

## 5. Can Agents Be Forced to Use Skills?

### 5.1 The Indirect Forcing Pattern

Agents cannot be directly forced to invoke a skill. However, they can be **indirectly forced** into the skill's workflow through a combination of mechanisms:

1. **Deny raw install commands** (permissions): Block `Bash(npm install *)`, etc.
2. **CLAUDE.md instructions**: "Always use `/safe-install` for package installation. Never install packages directly."
3. **Skill auto-invocation**: Set a descriptive `description` so Claude auto-invokes the skill when it detects a package installation context.
4. **PreToolUse hook as fallback**: If the agent tries to bypass both the skill and the deny rules, the hook catches and blocks the attempt.

The result: the agent's only viable path to installing packages is through the skill's approved workflow. Raw commands are blocked by deny rules + hooks. The skill provides the only path that satisfies permission rules.

### 5.2 Can Agents Invoke Skills Autonomously?

Yes. When `disable-model-invocation` is `false` (the default), Claude can autonomously decide to invoke a skill based on description matching. The skill's `description` and `when_to_use` fields are the primary trigger mechanism.

For a `/safe-install` skill, the description should explicitly mention package installation keywords:

```yaml
description: >
  Securely install a package with vulnerability checks. Use when installing
  any npm, pip, cargo, go, yarn, pnpm, poetry, uv, gem, or bundle package.
  Use when the user asks to "add a dependency", "install a library", or
  "add a package".
when_to_use: >
  Trigger when the conversation involves adding new dependencies,
  installing packages, or updating package versions.
```

However, auto-invocation is **probabilistic, not deterministic**. Claude may or may not match the context to the skill description. This is another reason why hooks (deterministic) must back up skills (probabilistic).

### 5.3 Can Skills Be Auto-Triggered by Installation Attempts?

Not directly. There is no mechanism to say "whenever Claude tries to run `npm install`, auto-invoke this skill instead." The closest pattern is:

1. A **PreToolUse hook** intercepts the install command
2. The hook **blocks** the command (exit code 2) with a message: "Use /safe-install instead"
3. Claude receives the denial feedback and (ideally) invokes the skill

This is a two-step redirect, not a direct auto-trigger. The agent sees the denial, reads the feedback message, and should then use the skill. But there is no guarantee — the agent might try a different approach or ask the user.

**A more robust pattern**: The PreToolUse hook itself performs the security check (rather than redirecting to the skill). The skill serves as the user-facing workflow for intentional installations, while the hook serves as the invisible safety net for all other cases.

---

## 6. Real-World Examples of Security-Focused Skills

### 6.1 Security Phoenix: Security Assessment Suite

The most comprehensive security skills toolkit found is [Security-Phoenix-demo/security-skills-claude-code](https://github.com/Security-Phoenix-demo/security-skills-claude-code). It includes:

- **Package installation gating**: PreToolUse hook that intercepts npm/yarn/pnpm/pip/uv/poetry/cargo/go/gem/bundle/composer/dotnet install commands
- **Known-malicious blocking**: Blocks packages on a known-malicious list
- **Typosquat detection**: Flags packages that look like typosquats of popular packages
- **New package flagging**: Flags brand-new packages for user confirmation
- **SessionStart audit**: Fingerprints the project and runs dependency audits at session start
- **PostToolUse scanning**: Pattern scans file writes for SQL injection, innerHTML, hardcoded secrets

The suite uses **hooks embedded in skills** — the `--full` install option enables all hooks alongside the skill workflows. The `--lite` option installs skills without hooks (advisory only).

This is the closest real-world implementation to the architecture described in this report.

**Source**: `docs/security-phoenix-skills-repo.md`

### 6.2 attach-guard Plugin

The attach-guard plugin (documented in `hooks-research.md`) uses a PreToolUse hook to intercept package install commands and query the Socket.dev API for supply chain scoring. It is hook-based, not skill-based, but demonstrates the pattern of external API validation at install time.

### 6.3 Anthropic Official Skills Repository

The [anthropics/skills](https://github.com/anthropics/skills) repository contains document-focused skills (docx, pdf, pptx, xlsx) and creative/development skills. **No security-focused skills exist in the official repository.** The security use case is addressed through hooks and permissions in Anthropic's documentation, not through skills.

**Source**: `docs/anthropic-skills-repository.md`

### 6.4 Community Guardrail Projects

Several community projects (dwarvesf/claude-guardrails, rulebricks/claude-code-guardrails, mafiaguy/claude-security-guardrails) provide security guardrails, but all use **hooks and permission rules**, not skills. This pattern is consistent: the community has converged on hooks as the enforcement mechanism and has not widely adopted skills for security enforcement.

---

## 7. Limitations and Edge Cases

### 7.1 Skills Are Compactable

After context compaction, skills are re-attached with a budget of 5,000 tokens each and 25,000 tokens combined. If a session invokes many skills, older skills may be dropped entirely. A security skill that gets compacted away loses its lifecycle-scoped hooks and contextual knowledge.

**Mitigation**: Keep the `/safe-install` skill concise (<500 lines). Use supporting files for reference material rather than embedding everything in SKILL.md.

### 7.2 Description Matching Is Imprecise

Claude's auto-invocation depends on matching the skill description to the current task context. This matching is probabilistic — Claude may fail to auto-invoke the security skill when it should, or invoke it when it shouldn't.

**Mitigation**: Write precise descriptions with explicit trigger keywords. Use `when_to_use` for additional matching context. But never rely on auto-invocation as the sole enforcement mechanism.

### 7.3 Skill Shell Execution Can Be Disabled

The `disableSkillShellExecution: true` setting prevents `` !`command` `` and `` ```! `` blocks from executing. If this is set in managed settings, dynamic context injection (pre-flight project state) is disabled for all non-managed skills.

**Mitigation**: Design the skill to work both with and without dynamic context. Use the validation script (run by Claude via Bash, not shell injection) as the primary check.

### 7.4 Enterprise vs. Personal Skills

Enterprise skills (managed settings) override personal and project skills. If an organization deploys a `/safe-install` skill via managed settings, individual developers cannot override it with their own version. This is desirable for security enforcement but may cause friction if the managed skill is too restrictive.

### 7.5 No Skill-Level Permission Restriction

The `allowed-tools` field can **grant** permissions but cannot **restrict** them. A skill cannot say "while I'm active, block all Bash commands except these." To achieve restriction, you must use permission deny rules at the settings level.

### 7.6 Plugin Skills vs. Project Skills

Skills distributed via plugins use namespaced identifiers (`plugin:skill-name`) and are subject to plugin trust model. Plugin skills from untrusted sources can grant themselves broad tool access via `allowed-tools` — review plugin skills before trusting a repository.

---

## 8. Recommended Architecture

### 8.1 Skills as Workflow Layer (Not Enforcement Layer)

The optimal role for skills in the package security architecture:

```
┌─────────────────────────────────────────────────────┐
│  Layer 3: SKILL (Workflow Orchestration)             │
│  /safe-install — structured check→decide→install     │
│  Auto-invokes on package install context             │
│  Carries security knowledge and project context      │
│  User-facing, discoverable, readable                 │
├─────────────────────────────────────────────────────┤
│  Layer 2: HOOKS (Programmatic Enforcement)           │
│  PreToolUse — intercepts ALL Bash commands            │
│  Parses install patterns, queries vuln DBs           │
│  Blocks via exit code 2 / JSON deny                  │
│  Always-on, deterministic, cannot be ignored         │
├─────────────────────────────────────────────────────┤
│  Layer 1: PERMISSIONS (Deterministic Blocking)       │
│  Deny rules block raw install commands               │
│  Allow rules permit skill scripts only               │
│  Fires before hooks, cannot be overridden by model   │
├─────────────────────────────────────────────────────┤
│  Layer 0: MCP SERVER (Validation Logic)              │
│  Socket.dev / Snyk / custom OSV.dev server           │
│  Provides check_package tool for hooks and skills    │
│  Reusable across projects, maintained separately     │
└─────────────────────────────────────────────────────┘
```

### 8.2 The Key Insight

**Skills answer the question "how should the agent install packages?" while hooks and permissions answer "what happens if the agent tries to install packages without following the approved workflow?"**

Skills are the carrot; hooks and permissions are the stick. Both are necessary. A skill without enforcement is just a suggestion. Enforcement without a skill leaves the agent with no approved path to install packages (resulting in it asking the user to install manually, which defeats the purpose of an AI coding assistant).

### 8.3 When Skills Add Value vs. When They Don't

**Skills add value when:**
- You want a discoverable, user-invocable installation workflow
- The security check requires multi-step reasoning (not just pass/fail)
- You need to present security findings to the user in a structured format
- Different package managers need different check workflows
- You want to carry organizational security policies into the agent's context

**Skills don't add value when:**
- You just need to block known-bad packages (use hooks)
- The validation is fully automated with no user decision points (use hooks + MCP)
- You don't need a user-facing installation command (use hooks)
- The project has no established package installation workflow (hooks suffice)

---

## 9. Depth Checklist

- [x] **Underlying mechanism explained**: Skills are prompt-based instructions loaded from SKILL.md files, auto-discovered by Claude based on description matching, with optional lifecycle-scoped hooks and dynamic context injection.
- [x] **Key tradeoffs and limitations identified**: Advisory not enforcement, compactable, imprecise auto-triggering, cannot restrict tools (only grant), dependent on hooks+permissions for actual enforcement.
- [x] **Compared to alternatives**: Detailed comparison with MCP servers and hooks across 12 dimensions. Skills are the workflow layer; hooks are the enforcement layer; MCP is the validation logic layer.
- [x] **Failure modes and edge cases described**: Compaction dropping skills, description mismatch, shell execution disabled, plugin trust issues, no skill-level permission restriction.
- [x] **Concrete examples found**: Security Phoenix suite (production security skills with hooks), attach-guard (hook-based), Anthropic official repo (no security skills), community guardrails (all hook-based).
- [x] **Report is standalone-readable**: Complete architectural guidance without requiring other reports (though references to hooks-research.md and mcp-server-research.md provide deeper detail on those layers).

---

## Sources

- [Claude Code Skills Official Documentation](https://code.claude.com/docs/en/skills) → `docs/claude-code-skills-official-docs.md`
- [Agent Skills Platform Overview](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/overview) → `docs/agent-skills-platform-overview.md`
- [Skills vs Slash Commands (MindStudio)](https://www.mindstudio.ai/blog/claude-code-skills-vs-slash-commands) → `docs/skills-vs-slash-commands-mindstudio.md`
- [Anthropic Skills Repository](https://github.com/anthropics/skills) → `docs/anthropic-skills-repository.md`
- [Claude Code Skills Customization Guide (alexop.dev)](https://alexop.dev/posts/claude-code-customization-guide-claudemd-skills-subagents/) → `docs/alexop-skills-customization-guide.md`
- [Security Phoenix Skills for Claude Code](https://github.com/Security-Phoenix-demo/security-skills-claude-code) → `docs/security-phoenix-skills-repo.md`
- [Hooks in Skills and Agents (Official Reference)](https://code.claude.com/docs/en/hooks) → `docs/hooks-in-skills-official-reference.md`
