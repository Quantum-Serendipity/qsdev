# Research Summary: gdev Agentic Workflow Patterns

## Overview

Agentic-first development workflow patterns for a CLI tool (gdev) that provides pre-built Claude Code workflows for consulting engineers. Research covers pre-built skills/commands for common tasks (code review, refactoring, testing, documentation, incident response, onboarding, migration), Claude Code agent files format, agentic task templates, context management strategies, guardrail integration with workflows, prior art (Trail of Bits, Security Phoenix, starter kits, cursor rules), and consulting-specific workflow differentiation.

Builds on two completed sibling spikes:
- `gdev-extension-design/` — gdev addon architecture, Claude Code config surface area, wizard UX, template engine
- `claude-code-agent-package-guardrails/` — hooks enforcement model, deny rules, skills as workflow orchestrators, MCP integration

## Topics

- **Pre-built Workflow Skills Catalog** — Complete. 7 agents + 15 skills across 7 categories (code review, refactoring, testing, documentation, incident response, onboarding, migration). Each with complete SKILL.md/agent definitions, structured checklists, verification steps, and consulting-appropriate output formatting. See [workflow-skills-catalog-research.md](workflow-skills-catalog-research.md).

- **Claude Code Agent Files** — Complete. Full format documentation for `.claude/agents/*.md` including all 15 frontmatter fields, priority ordering, built-in agents, multi-agent workflow patterns (sequential chaining, parallel subagents, foreground/background, agent teams, fork mode), and persistent memory system. Key finding: subagents cannot spawn other subagents. Trail of Bits principle: "Encode expertise in agents, procedures in skills." See [agent-files-research.md](agent-files-research.md).

- **Agentic Task Templates** — Complete. Four core templates (security review, add tests, upgrade dependency, onboard to codebase) with checklist-driven execution, evidence-anchored findings, graduated depth levels, self-verification gates, and structured output formats. Templates are enhanced skills with explicit pass/fail criteria. See [agentic-task-templates-research.md](agentic-task-templates-research.md).

- **Context Management** — Complete. Five-layer context architecture: minimal always-on CLAUDE.md (50-100 lines), conditional `.claude/rules/` with `paths:` frontmatter, on-demand skill loading, isolated agent context, and zero-cost external hooks/settings. 5% context budget target for generated config. Model-aware generation (Sonnet 200k vs Opus 1M). Anti-patterns: kitchen sink CLAUDE.md, verbose skill descriptions, auto-invoking all skills. See [context-management-research.md](context-management-research.md).

- **Guardrail-Workflow Integration** — Complete. The core tension: deny rules must not block test/build operations that skills need. Solution: precision-scoped deny rules targeting specific dangerous operations (not broad tool categories), explicit allow rules for workflow operations, agent tool restrictions as stronger guarantees, and conflict detection tests in gdev init/update. Recommended approach: `/upgrade-dep` works *with* guardrail hooks (hook validates package), not around them. See [guardrail-integration-research.md](guardrail-integration-research.md).

- **Prior Art Survey** — Complete. 12 sources from 6 projects: Trail of Bits skills (35+ plugins, marketplace distribution, security-focused), Trail of Bits claude-code-config (opinionated team defaults, anti-rationalization hook, custom commands), Security Phoenix (graduated security tiers, 12-role pipeline, hook integration), awesome-claude-code-toolkit (taxonomy of 135 agents), cursor rules (decomposition patterns for multi-file architecture), community best practices (context management, three-tier orchestration, skill design). See [prior-art-research.md](prior-art-research.md).

- **Consulting-Specific Differentiation** — Complete. Five structural differences from product development: unfamiliar codebases as default, variable client compliance, billable hour economics, mandatory handoff documentation, multi-client context switching. Top gdev differentiators: onboarding-first library, compliance profiles, handoff documentation pipeline, two-tier skills (user-level consultant standards + project-level client specifics), time-aware skill design with cost estimates. See [consulting-differentiation-research.md](consulting-differentiation-research.md).

## Open Questions

- How should gdev version its embedded skill library for updates? (embed.FS copies vs git-based remote library)
- Should gdev include a `/skill-improve` meta-skill that helps teams tune workflows to specific clients?
- What is the practical false-positive rate of auto-invoked skills in consulting contexts?
- How should agent memory be handled across engagement completion (archive vs delete)?
- Should gdev generate agent team configurations, or is that too experimental (v2.1.32+, Opus 4.6 required)?
- How to handle consulting firms that use Sonnet (200k context) vs Opus (1M context) differently?

## Conclusions

### Primary Finding

gdev can provide a comprehensive agentic workflow library through Claude Code's multi-file extension system: **7 agents** for context-isolated analysis, **15 skills** for procedural workflows, **language-specific rules** for conditional conventions, and **precise guardrails** that don't block legitimate operations. The architecture maps cleanly to Claude Code's extension taxonomy.

### Key Architectural Decisions

1. **Agent = Expertise, Skill = Procedure**: Agents (security-reviewer, codebase-explorer) run in isolated context with specialized tool restrictions. Skills (/review-pr, /add-tests, /upgrade-dep) run in main context with structured checklists. This follows Trail of Bits' proven principle.

2. **Five-Layer Context Architecture**: Always-on CLAUDE.md (50-100 lines) → conditional rules with `paths:` → on-demand skills → isolated agents → external hooks/settings. Generated config stays under 5% of context budget.

3. **Precision Guardrails**: Deny rules target specific dangerous operations (`npm install *`), not broad categories (`npm *`). Explicit allow rules for workflow operations (`npm test *`, `git *`). Conflict detection tests validate guardrail-workflow compatibility at `gdev init` time.

4. **Consulting Differentiation**: Onboarding-first library, compliance profiles, handoff documentation pipeline, two-tier skills (user-level + project-level), graduated cost tiers, zero-friction wizard quick-path.

5. **Task Templates as Enhanced Skills**: Four core templates with checklist-driven execution, evidence-anchored findings, self-verification gates, and graduated depth levels. Templates produce consistent, auditable output suitable for client deliverables.

### Risks and Limitations

- **Claude Code version dependency**: Agent features (teams, fork mode) are experimental and require recent versions. gdev should generate only stable features by default.
- **Context budget is tight on Sonnet**: With 200k context, generated config competes with user interaction. Model-aware generation is essential.
- **Skill auto-invocation is probabilistic**: Descriptions determine when Claude loads a skill. Poorly written descriptions cause misfires. gdev must write precise, keyword-rich descriptions.
- **Agent memory creates state**: Persistent memory accumulates across sessions, creating a maintenance burden. gdev should include a memory cleanup command.
- **Guardrail-workflow conflicts are possible**: Despite precision-scoping, new deny rules or custom hooks could block workflow operations. The conflict detection test is essential.

### Deliverables

- 7 research reports covering all requested topics (12 source docs saved)
- 7 complete agent definitions ready for embed.FS
- 15 complete skill definitions ready for embed.FS
- Context budget analysis with model-aware generation guidance
- Guardrail conflict detection test specification
- Consulting differentiation analysis with 5 gdev-specific differentiators
