# Research Summary: gdev Health & Compliance Reporting

## Overview

Research into project health and compliance reporting capabilities for gdev -- a Go CLI that bootstraps secure development environments with 27 language ecosystems, 16+ toggleable security tools, and 10 defense layers. Investigated what a `gdev report` / `gdev status` command should look like, how to present compliance posture for a consulting firm managing multiple client projects, machine-readable output formats (JSON/SARIF/OWASP ASVS), configuration drift detection, team-level aggregation across repos, prior art from DevSecOps tools, and security posture badge generation.

## Topics

- **Prior Art: DevSecOps Reporting Tools** -- Complete. Surveyed 6 tools (OpenSSF Scorecard, npm audit, cargo audit, Python Safety, govulncheck, OWASP Dependency-Check) for output formats, scoring models, UX patterns, and CI integration approaches. Extracted universal patterns (JSON output, severity levels, exit codes, summary-then-detail) and anti-patterns (paywalled JSON, unstable schemas). See [prior-art-research.md](prior-art-research.md).

- **Compliance Posture Model** -- Complete. Three-layer assessment model: defense coverage (weighted scoring like Scorecard), configuration health (hash-based drift detection), and dependency health (vulnerability counts per ecosystem). Dual-track evaluation: numeric score (0-100 with letter grades) AND conformance labels (baseline/enhanced PASS/FAIL). Full Go type definitions. See [compliance-posture-model-research.md](compliance-posture-model-research.md).

- **Status Command UX** -- Complete. Terminal output design following flutter doctor pattern (hierarchical checks with three-state indicators). Progressive disclosure from `--quiet` through default summary to `--verbose` detail. Subcommands for focused views (`gdev status tools`, `gdev status defense`). Performance model: fast path (< 1s, local only) vs slow path (5-30s, network scan). See [status-command-ux-research.md](status-command-ux-research.md).

- **Machine-Readable Output Formats** -- Complete. JSON as canonical format (versioned schema from day one), SARIF 2.1.0 for GitHub Code Scanning (maps discrete findings, not aggregate scores), shields.io badge JSON, JUnit XML for CI test infrastructure. Consumer matrix mapping formats to use cases. ASVS alignment for audit evidence. See [machine-readable-output-research.md](machine-readable-output-research.md).

- **Configuration Drift Detection** -- Complete. Six drift categories: unauthorized file modification, version drift, tool availability drift, section marker integrity, lock file drift, pre-commit hook drift. All detection is local-only (< 100ms). Builds on gdev's existing SHA256 hash tracking from the migration strategy. See [drift-detection-research.md](drift-detection-research.md).

- **Team-Level Reporting** -- Complete. CI artifact aggregation as recommended architecture (no new infrastructure). Markdown summary dashboard with score table, trend tracking, and attention-required alerts. GitHub issue auto-generation when scores drop (scorecard-monitor pattern). See [team-reporting-research.md](team-reporting-research.md).

- **Badge Generation** -- Complete. Static file in repo via CI (recommended), GitHub Pages endpoint, or offline SVG generation. shields.io endpoint badge protocol. Color mapping from score to badge color. See [badge-generation-research.md](badge-generation-research.md).

## Open Questions

- Should the scoring weights (defense 40%, config 30%, deps 30%) be configurable per-profile, or hardcoded with escape hatch?
- How should `gdev status` handle the first run before any tools are enabled (all zeros vs "not initialized" state)?
- Should vulnerability scan results be cached in `.gdev/cache/` (gitignored) or regenerated on every `gdev status --scan`?
- Is JUnit output worth building, or does SARIF cover the CI integration need sufficiently?
- Should `gdev team-report` be a separate binary/command or a subcommand of gdev?

## Conclusions

### Core Architecture Decision: Three-Layer Posture Model

gdev's health reporting should assess three independent dimensions:
1. **Defense Coverage** (40% weight): What percentage of applicable security layers are enabled and correctly configured?
2. **Configuration Health** (30% weight): Are generated configs current, intact, and matching the latest gdev version?
3. **Dependency Health** (30% weight): Are lock files present and valid? Are there known vulnerabilities?

This produces both a **numeric score** (0-100 with letter grade) for quick assessment and **conformance labels** (baseline PASS/FAIL, enhanced PASS/FAIL) for binary compliance checks. The dual-track model follows Scorecard v6's evolution from pure numeric scoring to conformance evaluation.

### Command Design: `gdev status`

The primary interface is `gdev status` with progressive disclosure:
- Default: colored summary with section scores, defense layer checklist, config health, vulnerability counts
- `--verbose`: per-check detail with remediation hints
- `--json`: complete PostureReport for CI/dashboards
- `--sarif`: security findings for GitHub Code Scanning
- `--audit-level <severity>`: exit-code-based CI gate

Performance split: defense and config checks are local-only (< 1s), dependency scanning is network-dependent (5-30s, cached by default).

### Output Format Priority

1. JSON (canonical, versioned schema) -- all other formats derive from this
2. Terminal text (developer primary interface)
3. SARIF 2.1.0 (GitHub Code Scanning integration)
4. Exit codes with `--audit-level` (CI gates)
5. Badge JSON (shields.io endpoint for README badges)
6. JUnit XML (optional, for Jenkins/GitLab CI)

### Drift Detection: Six Categories, All Local

Drift detection builds on gdev's existing SHA256 hash tracking and adds: version drift, tool availability drift, section marker integrity checks, lock file freshness, and pre-commit hook status verification. All checks are local-only and complete in < 100ms.

### Team Aggregation: CI Artifacts, Not a Server

Multi-repo aggregation via CI artifact collection (each project generates posture JSON as build artifact, central job collects and aggregates). No new infrastructure required. Markdown summary dashboard with score table, trends, and auto-generated GitHub issues for score drops.

### Key Design Principles (derived from prior art)

1. **Version the JSON schema from day one** -- cargo-audit's unstable JSON broke downstream tools
2. **Never paywall machine-readable output** -- Safety CLI's JSON-behind-API-key blocks automation
3. **Support `NO_COLOR`** -- standard accessibility convention
4. **Exit codes match severity thresholds** -- the universal CI gate pattern
5. **Include remediation hints** -- don't just report problems, suggest fixes
6. **Distinguish "disabled by choice" from "disabled by oversight"** -- scoring should account for intentionality
7. **Conformance is binary, score is nuanced** -- both are needed for different audiences
