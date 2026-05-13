# Tasks: gdev Agentic Workflow Patterns

## Phase 1: Research & Investigation

### Pending

### Active

### Completed
- [x] **Pre-built workflow skills catalog** — Research and design Claude Code skill files for 7 consulting workflow categories: code review (security/perf/a11y), refactoring with test validation, testing (coverage gaps, mutation testing), documentation (ADRs, API docs, runbooks), incident response, codebase onboarding, and migration workflows
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: Produced 7 agents + 15 skills catalog with complete SKILL.md and agent definitions. See workflow-skills-catalog-research.md.

- [x] **Claude Code agent files research** — Investigate `.claude/agents/*.md` format, multi-agent workflows, sub-agent delegation patterns, and how agents differ from skills/commands
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: Complete format documentation from official docs. Key finding: subagents cannot spawn other subagents. Agent = isolated context specialist, Skill = shared context procedure. See agent-files-research.md.

- [x] **Agentic task templates** — Design pre-configured task patterns that encode best practices for common operations (PR security review, add tests, upgrade dependency, onboard to codebase)
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: Four core templates with checklist-driven execution, evidence anchoring, graduated depth levels, and self-verification gates. See agentic-task-templates-research.md.

- [x] **Context management patterns** — Research how to structure CLAUDE.md, skills, rules, and agents to provide right context per task without bloating the context window
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: Five-layer architecture (always-on CLAUDE.md, conditional rules with paths:, on-demand skills, isolated agents, external hooks/settings). 5% context budget for generated config. Model-aware generation. See context-management-research.md.

- [x] **Guardrail-workflow integration** — How agentic workflows interact with security guardrails (deny rules, hooks, permissions) without blocking legitimate operations
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: Precision-scoped deny rules (target specific dangerous ops, not broad tool categories). Conflict detection test for gdev init/update. Recommended approach C for /upgrade-dep: work with guardrails, not around them. See guardrail-integration-research.md.

- [x] **Prior art survey** — Research Trail of Bits claude-code skills, Security Phoenix patterns, Claude Code starter kits, best practices repos, and .cursorrules cross-pollination
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: 12 sources saved to docs/. Trail of Bits (35+ plugins, config patterns), Security Phoenix (graduated security tiers, 12-role pipeline), awesome-claude-code-toolkit (taxonomy of 135 agents), cursor rules (decomposition into Claude Code multi-file architecture). See prior-art-research.md.

- [x] **Consulting-specific workflow differentiation** — What makes consulting workflows different: unfamiliar codebases, client compliance, time pressure, handoff documentation, multi-client context switching
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: 5 structural differentiators identified. Top gdev differentiators: onboarding-first library, compliance profiles, handoff documentation pipeline, two-tier skills (user + project), time-aware design. See consulting-differentiation-research.md.

## Phase 2: Synthesis & Design

### Pending
- [ ] **Skill library architecture** — Design the complete skill/agent/command library that gdev embeds, including categorization, naming conventions, and dependency model
  - Priority: high
  - Estimate: large
  - Depends: All Phase 1 tasks (complete)

- [ ] **gdev integration design** — How the workflow library integrates with the existing claudecode addon design from gdev-extension-design spike
  - Priority: high
  - Estimate: medium
  - Depends: Skill library architecture

### Active

### Completed
