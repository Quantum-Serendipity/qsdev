# Research Log: gdev Extension Design

## 2026-05-12 — Spike Created
- **Type**: decision
- **Status**: success
- **Depth**: surface
- **Summary**: Spike initialized. Goal: deep investigation of gdev (~/Repos/gdev, github.com/fastcat/gdev) architecture to understand its extension/module/plugin system and design new extensions that enable simple development environment configuration — potentially via installer wizard workflows — for devenv.sh, Claude Code, with boilerplate packages, settings, skills, and modules.
- **Next**: Define research question and create Phase 1 tasks. Start by reading gdev source to understand its architecture.

## 2026-05-12 — Wizard/Installer UX Patterns Survey Complete
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**: 13 docs saved to `docs/` — Yeoman (composability, runtime context), Copier comparisons, Cookiecutter hooks, create-next-app docs, create-t3-app overview, sv create docs, clack/inquirer/enquirer comparison, Go CLI prompt libraries (huh, survey, promptui, go-prompt), degit readme, CLI UX patterns article
- **Summary**: Surveyed the full landscape of CLI wizard/installer UX patterns across 6 categories: classic generators (Yeoman), modern scaffolding CLIs (create-next-app, create-t3-app, sv create), template generators (cookiecutter, copier), lightweight scaffolding (degit), JS prompt libraries (clack, inquirer, enquirer), and Go prompt libraries (huh, bubbletea, survey, promptui). Key finding: ecosystem has converged on "opinionated menu with escape hatches" — strong defaults first, progressive disclosure gate, CLI flags for non-interactive use. For gdev, charmbracelet/huh is the recommended Go library (Form > Group > Field maps to bootstrap steps, dynamic forms, accessibility mode, theming). Created comprehensive report with concrete UX mockups for gdev init wizard.
- **Next**: Complete devenv.sh and Claude Code surface area research, then synthesize all three into extension design recommendations.

## 2026-05-12 12:30 — Architecture, Docs & Inventory Complete
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**: gdev source at ~/Repos/gdev, GitHub raw files → `docs/gdev-readme.md`, `docs/gdev-addons-readme.md`, `docs/gdev-source-architecture.md`, `docs/gdev-extension-interfaces.md`, `docs/gdev-examples.md`, `docs/gdev-addons-pkg-godoc.md`
- **Summary**: Three parallel sub-agents completed architecture deep-read, upstream docs fetch, and modules inventory. Key findings:
  - gdev is a Go addon-framework toolkit (not a standalone app). Users create their own `main()`, configure addons, call `cmd.Main()`.
  - Two-phase lifecycle: customization phase (addon registration, config) → lockdown → initialization → runtime. Guards prevent misconfiguration.
  - 10 extension points: Addon, CLI Command, Resource, Context DI, Service, PreStartHook, Build Strategy, Diags Source/Collector, GoBuildCache Backend, Bootstrap Step.
  - 25+ built-in addons across 11 categories. No devenv.sh or Nix integration exists.
  - Bootstrap addon already has wizard infrastructure: steps with user input, skip handlers, plan composition, headless mode, reboot support.
  - Project is pre-stable (v0.14.0), no external docs beyond README and source. `_template` addon is the only extension authoring guide.
- **Next**: Research devenv.sh config surface area, Claude Code config surface area, and wizard UX patterns (all unblocked now).

## 2026-05-12 14:00 — devenv.sh Integration Surface Area Research Complete
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**: 28 docs saved to `docs/devenv-*.md` from devenv.sh official documentation, GitHub source, and blog posts
- **Summary**: Comprehensive analysis of devenv.sh configuration surface area. Key findings:
  - Two-file config model: `devenv.yaml` (YAML, inputs/deps) + `devenv.nix` (Nix, environment definition). Plus optional `.envrc` for direnv.
  - 60+ language modules, 45+ services, processes with supervision/probes/auto-ports, DAG tasks, scripts, git-hooks, containers, outputs, env vars, shell hooks, secretspec.
  - Consistent module pattern: `enable = true` + language/service-specific options. Go module source analyzed as canonical example.
  - Three activation modes: manual `devenv shell`, native `devenv hook` (auto-activation), direnv `.envrc`.
  - Native mode (devenv.yaml) recommended over Flake mode — more features, better caching.
  - Module system is NixOS module system internally — custom modules composable via imports.
  - devenv 2.0 adds native process manager, evaluation caching (sub-100ms), TUI, auto-port allocation, MCP server.
  - Addon wizard decision tree mapped: project type -> languages -> package managers -> services -> hooks -> direnv -> packages -> env vars.
