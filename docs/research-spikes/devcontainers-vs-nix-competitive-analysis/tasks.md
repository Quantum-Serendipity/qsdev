# Tasks: Dev Containers vs Nix Competitive Analysis

## Phase 1: Scoping & Initial Research

### Pending

### Active

### Completed
- [x] **Review existing corpus** — Read nix-consulting-environments spike's devcontainer analysis, Docker isolation comparison, and objections doc to establish baseline
  - Priority: high
  - Estimate: small
  - Outcome: success
  - Completed: 2026-03-20
  - Notes: Read docker-vs-nix-isolation-comparison.md and nix-vs-docker-comparison.md from prior spike. Established baseline understanding of Docker vs Nix isolation models, performance tradeoffs, and credential handling differences.

- [x] **Dev Containers deep dive** — Architecture, devcontainer.json spec, Features system, lifecycle hooks, IDE lock-in, multi-container setups
  - Priority: high
  - Estimate: medium
  - Outcome: success
  - Completed: 2026-03-20
  - Notes: Full report at `devcontainers-research.md`. 8 source docs saved to `docs/`.

- [x] **GitHub Codespaces deep dive** — Pricing model, prebuilds, org management, secrets handling, offline limitations, consulting-relevant constraints
  - Priority: high
  - Estimate: medium
  - Outcome: success
  - Completed: 2026-03-20
  - Notes: Full report at `codespaces-research.md`. 7 source docs saved to `docs/`.

- [x] **Coder deep dive** — Self-hosted model, template system, workspace provisioning, enterprise features, consulting fit
  - Priority: high
  - Estimate: medium
  - Outcome: success
  - Completed: 2026-03-20
  - Notes: Full report at `coder-research.md`. 13 source docs saved to `docs/`.

- [x] **DevPod deep dive** — Provider model, local/cloud flexibility, open-source positioning, maturity level
  - Priority: medium
  - Estimate: medium
  - Outcome: success
  - Completed: 2026-03-20
  - Notes: Full report at `devpod-research.md`. 8 source docs saved. Critical finding: unmaintained since mid-2025.

- [x] **Nix-adjacent alternatives survey** — Devbox, Flox, pixi — tools that wrap Nix or compete in the same space
  - Priority: medium
  - Estimate: medium
  - Outcome: success
  - Completed: 2026-03-20
  - Notes: Full report at `nix-adjacent-alternatives-research.md`. 11 source docs saved.

- [x] **Consulting scenario matrix** — Map each tool against consulting-specific requirements: multi-client isolation, onboarding speed, credential separation, offline capability, client-imposed constraints
  - Priority: high
  - Estimate: medium
  - Outcome: success
  - Completed: 2026-03-20
  - Notes: Eight-dimension comparison matrix at `consulting-scenario-matrix.md`. Five scenario recommendations for different consulting firm profiles.

- [x] **Craft the crisp answer** — Write the 2-sentence "why not devcontainers?" response plus the detailed decision framework
  - Priority: high
  - Estimate: small
  - Outcome: success
  - Completed: 2026-03-20
  - Notes: Crisp answer and decision framework written in Conclusions section of `research.md`. Covers when to lead with Nix, when to reach for containers, and six key findings.
