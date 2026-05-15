<!-- Source: https://raw.githubusercontent.com/lateos-ai/npm-scan/main/LICENSING.md -->
<!-- Retrieved: 2026-05-15 -->

# @lateos/npm-scan — LICENSING

## Model: Apache-2.0 core + Commons Clause premium

### Core (Apache-2.0):
- Static analysis engine, ATK-001-007 detectors, CLI, lockfile scanner, SBOM output (CycloneDX), GitHub Action, Docker images, JSON output, SQLite-backed local storage, basic HTML report.

### Premium (Apache-2.0 + Commons Clause):
- Dynamic sandbox (ATK-008+), advanced compliance reports (PDF, regulatory templates), SIEM connectors, reachability analysis, team dashboard, SSO, audit logs, API/webhooks, on-prem/air-gapped licenses, priority support.

## Commons Clause
The Commons Clause prohibits selling our open core software as a service. See https://commonsclause.com/ for details.

## Feature Flags
Premium features gated by license key validated at runtime. Keys issued per-seat CLI, per-org hosted.

## Note
LICENSING.md claims ATK-008+ are premium (dynamic sandbox), but the actual source code ships ATK-008 through ATK-011 as static regex detectors in the free tier. The dynamic sandbox (gVisor-based) is the premium component, not the static detectors themselves.
