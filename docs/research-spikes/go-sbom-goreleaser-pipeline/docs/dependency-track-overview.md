<!-- Source: https://dependencytrack.org/ -->
<!-- Retrieved: 2026-05-15 -->

# Dependency-Track: SBOM Analysis Platform Overview

## Core Capabilities

### SBOM Ingestion & Format Support

Dependency-Track accepts SBOMs published via REST API, Jenkins plugin, or uploaded through web interface. The platform specifically supports CycloneDX format as "an OWASP and industry standard" for bill of materials. Also supports SPDX ingestion.

### Vulnerability Intelligence Sources

The platform integrates with multiple threat databases:
- National Vulnerability Database (NVD)
- Sonatype OSS Index
- GitHub Advisories
- Snyk
- OSV
- VulnDB from Risk Based Security

### Policy Evaluation

Security, operational, and license policy compliance assessment to identify risk across development teams and supply chain partners.

## Advanced Features

### VEX Support

Dependency-Track produces and consumes CycloneDX Vulnerability Exploitability eXchange (VEX) exceeding CISA recommendations.

### API & Automation

Well-documented API-first design integrates easily with other systems providing endless possibilities for enterprise automation workflows.

### Continuous Monitoring

The platform continuously analyzes portfolios for risk and compliance, delivering real-time analysis and security events via webhooks and chat operations integrations.

## Key Differentiator

Unlike build-time scanners, Dependency-Track maintains a persistent component inventory and alerts when new CVEs affect deployed software (N-day vulnerability identification).

## Architecture

Operates as an "intelligent Component Analysis" solution emphasizing supply chain risk reduction through comprehensive component tracking across cloud, enterprise, and IoT environments.
