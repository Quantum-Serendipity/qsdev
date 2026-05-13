# Research Summary: gdev DX Polish & Observability

## Overview

Research DX improvements and observability capabilities that would make gdev feel like a complete, polished developer platform rather than a collection of configs. Covers eight areas: agentic session observability (claude-code-otel, cost tracking), task runner integration (devenv scripts vs mise vs just vs Taskfile), git workflow automation beyond pre-commit, multi-project environment switching, error recovery and self-healing (gdev repair/fix), shell integration ergonomics (starship, aliases, status), dependency freshness/update workflows, and critical evaluation of what NOT to include. Goal: identify features that genuinely reduce friction vs features that add complexity.

Context: gdev is a Go CLI that bootstraps secure development environments for a consulting firm. It generates devenv.nix, CLAUDE.md, settings.json, pre-commit hooks, CI workflows, etc. for 27 language ecosystems. Engineers work across many client projects.

## Topics

- **Agentic Session Observability** -- Complete. Claude Code has native OTEL support (April 2026). Include as optional profile-driven config for consulting firms with client billing needs. Not a default. See [observability-research.md](observability-research.md).

- **Task Runner Integration** -- Complete. devenv 2.0+ task system has parallel execution, dependency ordering, lifecycle hooks, caching, and process integration. Do NOT add a separate task runner. Generate devenv task definitions for common operations. See [task-runner-research.md](task-runner-research.md).

- **Git Workflow Automation** -- Complete. Four features to include: branch naming enforcement, PR template generation, commit ticket extraction (opt-in), automated PR labels. Two to exclude: merge queue config (API not files), release automation (complexity, already rejected in Phase 12). See [git-workflow-research.md](git-workflow-research.md).

- **Multi-Project Environment Switching** -- Complete. devenv 2.0 hook solves core switching. Genuine gaps: cross-project status view (`gdev projects`), SecretSpec credential management across client engagements. See [environment-switching-research.md](environment-switching-research.md).

- **Error Recovery and Self-Healing** -- Complete. `gdev doctor` (read-only diagnostic) + `gdev repair` (conservative auto-fix) pattern. Four failure categories mapped with detection and recovery strategies. See [error-recovery-research.md](error-recovery-research.md).

- **Shell Integration and Ergonomics** -- Complete. Include: starship config generation (opt-in), gdev env vars in devenv.nix, `gdev info` quick-status, devenv enterShell notification. Exclude: aliases, separate gdev shell hook. See [shell-integration-research.md](shell-integration-research.md).

- **Dependency Freshness and Update Workflow** -- Complete. Include: thin `gdev outdated` wrapper, `gdev update` for self+configs+devenv. Exclude: unified analysis, breaking change detection, application dependency updates. See [dependency-freshness-research.md](dependency-freshness-research.md).

- **What NOT to Include** -- Complete. Ten features explicitly rejected: task runner, container management, CI execution, deployment, code scaffolding, IDE config (except Claude Code), OTEL infrastructure, package manager installation, Git server API, vulnerability database. Three-test decision framework established. See [what-not-to-include-research.md](what-not-to-include-research.md).

## Open Questions

- Should `gdev projects` scan a configurable list of directories or discover projects automatically? (Auto-discovery could be slow on large filesystems.)
- Should gdev's OTEL profile include a recommended free-tier collector (Grafana Cloud free) or only support self-hosted?
- How does `gdev repair` interact with the tool lifecycle system (`gdev enable/disable`)? Should repair reinstall disabled tools?

## Conclusions

### The Core Insight

gdev's value proposition is **curation and configuration, not runtime behavior**. The features that make it feel like a polished platform are not new capabilities -- they are completions of the existing file-generation and diagnostic model. The strongest recommendations all follow the same pattern: generate one more file, add one more diagnostic check, or thin-wrap one more ecosystem command.

### Priority Matrix: What to Build

Features are ranked by friction-reduction value (how much developer pain they eliminate) against implementation complexity (effort relative to existing gdev infrastructure).

