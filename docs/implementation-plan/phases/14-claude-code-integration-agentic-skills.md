# Phase 14: Claude Code Integration & Agentic Skills Library

## Goal

Make gdev self-operating through Claude Code by deploying a library of skills and agents that let Claude Code invoke gdev commands, follow consulting best practices, and automate common development workflows. A developer can say "set up this repo for Python development with full security" and Claude Code does it via gdev skills. This is the key differentiator: gdev becomes the first secure devenv bootstrap tool that is fully operable through natural language via a curated agentic skills library.

## Dependencies

Phase 4 complete (Claude Code addon — settings.json, CLAUDE.md, hook deployment, skill/rule library, .mcp.json generation). Phase 11 complete (AI agent tooling — agent-postmortem, Version-Sentinel, semble, per-ecosystem verification commands). Phase 12 complete (tool lifecycle — `gdev enable/disable/status/list`, section markers, shared-file surgery). Phase 13 complete (project config — detection engine, profile system, non-interactive mode).

## Phase Outputs

- 10 gdev operation skills (`.claude/skills/gdev-*/SKILL.md`) — 6 user-only (side effects), 4 Claude-invocable (read-only diagnostics)
- 7 consulting workflow agents (`.claude/agents/*.md`) for context-isolated analysis tasks
- 8+ consulting workflow skills (`.claude/skills/*/SKILL.md`) for procedural task execution
- Context budget management ensuring CLAUDE.md stays under 5% of context window
- Deny rule conflict validation (generated deny rules don't block legitimate skill operations)
- devenv task definitions per detected ecosystem (build, test, lint, format)
- Dynamic context injection via `` !`command` `` preprocessor for live system state
- Model-aware generation (Sonnet 200K vs Opus 1M context budgets)
- Per-ecosystem verification commands surfaced in skills and CLAUDE.md

---

### Unit 14.1: gdev Operation Skills

**Description:** Deploy 10 skill files that expose gdev CLI operations to Claude Code, split into user-only skills (side effects requiring explicit `/slash-command` invocation) and Claude-invocable skills (read-only diagnostics safe for autonomous use). Each skill uses dynamic context injection to pre-capture system state before Claude begins reasoning, and `--json` output for structured data parsing.

**Context:** The gdev CLI from Phases 9-12 provides `init`, `doctor`, `setup`, `enable`, `disable`, `status`, `list`, and `detect` commands — all with `--json` output support. Phase 4 established the claudecode addon's embedded skill library using Go's `embed.FS`. The Claude Code skill format (`.claude/skills/<name>/SKILL.md`) supports YAML frontmatter with `disable-model-invocation`, `allowed-tools`, `arguments`, and dynamic context injection via the `` !`command` `` preprocessor. The research spike `gdev-claude-code-integration` validated that skills (not MCP servers) are the correct integration mechanism, and that dynamic state injection is the strongest CLI wrapper pattern. The five-layer safety architecture (skill-level → tool-level → gdev-level → permission-level → hooks-level) ensures side-effect operations require explicit user invocation.

**Code-Grounded Notes:** 6 skills already exist in `addons/claudecode/templates/skills/` (deploy, review-pr, security-review, generate-tests, refactor, db-migration), deployed via `deploySkills()` at `addons/claudecode/generate_skills.go:42-78`. The skill manifest at `templates/skills/manifest.yaml` must be extended with the 10 new operation skills. IMPORTANT: `` !`command` `` syntax is a Claude Code runtime feature — skills can use this syntax directly in deployed markdown files and Claude Code will preprocess them at invocation time. No gdev-side templating is needed for dynamic context injection; the raw `` !`command` `` strings are written verbatim into the SKILL.md files.

**Desired Outcome:** After `gdev init`, developers find 10 gdev skills in `.claude/skills/`. Typing `/gdev-doctor` runs health checks with analysis. Claude can autonomously invoke `gdev-doctor`, `gdev-status`, `gdev-tools`, and `gdev-detect` when troubleshooting. Side-effect operations (`/gdev-init`, `/gdev-onboard`, `/gdev-setup`, `/gdev-enable`, `/gdev-disable`, `/gdev-update`) require explicit user invocation via slash command.

**Steps:**

1. Create skill directory structure in the embedded library:
   ```
   internal/claudecode/skills/
   ├── gdev-init/SKILL.md
   ├── gdev-onboard/SKILL.md
   ├── gdev-setup/SKILL.md
   ├── gdev-enable/SKILL.md
   ├── gdev-disable/SKILL.md
   ├── gdev-update/SKILL.md
   ├── gdev-doctor/SKILL.md
   ├── gdev-status/SKILL.md
   ├── gdev-tools/SKILL.md
   └── gdev-detect/SKILL.md
   ```

2. Implement `/gdev-init` (user-only, side effects):
   ```yaml
   ---
   name: gdev-init
   description: Initialize a new project with gdev. Detects ecosystem, generates security-hardened devenv configs, Claude Code settings, and pre-commit hooks.
   disable-model-invocation: true
   allowed-tools: Bash(gdev *) Read Grep Glob
   argument-hint: [--profile <name>] [--yes]
   ---

   ## Current project state
   !`gdev devenv doctor --json 2>/dev/null || echo '{"installed": false}'`

   ## Current directory
   !`ls -la`

   ## Detected ecosystems
   !`gdev detect --json 2>/dev/null || echo '{"ecosystems": []}'`

   ## Instructions

   Initialize this project with gdev. Based on the detected state above:

   1. If gdev is not installed, tell the user to install it first and stop.
   2. If ecosystems are detected, confirm the detection with the user.
   3. First run `gdev init --dry-run --json` and show the user what files would be created/modified.
   4. Ask for confirmation before proceeding.
   5. Run `gdev init` with appropriate flags:
      - If $ARGUMENTS includes --yes or --profile, pass them through
      - Otherwise, use `gdev init --non-interactive` with detected ecosystems
      - If the user specified a profile (e.g., "set up for Python with full security"), map to flags
   6. After init completes, run `gdev devenv doctor --json` to verify everything is healthy.
   7. Summarize what was created and any manual steps needed.

   Natural language mappings:
   - "set up this repo for Python development" → `gdev init --ecosystem python --non-interactive`
   - "initialize for TypeScript with full security" → `gdev init --ecosystem javascript-typescript --profile security-full --non-interactive`
   - "bootstrap this Go project" → `gdev init --ecosystem go --non-interactive`
   ```

3. Implement `/gdev-onboard` (user-only, side effects):
   ```yaml
   ---
   name: gdev-onboard
   description: Onboard an existing project to gdev management. Detects what's already configured, identifies gaps, fills them without overwriting existing customizations.
   disable-model-invocation: true
   allowed-tools: Bash(gdev *) Read Grep Glob
   ---

   ## Project analysis
   !`gdev devenv doctor --json 2>/dev/null || echo '{"installed": false}'`

   ## Existing configuration files
   ```!
   ls -la .envrc devenv.yaml devenv.nix .pre-commit-config.yaml .claude/settings.json CLAUDE.md 2>/dev/null || echo "No existing configs found"
   ```

   ## Detected ecosystems
   !`gdev detect --json 2>/dev/null || echo '{"ecosystems": []}'`

   ## Instructions

   Onboard this existing project to gdev management:

   1. Analyze what's already configured vs what's missing from the state above.
   2. Present a gap analysis to the user:
      - What security hardening is already in place
      - What's missing or could be improved
      - What gdev would add/modify
   3. Ask the user to confirm before making changes.
   4. Run: `gdev init --merge --non-interactive`
   5. Run `gdev devenv doctor --json` to verify the result.
   6. Summarize what was added and what was preserved.
   ```

4. Implement `/gdev-setup` (user-only, side effects):
   ```yaml
   ---
   name: gdev-setup
   description: Install missing prerequisites detected by gdev devenv doctor. Requires user confirmation before installing system packages.
   disable-model-invocation: true
   allowed-tools: Bash(gdev *)
   ---

   ## Prerequisites check
   !`gdev devenv doctor --json 2>/dev/null | jq '{overall, checks: [.checks[] | select(.status != "pass")]}' || echo '{"error": "gdev not found"}'`

   ## Instructions

   Install missing prerequisites:

   1. Show the failing/warning checks from the status above.
   2. Run: `gdev devenv setup --dry-run` to show what would be installed.
   3. Ask the user: "The following packages will be installed: [list]. This may require sudo. Proceed?"
   4. If confirmed: `gdev devenv setup`
   5. Run `gdev devenv doctor --json` to verify everything is healthy after installation.
   ```

5. Implement `/gdev-enable` (user-only, side effects):
   ```yaml
   ---
   name: gdev-enable
   description: Enable a security or development tool in the current gdev-managed project. Adds tool configuration, updates shared config files, and verifies the tool is working.
   disable-model-invocation: true
   allowed-tools: Bash(gdev *) Read Grep
   argument-hint: <tool-name>
   arguments: [tool]
   ---

   ## Currently enabled tools
   !`gdev status --json 2>/dev/null || echo '{"tools": []}'`

   ## Available tools
   !`gdev list --json 2>/dev/null || echo '{"available": []}'`

   ## Instructions

   Enable the tool: $tool

   1. Check if $tool is already enabled (from status above).
   2. Check if $tool is in the available list.
   3. If valid, run: `gdev enable $tool`
   4. Verify with: `gdev status --json`
   5. Report what changed (files modified, configs updated).
   6. Note any additional setup steps the user needs to take.

   If $tool is empty, show the available tools list and ask what to enable.
   ```

6. Implement `/gdev-disable` (user-only, side effects):
   ```yaml
   ---
   name: gdev-disable
   description: Disable and cleanly remove a tool from the current gdev-managed project. Removes configuration, cleans up shared config files.
   disable-model-invocation: true
   allowed-tools: Bash(gdev *) Read Grep
   argument-hint: <tool-name>
   arguments: [tool]
   ---

   ## Currently enabled tools
   !`gdev status --json 2>/dev/null || echo '{"tools": []}'`

   ## Instructions

   Disable the tool: $tool

   1. Check if $tool is currently enabled.
   2. If enabled, run: `gdev disable $tool`
   3. Verify with: `gdev status --json`
   4. Report what changed (files removed, configs cleaned up).
   5. Note if any other tools depended on the disabled tool.

   If $tool is empty, show currently enabled tools and ask what to disable.
   ```

7. Implement `/gdev-update` (user-only, side effects):
   ```yaml
   ---
   name: gdev-update
   description: Update gdev-managed configuration files after project changes. Re-detects ecosystems, updates templates, preserves user customizations.
   disable-model-invocation: true
   allowed-tools: Bash(gdev *) Read Grep
   ---

   ## Current state
   !`gdev status --json 2>/dev/null`

   ## Recent changes
   !`git diff --name-only HEAD~5 2>/dev/null || echo "Not a git repo or no recent commits"`

   ## Instructions

   Update gdev configuration:

   1. Show what has changed since last gdev update.
   2. Run: `gdev init --update --dry-run --json` to preview changes.
   3. Show the user what would be updated.
   4. Ask for confirmation.
   5. Run: `gdev init --update`
   6. Report what was updated, preserved, and newly detected.
   7. Run `gdev devenv doctor --json` to verify health after update.
   ```

8. Implement `gdev-doctor` (Claude-invocable, read-only):
   ```yaml
   ---
   name: gdev-doctor
   description: Run gdev health checks on the current project. Reports missing prerequisites, configuration issues, and security gaps. Use when diagnosing development environment problems.
   allowed-tools: Bash(gdev *) Read
   ---

   ## System health
   !`gdev devenv doctor --json 2>/dev/null || echo '{"error": "gdev not found"}'`

   ## Instructions

   Analyze the health check results above and:

   1. Report any FAIL or WARN items clearly.
   2. For each issue, explain what it means and how to fix it.
   3. If gdev is not installed, provide installation instructions.
   4. If all checks pass, confirm the environment is healthy.
   5. Suggest `/gdev-setup` if prerequisites are missing.

   If the user has made changes since this status was captured, re-run `gdev devenv doctor --json` to get current state.
   ```

9. Implement `gdev-status` (Claude-invocable, read-only):
   ```yaml
   ---
   name: gdev-status
   description: Show current gdev configuration status. Lists enabled tools, detected ecosystems, security posture. Use when asked about development environment state.
   allowed-tools: Bash(gdev *) Read
   ---

   ## Current status
   !`gdev status --json 2>/dev/null || echo '{"error": "gdev not found"}'`

   ## Instructions

   Present the gdev status clearly:
   - Enabled tools and their health
   - Detected ecosystems
   - Security posture summary
   - Any warnings or issues

   If $ARGUMENTS specifies a particular tool or area, focus on that.
   ```

10. Implement `gdev-tools` (Claude-invocable, read-only):
    ```yaml
    ---
    name: gdev-tools
    description: List available gdev tools with descriptions and enabled/disabled state. Use when the user asks what tools are available or what can be enabled.
    allowed-tools: Bash(gdev *) Read
    ---

    ## Available tools
    !`gdev list --json 2>/dev/null || echo '{"available": []}'`

    ## Instructions

    Present the available tools organized by category:
    - Security tools (Semgrep, Gitleaks, Grype, etc.)
    - AI agent tools (semble, Version-Sentinel, etc.)
    - Developer experience tools (changelog, commitlint, etc.)
    - Infrastructure tools (CI workflows, SecretSpec, etc.)

    Show which are enabled, disabled, and their default state.
    If the user wants to enable something, suggest `/gdev-enable <tool>`.
    ```

11. Implement `gdev-detect` (Claude-invocable, read-only):
    ```yaml
    ---
    name: gdev-detect
    description: Detect project ecosystems (languages, frameworks, package managers) in the current directory. Use when understanding what a project uses.
    allowed-tools: Bash(gdev *) Read
    ---

    ## Detected ecosystems
    !`gdev detect --json 2>/dev/null || echo '{"ecosystems": []}'`

    ## Instructions

    Report what gdev detected:
    - Languages and their versions
    - Package managers and manifest files
    - Frameworks detected
    - Recommended gdev ecosystem modules

    If detection seems wrong, suggest the user check specific files or use `gdev init --ecosystem <name>` to override.
    ```

12. Register all 10 skills in the tool registry (Phase 12 lifecycle system) with category `gdev-operations`. User-only skills get `disable-model-invocation: true`. Claude-invocable skills have no such restriction.

13. Wire skill deployment into the claudecode addon's `Init()` method: extract all 10 skills from `embed.FS` to `.claude/skills/gdev-*/SKILL.md`.

14. Add skill update support: `claudecode` addon's `Update()` method replaces skill files with current gdev version (merge strategy: `Overwrite` — skills are machine-owned).

15. Write unit tests: verify each skill file is valid YAML frontmatter + markdown, verify `allowed-tools` patterns match actual gdev command patterns, verify `disable-model-invocation` is set correctly for each category.

**Acceptance Criteria:**
- [ ] 6 user-only skills deploy to `.claude/skills/gdev-{init,onboard,setup,enable,disable,update}/SKILL.md` with `disable-model-invocation: true`
- [ ] 4 Claude-invocable skills deploy to `.claude/skills/gdev-{doctor,status,tools,detect}/SKILL.md` without `disable-model-invocation`
- [ ] All skills use `` !`gdev <cmd> --json` `` dynamic context injection with `|| echo` fallbacks
- [ ] All skills use `allowed-tools: Bash(gdev *)` to scope tool access
- [ ] `/gdev-init` shows dry-run preview before executing
- [ ] `/gdev-setup` confirms before installing system packages
- [ ] Claude can autonomously invoke `gdev-doctor` when troubleshooting build failures
- [ ] Skills are registered in the tool lifecycle registry (individually toggleable)
- [ ] Skill files are embedded via `embed.FS` and versioned with gdev binary
- [ ] `gdev init --update` replaces skill files with current version

**Research Citations:**
- `research-spikes/gdev-claude-code-integration/claude-code-integration-research.md § 1-2` — SKILL.md format, frontmatter fields, dynamic context injection syntax
- `research-spikes/gdev-claude-code-integration/claude-code-integration-research.md § 3` — 10 gdev operations mapped to skills with concrete SKILL.md implementations
- `research-spikes/gdev-claude-code-integration/claude-code-integration-research.md § 6` — 5-layer safety architecture, operation classification (read-only vs side-effects)
- `research-spikes/gdev-claude-code-integration/claude-code-integration-research.md § 4.3` — dynamic state injection as strongest CLI wrapper pattern
- `research-spikes/gdev-claude-code-integration/claude-code-integration-research.md § 7.1-7.2` — embedding skills via embed.FS, skill directory layout
- `phases/04-claude-code-addon-core-generation.md § Unit 3.4` — embedded skill library deployment via embed.FS copy

**Status:** Not Started

---

### Unit 14.2: Consulting Workflow Agents

**Description:** Deploy 7 agent files (`.claude/agents/*.md`) for context-isolated analysis tasks. Agents encode domain expertise (security review, codebase exploration, test coverage analysis) and run in their own context windows, keeping the main conversation uncluttered. Each agent has YAML frontmatter controlling tools, model, permissions, and memory.

**Context:** Claude Code agent files are markdown files with YAML frontmatter that define specialized subagents with their own context windows, tool access, permission modes, and persistent memory. The Trail of Bits principle — "Encode expertise in agents, procedures in skills" — drives the split: agents are specialized personas for analysis work, skills are repeatable procedures for task execution. Key architectural constraint: subagents cannot spawn other subagents. Agents cost zero main-context tokens (they run in isolated context windows). The research spike `gdev-agentic-workflows` produced complete agent definitions ready for embedding.

**Code-Grounded Notes:** ZERO agent infrastructure currently exists in the codebase. There are no `.claude/agents/` templates, no agent manifest file, and no deployment function. This unit must create the entire agent deployment pipeline from scratch, modeled on the existing skill deployment in `generate_skills.go`. Specifically: create `addons/claudecode/templates/agents/` with a `manifest.yaml`, create a `deployAgents()` function analogous to `deploySkills()`, and wire it into the addon's Init/Update paths. Agent files should use the LibraryManaged merge strategy (always overwritten on update), matching how skills are currently handled.

**Desired Outcome:** After `gdev init`, developers find 7 agents in `.claude/agents/`. Claude can delegate to these agents when the task matches (e.g., "review this PR for security issues" auto-delegates to `security-reviewer`). Each agent produces structured output suitable for client deliverables or team handoff.

**Steps:**

1. Create agent file directory in the embedded library:
   ```
   internal/claudecode/agents/
   ├── security-reviewer.md
   ├── codebase-explorer.md
   ├── test-gap-analyzer.md
   ├── onboarding-guide.md
   ├── migration-planner.md
   ├── handoff-doc-generator.md
   └── incident-debugger.md
   ```

2. Implement `security-reviewer.md`:
   ```yaml
   ---
   name: security-reviewer
   description: Security-focused code review specialist. Checks for injection vulnerabilities, auth flaws, secrets exposure, and OWASP Top 10 issues. Use proactively after code changes or when reviewing PRs.
   tools: Read, Grep, Glob, Bash
   disallowedTools: Write, Edit
   model: inherit
   permissionMode: default
   maxTurns: 50
   memory: project
   ---

   You are a senior security engineer conducting a security-focused code review.

   ## Review Process

   1. Run `git diff --name-only HEAD~1` to identify changed files.
   2. For each changed file, analyze for:
      - Injection vulnerabilities (SQL, XSS, command injection, path traversal)
      - Authentication and authorization flaws
      - Secrets or credentials in code or config
      - Insecure data handling (PII exposure, missing encryption)
      - Missing input validation
      - Insecure deserialization
      - SSRF vectors
      - Race conditions in concurrent code
   3. Check dependency changes against known vulnerabilities.
   4. Review configuration changes for security implications.

   ## Output Format

   Organize findings by severity:

   ### Critical (must fix before merge)
   - [file:line] Description of vulnerability
   - Impact: What an attacker could achieve
   - Fix: Specific code change to remediate

   ### High (should fix before merge)
   - [file:line] Description — Impact — Fix

   ### Medium (fix in next sprint)
   - [file:line] Description — Impact — Fix

   ### Informational (consider improving)
   - [file:line] Description — Suggestion

   ## Memory Management
   After each review, update your agent memory with:
   - Recurring vulnerability patterns in this codebase
   - Security-sensitive files and modules identified
   - Authentication/authorization architecture notes
   ```

3. Implement `codebase-explorer.md`:
   ```yaml
   ---
   name: codebase-explorer
   description: Rapid codebase exploration specialist. Provides architectural overview, key patterns, and navigation guidance. Use when onboarding to a new project or understanding unfamiliar code.
   tools: Read, Grep, Glob, Bash
   disallowedTools: Write, Edit
   model: haiku
   permissionMode: default
   maxTurns: 30
   memory: project
   ---

   You are a codebase exploration specialist helping an engineer understand a new codebase quickly.

   ## Exploration Process

   1. **Structure**: Map the top-level directory structure and identify source code, tests, config, docs, build/deploy files.

   2. **Architecture**: Identify entry points, routing/dispatch, business logic, data access, and external integration layers.

   3. **Patterns**: Catalog frameworks and versions, ORM/database access, auth approach, error handling, logging, and dependency injection patterns.

   4. **Key Files**: Identify the 10 most important files to read first.

   5. **Gotchas**: Note anything surprising, non-standard, or potentially confusing.

   ## Output

   Provide a structured onboarding guide optimized for a consulting engineer who has 30 minutes to understand this codebase well enough to contribute.

   ## Memory
   After exploration, save to agent memory:
   - Architecture diagram (text-based)
   - Key file index
   - Technology stack summary
   - Non-obvious conventions
   ```

4. Implement `test-gap-analyzer.md`:
   ```yaml
   ---
   name: test-gap-analyzer
   description: Identifies untested code paths and generates test recommendations. Analyzes coverage gaps by comparing source modules to test files. Use when improving test coverage or auditing testing completeness.
   tools: Read, Grep, Glob, Bash
   disallowedTools: Write, Edit
   model: inherit
   permissionMode: default
   maxTurns: 40
   ---

   You are a test coverage analyst. Your job is to find code that lacks test coverage and recommend what tests to write.

   ## Analysis Process

   1. Identify the project's test framework and test directory structure.
   2. For each source module, find its corresponding test file (if any).
   3. For modules without tests, analyze complexity to prioritize:
      - Public API surface (exported functions/classes)
      - Error handling paths
      - Branching logic (if/else, switch)
      - Data transformation functions
      - Integration points (DB, API, filesystem)
   4. For modules with tests, identify gaps:
      - Untested public functions
      - Missing edge case coverage (null, empty, boundary)
      - Missing error path coverage
      - Missing integration test coverage

   ## Output Format

   ### Coverage Summary
   | Module | Has Tests | Public Functions | Tested | Coverage Gap |
   |--------|-----------|-----------------|--------|-------------|

   ### Priority Test Recommendations
   1. [HIGH] `module.function()` - No tests. Handles user input validation.
   2. [HIGH] `module.process()` - Error path untested. Could crash on null input.
   3. [MED] `module.transform()` - Edge cases missing. Empty array not tested.
   ```

5. Implement `onboarding-guide.md`:
   ```yaml
   ---
   name: onboarding-guide
   description: Guide a new engineer through understanding a codebase. Interactive exploration with Q&A, structured notes, and progressive depth. Use when a new team member needs to get up to speed.
   tools: Read, Grep, Glob, Bash
   disallowedTools: Write, Edit
   model: inherit
   permissionMode: default
   maxTurns: 60
   memory: project
   ---

   You are a patient, thorough technical mentor helping a new engineer understand this codebase.

   ## Approach

   Guide the engineer through the codebase in a structured way:

   1. **Start with the big picture**: What does this system do? Who uses it? What problem does it solve?
   2. **Architecture tour**: Walk through the high-level components, their responsibilities, and how they communicate.
   3. **Build & run**: Help them build, test, and run the project locally.
   4. **First contribution area**: Identify the easiest area to make a first change.
   5. **Deep dives**: Based on their questions, explore specific modules in detail.

   ## Guidelines
   - Answer questions by reading actual code, not guessing
   - When explaining a module, show the key functions and data flow
   - Point out non-obvious conventions and gotchas
   - Track what has been explored in agent memory for continuity across sessions

   ## Memory
   Save to agent memory after each session:
   - Topics covered and depth reached
   - Questions the engineer asked (reveals knowledge gaps)
   - Areas not yet explored
   ```

6. Implement `migration-planner.md`:
   ```yaml
   ---
   name: migration-planner
   description: Plan framework or library upgrades with risk assessment. Analyzes codebase impact, identifies breaking changes, produces phased migration plans. Use when planning major dependency or framework upgrades.
   tools: Read, Grep, Glob, Bash
   disallowedTools: Write, Edit
   model: inherit
   permissionMode: default
   maxTurns: 40
   ---

   You are a migration planning specialist. Your job is to analyze a codebase and produce a detailed, phased migration plan for a framework or library upgrade.

   ## Analysis Process

   1. **Scope Assessment**: Count affected files, map dependency chains, identify parallel migration paths, assess risk profile.
   2. **Breaking Change Research**: Check changelogs between current and target versions. List every breaking change, deprecation, and new requirement.
   3. **Impact Mapping**: For each breaking change, identify which files and modules are affected.
   4. **Risk Assessment**: Classify each change as high/medium/low risk based on blast radius and test coverage.

   ## Output Format

   ### Migration Plan: [from] → [to]

   **Estimated Effort**: Based on codebase analysis
   **Risk Level**: High / Medium / Low
   **Recommended Approach**: Big bang / Strangler fig / Parallel run

   ### Phase 1: Preparation (no behavior change)
   - Set up compatibility layers
   - Add test coverage for affected areas
   - Rollback: None needed

   ### Phase 2: Incremental Migration
   - Migrate low-risk areas first
   - Verification steps per area
   - Rollback: Revert to Phase 1

   ### Phase 3: Cleanup
   - Remove compatibility layers
   - Update documentation
   - Rollback: Keep compat layers

   ### Risks
   | Risk | Likelihood | Impact | Mitigation |
   ```

7. Implement `handoff-doc-generator.md`:
   ```yaml
   ---
   name: handoff-doc-generator
   description: Generate comprehensive engagement handoff documentation. Synthesizes architecture, decisions, operations, and known issues into a client-facing deliverable. Use at the end of a consulting engagement.
   tools: Read, Grep, Glob, Bash
   disallowedTools: Write, Edit
   model: inherit
   permissionMode: default
   maxTurns: 50
   memory: project
   ---

   You are a technical writer producing a client handoff document for a consulting engagement.

   ## Document Structure

   Generate a comprehensive handoff covering:

   ### 1. Executive Summary
   What was built, why, key decisions, current state.

   ### 2. Architecture Overview
   System diagram (text-based), component responsibilities, data flow, external integrations.

   ### 3. Technology Stack
   Languages, frameworks, versions, why each was chosen, known limitations.

   ### 4. Development Guide
   Environment setup, build/test/deploy commands, code organization, conventions.

   ### 5. Operations Guide
   Deployment steps, health monitoring, incident response, backup/recovery.

   ### 6. Decisions Log
   Reference existing ADRs, summarize undocumented decisions.

   ### 7. Known Issues and Technical Debt
   Current bugs, workarounds, prioritized tech debt, planned improvements not completed.

   ### 8. Recommended Next Steps
   What to do after engagement ends, suggested improvements, areas needing attention.

   ## Sources
   Pull from: ADRs, runbooks, README, CLAUDE.md, git log, package manifests, CI configs.

   ## Style
   Write for the client's engineering team — no internal jargon, clear explanations, actionable guidance.
   ```

8. Implement `incident-debugger.md`:
   ```yaml
   ---
   name: incident-debugger
   description: Systematic production incident debugging specialist. Follows hypothesis-test-conclude methodology. Use when investigating production errors, outages, or performance degradation.
   tools: Read, Grep, Glob, Bash
   model: inherit
   permissionMode: default
   maxTurns: 40
   memory: project
   ---

   You are a production incident debugger following a structured hypothesis-test-conclude methodology.

   ## Debugging Process

   ### 1. Triage
   - When did the issue start? Check recent deployments, config changes.
   - What is the blast radius? Which users, endpoints, services affected.
   - Is there an obvious recent change? `git log --oneline -20`

   ### 2. Hypothesize
   Generate 3-5 ranked hypotheses:
   1. [Most likely] — Evidence needed: [what to check]
   2. [Second likely] — Evidence needed: [what to check]
   3. [Less likely, high impact] — Evidence needed: [what to check]

   ### 3. Test Each Hypothesis
   For each, starting with most likely:
   - State what you're checking and why
   - Execute the check (read logs, inspect code, trace data flow)
   - Record: CONFIRMED / REFUTED / INCONCLUSIVE
   - If CONFIRMED: proceed to root cause
   - If all REFUTED: generate new hypotheses

   ### 4. Root Cause
   - What happened (specific technical description)
   - Why it happened (underlying cause)
   - When it started (timestamp and trigger)
   - What's affected (scope)

   ### 5. Remediation Options
   Ranked by speed vs completeness:
   - Immediate (minutes): quickest fix, possibly temporary
   - Short-term (hours): proper fix for this instance
   - Long-term (days): prevent recurrence

   ## Important
   - DO NOT make production changes without user approval
   - Present options, let the user decide
   - If unsure, say so — do not guess at root causes

   ## Memory
   After each incident, save to agent memory:
   - Root cause and fix applied
   - Similar symptoms to watch for
   - System components involved
   ```

9. Register all 7 agents in the tool registry with category `consulting-agents`. Agents use merge strategy `Overwrite` (machine-managed).

10. Wire agent deployment into claudecode addon's `Init()` method: extract agent files from `embed.FS` to `.claude/agents/`.

11. Write unit tests: verify each agent file has valid YAML frontmatter with required fields (`name`, `description`), verify `disallowedTools` is set on read-only agents, verify no agent specifies `Write` or `Edit` in its `tools` list when it's analysis-only.

**Acceptance Criteria:**
- [ ] 7 agent files deploy to `.claude/agents/{security-reviewer,codebase-explorer,test-gap-analyzer,onboarding-guide,migration-planner,handoff-doc-generator,incident-debugger}.md`
- [ ] `security-reviewer` has `disallowedTools: Write, Edit` (cannot modify code during review)
- [ ] `codebase-explorer` uses `model: haiku` for fast, cheap exploration
- [ ] Agents with `memory: project` create persistent memory in `.claude/agent-memory/<name>/`
- [ ] Agent descriptions are concise and keyword-rich for accurate auto-delegation
- [ ] No agent attempts to spawn subagents (architectural constraint documented)
- [ ] All agents produce structured output suitable for client deliverables
- [ ] Agents are registered in the tool lifecycle registry (individually toggleable via `gdev enable/disable`)
- [ ] `gdev disable security-reviewer` removes the agent file cleanly

**Research Citations:**
- `research-spikes/gdev-agentic-workflows/agent-files-research.md § 1` — agent file format, 15 frontmatter fields, priority ordering
- `research-spikes/gdev-agentic-workflows/agent-files-research.md § 2` — agents vs skills vs commands comparison, Trail of Bits principle
- `research-spikes/gdev-agentic-workflows/agent-files-research.md § 4` — persistent memory system (user, project, local scopes)
- `research-spikes/gdev-agentic-workflows/agent-files-research.md § 5` — recommended agents for consulting (security-reviewer, codebase-explorer, test-gap-analyzer)
- `research-spikes/gdev-agentic-workflows/workflow-skills-catalog-research.md § Category 1, 3, 5, 6` — complete agent definitions (security-reviewer, test-gap-analyzer, codebase-explorer, incident-investigator)
- `research-spikes/gdev-agentic-workflows/consulting-differentiation-research.md § 1` — unfamiliar codebases as default, onboarding as highest-frequency workflow

**Status:** Not Started

---

### Unit 14.3: Consulting Workflow Skills

**Description:** Deploy 8+ procedural skills for common consulting tasks: PR review, test generation, dependency upgrades, codebase onboarding, ADR writing, incident debugging, migration planning, and handoff documentation. Each skill includes per-ecosystem verification commands, structured checklists, and self-verification gates.

**Context:** Skills (`.claude/skills/<name>/SKILL.md`) are repeatable procedures that run in the main conversation context with `disable-model-invocation: true` to require explicit user invocation. Unlike agents (isolated context, analysis focus), skills share context with the main conversation and orchestrate multi-step workflows with side effects. The research spike `gdev-agentic-workflows` produced complete skill definitions with per-ecosystem verification commands drawn from the `EcosystemModule.VerificationCommands()` interface (Phase 11 Unit 11.5). The consulting differentiation research identified onboarding, handoff documentation, and time-aware design as key differentiators.

**Desired Outcome:** After `gdev init`, developers have 8+ consulting workflow skills available via `/slash-command`. Each skill produces structured, client-deliverable output with built-in verification steps. Skills use per-ecosystem verification commands appropriate to the detected project type.

**Steps:**

1. Create skill directories in the embedded library:
   ```
   internal/claudecode/skills/
   ├── review-pr/SKILL.md
   ├── add-tests/SKILL.md
   ├── upgrade-dep/SKILL.md
   ├── onboard-me/SKILL.md
   ├── write-adr/SKILL.md
   ├── incident-debug/SKILL.md
   ├── migration-plan/SKILL.md
   └── handoff-doc/SKILL.md
   ```

2. Implement `/review-pr`:
   ```yaml
   ---
   name: review-pr
   description: Comprehensive PR review across security, performance, and code quality. Use when reviewing pull requests or preparing code for merge.
   disable-model-invocation: true
   allowed-tools: Bash(git *) Bash(gh *) Read Grep Glob
   arguments: [pr-number]
   ---

   # PR Review: $pr-number

   ## Context
   !`gh pr view $0 --json title,body,additions,deletions,changedFiles,baseRefName,headRefName 2>/dev/null || echo "PR not found or gh not authenticated"`
   !`gh pr diff $0 --name-only 2>/dev/null || echo "Cannot fetch PR diff"`

   ## Review Dimensions

   Evaluate the PR proportionally across:

   ### Code Quality
   - Readability, structure, naming
   - Duplicated patterns that should be extracted
   - Comprehensive error handling
   - Unnecessary changes (formatting-only, unrelated refactors)

   ### Security
   - Injection risks (SQL, XSS, command)
   - Input validation on user-facing inputs
   - Secrets or credentials exposed
   - Auth/authz checks present where needed
   - Dependency updates checked for known vulns

   ### Performance
   - N+1 query patterns
   - Unnecessary allocations in hot paths
   - Appropriate index usage
   - Caching opportunities

   ### Testing
   - New code paths covered
   - Edge cases tested (null, empty, boundary)
   - Error paths tested
   - Run the test suite to confirm: `{{verification_test_cmd}}`

   ### Maintainability
   - Consistency with existing patterns
   - Documentation updated
   - Breaking changes communicated

   ## Synthesis

   **Verdict**: APPROVE / REQUEST CHANGES / NEEDS DISCUSSION

   **Summary**: 2-3 sentences on what this PR does and quality.

   **Blocking Issues** (must fix):
   **Suggestions** (should consider):
   **Praise** (well done):
   ```

3. Implement `/add-tests`:
   ```yaml
   ---
   name: add-tests
   description: Generate tests for uncovered code. Follows existing test patterns in the codebase. Use when adding test coverage to a module or function.
   disable-model-invocation: true
   allowed-tools: Bash(*) Read Write Edit Grep Glob
   arguments: [target]
   ---

   # Add Tests for $target

   ## Step 1: Understand Test Patterns
   Find existing test files and understand:
   - Test framework (detect from project config)
   - File naming convention (*.test.ts, *_test.go, test_*.py)
   - Directory structure (co-located, separate test dir)
   - Common patterns (describe/it, table-driven, test classes)
   - Mock/stub patterns used

   ## Step 2: Analyze Target
   Read $target and identify:
   - All public functions/methods and their signatures
   - Input types and valid ranges
   - Error conditions and error types
   - Side effects (DB, filesystem, network, state mutation)
   - Dependencies that need mocking

   ## Step 3: Generate Tests
   Write tests following existing project patterns:
   - **Happy path**: Normal inputs, expected outputs
   - **Edge cases**: Empty, boundary, null/undefined
   - **Error paths**: Invalid inputs, external failures
   - **Type safety**: Verify type constraints (typed languages)

   ## Step 4: Verify
   1. Run the new tests — they must pass
   2. Run the full test suite — no regressions
   3. Report: tests added, any gaps intentionally skipped

   ## Guidelines
   - Match existing test style exactly
   - Descriptive test names explaining the scenario
   - One assertion per test when feasible
   - Mock external dependencies, not internal functions
   - Do not test private/internal functions directly
   ```

4. Implement `/upgrade-dep`:
   ```yaml
   ---
   name: upgrade-dep
   description: Upgrade a dependency to a new version with verification. Checks changelogs, identifies breaking changes, updates code, validates tests.
   disable-model-invocation: true
   allowed-tools: Bash(*) Read Write Edit Grep Glob
   arguments: [package, target-version]
   ---

   # Upgrade $package to $target-version

   ## Step 1: Current State
   - Find current version in dependency files
   - Record all files importing/using $package
   - Run tests and confirm baseline passes

   ## Step 2: Research Breaking Changes
   - Check changelog between current and target version
   - Identify breaking changes, deprecations, new requirements
   - List migration steps from official migration guide

   ## Step 3: Plan
   Present a migration plan:
   1. Update dependency version in manifest
   2. Address each breaking change (list specific code changes)
   3. Update configuration if needed
   4. Update imports if API surface changed

   Ask user to approve before proceeding.

   ## Step 4: Execute
   For each planned change:
   1. Make the change
   2. Run the build
   3. Fix type errors or build failures
   4. Run tests
   5. If tests fail: analyze, fix, re-run

   ## Step 5: Verify
   1. Full test suite passes
   2. Linter passes
   3. No deprecation warnings
   4. Summary of changes and what needs manual testing

   ## Important
   - If unsure about a code change, STOP and ask
   - Document behavior changes in PR description
   - List what needs manual verification if tests can't fully cover it
   ```

5. Implement `/onboard-me`:
   ```yaml
   ---
   name: onboard-me
   description: Systematic codebase onboarding. Explores architecture, patterns, conventions, key files. Produces a structured orientation document.
   disable-model-invocation: true
   context: fork
   agent: codebase-explorer
   ---

   Perform a comprehensive onboarding exploration of this codebase.

   Produce a document covering:
   1. **Technology Stack**: Languages, frameworks, key dependencies with versions
   2. **Architecture**: High-level component diagram, data flow, key abstractions
   3. **Build & Test**: How to build, test, lint, and run locally
   4. **Directory Guide**: What lives where, key files to read first
   5. **Patterns & Conventions**: Coding style, naming, error handling, logging
   6. **Data Model**: Key entities and their relationships
   7. **External Integrations**: APIs, databases, queues, caches
   8. **Gotchas**: Non-obvious behaviors, known issues, technical debt
   9. **Key Contributors**: From git log, who owns what

   Write the output to `docs/ONBOARDING.md`.

   $ARGUMENTS
   ```

6. Implement `/write-adr`:
   ```yaml
   ---
   name: write-adr
   description: Generate an Architecture Decision Record (ADR) in MADR format. Use when documenting why a technical choice was made.
   disable-model-invocation: true
   allowed-tools: Read Write Edit Grep Glob Bash(ls *) Bash(find *) Bash(git log *)
   arguments: [decision-title]
   ---

   # ADR: $decision-title

   ## Step 1: Gather Context
   - Recent commits related to the topic
   - Configuration changes
   - Dependency additions/removals
   - Architecture patterns in use

   ## Step 2: Interview
   Ask the user:
   1. What problem prompted this decision?
   2. What alternatives were considered?
   3. Key constraints (time, budget, team skills, compliance)?
   4. What was the deciding factor?

   ## Step 3: Generate ADR
   Write to `docs/adr/NNNN-$decision-title.md` (increment based on existing ADRs):

   # NNNN. $decision-title

   ## Status: Accepted
   ## Date: YYYY-MM-DD
   ## Context: [forces at play]
   ## Decision: [what we chose]
   ## Consequences
   ### Positive / Negative / Neutral
   ## Alternatives Considered
   ### Alternative N: [name] — Pros / Cons / Why rejected

   ## Step 4: Verify
   - ADR consistent with observable codebase evidence
   - Alternatives are genuinely different approaches
   - Consequences are concrete and actionable
   ```

7. Implement `/incident-debug`:
   ```yaml
   ---
   name: incident-debug
   description: Systematic production incident debugging. Follows hypothesis-test-conclude loop. Use when investigating production errors or performance degradation.
   disable-model-invocation: true
   allowed-tools: Bash(*) Read Grep Glob
   arguments: [symptom]
   ---

   # Incident Debug: $symptom

   ## Step 1: Triage
   - When did the issue start? Check recent deployments, config changes.
   - What is the blast radius? Which users, endpoints, services.
   - Obvious recent change? `git log --oneline -20`

   ## Step 2: Hypothesize
   Generate 3-5 ranked hypotheses:
   1. [Most likely] — Evidence needed: [check]
   2. [Second likely] — Evidence needed: [check]
   3. [Less likely, high impact] — Evidence needed: [check]

   ## Step 3: Test Each Hypothesis
   For each, starting with most likely:
   1. State what you're checking and why
   2. Execute the check
   3. Record: CONFIRMED / REFUTED / INCONCLUSIVE
   4. If CONFIRMED: proceed to root cause
   5. If all REFUTED: broaden search

   ## Step 4: Root Cause
   - **What happened**: specific technical description
   - **Why**: underlying cause
   - **When**: timestamp and trigger
   - **Scope**: what's affected

   ## Step 5: Remediation Options
   - Immediate (minutes): quickest fix
   - Short-term (hours): proper fix
   - Long-term (days): prevent recurrence

   ## Step 6: Verification
   After fix: confirm symptom resolved, check for side effects, set up monitoring.

   ## Important
   - DO NOT make production changes without user approval
   - If unsure, say so — don't guess at root causes
   ```

8. Implement `/migration-plan`:
   ```yaml
   ---
   name: migration-plan
   description: Generate a phased migration plan for framework, library, or architecture changes. Produces executable plan with phases, dependencies, and rollback strategies.
   disable-model-invocation: true
   allowed-tools: Read Grep Glob Bash(git *) Bash(find *) Bash(wc *)
   arguments: [migration-description]
   ---

   # Migration Plan: $migration-description

   ## Step 1: Scope Assessment
   - How many files affected?
   - Dependency chains?
   - Parallel migration paths?
   - Risk profile (critical paths vs low-risk)?

   ## Step 2: Interview
   Ask the user:
   1. Timeline for this migration?
   2. Can old and new coexist (strangler fig)?
   3. Compliance or deployment constraints?
   4. Rollback strategy requirements?

   ## Step 3: Generate Plan
   Write to `docs/migrations/$migration-description.md`:

   ### Overview
   - Goal, current state, estimated effort, risk level

   ### Phase 1: Preparation (no behavior change)
   - Set up parallel infrastructure, add compatibility layers, ensure test coverage
   - Rollback: None needed

   ### Phase 2: Incremental Migration
   - Migrate low-risk areas first, then higher-risk
   - Rollback: Revert to Phase 1

   ### Phase 3: Cleanup
   - Remove compat layers, old code, update docs
   - Rollback: Keep compat layers

   ### Dependencies, Risks, Verification per Phase
   ```

9. Implement `/handoff-doc`:
   ```yaml
   ---
   name: handoff-doc
   description: Generate comprehensive client handoff documentation. Synthesizes ADRs, runbooks, architecture, and maintenance guidance into a single deliverable. Use at the end of a consulting engagement.
   disable-model-invocation: true
   context: fork
   agent: handoff-doc-generator
   ---

   Generate a comprehensive client handoff document for this project.

   Pull information from:
   - Existing ADRs in docs/adr/
   - Existing runbooks in docs/runbooks/
   - README.md and CLAUDE.md
   - Git log for recent changes and contributors
   - Package manifests for dependency information
   - CI configuration for deployment details

   Write the output to `docs/HANDOFF.md`.

   $ARGUMENTS
   ```

10. Template skill verification commands per ecosystem. During `gdev init`, the claudecode addon reads `EcosystemModule.VerificationCommands()` and injects the appropriate build/test/lint commands into skill templates where `{{verification_test_cmd}}` placeholders appear. For example:
    - Go project: `go test ./...`
    - Node project: `npm test`
    - Python project: `pytest`
    - Rust project: `cargo test`
    - Multi-ecosystem: combined command list

11. Register all 8 skills in the tool registry with category `consulting-workflows`. All use `disable-model-invocation: true`. Wire deployment into `Init()`.

12. Write unit tests: verify skill file validity, verify `disable-model-invocation: true` on all skills, verify `allowed-tools` don't conflict with deny rules (see Unit 14.5).

**Acceptance Criteria:**
- [ ] 8 skill files deploy to `.claude/skills/{review-pr,add-tests,upgrade-dep,onboard-me,write-adr,incident-debug,migration-plan,handoff-doc}/SKILL.md`
- [ ] All consulting skills have `disable-model-invocation: true` (user must explicitly invoke)
- [ ] `/review-pr` uses `Bash(gh *)` and `Bash(git *)` for PR context gathering
- [ ] `/add-tests` runs per-ecosystem test command after generating tests
- [ ] `/upgrade-dep` works with guardrail hooks (hook validates package, doesn't bypass)
- [ ] `/onboard-me` delegates to `codebase-explorer` agent via `context: fork` + `agent: codebase-explorer`
- [ ] `/handoff-doc` delegates to `handoff-doc-generator` agent
- [ ] Per-ecosystem verification commands injected during `gdev init` based on detected ecosystems
- [ ] All skills are individually toggleable via `gdev enable/disable`

**Research Citations:**
- `research-spikes/gdev-agentic-workflows/workflow-skills-catalog-research.md` — complete skill definitions for all 7 categories (review, testing, documentation, incident, onboarding, migration, refactoring)
- `research-spikes/gdev-agentic-workflows/agentic-task-templates-research.md` — checklist-driven execution, evidence-anchored findings, self-verification gates
- `research-spikes/gdev-agentic-workflows/consulting-differentiation-research.md § 1, 3, 4` — onboarding-first, time-aware design, mandatory handoff documentation
- `research-spikes/gdev-agentic-workflows/guardrail-integration-research.md § 2.4` — `/upgrade-dep` works WITH guardrail hooks, not around them
- `phases/11-ai-agent-tooling-integration.md § Unit 11.5` — `EcosystemModule.VerificationCommands()` interface for per-ecosystem commands

**Status:** Not Started

---

### Unit 14.4: Context Budget Management

**Description:** Implement model-aware context budget management ensuring gdev's generated configuration (CLAUDE.md, rules, skill descriptions, agent definitions) stays under 5% of the model's context window. Generate different configurations for Sonnet (200K context, ~10K char budget) and Opus (1M context, ~50K char budget).

**Context:** Claude Code's context window fills fast — community data suggests performance degradation begins at ~40% utilization. Everything gdev generates (CLAUDE.md content, rules files, skill descriptions) consumes context tokens every session. The research spike `gdev-agentic-workflows/context-management-research.md` established a five-layer context architecture: minimal always-on CLAUDE.md (50-100 lines), conditional `.claude/rules/` with `paths:` frontmatter, on-demand skill loading, isolated agent context (zero main-context cost), and zero-cost external hooks/settings. The target is 5% of context window: ~10,000 tokens for Sonnet, ~50,000 tokens for Opus.

**Code-Grounded Notes:** The current CLAUDE.md template is 77 lines (`templates/claude-md.tmpl`). With language conventions (~15-17 lines each for up to 8 languages), a typical project generates ~200-300 lines. This is within the 5% budget for both Sonnet (200K context window → ~350 lines at 5%) and Opus (1M context window → ~1,750 lines at 5%). Current section markers are a SINGLE pair: `<!-- BEGIN GENERATED SECTION -->` / `<!-- END GENERATED SECTION -->`. The merge function `merge.SectionMarkers()` at `internal/merge/section.go:24-76` handles only ONE section. Phase 14 must either: (a) extend SectionMarkers to support multiple named sections, OR (b) keep a single generated section with structured subsections inside it. Approach (b) is recommended as simpler — add `<!-- gdev:skills -->` etc. as NESTED markers within the existing generated section, with a new `merge.NamedSections()` function that operates within the generated block.

**Desired Outcome:** `gdev init` generates context-efficient configuration. CLAUDE.md is 50-100 lines of concise reference. Ecosystem-specific rules load lazily via `paths:` frontmatter. Skill descriptions are keyword-rich and under 200 characters each. `gdev check` validates the context budget. Model selection in the wizard adjusts generation.

**Steps:**

1. Define `ContextBudget` tracking struct:
   ```go
   type ContextBudget struct {
       ClaudeMDLines    int     // lines in CLAUDE.md
       ClaudeMDChars    int     // characters in CLAUDE.md
       RulesCount       int     // number of rules files
       RulesTotalChars  int     // total characters across all unconditional rules
       SkillDescChars   int     // total characters of all skill descriptions
       AgentDescChars   int     // total characters of all agent descriptions
       EstTokens        int     // estimated total tokens (chars / 4)
       ModelContextSize int     // target model context (200000 or 1000000)
       BudgetPct        float64 // estimated % of context window
   }
   ```

2. Implement `ContextBudget.Validate(modelSize int) error` — returns error if budget exceeds 5% threshold. Include breakdown of what's consuming context.

3. Add model selection to wizard: "Target model: Sonnet (200K) / Opus (1M) / Auto-detect". Default: Auto-detect (check `claude --version` or env vars if available, fall back to Sonnet for conservative default).

4. Generate model-aware CLAUDE.md:
   - **Sonnet (200K)**: 50 lines max. Build/test/lint commands only. Available skills list (names only, no descriptions). Security policy summary. `@`-imports for rules.
   - **Opus (1M)**: 100 lines max. Build/test/lint commands. Architecture overview paragraph. Available skills with one-line descriptions. Security policy. Conventions summary. `@`-imports for detailed references.

5. Generate `.claude/rules/*.md` with `paths:` frontmatter for conditional loading:
   ```yaml
   ---
   paths: "**/*.go"
   ---
   # Go Conventions
   - Use table-driven tests
   - Handle all errors explicitly (no _ = err)
   - Use context.Context as first parameter
   ```
   Each ecosystem module contributes its rules file. Rules without `paths:` (security rules) load unconditionally — keep these under 30 lines.

6. Implement model-aware skill description management:
   - **Sonnet**: Set `disable-model-invocation: true` on consulting workflow skills (remove descriptions from context). Keep only gdev diagnostic skills auto-invocable (4 descriptions × ~200 chars = ~800 chars).
   - **Opus**: All skills keep descriptions. Total: ~18 skills × ~200 chars = ~3,600 chars (well within 1% budget).

7. Implement `@`-import pattern for detailed reference docs:
   ```markdown
   ## Development Environment
   @.claude/gdev-reference.md
   ```
   Where `.claude/gdev-reference.md` is a gdev-generated file with full command docs, tool status, and project-specific information. Loaded by `@`-import, not inline in CLAUDE.md.

8. Add `gdev check --context-budget` subcommand: scans generated files, calculates `ContextBudget`, reports per-component breakdown, warns if over 5%.

9. Wire budget validation into `gdev init` and `gdev update`: after generation, validate budget and warn if exceeded.

10. Write unit tests: verify Sonnet CLAUDE.md is <=50 lines, Opus <=100 lines. Verify all skill descriptions are <=200 chars. Verify rules files have `paths:` frontmatter (except security rules). Verify budget calculation is accurate.

**Acceptance Criteria:**
- [ ] CLAUDE.md generated for Sonnet is <=50 lines (~2KB)
- [ ] CLAUDE.md generated for Opus is <=100 lines (~4KB)
- [ ] All `.claude/rules/*.md` for ecosystem conventions use `paths:` frontmatter (lazy loading)
- [ ] Security rules load unconditionally (no `paths:` frontmatter) and are <=30 lines
- [ ] Each skill description is <=200 characters and keyword-rich
- [ ] Total generated config is <5% of target model context window
- [ ] `gdev check --context-budget` reports per-component breakdown
- [ ] Model selection in wizard adjusts generation (Sonnet vs Opus)
- [ ] Sonnet config disables model-invocation on consulting skills (reduces description budget)
- [ ] `@`-import pattern used for detailed gdev reference doc (not inline in CLAUDE.md)
- [ ] Budget validation runs automatically during `gdev init` and `gdev update`

**Research Citations:**
- `research-spikes/gdev-agentic-workflows/context-management-research.md § 1` — token budget analysis, 5% target, Sonnet 10K chars / Opus 50K chars
- `research-spikes/gdev-agentic-workflows/context-management-research.md § 2` — five-layer context architecture (always-on, conditional, on-demand, isolated, external)
- `research-spikes/gdev-agentic-workflows/context-management-research.md § 3` — skill description budget management, progressive disclosure, dynamic context injection
- `research-spikes/gdev-agentic-workflows/context-management-research.md § 4` — anti-patterns (kitchen sink CLAUDE.md, verbose descriptions, auto-invoking all skills)
- `research-spikes/gdev-agentic-workflows/context-management-research.md § 5` — ContextBudget Go struct, model-aware generation table
- `research-spikes/gdev-claude-code-integration/claude-code-integration-research.md § 5.2-5.4` — CLAUDE.md section design, section markers, @-import pattern

**Status:** Not Started

---

### Unit 14.5: Deny Rule Conflict Validation

**Description:** Implement a test matrix validating that generated deny rules don't block operations required by generated skills. Deny rules must block `npm install *`, `pip install *`, etc. but NOT `npm test`, `npm run build`, etc. The validation runs during `gdev init`, `gdev update`, and `gdev check` to catch conflicts before they cause runtime failures.

**Context:** The core tension identified in the guardrails research: security deny rules and workflow skills share the same tool namespace. A deny rule `Bash(npm install *)` correctly blocks unauthorized package installs, but `Bash(npm *)` would also block `npm test` — breaking the `/review-pr` and `/add-tests` skills that need to run tests. The research established a precision-scoping principle: deny rules target specific dangerous operations, not broad tool categories. The conflict detection test from `guardrail-integration-research.md § 4` provides the Go implementation pattern. Additionally, skills that invoke package managers (like `/upgrade-dep`) should work WITH guardrail hooks (the hook validates the package being installed), not bypass them.

**Code-Grounded Notes:** Deny rules are defined as string slices in `addons/claudecode/generate_settings.go:54-291` across 13 categories (~120 rules total). The `allBaseDenyRules()` function concatenates all categories. `collectEcosystemDenyRules()` pulls additional rules from module `DenyRules()` methods. There is NO existing conflict detection code — Phase 14 builds it entirely from scratch. Rules use patterns like `Bash(npm install *)`, `Read(./.env)`. The pattern matching logic for detecting conflicts (glob-style matching of deny rules against skill allowed-tools) must be purpose-built.

**Desired Outcome:** `gdev init` validates all deny rule × skill operation combinations and fails with clear diagnostics if conflicts exist. A regression test suite prevents future deny rules from blocking legitimate skill operations. `gdev check --deny-rules` is available as a standalone validation command.

**Steps:**

1. Define the conflict detection function:
   ```go
   type DenyRuleConflict struct {
       Skill     string // Skill name (e.g., "review-pr")
       Operation string // Operation the skill needs (e.g., "Bash(npm test *)")
       DenyRule  string // Deny rule that would block it (e.g., "Bash(npm *)")
       Message   string // Human-readable explanation
   }

   func ValidateDenyRuleConflicts(
       denyRules []string,
       skills []SkillDefinition,
   ) []DenyRuleConflict {
       var conflicts []DenyRuleConflict
       for _, skill := range skills {
           for _, tool := range skill.AllowedTools {
               for _, deny := range denyRules {
                   if globMatch(deny, tool) {
                       conflicts = append(conflicts, DenyRuleConflict{
                           Skill:     skill.Name,
                           Operation: tool,
                           DenyRule:  deny,
                           Message: fmt.Sprintf(
                               "Skill %q needs %q but deny rule %q would block it",
                               skill.Name, tool, deny),
                       })
                   }
               }
           }
       }
       return conflicts
   }
   ```

2. Build the skill operations registry — extract `allowed-tools` patterns from all generated skills:
   ```go
   var skillOperations = map[string][]string{
       "review-pr":     {"Bash(git *)", "Bash(gh *)"},
       "add-tests":     {"Bash(npm test *)", "Bash(go test *)", "Bash(pytest *)", "Bash(cargo test *)"},
       "upgrade-dep":   {"Bash(npm install *)", "Bash(pip install *)", "Bash(cargo add *)"},
       "refactor-safe": {"Bash(npm test *)", "Bash(go test *)", "Bash(make *)"},
       "gdev-init":     {"Bash(gdev *)"},
       "gdev-doctor":   {"Bash(gdev *)"},
       // ... all skills
   }
   ```

3. Build the deny rules registry — all generated deny rules from Phase 4:
   ```go
   var precisionDenyRules = []string{
       "Bash(npm install *)", "Bash(npm uninstall *)", "Bash(npm link *)",
       "Bash(npm publish *)", "Bash(npx *)",
       "Bash(yarn add *)", "Bash(yarn remove *)",
       "Bash(pnpm add *)", "Bash(pnpm remove *)",
       "Bash(bun add *)", "Bash(bun remove *)",
       "Bash(pip install *)", "Bash(pip uninstall *)",
       "Bash(uv pip install *)",
       "Bash(cargo install *)",
       "Bash(go install *)",
       "Bash(curl * | sh)", "Bash(curl * | bash)",
       "Bash(wget * | sh)", "Bash(wget * | bash)",
       // ... all 48+ rules from reference-deny-rules.md
   }
   ```

4. Identify expected conflicts: `/upgrade-dep` legitimately needs to run install commands. This is an expected conflict — the skill works WITH the guardrail hook (which validates the package), not around it. Document expected conflicts with explanations:
   ```go
   var expectedConflicts = map[string]string{
       "upgrade-dep:Bash(npm install *)": "Expected: /upgrade-dep installs packages through the PreToolUse guardrail hook which validates each package against OSV.dev before allowing",
   }
   ```

5. Implement `gdev check --deny-rules` command:
   - Load current deny rules from generated settings.json
   - Load skill operations from all generated SKILL.md files
   - Run `ValidateDenyRuleConflicts()`
   - Filter out expected conflicts
   - Report unexpected conflicts with clear diagnostics
   - Exit 1 if unexpected conflicts found

6. Wire validation into `gdev init` and `gdev update`: after generating settings.json and skills, run conflict validation. Fail with clear message if unexpected conflicts detected.

7. Generate a conflict test file at `.claude/tests/deny-rule-conflicts_test.go` (or equivalent) that can be run independently as a regression check.

8. Verify allow-rule compatibility: ensure generated allow rules (from Phase 12 guardrail strategy) don't inadvertently override deny rules. Remember: deny always wins over allow in Claude Code's permission model.

9. Write comprehensive tests:
   - Test that `Bash(npm install *)` does NOT match `Bash(npm test *)`
   - Test that `Bash(npm *)` DOES match `Bash(npm test *)` (this is the overly-broad case)
   - Test that all current deny rules × all current skill operations produce zero unexpected conflicts
   - Test that adding a new overly-broad deny rule is caught
   - Test that expected conflicts (upgrade-dep) are documented and filtered

**Acceptance Criteria:**
- [ ] `ValidateDenyRuleConflicts()` correctly identifies when a deny rule would block a skill operation
- [ ] `Bash(npm install *)` does NOT conflict with `Bash(npm test *)`, `Bash(npm run *)`, `Bash(npm audit *)`
- [ ] `Bash(npm *)` DOES conflict with test commands (overly-broad deny rule detected)
- [ ] `gdev check --deny-rules` reports zero unexpected conflicts with current configuration
- [ ] `/upgrade-dep` expected conflict documented: works with guardrail hook, not around it
- [ ] Conflict validation runs automatically during `gdev init` and `gdev update`
- [ ] Adding a new deny rule that blocks a skill operation fails the check with clear diagnostics
- [ ] Adding a new skill with operations that conflict with deny rules fails the check
- [ ] Regression test suite prevents future deny rule regressions
- [ ] Allow rules don't inadvertently override deny rules (deny-wins-over-allow verified)

**Research Citations:**
- `research-spikes/gdev-agentic-workflows/guardrail-integration-research.md § 1` — permission interaction model, deny-wins-over-allow rule
- `research-spikes/gdev-agentic-workflows/guardrail-integration-research.md § 2` — four common conflict points (package manager, read-deny, sandbox, hooks)
- `research-spikes/gdev-agentic-workflows/guardrail-integration-research.md § 3` — precision-scoping principle, safe vs unsafe deny rule examples
- `research-spikes/gdev-agentic-workflows/guardrail-integration-research.md § 4` — TestGuardrailWorkflowCompatibility Go implementation
- `research-spikes/claude-code-agent-package-guardrails/reference-deny-rules.md` — 48 deny rules covering 15+ package managers
- `phases/04-claude-code-addon-core-generation.md § Unit 3.1` — deny rule generation with individual entries

**Status:** Not Started

---

### Unit 14.6: devenv Task Definitions

**Description:** Generate devenv.nix task declarations per detected ecosystem, providing standardized build/test/lint/format/type-check/security-scan tasks. Tasks use devenv 2.0's task system with parallel execution and dependency ordering, are Claude-readable via CLAUDE.md, and map to CI workflow steps.

**Context:** devenv 2.0 introduced a task system (`devenv.tasks`) that declares named tasks with commands, dependencies, and parallel execution support. Each ecosystem module knows its build/test/lint/format commands via the `VerificationCommands()` interface (Phase 11 Unit 11.5). This unit generates task definitions that compose across multiple detected ecosystems. The tasks serve three audiences: developers running `devenv task <name>`, Claude Code reading CLAUDE.md to know how to build/test, and CI workflows referencing standardized task names.

**Code-Grounded Notes:** The devenv.nix template at `addons/devenv/templates/devenv.nix.tmpl` (79 lines) currently has NO tasks, scripts, or processes sections. The template data struct `DevenvNixTemplateData` at `addons/devenv/devenv_nix_data.go` has no task-related fields. This unit must add task support to both the template (Nix task declarations) and the data structure (Go fields to feed the template). The existing `CICommands()` method on ecosystem modules returns phase-aware commands (Install/Test/Scan) that can seed task definitions, but explicit build/test/lint/format tasks require `VerificationCommands()` which is not yet on the ecosystem module interface — this is a Phase 11 prerequisite that must be completed first.

**Desired Outcome:** A Go+TypeScript project gets `devenv task test` that runs both `go test ./...` and `npm test`. `devenv task lint` runs `golangci-lint run` and `eslint .`. CLAUDE.md lists all available tasks with their commands. CI workflows reference the same task names.

**Steps:**

1. Define task declaration types:
   ```go
   type TaskDefinition struct {
       Name        string   // "test", "build", "lint", "format", "typecheck", "security-scan"
       Description string   // Human-readable description
       Commands    []string // Commands to execute (from all detected ecosystems)
       DependsOn   []string // Task dependencies (e.g., "build" depends on nothing, "test" depends on "build")
       Parallel    bool     // Whether commands can run in parallel (true for independent ecosystems)
   }
   ```

2. Implement task aggregation from ecosystem modules. For each detected ecosystem, call `VerificationCommands()` and map to standard task names:
   ```go
   func AggregateTaskDefinitions(modules []EcosystemModule) []TaskDefinition {
       tasks := map[string]*TaskDefinition{
           "build":         {Name: "build", Description: "Build all projects", Parallel: true},
           "test":          {Name: "test", Description: "Run all test suites", Parallel: true, DependsOn: []string{"build"}},
           "lint":          {Name: "lint", Description: "Run all linters", Parallel: true},
           "format":        {Name: "format", Description: "Format all source code", Parallel: true},
           "typecheck":     {Name: "typecheck", Description: "Run type checkers", Parallel: true},
           "security-scan": {Name: "security-scan", Description: "Run security scanners", Parallel: true},
       }
       for _, mod := range modules {
           cmds := mod.VerificationCommands()
           tasks["build"].Commands = append(tasks["build"].Commands, cmds.Build...)
           tasks["test"].Commands = append(tasks["test"].Commands, cmds.Test...)
           tasks["lint"].Commands = append(tasks["lint"].Commands, cmds.Lint...)
           tasks["format"].Commands = append(tasks["format"].Commands, cmds.Format...)
           tasks["typecheck"].Commands = append(tasks["typecheck"].Commands, cmds.TypeCheck...)
       }
       // Add security scanners from enabled tools (Semgrep, Gitleaks)
       if toolEnabled("semgrep") {
           tasks["security-scan"].Commands = append(tasks["security-scan"].Commands, "semgrep --config auto --error .")
       }
       return filterEmpty(tasks)
   }
   ```

3. Generate task declarations in devenv.nix (shared file, `tasks` section):
   ```nix
   # --- tasks ---
   tasks = {
     "test" = {
       exec = ''
         go test ./...
         npm test
       '';
       after = [ "build" ];
     };
     "build" = {
       exec = ''
         go build ./...
         npm run build
       '';
     };
     "lint" = {
       exec = ''
         golangci-lint run
         eslint .
       '';
     };
     "format" = {
       exec = ''
         gofmt -w .
         prettier --write .
       '';
     };
   };
   # --- end tasks ---
   ```

4. Add task reference to CLAUDE.md (within gdev section markers):
   ```markdown
   <!-- gdev:tasks -->
   ## Development Tasks
   - `devenv task build` — Build all projects (go build ./..., npm run build)
   - `devenv task test` — Run all test suites (go test ./..., npm test)
   - `devenv task lint` — Run all linters (golangci-lint run, eslint .)
   - `devenv task format` — Format all code (gofmt -w ., prettier --write .)
   <!-- /gdev:tasks -->
   ```

5. Generate CI task mapping: map devenv task names to CI workflow step commands. Each task name maps to the same commands used in CI workflows (Phase 12 Unit 12.7).

6. Handle polyglot composition: when multiple ecosystems are detected, tasks aggregate commands from all ecosystems. Order by ecosystem detection priority (Tier 1 first).

7. Handle tool-contributed tasks: enabled tools (Semgrep, Gitleaks) contribute to the `security-scan` task. When a tool is enabled/disabled via lifecycle (Phase 12), tasks are regenerated.

8. Wire into lifecycle system: task section in devenv.nix is a shared-file section. `gdev enable/disable` triggers task regeneration.

9. Write unit tests: verify single-ecosystem task generation, multi-ecosystem aggregation, tool-contributed tasks, empty task filtering (don't generate `typecheck` if no ecosystem provides type-check commands).

**Acceptance Criteria:**
- [ ] `devenv task test` runs per-ecosystem test commands for all detected ecosystems
- [ ] `devenv task lint` runs per-ecosystem lint commands
- [ ] `devenv task build` runs per-ecosystem build commands
- [ ] Polyglot project (Go + TypeScript) gets combined tasks with both ecosystems' commands
- [ ] CLAUDE.md `<!-- gdev:tasks -->` section lists all tasks with their commands
- [ ] Tasks use devenv 2.0's task system (`devenv.tasks` in devenv.nix)
- [ ] `gdev enable semgrep` adds Semgrep to the `security-scan` task
- [ ] `gdev disable semgrep` removes Semgrep from the `security-scan` task
- [ ] Empty tasks are not generated (no `typecheck` task if no type checker detected)
- [ ] CI workflows use the same commands as devenv tasks (single source of truth)

**Research Citations:**
- `phases/11-ai-agent-tooling-integration.md § Unit 11.5` — `VerificationCommands()` interface, per-ecosystem build/test/lint/format commands
- `phases/11-ai-agent-tooling-integration.md § Unit 11.1` — per-ecosystem verification command registry (Go, JS/TS, Python, Rust, Java, .NET, Docker, Terraform)
- `phases/12-extended-integrations-lifecycle.md § Unit 12.1` — section markers in devenv.nix, tool lifecycle management
- `phases/12-extended-integrations-lifecycle.md § Unit 12.7` — CI workflow generation composing steps from enabled tools
- `research-spikes/gdev-claude-code-integration/claude-code-integration-research.md § 5.2` — CLAUDE.md build/test/lint commands reference

**Status:** Not Started

---

### Unit 14.7: CLAUDE.md Section Enhancement

**Description:** Enhance the generated CLAUDE.md with section markers for all new Phase 14 content areas: available skills directory, available agents directory, gdev command reference, devenv task reference, and per-ecosystem build/test/lint commands. Support both Sonnet and Opus context budgets with model-aware content generation. Use `@`-imports for detailed reference docs to keep CLAUDE.md lean.

**Context:** Phase 4 (Unit 3.2) established CLAUDE.md generation with `<!-- BEGIN/END GENERATED SECTION -->` markers. Phase 12 (Unit 12.1) extended this with per-tool section markers (`<!-- gdev:semgrep -->`, etc.). Phase 14 adds four new content areas that must integrate with the existing section marker system. The context management research established that CLAUDE.md should be 50-100 lines of concise reference — detailed content belongs in skills (loaded on demand), rules (loaded conditionally), or `@`-imported reference docs (loaded by reference). All section markers must support the lifecycle system's enable/disable operations.

**Code-Grounded Notes:** The template data struct `ClaudeMdTemplateData` at `addons/claudecode/generate_claude_md.go:13-23` currently has: ProjectName, ProjectDescription, ArchitectureNotes, Languages, BuildCommands, TestCommands, LintCommands, HasSecurityHooks, PackageManagers. New fields needed for Phase 14: AvailableSkills, AvailableAgents, GdevCommands, DevenvTasks. The template at `templates/claude-md.tmpl` uses conditional rendering with `{{ if hasAny .Languages }}` — this pattern extends naturally to the new fields (e.g., `{{ if hasAny .AvailableSkills }}`). No new template functions are needed, just new data fields and corresponding template blocks.

**Desired Outcome:** CLAUDE.md contains a compact, well-organized reference to all gdev capabilities. Developers and Claude can quickly see what skills, agents, commands, and tasks are available. Content stays within the model-aware context budget. `gdev enable/disable` correctly adds/removes sections.

**Steps:**

1. Define section markers for new content areas:
   ```
   <!-- gdev:skills -->    / <!-- /gdev:skills -->    — Available skills directory
   <!-- gdev:agents -->    / <!-- /gdev:agents -->    — Available agents directory
   <!-- gdev:commands -->  / <!-- /gdev:commands -->  — gdev command reference
   <!-- gdev:tasks -->     / <!-- /gdev:tasks -->     — devenv task reference
   ```

2. Generate the skills directory section:
   ```markdown
   <!-- gdev:skills -->
   ## Available Skills
   ### gdev Operations
   - `/gdev-init` — Initialize project with gdev
   - `/gdev-onboard` — Onboard existing project
   - `/gdev-setup` — Install prerequisites
   - `/gdev-enable <tool>` — Enable a tool
   - `/gdev-disable <tool>` — Disable a tool
   - `/gdev-update` — Update configs to latest

   ### Consulting Workflows
   - `/review-pr <number>` — Comprehensive PR review
   - `/add-tests <target>` — Generate tests for module
   - `/upgrade-dep <pkg> <version>` — Upgrade dependency
   - `/onboard-me` — Codebase onboarding
   - `/write-adr <title>` — Architecture Decision Record
   - `/incident-debug <symptom>` — Incident debugging
   - `/migration-plan <desc>` — Migration planning
   - `/handoff-doc` — Engagement handoff document
   <!-- /gdev:skills -->
   ```

3. Generate the agents directory section:
   ```markdown
   <!-- gdev:agents -->
   ## Available Agents
   - `@security-reviewer` — Security-focused code review (OWASP, injection, auth)
   - `@codebase-explorer` — Rapid codebase understanding (uses Haiku for speed)
   - `@test-gap-analyzer` — Find untested code paths
   - `@onboarding-guide` — Interactive codebase mentoring
   - `@migration-planner` — Framework upgrade risk assessment
   - `@handoff-doc-generator` — Client handoff documentation
   - `@incident-debugger` — Systematic incident investigation
   <!-- /gdev:agents -->
   ```

4. Generate the commands section:
   ```markdown
   <!-- gdev:commands -->
   ## gdev Commands
   - `gdev init` — Initialize or re-initialize project
   - `gdev devenv doctor` — Check system and project health
   - `gdev devenv setup` — Install missing prerequisites
   - `gdev enable <tool>` — Enable a tool
   - `gdev disable <tool>` — Disable a tool
   - `gdev status` — Show configuration state
   - `gdev list` — Show available tools
   - `gdev check` — Validate configuration

   ### Security Policy
   - Package installations go through gdev's security pipeline
   - Always use `gdev enable` to add tools, never configure manually
   - Run `gdev devenv doctor` after configuration changes
   <!-- /gdev:commands -->
   ```

5. Generate the tasks section (from Unit 14.6 output):
   ```markdown
   <!-- gdev:tasks -->
   ## Development Tasks
   - `devenv task build` — {{build commands from detected ecosystems}}
   - `devenv task test` — {{test commands from detected ecosystems}}
   - `devenv task lint` — {{lint commands from detected ecosystems}}
   - `devenv task format` — {{format commands from detected ecosystems}}
   <!-- /gdev:tasks -->
   ```

6. Generate per-ecosystem build/test/lint commands inline (from `EcosystemModule.VerificationCommands()`):
   ```markdown
   ## Build & Test
   - Build: `go build ./...`
   - Test: `go test ./...`
   - Lint: `golangci-lint run`
   - Format: `gofmt -w .`
   ```

7. Generate `@`-import for detailed reference:
   ```markdown
   @.claude/gdev-reference.md
   ```
   The gdev-reference.md file contains full command docs, tool descriptions, configuration reference, and troubleshooting — content too detailed for always-on CLAUDE.md but useful when Claude needs it.

8. Generate `.claude/gdev-reference.md` (machine-owned, overwritten on update):
   - Full gdev CLI reference with all commands and flags
   - All tool descriptions with status (enabled/disabled)
   - Configuration file locations and formats
   - Troubleshooting guide (common issues and fixes)
   - Size: 200-500 lines (loaded only via `@`-import, not always-on)

9. Implement model-aware content selection:
   - **Sonnet (200K)**: Skills section shows names only (no descriptions). Agents section shows names only. Commands section included. Tasks section included. Total ~50 lines.
   - **Opus (1M)**: Skills section with one-line descriptions. Agents section with one-line descriptions. Commands with security policy. Tasks with full command detail. Total ~100 lines.

10. Wire section markers into lifecycle system: `gdev enable/disable` for skills and agents updates the corresponding section. When a skill/agent is disabled, its entry is removed from the directory section. When re-enabled, it's added back.

11. Write unit tests: verify section markers are balanced (every `<!-- gdev:X -->` has `<!-- /gdev:X -->`). Verify Sonnet version <=50 lines. Verify Opus version <=100 lines. Verify lifecycle enable/disable correctly updates sections.

**Acceptance Criteria:**
- [ ] CLAUDE.md contains `<!-- gdev:skills -->`, `<!-- gdev:agents -->`, `<!-- gdev:commands -->`, `<!-- gdev:tasks -->` sections
- [ ] Each section marker pair is balanced and parseable by the lifecycle system
- [ ] Skills directory lists all deployed skills with invocation syntax
- [ ] Agents directory lists all deployed agents with `@agent-name` syntax
- [ ] Commands section includes gdev CLI reference and security policy
- [ ] Tasks section shows per-ecosystem build/test/lint/format commands
- [ ] `@.claude/gdev-reference.md` import present for detailed docs
- [ ] `.claude/gdev-reference.md` generated with full reference content
- [ ] Sonnet CLAUDE.md is <=50 lines total
- [ ] Opus CLAUDE.md is <=100 lines total
- [ ] `gdev enable/disable <skill|agent>` updates the corresponding directory section
- [ ] User content outside of section markers is preserved on `gdev init --update`
- [ ] Per-ecosystem build/test/lint commands injected from detected ecosystems

**Research Citations:**
- `research-spikes/gdev-claude-code-integration/claude-code-integration-research.md § 5.1-5.4` — CLAUDE.md vs skills content split, recommended section, section markers, @-import pattern
- `research-spikes/gdev-agentic-workflows/context-management-research.md § 2` — five-layer context architecture, always-on CLAUDE.md target of 50-100 lines
- `research-spikes/gdev-agentic-workflows/context-management-research.md § 5` — model-aware generation (Sonnet 50 lines / Opus 100 lines)
- `phases/04-claude-code-addon-core-generation.md § Unit 3.2` — CLAUDE.md generation with section markers, language-specific conventions
- `phases/12-extended-integrations-lifecycle.md § Unit 12.1` — per-tool section markers in CLAUDE.md, lifecycle-driven content management
- `research-spikes/gdev-extension-design/migration-strategy-design.md § CLAUDE.md Section Markers` — marker-based section management

**Status:** Not Started

---

## Phase Completion Criteria

- [ ] All seven units pass acceptance criteria
- [ ] 10 gdev operation skills deploy correctly during `gdev init`
- [ ] 7 consulting workflow agents deploy correctly during `gdev init`
- [ ] 8 consulting workflow skills deploy correctly during `gdev init`
- [ ] User-only skills cannot be auto-invoked by Claude (verified with `disable-model-invocation: true`)
- [ ] Claude can autonomously invoke `gdev-doctor`, `gdev-status`, `gdev-tools`, `gdev-detect`
- [ ] `/gdev-init` with dynamic context injection correctly captures and presents system state
- [ ] `/review-pr` successfully gathers PR context via `gh` and produces structured review
- [ ] `/onboard-me` delegates to `codebase-explorer` agent and produces onboarding doc
- [ ] Context budget stays under 5% of target model for both Sonnet and Opus
- [ ] `gdev check --deny-rules` reports zero unexpected conflicts
- [ ] `devenv task test` runs correct test commands for detected ecosystems
- [ ] CLAUDE.md has all section markers and stays within line count limits
- [ ] `gdev enable/disable` correctly updates skill/agent directory sections in CLAUDE.md
- [ ] All skills and agents are individually toggleable via tool lifecycle system
- [ ] `gdev init --update` replaces skills/agents with current gdev version
- [ ] Full enable → disable → re-enable cycle works for all skills and agents
