# Tasks: gdev Extension Design

## Phase 2: Extension Design & Synthesis

### Pending

### Active

### Completed
- [x] **Config template engine design** — How to produce Nix code (devenv.nix), YAML (devenv.yaml), structured JSON (settings.json), and markdown (CLAUDE.md) from wizard answers.
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: Per-format strategy: text/template + Nix helper funcs for .nix, yaml.Marshal for .yaml, json.MarshalIndent for .json, text/template for .md, embed.FS copy for skills/rules. Unified GeneratedFile pipeline with atomic writes and post-generation validation. See config-template-engine-design.md.

- [x] **Re-runnability and migration strategy** — How `qsdev init` handles existing projects: merging generated config with user customizations, versioning team standards, update workflows.
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: SHA256 hash tracking for change detection. Per-file strategies: regenerate machine-owned, section markers for CLAUDE.md, three-way merge for settings.json, library versioning for skills. See migration-strategy-design.md.

### Completed
- [x] **Addon architecture design** — Define addon boundaries (devenv vs claude-code vs combined), composition model, how the init/wizard command works, which gdev extension points each addon uses.
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: Three-addon architecture: devenv, claudecode, devinit (orchestration). Profile system, detection engine, inter-addon communication via direct calls. See addon-architecture-design.md.

- [x] **devenv addon detailed design** — Bootstrap steps, config keys, commands, template strategy for devenv.yaml/devenv.nix/.envrc. Wizard question sequence with huh form groups.
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: 1505-line design. 5 bootstrap steps, 4 commands, YAML marshaling for devenv.yaml, text/template for devenv.nix. 3 concrete template examples. See devenv-addon-design.md.

- [x] **Claude Code addon detailed design** — Bootstrap steps, config keys, commands, template strategy for CLAUDE.md/settings.json/skills/hooks. Wizard question sequence.
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: 1776-line design. 7 bootstrap steps, 5 commands, text/template for CLAUDE.md, json.Marshal for settings/MCP, embed.FS for skills. Two-tier skill library. See claude-code-addon-design.md.

- [x] **Wizard flow integration design** — How huh forms integrate with gdev's bootstrap step system, progressive disclosure implementation, non-interactive/CI mode via flags, plan preview.
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: Quick path (1 question) vs customize (5 groups). Detection engine pre-populates. Merge mode for existing projects. Full flag mapping. See wizard-flow-integration-design.md.

### Completed

## Phase 1: Scoping & Initial Research

### Pending

### Active

### Completed
- [x] **gdev architecture deep-read** — Read the full gdev source at ~/Repos/gdev to understand its module/plugin system, extension points, and configuration model
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: Go addon framework with two-phase lifecycle (customization → lockdown). 10 extension points identified. See gdev-architecture-research.md.

- [x] **gdev upstream docs & README analysis** — Fetch and save gdev GitHub docs, README, any wiki/issues that describe the extension model
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: 6 docs saved to docs/. No external docs exist beyond README and source — project is pre-stable (v0.14.0). _template addon is the only extension authoring guide.

- [x] **Existing gdev extensions/modules inventory** — Catalog what extensions/modules gdev already ships or supports, to understand the conventions and gaps
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: 25+ addons across 11 categories cataloged. Bootstrap addon has wizard infrastructure (steps, plans, user input, skip handlers). See gdev-modules-inventory-research.md.

- [x] **devenv.sh integration surface area** — Identify what devenv.sh configuration a gdev extension would need to manage (packages, services, languages, scripts, env vars)
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: Two-file config model (devenv.yaml + devenv.nix), 60+ language modules, 45+ services, processes, DAG tasks, git-hooks, containers. Wizard decision tree mapped. See devenv-surface-area-research.md.

- [x] **Claude Code configuration surface area** — Identify what Claude Code config a gdev extension would manage (CLAUDE.md, settings.json, skills, hooks, MCP servers, permissions)
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: 7 config domains (CLAUDE.md, settings.json, skills, hooks, MCP servers, permissions, directory structure) across 3 scope layers (managed/project/user). See claude-code-config-surface-area-research.md.

- [x] **Wizard/installer UX patterns survey** — Research existing CLI wizard/installer patterns (yeoman, create-*, degit, cookiecutter) for guided project setup, to inform the gdev extension UX
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: Surveyed 6 categories of wizard patterns across JS and Go ecosystems. Key recommendation: use charmbracelet/huh for Go wizard UI, adopt "opinionated menu with escape hatches" pattern (strong defaults + progressive disclosure gate + CLI flags). 13 source docs saved. See wizard-ux-patterns-research.md.
