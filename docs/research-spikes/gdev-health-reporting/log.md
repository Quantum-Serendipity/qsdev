# Research Log: gdev Health & Compliance Reporting

## 2026-05-12 — Spike Created
- **Type**: decision
- **Status**: success
- **Depth**: surface
- **Summary**: Spike initialized. Research project health and compliance reporting capabilities for gdev.
- **Next**: Define research question and create Phase 1 tasks.

## 2026-05-12 — Phase 1 Complete: All 7 Research Tasks
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [OpenSSF Scorecard README](https://github.com/ossf/scorecard) -> `docs/openssf-scorecard-readme.md`
  - [Scorecard v6 2026 Roadmap](https://github.com/ossf/scorecard/pull/4952) -> `docs/scorecard-v6-2026-roadmap.md`
  - [Scorecard Monitor](https://github.com/ossf/scorecard-monitor) -> `docs/scorecard-monitor-readme.md`
  - [npm audit docs](https://docs.npmjs.com/cli/v11/commands/npm-audit/) -> `docs/npm-audit-docs.md`
  - [govulncheck docs](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck) -> `docs/govulncheck-docs.md`
  - [Safety CLI output formats](https://docs.safetycli.com/safety-2/safety-cli-2-scanner/output-formats) -> `docs/safety-cli-output-formats.md`
  - [SARIF v2 specification](https://docs.oasis-open.org/sarif/sarif/v2.0/sarif-v2.0.html) -> `docs/sarif-v2-specification.md`
  - [OWASP ASVS](https://github.com/OWASP/ASVS) -> `docs/owasp-asvs-readme.md`
  - [shields.io endpoint badge](https://shields.io/badges/endpoint-badge) -> `docs/shields-io-endpoint-badge.md`
- **Summary**: Completed all 7 Phase 1 research tasks in a single session:
  1. **Prior art survey** of 6 DevSecOps tools. Extracted universal patterns (JSON output, severity levels, exit codes, summary-then-detail, remediation hints) and anti-patterns (paywalled JSON, unstable schemas, inconsistent exit codes).
  2. **Compliance posture model** with three-layer assessment (defense coverage, config health, dependency health), weighted scoring (following Scorecard's risk-based model), and dual-track evaluation (numeric score + conformance PASS/FAIL). Full Go types.
  3. **Status command UX** following flutter doctor's hierarchical check pattern with progressive disclosure. Performance split: local checks (< 1s) vs network scans (cached, 5-30s on demand).
  4. **Machine-readable output formats** prioritized: JSON (canonical, versioned), SARIF (Code Scanning), badge JSON (shields.io), JUnit (optional). Consumer matrix mapping formats to use cases.
  5. **Drift detection** covering 6 categories, all local-only (< 100ms). Builds on existing SHA256 hash tracking from migration strategy design.
  6. **Team aggregation** via CI artifact collection. Markdown dashboard with scores, trends, and auto-generated GitHub issues for degraded posture.
  7. **Badge generation** via static JSON file in repo (CI-generated), consumed by shields.io endpoint badge.
- **Next**: Phase 2 design tasks synthesize research into implementation-ready specifications. Phase 1 reports are already design-grade (include Go types, terminal mockups, CI pipeline examples), so Phase 2 primarily synthesizes and resolves remaining open questions.
