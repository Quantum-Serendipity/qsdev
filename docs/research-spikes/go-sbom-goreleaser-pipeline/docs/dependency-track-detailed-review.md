<!-- Source: https://appsecsanta.com/dependency-track -->
<!-- Retrieved: 2026-05-15 -->

# OWASP Dependency-Track: Technical Architecture & Enterprise Capabilities

## Core Architecture

Dependency-Track operates as a persistent SBOM management platform with two primary components: a Java-based API server and a Vue.js frontend. The system maintains a continuous component inventory rather than producing point-in-time reports. The architecture uses "an API-first design where every feature is accessible through REST endpoints" with webhook support for external system notifications.

### Infrastructure Requirements
- API server: 2-8 GB RAM, 2-4 CPU cores (production)
- Frontend: 128-512 MB RAM, 0.5-1 core
- Database: H2 (evaluation only); PostgreSQL, MySQL, or SQL Server for production
- Deployment: Docker containers with persistent volume storage

## SBOM Ingestion & Format Support

**Supported Formats:**
- CycloneDX JSON and XML (full support, preferred)
- SPDX JSON (import/export)
- SPDX tag-value (import only)

The system ingests SBOMs as "living documents" that track component versions over time. Upload methods include web UI, REST API (`/api/v1/bom` endpoint), or the `dtrack-cli` client.

## Vulnerability Intelligence Sources

- **NVD (NIST)**: National Vulnerability Database
- **GitHub Security Advisories**: Community-reviewed ecosystem-specific data
- **Sonatype OSS Index**: Java, JavaScript, .NET, Go coverage
- **OSV**: Multi-ecosystem open-source vulnerabilities
- **Optional feeds**: Snyk, VulnDB (requires API keys)

## Continuous Monitoring Model

The platform "correlates your component inventory against updated vulnerability feeds around the clock." When new CVEs publish, the system identifies all affected projects automatically without requiring rescans. "A CVE disclosed a month after your last build still fires alerts against every affected project."

## Policy Engine & Compliance

- Suppressing false positives
- Escalating critical findings
- Requiring approval for specific risk profiles
- Flagging license compliance violations (GPL, AGPL, copyleft, permissive-only)

## API Capabilities

Official plugins for GitHub Actions (`gh-upload-sbom`), Jenkins (`dependencyTrackPublisher`), and GitLab CI.

## Key Limitations

- Cannot independently scan source code or binaries — strictly consumes externally-generated SBOMs
- No automated remediation or fix pull requests
- Requires self-hosted infrastructure with database management overhead

## Licensing

Fully open-source under Apache 2.0, 3,600+ GitHub stars, 167 contributors.
