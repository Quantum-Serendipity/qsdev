<!-- Source: https://github.com/lateos-ai/npm-scan -->
<!-- Retrieved: 2026-05-15 -->

# @lateos/npm-scan — GitHub Repository Page

## Repository Metadata
- **Owner**: lateos-ai
- **Repository Name**: npm-scan
- **Stars**: 4
- **Forks**: 0
- **License**: Apache-2.0 + Commons Clause
- **Primary Languages**: JavaScript (83.6%), Python (13.4%), PLpgSQL (2.1%)
- **Node Version**: 18+
- **Latest Release**: v1.0.0 (May 13, 2026)

## Project Description
Modern supply chain security for npm packages through combined static and behavioral analysis. Detects threats that traditional scanners miss, including obfuscated payloads, credential stealers, conditional triggers, sandbox evasion, and worm-like propagation.

## Core Features

### Detection Capabilities
- 11 classified attack types (ATK-001 through ATK-011)
- AST-level heuristic analysis for obfuscation detection
- Behavioral detection for dormant activation patterns
- NIST 800-161 control mappings

### Output Formats
- JSON, HTML, text reports
- SBOM generation (CycloneDX, SPDX)
- SARIF v2.1 for GitHub Code Scanning
- Compliance reports (NIST, EU CRA)
- SIEM exports (Splunk, Elastic, Sentinel, QRadar - premium)

### Integrations
- GitHub Action for PR scanning
- Docker images (multi-arch: amd64, arm64)
- Pre-commit hooks via Husky
- Direct CLI usage

## Licensing & Tiers

**Free**: All 11 detectors, SBOM, HTML/text reports, policy engine, GitHub Action, Docker support

**Premium** (license key): PDF reports, SIEM export, dynamic sandbox, reachability analysis

**Enterprise**: SSO/SAML, REST API, team RBAC, Kubernetes Helm charts ($10k/year)

## Testing Infrastructure

Uses Node.js native test runner with 33 malicious and 50 clean tarball samples, achieving 90% coverage per badge. Tests cover database operations, detector edge cases, policy handling, lockfile parsing, and CLI integration.

## Maintainer

Roongrunchai Chongolnee, certified security professional (CISSP, CEH) with infrastructure and application security background at Philips.
