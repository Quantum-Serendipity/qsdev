# Research Summary: gdev Team Configuration & Onboarding

## Overview

Research into team configuration management, developer onboarding workflows, standards propagation, configuration versioning/drift, CI validation, and consulting-specific lifecycle needs for the gdev CLI tool that bootstraps secure development environments. gdev is a Go CLI for a consulting firm where engineers work across many client projects, each getting a gdev-managed development environment with security hardening, Claude Code configuration, pre-commit hooks, and CI workflows.

This spike builds on prior research in `gdev-extension-design` (addon architecture, migration strategy, wizard UX) and the `qsdev` implementation plan. It focuses on the team/org-level concerns that sit above the per-project generation layer.

## Topics

- **Team Configuration Sharing Models** -- Complete. Surveyed 6 patterns across 10+ tools (EditorConfig, ESLint, Biome, Renovate, Copier, Projen, Nx, Dev Containers, mise, proto). Recommends three-layer hierarchy: compiled org defaults -> `.qsdev.yaml` in repo -> `.qsdev.local.yaml` gitignored. See [team-config-sharing-research.md](team-config-sharing-research.md).

- **Developer Onboarding Workflow** -- Complete. Designed four onboarding modes (Create, Join, Update, Repair) with detection engine. Target: 3 commands, under 2 minutes for returning engineers. Mapped machine-specific vs project-specific setup. See [developer-onboarding-research.md](developer-onboarding-research.md).

- **Configuration Versioning and Drift** -- Complete. Designed three-axis versioning (binary, config schema, template), Terraform-style version constraints (`gdev_version`), incremental migration chain, and version ratchet strategy to prevent downgrades. See [config-versioning-drift-research.md](config-versioning-drift-research.md).

- **Standards Enforcement in CI** -- Complete. Designed `qsdev check` command with 5 check categories, 4 output formats (human, JSON, SARIF, JUnit), auto-fix mode for safe changes. Distinguished from `qsdev devenv doctor` (machine state) vs `qsdev check` (project compliance). See [standards-enforcement-ci-research.md](standards-enforcement-ci-research.md).

- **Prior Art Survey** -- Complete. Deep analysis of 7 tools (Nx, Yeoman, Copier, Projen, Dev Containers, mise, proto) plus cross-cutting pattern extraction. Identified 10 patterns, recommended adoption priorities. See [prior-art-survey-research.md](prior-art-survey-research.md).

- **Consulting-Specific Lifecycle** -- Complete. Designed client-specific profiles with compliance level mapping, `qsdev teardown` with three profiles (quick/default/compliance), project archival with re-engagement workflow, and compliance evidence generation. See [consulting-lifecycle-research.md](consulting-lifecycle-research.md).

## Open Questions

- Should `.qsdev.yaml` support an `extends` field referencing a remote config repo (like Renovate's preset repos)? This would allow cross-repo standardization without recompiling the binary, but adds network dependency and complexity.
- How should gdev handle monorepo scenarios where subdirectories need different configurations? The current design operates at the project root only.
- Should compliance evidence reports be cryptographically signed for non-repudiation, or is integrity verification (hash-based) sufficient?
- What is the right mechanism for distributing new profiles without requiring a full binary rebuild? Possible approaches: WASM plugins, HTTPS-fetched profile registry, or accept that binary updates are the distribution channel.

## Conclusions

### Core Design: Three-Layer Configuration Hierarchy

gdev should implement a three-layer configuration hierarchy that combines the best patterns from the ecosystem:

1. **Org defaults (compiled into binary)** -- Non-negotiable security baselines, default profiles, and infrastructure settings. These work offline, cannot be bypassed, and version-lock with the binary. Updated quarterly via binary releases.

2. **Project config (`.qsdev.yaml` in repo)** -- Profile selection, language/service overrides, client-specific settings, gdev version constraint. Travels with the repo via git. Provides the team standard.

3. **Local overrides (`.qsdev.local.yaml`, gitignored)** -- Developer-specific preferences (editor tools, permission levels, extra packages). Never committed.

Resolution: deep merge, later layers override earlier, with security level acting as a floor that cannot be lowered by project or local overrides.

### Onboarding: 3 Commands, 2 Minutes

The clone-to-productive gap for a returning engineer should be: `git clone`, `cd project`, `qsdev init` (detects `.qsdev.yaml`, verifies state, does local setup), `devenv shell`. A detection engine determines one of four modes (Create/Join/Update/Repair) and adapts the UI accordingly. Machine-specific setup (`qsdev devenv setup`) runs once per machine, not per project.

### Version Compatibility: Terraform Pattern

`.qsdev.yaml` includes `gdev_version: ">= 0.15.0"` (semver constraint) and `version: 1` (config schema integer). Binary version is checked before any operation. Config migrations chain incrementally (v1->v2->v3). Older binaries refuse to downgrade files generated by newer binaries (ratchet strategy).

### CI Enforcement: `qsdev check`

A read-only validation command that verifies project compliance against org policy. Checks: binary compatibility, config integrity, required tools, generated file state, security hardening. Outputs human-readable, JSON, SARIF, or JUnit. Exit code 0 (pass) or 1 (fail). Auto-fix mode for safe additive changes. Complemented by `qsdev devenv doctor` (machine health) which serves a different purpose.

### Consulting Lifecycle: Teardown, Archive, Evidence

Client-specific profiles encode compliance requirements as security level floors. `qsdev teardown` handles clean project exit (archive config, GC Nix store, revoke tokens, remove trust). Archives preserve enough state for re-engagement months later. `qsdev evidence` generates machine-readable compliance evidence mapping to SOC2/HIPAA controls.

### Key Risks

- **Profile distribution without binary rebuild** is the biggest open question. Compiled-in profiles are simple and reliable but create a release bottleneck. A remote profile registry would decouple these but adds complexity and a network dependency.
- **Config migration chain maintenance** grows linearly with schema versions. Need a clear support window (e.g., current + 2 previous versions) and migration test coverage.
- **Trust mechanism adoption** depends on mise-style UX that developers are not yet accustomed to. Must be low-friction with escape hatches for automated environments.