- **Next**: Complete devenv surface area task revision cycle. Research Claude Code config surface area.

## 2026-05-12 15:00 — Phase 1 Complete: All 6 Tasks Done
- **Type**: analysis
- **Status**: success
- **Depth**: moderate
- **Summary**: Claude Code configuration surface area research completed, finishing all 6 Phase 1 tasks. 7 config domains identified across 3 scope layers. All findings written to claude-code-config-surface-area-research.md. Combined with the other 5 completed tasks, we now have comprehensive understanding of: gdev architecture (10 extension points), existing modules (25+ addons), devenv.sh surface area (60+ language modules, 45+ services), Claude Code config (7 domains), and wizard UX patterns (recommends huh + opinionated defaults).
- **Next**: Move to Phase 2 — synthesize findings into concrete extension designs. Key design decisions: addon boundaries (devenv vs claude-code vs combined), wizard flow architecture (bootstrap steps vs standalone command), template/generation strategy for config files.

## 2026-05-12 16:00 — Phase 2 Complete: All 6 Design Documents Written
- **Type**: analysis
- **Status**: success
- **Depth**: deep
- **Summary**: All Phase 2 design work complete. Produced 6 design documents:
  1. **Addon architecture** — Three-addon split (devenv, claudecode, devinit). Profile system for named presets. Detection engine pre-populates wizard. Direct inter-addon calls for orchestration.
  2. **devenv addon** (1505 lines) — 5 bootstrap steps, 4 commands, YAML marshaling for devenv.yaml, text/template with Nix helpers for devenv.nix. 3 project-type template examples (Go+Postgres, TS+pnpm+Redis, Python+uv).
  3. **Claude Code addon** (1776 lines) — 7 bootstrap steps, 5 commands, text/template for CLAUDE.md, json.Marshal for settings.json, embed.FS for skills. Two-tier skill library (embedded + remote git).
  4. **Wizard flow integration** — Quick path (1 question) vs customize (5 huh form groups). Detection pre-population. Merge mode for existing projects. Full CLI flag mapping for CI/non-interactive use.
  5. **Config template engine** — Per-format strategy: text/template for Nix/markdown, struct marshaling for YAML/JSON, embed.FS copy for skills/rules. Unified GeneratedFile pipeline with atomic writes and post-generation validation.
  6. **Migration strategy** — SHA256 hash tracking per file. Machine-owned files regenerated safely. Human-edited files: section markers (CLAUDE.md), three-way merge (settings.json), library versioning (skills/rules). devenv.nix never auto-overwritten.
- **Next**: Phase 3 synthesis — write conclusions in research.md. Review all designs against depth checklist. Identify remaining gaps.

## 2026-05-12 17:00 — Spike Completed
- **Type**: analysis
- **Status**: success
- **Depth**: deep
- **Summary**: Spike finalized. Depth checklist audit: 9/11 documents pass all criteria; 2 accepted with noted limitations (modules inventory missing failure modes, Claude Code surface area missing concrete examples — both gaps covered by sibling design docs). Conclusions written to research.md. 4 follow-on candidates flushed to proposed-spikes.md: implementation of the three designed addons (large), team skill library format (small), monorepo devenv patterns (small), config lint command (small). Spike produced 5 research reports (28+ source docs), 6 design documents (~3,300 lines of specification with Go code sketches), and a complete architecture for three gdev addons (devenv, claudecode, devinit) ready for implementation.
