# Tasks: Package Supply Chain Security

## Phase 1: Scoping & Initial Research

### Pending

### Active

### Completed
- [x] **Install script sandboxing & runtime protections** — Sandboxing pre/post-install scripts (npm ignore-scripts, Deno permissions), network isolation during builds
  - Priority: medium
  - Estimate: medium
  - Started: 2026-05-12
  - Completed: 2026-05-12
  - Outcome: success
  - Report: `install-sandboxing-research.md`

- [x] **Lock file integrity & reproducible installs** — Lock file best practices, --frozen-lockfile enforcement, hash pinning, reproducible build strategies
  - Priority: medium
  - Estimate: medium
  - Started: 2026-05-12
  - Completed: 2026-05-12
  - Outcome: success
  - Report: `lockfile-integrity-research.md`

- [x] **Landscape survey: per-ecosystem attack surface** — Map major package managers (npm, PyPI, cargo, Go modules, Maven/Gradle, NuGet, RubyGems) and their known supply chain attack vectors (typosquatting, account takeover, malicious install scripts, dependency confusion, etc.)
  - Priority: high
  - Estimate: large
  - Started: 2026-05-12
  - Completed: 2026-05-12
  - Outcome: success
  - Report: `attack-surface-landscape-research.md`

- [x] **Private registries & validated mirrors** — Research tools (Artifactory, Verdaccio, Devpi, Cloudsmith, etc.) that proxy/cache upstream registries and enforce validation policies before serving packages
  - Priority: high
  - Estimate: large
  - Started: 2026-05-12
  - Completed: 2026-05-12
  - Outcome: success
  - Report: `private-registries-research.md`

- [x] **Signature verification & provenance** — SLSA, Sigstore, npm provenance, PyPI Trusted Publishers, cargo-vet, Go module checksums — what's available per ecosystem and how to enforce it
  - Priority: high
  - Estimate: large
  - Started: 2026-05-12
  - Completed: 2026-05-12
  - Outcome: success
  - Report: `signature-provenance-research.md`

- [x] **Organizational tooling & policy enforcement** — Tools like Socket.dev, Snyk, Dependabot, Renovate, OSV Scanner, Scorecard — things that run in CI or as pre-commit gates
  - Priority: high
  - Estimate: large
  - Started: 2026-05-12
  - Completed: 2026-05-12
  - Outcome: success
  - Report: `org-tooling-research.md`

- [x] **Publication age / quarantine gates** — Research "package hold" or quarantine period strategies (Socket.dev approach, configurable hold-back policies, registry-level quarantine)
  - Priority: high
  - Estimate: medium
  - Started: 2026-05-12
  - Completed: 2026-05-12
  - Outcome: success
  - Report: `quarantine-gates-research.md`

## Scope Notes
- NixOS-specific considerations excluded — this research targets non-NixOS systems
