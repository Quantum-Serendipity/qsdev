<!-- Source: https://trivy.dev/docs/latest/supply-chain/sbom/ -->
<!-- Retrieved: 2026-05-15 -->

# Trivy SBOM Capabilities: Comprehensive Overview

## Supported Formats

Trivy generates SBOMs in two standardized formats:

**CycloneDX**: Trivy supports JSON output (XML is not currently supported). By default, `--format cyclonedx` generates software bill of materials without vulnerability data. To include vulnerabilities, users must explicitly enable scanning with `--scanners vuln`.

**SPDX**: Available in two variants -- tag-value format (`--format spdx`) and JSON format (`--format spdx-json`). Both represent software composition comprehensively with package relationships and dependencies.

## SBOM Generation

### CLI Usage
Users generate SBOMs using the `--format` option across subcommands:
- `trivy image --format spdx-json --output result.json alpine:3.15`
- `trivy fs --format cyclonedx --output result.json /app/myproject`

### Supported Packages
Trivy catalogs OS packages and language-specific dependencies, following its standard vulnerability scanning package detection logic.

## Scanning Modes

Trivy can scan via:
- **Container images** (with automatic SBOM discovery capability)
- **Filesystems**
- **Rootfs** environments
- **VM images**
- **Kubernetes clusters**

## Key Distinction: SBOM vs. Vulnerability Scanning

By default, SBOM generation disables security scanning. The documentation explicitly states: "'--format cyclonedx' disables security scanning." Users who want vulnerability information alongside SBOM data must explicitly activate vulnerability scanning through additional flags.

## SBOM Detection Features

Trivy automatically searches container images for embedded SBOM files with extensions `.spdx`, `.spdx.json`, `.cdx`, and `.cdx.json`. This detection is enabled for container images and rootfs targets, with special support for Bitnami images.

## Output Enrichment

Both formats include metadata like package relationships, dependencies, licensing information, layer digests, and custom Trivy properties for enhanced supply chain visibility.
