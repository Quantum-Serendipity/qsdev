# Tasks: gdev Team Configuration & Onboarding

## Phase 1: Scoping & Initial Research

### Pending

### Active

### Completed
- [x] **Team configuration sharing models** -- How should gdev preferences propagate across a team? Profile inheritance, project-level config files, shared config packages. Prior art from editorconfig, eslint, biome, renovate.
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: Surveyed 6 patterns across 10+ tools. Recommends three-layer hierarchy. See team-config-sharing-research.md.

- [x] **Developer onboarding workflow** -- What happens when a new engineer runs `gdev init` on an existing project? Detection vs local setup, clone-to-productive gap measurement, UX audit.
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: Designed four onboarding modes with detection engine. Target: 3 commands, 2 minutes. See developer-onboarding-research.md.

- [x] **Configuration versioning and drift** -- Binary vs config version mismatch, template update propagation, config format migrations, team member version skew.
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: Three-axis versioning, Terraform-style constraints, migration chain, ratchet strategy. See config-versioning-drift-research.md.

- [x] **Standards enforcement in CI** -- `gdev check`/`gdev validate` command design. Verify required tools, config state, security hardening, pre-commit hooks.
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: gdev check with 5 categories, 4 output formats, auto-fix mode. See standards-enforcement-ci-research.md.

- [x] **Prior art survey** -- Deep comparison of Nx generators, Yeoman, Copier, devcontainer features, mise, proto for team-level developer tooling patterns.
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: 7 tools analyzed, 10 cross-cutting patterns extracted, adoption priorities mapped. See prior-art-survey-research.md.

- [x] **Consulting-specific lifecycle** -- Project teardown, archival, client-specific profiles, compliance evidence generation, re-engagement workflows.
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: Client profiles, teardown command, archival format, compliance evidence. See consulting-lifecycle-research.md.