#### Tier 1: High Value, Low Complexity (Include in Existing Phases)

| Feature | Value | Complexity | Where It Fits |
|---------|-------|------------|---------------|
| `gdev doctor` (diagnostic checks) | Very High | Medium | Phase 9 (already planned, expand scope) |
| `gdev repair` (conservative auto-fix) | Very High | Medium | Phase 9 companion |
| PR template generation | High | Very Low | Phase 5 (file generation, one template) |
| Branch naming enforcement | High | Low | Phase 5 (one more git hook) |
| `gdev info` (quick status) | High | Low | Phase 9 (reads config, no evaluation) |
| gdev env vars in devenv.nix | High | Very Low | Phase 3 (add to devenv addon template) |

#### Tier 2: Medium Value, Low-Medium Complexity (Include in Later Phases)

| Feature | Value | Complexity | Where It Fits |
|---------|-------|------------|---------------|
| Starship config generation | Medium | Low | Phase 3 (devenv addon, opt-in) |
| devenv enterShell notification | Medium | Very Low | Phase 3 (one task in devenv.nix) |
| Commit ticket extraction | Medium | Low | Phase 5 (prepare-commit-msg hook) |
| Automated PR labels | Medium | Low | Phase 5 (labeler.yml + workflow) |
| `gdev outdated` (thin wrapper) | Medium | Medium | Phase 12 (iterate detected ecosystems) |
| `gdev update` (self+configs+devenv) | Medium | Medium | Phase 8 or 10 (coordinated update) |
| devenv task generation (per-ecosystem) | Medium | Medium | Phase 3 (build/test/lint tasks) |

#### Tier 3: Situational Value, Medium Complexity (Include as Profile-Driven Options)

| Feature | Value | Complexity | Where It Fits |
|---------|-------|------------|---------------|
| OTEL env var configuration | Medium (consulting) | Low | Phase 4 (Claude Code addon, profile-gated) |
| `gdev projects` (cross-project view) | Medium (multi-project) | Medium | New command, Phase 12+ |
| SecretSpec credential templates | Medium-High (multi-client) | Medium | Phase 3 (devenv addon) |

#### Explicitly Excluded (The "Won't Have" List)

| Feature | Reason |
|---------|--------|
| Standalone task runner (just/Taskfile/mise) | devenv tasks are sufficient |
| Docker/container management | devenv process manager handles this |
| CI/CD pipeline execution | nektos/act exists |
| Deployment automation | Terraform/Pulumi territory |
| Project scaffolding | Ecosystem-native tools do this |
| IDE config beyond Claude Code | Too personal, too variable |
| OTEL infrastructure (collector/Grafana) | Infrastructure ops, not dev env |
| Package manager installation | devenv/Nix provides this |
| Git server API integration | Terraform has GitHub/GitLab providers |
| Vulnerability database | OSV.dev, GitHub Advisory DB exist |

### The Decision Framework

For any future proposed feature, apply three tests:

1. **Is there a purpose-built tool?** If yes, integrate (generate config) rather than reimplement.
2. **Is it file generation or runtime behavior?** gdev generates files. Runtime behavior requires strong justification.
3. **Does it compound with existing features?** Good features multiply value of existing capabilities. Orthogonal features add maintenance without synergy.

### What "Polished" Actually Means

A polished gdev is not one with more features. It is one where:

- `gdev init` produces a complete, working environment in one command (already designed)
- `gdev doctor` tells you when something is wrong (expand Phase 9)
- `gdev repair` fixes it without manual intervention (new, companion to doctor)
- `gdev info` tells you where you are and what's active (new, lightweight)
- `gdev outdated` gives a cross-ecosystem freshness view (new, thin wrapper)
- `gdev update` keeps gdev infrastructure current (new, coordinated)
- Every generated file includes appropriate git workflow automation (expand Phase 5)
- The prompt tells you what environment you're in (starship integration)
- Switching between client projects "just works" (devenv 2.0 already handles this)

That is 6 new commands/features on top of the existing 17-phase plan. Not a new product direction -- a completion of the existing vision.
