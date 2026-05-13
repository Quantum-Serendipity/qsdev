# Tasks: gdev Health & Compliance Reporting

## Phase 1: Scoping & Initial Research

### Pending

### Active

### Completed
- [x] **Prior art: DevSecOps reporting tools** -- Surveyed OpenSSF Scorecard, npm audit, cargo audit, Python Safety, govulncheck, OWASP Dependency-Check. Extracted universal patterns and anti-patterns.
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: 9 source docs saved to docs/. See [prior-art-research.md](prior-art-research.md).

- [x] **Status/report command UX patterns** -- Designed `gdev status` terminal output with flutter doctor pattern, progressive disclosure, and performance model.
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: Full terminal mockups, CLI flag inventory, CI examples. See [status-command-ux-research.md](status-command-ux-research.md).

- [x] **Compliance posture model** -- Three-layer assessment model (defense/config/deps) with weighted scoring and conformance tracks.
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: Go type definitions, scoring algorithm, conformance check lists. See [compliance-posture-model-research.md](compliance-posture-model-research.md).

- [x] **Machine-readable output formats** -- JSON (canonical, versioned), SARIF 2.1.0, badge JSON, JUnit XML. Consumer matrix.
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: Full format examples with implementation priority. See [machine-readable-output-research.md](machine-readable-output-research.md).

- [x] **Configuration drift detection** -- Six drift categories, all local-only (< 100ms). Builds on existing SHA256 hash tracking.
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: State file schema, detection performance table, remediation matrix. See [drift-detection-research.md](drift-detection-research.md).

- [x] **Team-level reporting & aggregation** -- CI artifact aggregation architecture. Markdown dashboard, trend tracking, GitHub issue generation.
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: Three architecture options evaluated. CI artifacts recommended. See [team-reporting-research.md](team-reporting-research.md).

- [x] **Badge generation** -- Static file via CI (recommended), shields.io endpoint protocol, color mapping, multiple badge variants.
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: CI pipeline examples, JSON schema, README markdown. See [badge-generation-research.md](badge-generation-research.md).

## Phase 2: Synthesis & Design

### Pending
- [ ] **Design: `gdev status` command specification** -- Full command design with subcommands, flags, output formats, and integration points
  - Priority: high
  - Estimate: large
  - Depends: All Phase 1 tasks
  - Notes: Phase 1 research reports are design-grade already; this task synthesizes into implementation-ready spec

- [ ] **Design: Compliance posture data model** -- Go types, scoring algorithm, serialization formats
  - Priority: high
  - Estimate: medium
  - Depends: Compliance posture model, Machine-readable output formats
  - Notes: Go types already drafted in compliance-posture-model-research.md

- [ ] **Design: Drift detection engine** -- Detection algorithm, notification UX, auto-remediation options
  - Priority: medium
  - Estimate: medium
  - Depends: Configuration drift detection

- [ ] **Design: Team aggregation architecture** -- Collection mechanism, storage, dashboard/CLI
  - Priority: medium
  - Estimate: medium
  - Depends: Team-level reporting & aggregation

### Active

### Completed

## Phase 3: Review & Finalization

### Pending
- [ ] **Depth checklist review** -- Run revision cycle on all research reports
  - Priority: high
  - Estimate: medium
  - Depends: All Phase 2 tasks

- [ ] **Write conclusions** -- Final synthesis in research.md
  - Priority: high
  - Estimate: small
  - Depends: Depth checklist review

### Active

### Completed
