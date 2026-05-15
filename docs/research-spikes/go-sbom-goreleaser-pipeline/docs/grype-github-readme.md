<!-- Source: https://github.com/anchore/grype -->
<!-- Retrieved: 2026-05-15 -->

# Grype: SBOM Scanning and Vulnerability Detection

## Supported SBOM Input Formats

Grype can scan SBOMs for known vulnerabilities:

```bash
grype sbom:./sbom.json
```

SBOM piping capability:

```bash
cat ./sbom.json | grype
```

The documentation specifically mentions "scan a Syft SBOM" as a supported input method, indicating compatibility with Syft-generated SBOMs. Supports SPDX and CycloneDX formats.

## Go Module Scanning

The repository is written in Go (96.7% of codebase). Supports Go modules as one of its language ecosystems.

## Vulnerability Databases

Grype uses its own curated vulnerability database that aggregates from multiple sources including NVD, GitHub Advisory Database, and others.

## VEX Support

OpenVEX support for filtering and augmenting scan results. VEX integration enables vulnerability exclusion records.

## Risk Prioritization

Grype offers EPSS, KEV, and risk scoring for threat prioritization.

## Output Formats

Multiple output formats supported including table, JSON, CycloneDX, and template-based formats.

## CLI Usage

```bash
# Scan a container image
grype <image>

# Scan a directory
grype dir:/path/to/dir

# Scan an SBOM
grype sbom:./sbom.json

# Pipe an SBOM
cat ./sbom.json | grype

# Specify output format
grype sbom:./sbom.json -o json
grype sbom:./sbom.json -o cyclonedx

# Use VEX to filter results
grype sbom:./sbom.json --vex ./vex.json
```
