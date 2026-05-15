<!-- Source: https://raw.githubusercontent.com/lateos-ai/npm-scan/main/README.md -->
<!-- Retrieved: 2026-05-15 -->

# @lateos/npm-scan README

[![npm version](https://img.shields.io/npm/v/@lateos/npm-scan?style=flat-square)](https://www.npmjs.com/package/@lateos/npm-scan)
[![License](https://img.shields.io/badge/license-Apache%202.0%20%2B%20Commons%20Clause-blue?style=flat-square)](LICENSING.md)
[![Node](https://img.shields.io/badge/node-%3E%3D18-brightgreen?style=flat-square)](package.json)
[![Tests](https://img.shields.io/badge/tests-324%20passing-brightgreen?style=flat-square)](https://github.com/lateos-ai/npm-scan)
[![Coverage](https://img.shields.io/badge/coverage-90%25-brightgreen?style=flat-square)](https://github.com/lateos-ai/npm-scan)
[![Docker](https://img.shields.io/badge/docker-lateos%2Fnpm--scan-2496ED?style=flat-square&logo=docker)](https://hub.docker.com/r/lateos/npm-scan)
[![Sigstore](https://img.shields.io/static/v1?label=Sigstore&message=Provenance&color=green&style=flat-square&logo=sigstore)](https://github.com/lateos-ai/npm-scan/actions/workflows/publish.yml)

**Modern supply chain security for the npm ecosystem.**
Static + behavioral analysis that catches what npm audit, Snyk, and Socket miss — obfuscated payloads, credential stealers, conditional triggers, sandbox evasion, and worm-like propagation.

---

## The Problem

The 2025-2026 wave of npm supply chain attacks proved that traditional tooling is no longer enough.

Attackers have moved past simple typosquatting. They now ship **obfuscated preinstall hooks**, **credential harvesters hidden behind environment detection**, **dormant backdoors with time-based activation**, and **worm-style transitive propagation** that spreads through peer dependencies.

**npm audit** checks known CVEs. **Snyk** scans for vulnerabilities. **Socket** looks at package behavior. None of them were designed for the generation of attacks that emerged in 2025 — attacks that look benign until they reach production.

**@lateos/npm-scan** was built for this moment.

---

## Why @lateos/npm-scan?

| Capability | npm audit | Snyk | Socket | **@lateos/npm-scan** |
|---|---|---|---|---|
| Known CVE matching | Yes | Yes | No | Yes |
| Static analysis | No | Yes | Yes | Yes |
| Obfuscated payload detection | No | No | No | Yes |
| AST-level heuristic analysis | No | No | No | Yes |
| Runtime behavioral sandbox | No | No | Yes | Yes |
| Conditional trigger detection (ATK-009) | No | No | No | Yes |
| Sandbox evasion detection (ATK-010) | No | No | No | Yes |
| Transitive worm propagation (ATK-011) | No | No | No | Yes |
| Attack taxonomy (ATK series) | No | No | No | Yes |
| SBOM output (CycloneDX + SPDX) | No | Yes | No | Yes |
| SARIF v2.1 (GitHub Code Scanning) | No | No | No | Yes |
| NIST 800-161 compliance reporting | No | No | No | Yes |
| EU CRA compliance reporting | No | No | No | Yes |
| SIEM export (CEF / ECS / Sentinel / QRadar) | No | No | No | Yes |
| Runs entirely locally — no telemetry | Yes | No | No | Yes |
| Policy-as-code (YAML allowlists) | No | No | No | Yes |

> **Privacy first.** All scanning happens on your machine. No code leaves your environment. No telemetry. No cloud dependency.

---

## Key Features

- **Heuristic static analysis** — AST-level inspection catches obfuscation, eval chains, env probing, and suspicious lifecycle scripts that regex-based tools miss
- **Behavioral detection** — Identifies conditional triggers (time-based, CI-aware), sandbox evasion, and dormant activation patterns
- **ATK attack taxonomy** — 11 classified attack types with NIST 800-161 mappings — versioned, documented, and PR-able
- **SBOM generation** — CycloneDX 1.5 and SPDX 2.3 with findings embedded as vulnerabilities
- **SARIF output** — GitHub Advanced Security / CodeQL compatible SARIF v2.1 — shows findings directly in Security tab
- **Compliance reporting** — NIST SP 800-161 traceability matrix + EU Cyber Resilience Act mapping (free tier)
- **SIEM export** — Splunk CEF, Elastic ECS, Microsoft Sentinel, IBM QRadar formats (premium)
- **Policy-as-code** — YAML/JSON policy engine with allowlists, severity overrides, suppressions, and fail-on thresholds
- **Docker + GitHub Action** — Multi-arch images, one-command Compose pipeline, PR scan action
- **Zero telemetry** — No data leaves your machine. No cloud. No callbacks.
- **Local scan history** — SQLite-backed persistence, zero external dependencies
- **Pre-commit hook** — Block threats before commit — one-liner install, scans package-lock.json changes
- **Yarn + pnpm support** — scan-lockfile parses yarn.lock and pnpm-lock.yaml alongside package-lock.json

---

## Quick Start

```bash
# Install globally
npm install -g @lateos/npm-scan

# Scan a single package
npm-scan scan lodash

# Scan your lockfile
npm-scan scan-lockfile

# View latest scans
npm-scan report
```

**No install? No problem:**

```bash
npx @lateos/npm-scan scan commander
```

---

## Docker

```bash
# Pull and run a single scan — no Node.js or npm required
docker run --rm lateos/npm-scan:cli scan lodash

# Full pipeline with persistent storage and Compose
docker compose --profile pipeline up -d
```

No Node.js. No npm install. No global packages. Works on any system with Docker — CI servers, air-gapped environments, Kubernetes clusters. Multi-arch images for linux/amd64 and linux/arm64.

---

## Government & SOC 2 Ready

| Feature | SOC 2 Controls | NIST 800-161 | STIG/FedRAMP Alignment |
|---------|-------|--------------|--------------|
| Audit logs (--audit-log) | CC6.8 | AU-2 | Yes |
| FIPS crypto (--fips) | CC6.1 | SC-13 | Yes |
| STIG report (--stig) | CC7.3 | RA-5 | Yes |
| Offline cache (--cache-dir) | A1.2 | SC-8 | Yes |
| Sigstore provenance | CC6.2 | SI-7 | Yes |
| SBOM (SPDX/CycloneDX) | CC7.4 | SA-10 | Yes |

---

## Detection Capabilities (ATK Taxonomy)

| ID | Attack Class | Detection Method | Severity | NIST 800-161 |
|---|---|---|---|---|
| **ATK-001** | Malicious lifecycle scripts (preinstall, postinstall, install) | Static | high | SR-3.1 |
| **ATK-002** | Obfuscated payload delivery (hex, base64, eval chains) | Static | medium | SR-4.2 |
| **ATK-003** | Credential harvesting (env vars, .npmrc, SSH keys) | Static + Dynamic | high | SR-5.3 |
| **ATK-004** | Persistence via editor/config dirs (.vscode, .claude, .cursor) | Static | high | SR-6.4 |
| **ATK-005** | Network exfiltration (GitHub API, DNS tunneling, HTTP C2) | Static + Dynamic | critical | SR-7.5 |
| **ATK-006** | Dependency confusion / namespace squatting | Static (lockfile) | medium | SR-2.2 |
| **ATK-007** | Typosquatting (edit-distance matching) | Static | low | SR-2.1 |
| **ATK-008** | Tarball tampering (published != source) | Static | high | SR-8.1 |
| **ATK-009** | Conditional/dormant triggers (CI detection, time-based) | Behavioral | high | SR-9.2 |
| **ATK-010** | Sandbox evasion / anti-analysis | Behavioral | medium | SR-10.3 |
| **ATK-011** | Transitive propagation (worm-style lateral spread) | Behavioral | high | SR-11.4 |

---

## Output & Reports

| Format | Availability | Description |
|--------|-------------|-------------|
| JSON | Free | Structured machine-readable findings |
| HTML | Free | Rich HTML report with NIST compliance table, severity badges, control matrix |
| Text | Free | Clean terminal-friendly text report |
| CycloneDX SBOM | Free | Industry-standard SBOM with findings as vulnerabilities |
| SPDX SBOM | Free | SPDX 2.3 document format |
| NIST 800-161 | Free | Control traceability matrix (SR-2.1 to SR-11.4) |
| EU CRA | Free | Cyber Resilience Act article mapping |
| PDF | Premium | Multi-page PDF with title page, findings table, NIST compliance matrix |
| Splunk CEF | Premium | Common Event Format for Splunk ingestion |
| Elastic ECS | Premium | Elastic Common Schema format |
| Microsoft Sentinel | Premium | Sentinel-ready formatted output |
| IBM QRadar | Premium | QRadar DSM-ready format with QID mappings |

---

## Configuration & Advanced Usage

### Policy-as-code

```yaml
# .npm-scan.yml
allowlist:
  - lodash
  - chalk

severity_overrides:
  - id: ATK-001
    severity: medium

suppress:
  - atk_id: ATK-009
  - package: some-package

fail_on: high
```

### Environment variables

| Variable | Description | Default |
|----------|-------------|---------|
| NPM_SCAN_LICENSE_KEY | Premium / enterprise license key | — |
| NPM_SCAN_DATA_DIR | Scan history directory | ./.npm-scan |
| NPM_SCAN_LOG_LEVEL | Log verbosity | info |

---

## Integrations

- GitHub Actions CI (tests across Node 18, 20, 22)
- GitHub Action for downstream users (SARIF upload to Security tab)
- CI/CD pipeline (direct CLI integration)
- Pre-commit hook (husky + lint-staged)
- Docker (multi-arch images, Compose pipeline)

---

## Roadmap & Enterprise Features

### Free tier (shipped)
- All 11 ATK detectors (static + behavioral)
- SBOM output (CycloneDX + SPDX)
- HTML, text, and compliance reports (NIST + EU CRA)
- Policy-as-code engine (YAML)
- Local SQLite scan history
- GitHub Action
- Pre-commit hook (husky + lint-staged)
- Docker images + Compose pipeline
- Watch mode (--watch / --monorepo for auto-rescan)

### Premium (license key)
- PDF compliance reports with NIST traceability matrix
- SIEM export (Splunk CEF, Elastic ECS, Microsoft Sentinel, IBM QRadar)
- Dynamic sandbox (gVisor-based — ATK-008-010)
- Reachability analysis (call graph filtering)

### Enterprise (custom license)
- SAML 2.0 SSO
- REST API + webhooks
- Team RBAC + audit logs
- Helm chart for Kubernetes deployment
- PostgreSQL backend
- SLA-backed priority support

---

## License

Apache-2.0 core + Commons Clause.

## Maintainer

**Roongrunchai Chongolnee** — CISSP, CEH, Cisco Security, AWS Cloud Practitioner. Decade of infrastructure and application security experience at Philips.

Copyright (C) 2026 Lateos
