# Tasks: gdev DX Polish & Observability

## Phase 1: Scoping & Initial Research

### Pending

### Active

### Completed
- [x] **Agentic session observability** — claude-code-otel and similar: OpenTelemetry integration for tracking Claude Code usage/costs/outcomes, session recording/replay for audit trails, cost tracking per project/client, token usage monitoring.
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: Native OTEL support launched April 2026. Include as optional profile-driven config, not default. See observability-research.md

- [x] **Task runner integration** — Should gdev include a task runner? devenv.sh already has scripts/tasks, mise has task running, just (casey/just) as command runner, Taskfile.yml (go-task).
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: devenv 2.0 task system is sufficient -- do NOT add a separate task runner. Generate devenv task definitions instead. See task-runner-research.md

- [x] **Git workflow automation** — Beyond pre-commit hooks: branch naming conventions, commit message enforcement, PR template generation, changelog automation.
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: 4 features to include (branch naming, PR templates, ticket extraction, PR labels), 2 to exclude (merge queue config, release automation). See git-workflow-research.md

- [x] **Multi-project environment switching** — For consultants working across multiple client projects: how devenv handles multiple projects, direnv per-directory switching, gaps in multi-project workflow.
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: devenv 2.0 hook solves core switching. Gaps: cross-project status view (gdev projects), SecretSpec credential management. See environment-switching-research.md

- [x] **Error recovery and self-healing** — What happens when things break? gdev repair/fix command, broken devenv shell recovery, corrupted config recovery.
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: gdev doctor (diagnostic) + gdev repair (auto-fix) design. 4 failure categories, conservative-by-default repair. See error-recovery-research.md

- [x] **Shell integration and ergonomics** — Making gdev pleasant to use: aliases/abbreviations, status bar integration (starship), quick-info commands.
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: Include starship config gen (opt-in), gdev env vars, gdev info command. Exclude aliases and separate shell hook. See shell-integration-research.md

- [x] **Dependency freshness and update workflow** — Beyond Renovate: gdev outdated to check all deps, coordinated updates, breaking change detection.
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: Include thin gdev outdated wrapper and gdev update (self+configs+devenv). Exclude unified analysis and breaking change detection. See dependency-freshness-research.md

- [x] **What NOT to include** — Features that would bloat the tool or duplicate existing capabilities.
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: 10 features explicitly rejected with rationale. Three-test decision framework. See what-not-to-include-research.md

## Phase 2: Synthesis & Design

### Pending

### Active

### Completed
- [x] **Synthesis: DX feature priority matrix** — Rank all researched features by friction-reduction value vs implementation complexity
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: Priority matrix in research.md conclusions

- [x] **Design: Recommended gdev DX additions** — Concrete command designs, integration points, and implementation sketches
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: Integrated into research.md conclusions and individual reports

## Phase 3: Review & Finalization

### Pending

### Active

### Completed
- [x] **Depth checklist review** — Run revision cycle on all research reports
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: All 8 reports pass depth checklist

- [x] **Write conclusions** — Final synthesis in research.md
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: Full conclusions with priority matrix written
