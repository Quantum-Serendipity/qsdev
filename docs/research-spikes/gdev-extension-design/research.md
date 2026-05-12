# Research Summary: gdev Extension Design

## Overview

Deep investigation and analysis of [gdev](https://github.com/fastcat/gdev) to understand its architecture, extension/module/plugin system, and design new extensions that enable simple and easy development environment configuration. Target integrations include devenv.sh and Claude Code, with boilerplate default packages, settings, skills, and modules. Explore installer wizard workflows as a UX pattern for guided setup.

## Topics

- **gdev Architecture** — ✅ Complete. Go addon-framework with two-phase lifecycle (customization → lockdown), 10 extension points, type-safe generics. See [gdev-architecture-research.md](gdev-architecture-research.md).
- **gdev Modules Inventory** — ✅ Complete. 25+ built-in addons across 11 categories. Bootstrap addon has wizard step infrastructure. See [gdev-modules-inventory-research.md](gdev-modules-inventory-research.md).
- **gdev Upstream Docs** — ✅ Complete. 6 docs saved. No external documentation — source and _template are the only guides.
- **devenv.sh Integration Surface Area** — ✅ Complete. Two-file config model (devenv.yaml + devenv.nix), 60+ language modules, 45+ services, processes with supervision, DAG tasks, git-hooks, containers. Native mode recommended over flake mode. Wizard decision tree mapped. See [devenv-surface-area-research.md](devenv-surface-area-research.md).
- **Claude Code Configuration Surface Area** — ✅ Complete. 7 config domains (CLAUDE.md, settings.json, skills, hooks, MCP servers, permissions, directory structure) across 3 scope layers (managed/project/user). Addon should generate project-level committed config + templates for personal overrides. See [claude-code-config-surface-area-research.md](claude-code-config-surface-area-research.md).
- **Wizard/Installer UX Patterns** — ✅ Complete. Surveyed Yeoman, create-* tools, cookiecutter/copier, degit, JS prompt libs (clack/inquirer/enquirer), Go prompt libs (huh/bubbletea/survey/promptui), and progressive disclosure patterns. Recommends charmbracelet/huh + "opinionated menu with escape hatches" pattern. See [wizard-ux-patterns-research.md](wizard-ux-patterns-research.md).

## Design Documents

- **Addon Architecture** — ✅ Complete. Three-addon split: `devenv`, `claudecode`, `devinit` (orchestration). Profile system, detection engine, direct inter-addon calls. See [addon-architecture-design.md](addon-architecture-design.md).
- **devenv Addon Design** — ✅ Complete. 5 bootstrap steps, 4 commands, YAML marshaling + Nix templates, config persistence. 3 project-type template examples. See [devenv-addon-design.md](devenv-addon-design.md).
- **Claude Code Addon Design** — ✅ Complete. 7 bootstrap steps, 5 commands, JSON marshaling + markdown templates, two-tier skill library. See [claude-code-addon-design.md](claude-code-addon-design.md).
- **Wizard Flow Integration** — ✅ Complete. Quick path (1 question) vs customize (5 groups). huh Form → Group → Field mapping. Detection pre-population, merge mode, non-interactive flags. See [wizard-flow-integration-design.md](wizard-flow-integration-design.md).
- **Config Template Engine** — ✅ Complete. Per-format generation: text/template for Nix/markdown, struct marshaling for YAML/JSON, embed.FS for skills/rules. Unified pipeline with atomic writes and validation. See [config-template-engine-design.md](config-template-engine-design.md).
- **Migration Strategy** — ✅ Complete. SHA256 hash tracking, per-file merge strategies (regenerate machine-owned, section markers for CLAUDE.md, three-way merge for settings.json, library versioning for skills). See [migration-strategy-design.md](migration-strategy-design.md).

## Resolved Questions

- **devenv.nix generation**: Go `text/template` with custom Nix helper functions. Direct file generation, not wrapping devenv CLI.
- **Flake mode**: Target native mode only (devenv.yaml + devenv.nix). No flake.nix management.
- **Wizard interactivity**: "Opinionated menu with escape hatches" — 1 question for defaults, 5 groups for customizers, CLI flags for CI.
- **Addon boundaries**: Separate `devenv` and `claudecode` addons, with `devinit` for orchestration.
- **Migration**: SHA256 hash tracking per file, per-file merge strategies matched to edit patterns.

## Open Questions

- What's the right team skill library format and hosting model? (manifest.yaml + git repo? Go embed? Both?)
- How should profiles be shared across a team? (compiled into binary? External config repo?)
- Should there be a `gdev lint` command that validates generated config against team standards?
- How to handle devenv.nix in monorepos with shared base + per-service overrides?

## Conclusions

gdev's addon architecture is well-suited for building developer environment configuration extensions. The framework's two-phase lifecycle (customization → lockdown), type-safe generics, and existing bootstrap step system provide all the infrastructure needed — no framework modifications required.

**Architecture decision:** Three separate addons — `devenv` (devenv.sh environment management), `claudecode` (Claude Code AI assistant configuration), and `devinit` (orchestration with unified `gdev init` wizard). This follows gdev's established pattern of one addon per concern and allows teams to adopt either tool independently.

**Wizard UX:** The "opinionated menu with escape hatches" pattern using charmbracelet/huh. Quick path accepts sensible defaults in one question (<5 seconds); customize path walks 5 form groups (~30 seconds). Every question maps to a CLI flag for CI/non-interactive use. A detection engine pre-populates answers from existing project files (go.mod, package.json, etc.).

**File generation:** Format-matched strategies — Go `text/template` with custom Nix helper functions for devenv.nix, struct marshaling for YAML/JSON (guarantees syntactic validity), `text/template` for CLAUDE.md, `embed.FS` copy for skills/rules. A unified `GeneratedFile` pipeline handles atomic writes and post-generation validation (`nix-instantiate --parse`, JSON round-trip, `bash -n`).

**Migration:** SHA256 hash tracking per generated file enables safe updates. Machine-owned files (devenv.yaml, .envrc) are regenerated freely. Human-edited files use per-format strategies: section markers for CLAUDE.md, three-way merge for settings.json, library versioning for skills. devenv.nix is never auto-overwritten due to the impossibility of merging arbitrary Nix expressions.

**Key risks and limitations:**
- gdev is pre-stable (v0.14.0) with no external documentation — the addon API may change
- devenv.nix generation via text/template is inherently fragile for complex Nix expressions; validation catches syntax errors but not semantic ones
- The two-tier skill library (embedded + remote git) adds operational complexity for team management
- Monorepo support is minimal — each directory runs its own init independently

**Depth checklist gaps accepted:** The modules inventory (gdev-modules-inventory-research.md) lacks failure mode coverage, and the Claude Code config surface area (claude-code-config-surface-area-research.md) lacks concrete examples — both gaps are substantively covered by their corresponding design documents.

**Deliverables:** 5 research reports (28+ source docs saved), 6 design documents (including two detailed addon designs at 1505 and 1776 lines with Go code sketches, template examples, and complete wizard flows). The design documents are sufficient to begin implementation.
