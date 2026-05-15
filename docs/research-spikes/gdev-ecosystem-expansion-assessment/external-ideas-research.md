# External Ideas Analysis: Yaw Labs, Learning Opportunities, Fight Slop with Clarity

## Research Question

What ideas, concepts, implementations, libraries, utilities, or features from these three sources are good candidates for inclusion in gdev?

---

## Source 1: Yaw Labs — Terminal Context Management

**Source**: [siliconsnark.com](https://www.siliconsnark.com/yaw-labs-built-a-terminal-startup-for-people-who-treat-context-like-ammunition/)

### Candidates for Inclusion

#### 1a. Per-Session Context Overlays
**Concept**: "Yaw Mode" layers rules, skills, and agents onto Claude Code per-session without altering `~/.claude/` config.

**gdev relevance**: HIGH. gdev already generates per-project `.claude/` config. But the per-session overlay idea extends this — when an engineer enters a devenv shell, Claude Code's behavior should automatically shift to match that project's context. This is partially achieved by `.claude/settings.json` in the project directory, but the dynamic overlay concept (session-scoped, not file-scoped) could inform how gdev's `enterShell` hooks configure the Claude Code environment.

**Recommendation**: Investigate whether devenv `enterShell` can set Claude Code environment variables or session-scoped config that provides per-project context overlays. If Claude Code supports `CLAUDE_SKILLS_PATH` or similar env-driven config, gdev could set it per-project in devenv.nix. This would make project context truly automatic — enter the devenv, get the right Claude Code behavior.

**Implementation fit**: claudecode addon (Phase 4) — extend enterShell hook generation.

#### 1b. mcph MCP Orchestrator
**Concept**: Open-source MCP server orchestrator with compliance test suite. Addresses MCP sprawl.

**gdev relevance**: MEDIUM. Our Phase 4 MCP expansion design (Unit 3.5.1) builds a custom `McpServerDef` registry. Before implementing, evaluate whether `mcph` provides a reusable foundation. If it handles server lifecycle, health checks, and config composition, gdev could delegate to it rather than building from scratch.

**Recommendation**: Research `mcph` (github.com/mcp-hosting/mcph) — architecture, maturity, whether it can be embedded as a library or if it's a standalone daemon. If it's a Go library or CLI wrapper, it could replace our custom registry infrastructure.

**Implementation fit**: Phase 4/12 — potential replacement for custom MCP registry. Investigate before building.

#### 1c. Open-Source MCP Servers (AWS, Tailscale, SSH)
**Concept**: Yaw Labs ships open-source MCP servers for AWS, Tailscale, SSH, npm, Caddy.

**gdev relevance**: MEDIUM. The AWS MCP server is directly relevant to our cloud module design. Tailscale MCP could be valuable for consulting engineers working across client VPNs. SSH MCP could complement remote development workflows.

**Recommendation**: Add AWS MCP and Tailscale MCP to our optional MCP catalog (Phase 12, Unit 12.8.X). Evaluate quality and maturity before recommending.

**Implementation fit**: Phase 12 MCP optional catalog expansion.

#### 1d. Centralized MCP Config Syncing
**Concept**: "One config for every MCP server" synced across clients and machines.

**gdev relevance**: LOW-MEDIUM. gdev's three-layer config (binary → project → local) handles per-project MCP config. Team-level sync is partially addressed by `.qsdev.yaml` in shared repos. Full cross-machine sync (like mcp.hosting provides) is a SaaS concern, not a CLI tool concern.

**Recommendation**: Note as a future consideration. If gdev's team config (Phase 13) includes shared MCP server definitions in `.qsdev.yaml`, that provides 80% of the value without requiring a sync service.

**Implementation fit**: Already covered by Phase 13 team config. No new work needed.

---

## Source 2: Learning Opportunities — Developer Skill Building

**Source**: [github.com/DrCatHicks/learning-opportunities](https://github.com/DrCatHicks/learning-opportunities)

### Candidates for Inclusion

#### 2a. Learning Opportunities Skill Plugin (STRONG CANDIDATE)
**Concept**: Claude Code skill that prompts evidence-based learning exercises after significant development work — prediction/observation/reflection, generation/comparison, teach-it-back, retrieval check-ins.

**gdev relevance**: HIGH. A consulting firm has a direct business interest in engineers who genuinely understand what they build, not just engineers who produce AI-accelerated output. The five learning science risks (generation effect, fluency illusion, spacing effect, metacognition gap, testing deficit) are amplified in consulting where engineers frequently work on unfamiliar codebases.

**Recommendation**: Include `learning-opportunities` as an optional skill in gdev's claudecode addon skill library. Deploy via `qsdev enable learning-opportunities`. This is a ready-made, research-backed, CC-BY-4.0 licensed skill that requires no custom development — just curation and deployment.

**Implementation fit**: Phase 4 (claudecode addon skill library) or Phase 14 (agentic skills). Add as opt-in skill alongside existing Trail of Bits security skills. Could be default-enabled for junior engineers via client profile compliance levels.

#### 2b. Orient — Repository Orientation Generator (STRONG CANDIDATE)
**Concept**: Companion plugin that generates orientation lessons for unfamiliar codebases using codebase sampling strategies.

**gdev relevance**: HIGH. Consulting engineers onboard to unfamiliar client codebases regularly. This directly addresses the "Join" onboarding mode (Phase 13, Unit 13.4). When an engineer runs `qsdev init` in Join mode on an existing repo, `orient` could automatically generate an orientation lesson.

**Recommendation**: Include `orient` as an optional skill, potentially auto-triggered during Join mode onboarding. Research the actual implementation quality before committing.

**Implementation fit**: Phase 13 (Join mode) + Phase 14 (onboarding-guide agent could invoke orient).

#### 2c. Post-Commit Learning Hook
**Concept**: `learning-opportunities-auto` triggers learning prompts after commits.

**gdev relevance**: MEDIUM. Fits gdev's hook deployment infrastructure. However, post-commit hooks that interrupt flow may face resistance from experienced engineers. Better as opt-in.

**Recommendation**: Include as an optional hook via `qsdev enable learning-hooks`. Do not make default.

**Implementation fit**: Phase 4 (hook deployment) — optional post-commit hook.

#### 2d. Measurement Framework (MEASURE-THIS.md)
**Concept**: Validated survey instruments for developer thriving and AI skill threat, with statistical rigor guidance.

**gdev relevance**: MEDIUM. For a consulting firm tracking engineer effectiveness across engagements, this provides research-backed measurement. Could integrate with gdev's health/compliance reporting (Phase 15) or be referenced in consulting workflow skills.

**Recommendation**: Reference in generated CLAUDE.md as a team health measurement resource. Not a tool gdev deploys, but a methodology gdev's consulting agents should know about.

**Implementation fit**: Phase 14 (consulting agent knowledge) — agent prompts should reference learning science when relevant.

#### 2e. Five Learning Science Risks as Design Principles
**Concept**: Generation effect, fluency illusion, spacing effect, metacognition gap, testing deficit.

**gdev relevance**: HIGH (design-level, not feature-level). These risks should inform how gdev's consulting agents are designed. The agents should encourage understanding, not just output. Specifically:
- The **handoff-doc-generator** agent should use "Teach It Back" patterns
- The **onboarding-guide** agent should use "Prediction → Observation → Reflection"
- The **code-review** agent should flag fluency illusion risks (clean-looking AI code that hides complexity)

**Recommendation**: Incorporate these principles into consulting agent prompt design (Phase 14). Not a separate feature — a design constraint on existing features.

**Implementation fit**: Phase 14 agent prompt design.

---

## Source 3: Fight Slop with Clarity — Engineering Philosophy

**Source**: [blog.vtemian.com](https://blog.vtemian.com/post/fight-slop-with-clarity/)

### Candidates for Inclusion

#### 3a. The Tarpit Test as Tool Evaluation Framework
**Concept**: "If a tool sells itself as a replacement for thinking clearly, it's a tarpit."

**gdev relevance**: HIGH (design-level). This validates gdev's existing design philosophy (principle #4: "Curate, don't reinvent") and the 13 explicitly rejected features. Every gdev feature should amplify existing clarity, not substitute for it.

**Recommendation**: Add as a documented evaluation criterion for future feature proposals. When evaluating whether gdev should include a new tool or feature, apply the tarpit test: does it amplify clear thinking, or replace it?

**Implementation fit**: Design principle — add to plan.md Design Principles or gdev's generated CLAUDE.md conventions.

#### 3b. Project Clarity Questions in qsdev init
**Concept**: "What problem am I solving? Whose problem is it? What does concrete success look like? What am I willing to exclude?"

**gdev relevance**: MEDIUM. These questions could be codified in gdev's project initialization. When creating a new project (`qsdev init` in Create mode), gdev could prompt for or generate a "project clarity" section in CLAUDE.md or README that forces articulation of purpose.

**Recommendation**: Add a `## Project Context` section to generated CLAUDE.md that includes these clarity questions as prompts for the engineer to fill in. This gives Claude Code project context and forces the engineer to think clearly about purpose.

**Implementation fit**: Phase 4 (CLAUDE.md generation) — add project clarity section template.

#### 3c. Perception-Reality Productivity Gap Awareness
**Concept**: Engineers using AI felt 20% faster but were actually 19% slower (METR 2025 trial). 40-point perception-reality gap.

**gdev relevance**: MEDIUM. Reinforces why learning-opportunities (2a) matters. Also informs how gdev's health reporting (Phase 15) should avoid pure velocity metrics — measuring "commits per day" or "PRs merged" without quality assessment creates the exact Goodhart's Law problem the article describes.

**Recommendation**: gdev's health scoring (Phase 15) should include quality indicators (test coverage, security posture, review thoroughness) alongside productivity indicators. Reference the METR finding in consulting agent training.

**Implementation fit**: Phase 15 (scoring weights) + Phase 14 (agent awareness).

---

## Summary: Actionable Candidates

### Include in gdev (new features/skills)

| # | Candidate | Source | Priority | Phase | Effort |
|---|-----------|--------|----------|-------|--------|
| 2a | Learning Opportunities skill plugin | DrCatHicks | High | 4/14 | Small (deploy existing) |
| 2b | Orient codebase orientation plugin | DrCatHicks | High | 13/14 | Small (deploy existing) |
| 1a | Per-session context overlays via enterShell | Yaw Labs | Medium | 4 | Small (env var config) |
| 3b | Project clarity questions in CLAUDE.md | vtemian | Medium | 4 | Small (template addition) |

### Investigate before including

| # | Candidate | Source | What to investigate |
|---|-----------|--------|-------------------|
| 1b | mcph MCP orchestrator | Yaw Labs | Architecture, maturity, Go embeddability — could replace custom MCP registry |
| 1c | AWS/Tailscale MCP servers | Yaw Labs | Quality, maintenance, security posture |
| 2c | Post-commit learning hook | DrCatHicks | Engineer acceptance, opt-in UX |

### Incorporate as design principles (not features)

| # | Principle | Source | Where to apply |
|---|-----------|--------|---------------|
| 2e | Five learning science risks | DrCatHicks | Phase 14 agent prompt design |
| 3a | Tarpit test for tool evaluation | vtemian | Plan.md design principles, future feature proposals |
| 3c | Perception-reality gap awareness | vtemian | Phase 15 scoring, Phase 14 agent awareness |
