<!-- Source: https://raw.githubusercontent.com/lateos-ai/npm-scan/main/CHANGELOG.md -->
<!-- Retrieved: 2026-05-15 -->

# @lateos/npm-scan — CHANGELOG

All notable changes to [@lateos/npm-scan](https://github.com/lateos-ai/npm-scan).

## [Unreleased]
- scan --file <path> for local .tgz analysis
- scan --fail-on <level> for CI/CD exit codes
- scan --sarif for GitHub Advanced Security
- scan --csv for tabular export
- scan --score-only for dashboards
- Government/SOC 2: --audit-log, --fips, --stig, --cache-dir
- BYOC Helm chart v1.0.0

## [0.9.7] — 2026-05-12
- Sigstore provenance attestation
- SECURITY.md with PGP key

## [0.9.6] — 2026-05-12
- Docker badge and quick-start in all 5 READMEs

## [0.9.5] — 2026-05-12
- Fix literal \n in LICENSING.md

## [0.9.4] — 2026-05-11
- Fix language badge links, org links

## [0.9.3] — 2026-05-11
- Multi-language README (zh, ja, fr, de)

## [0.9.2] — 2026-05-11
- 222 tests (212 passing, 10 skipped), 85% coverage
- GitHub Actions CI with Node 18/20/22 matrix

## [0.9.1] — 2026-05-11
- Remove node-fetch dependency

## [0.9.0] — 2026-05-11
- Replace node-fetch with native fetch (Node 18+)
- Replace better-sqlite3 with sql.js (WASM, zero native compile)
- Reduce false positives on ATK-002/009/011

## [0.8.0] — 2026-05-11
- YAML/JSON policy-as-code engine
- Text + PDF report generators
- Docker: multi-stage builds, Compose profiles
- Comprehensive README rewrite

## [0.7.6] — 2026-05-10
- GitHub Action (action.yml)
- 28 comprehensive tests (SIEM, CRA, SBOM, License)

## [0.7.5] — 2026-05-10
- Elastic ECS, Microsoft Sentinel, IBM QRadar SIEM exporters

## [0.7.0] — 2026-05-10
- Enterprise SAML SSO integration

## [0.6.0] — 2026-05-10
- License key enforcement (HMAC-signed)
- PostgreSQL schema, FastAPI REST API, Webhook engine, Helm chart

## [0.5.0] — 2026-05-10
- ATK-011 (Transitive Propagation) detector
- SIEM CEF export, EU CRA compliance report

## [0.4.0] — 2026-05-10
- ATK-008/009/010 detectors, SPDX 2.3, NIST 800-161 report

## [0.3.0] — 2026-05-10
- ATK-001 through ATK-007 detectors (first real implementation)

## [0.2.2] — 2026-05-10
- Corpus test suite (50 clean + 22 malicious), HTML report, edit-distance typosquatting
- Phase 1 exit: FP < 2%

## [0.2.0] — 2026-05-10
- Commander.js CLI, SQLite persistence, CycloneDX SBOM

## [0.1.0] — 2026-05-09
- Initial foundation, monorepo structure, LICENSING.md, attack taxonomy stubs

## Key Timeline Observation
The ENTIRE project from 0.1.0 to 0.9.7 was developed in 4 days (May 9-12, 2026).
v0.1.0 through v0.6.0 all landed on May 10 alone (8 releases in one day).
