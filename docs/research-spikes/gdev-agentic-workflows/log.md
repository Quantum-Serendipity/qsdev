# Research Log: gdev Agentic Workflow Patterns

## 2026-05-12 — Spike Created
- **Type**: decision
- **Status**: success
- **Depth**: surface
- **Summary**: Spike initialized. Research focus: agentic-first development workflow patterns for gdev CLI, pre-built Claude Code skills/commands for consulting engineers, agent file formats, task templates, context management, guardrail integration, prior art, and consulting-specific differentiation. Builds on completed gdev-extension-design spike (addon architecture, claude code config surface area) and claude-code-agent-package-guardrails spike (hooks, permissions, skills enforcement model).
- **Next**: Define Phase 1 tasks and begin parallel sub-agent research.

## 2026-05-12 — Phase 1 Complete: All 7 Research Tasks
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**: 
  - [Claude Code subagents docs](https://code.claude.com/docs/en/sub-agents) → `docs/claude-code-subagents-official-docs.md`
  - [Claude Code skills docs](https://code.claude.com/docs/en/skills) → `docs/claude-code-skills-official-docs.md`
  - [Claude Code best practices](https://code.claude.com/docs/en/best-practices) → `docs/claude-code-best-practices-official.md`
  - [Claude Code common workflows](https://code.claude.com/docs/en/common-workflows) → `docs/claude-code-common-workflows.md`
  - [Claude Code features overview](https://code.claude.com/docs/en/features-overview) → `docs/claude-code-features-overview.md`
  - [Trail of Bits skills](https://github.com/trailofbits/skills) → `docs/trailofbits-skills-repo.md`
  - [Trail of Bits claude-code-config](https://github.com/trailofbits/claude-code-config) → `docs/trailofbits-claude-code-config.md`
  - [Security Phoenix skills](https://github.com/Security-Phoenix-demo/security-skills-claude-code) → `docs/security-phoenix-skills-repo.md`
  - [MindStudio workflow patterns](https://www.mindstudio.ai/blog/claude-code-agentic-workflow-patterns) → `docs/agentic-workflow-patterns-mindstudio.md`
  - [awesome-claude-code-toolkit](https://github.com/rohitg00/awesome-claude-code-toolkit) → `docs/awesome-claude-code-toolkit.md`
  - [awesome-cursorrules](https://github.com/PatrickJS/awesome-cursorrules) → `docs/awesome-cursorrules.md`
  - [Agent teams guide](https://github.com/FlorianBruniaux/claude-code-ultimate-guide) → `docs/agent-teams-guide.md`
- **Summary**: Completed all 7 Phase 1 research tasks in parallel. Produced 7 detailed research reports:
  1. `workflow-skills-catalog-research.md` — 7 agents + 15 skills across 7 categories with complete definitions
  2. `agent-files-research.md` — Full agent format documentation, multi-agent patterns, agent vs skill distinction
  3. `agentic-task-templates-research.md` — 4 core task templates with checklists, verification, graduated depth
  4. `context-management-research.md` — 5-layer architecture, 5% budget target, model-aware generation
  5. `guardrail-integration-research.md` — Precision-scoped deny rules, conflict detection, managed settings
  6. `prior-art-research.md` — 6 projects surveyed, transferable patterns cataloged
  7. `consulting-differentiation-research.md` — 5 structural differences, top 5 gdev differentiators
- **Next**: Phase 2 synthesis (skill library architecture, gdev integration design) if requested.
