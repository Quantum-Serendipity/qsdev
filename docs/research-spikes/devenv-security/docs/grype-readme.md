<!-- Source: https://github.com/anchore/grype -->
<!-- Retrieved: 2026-05-12 -->

# Grype Vulnerability Scanner - README

## Input Formats Supported
- **Container images**: Docker, OCI, and Singularity formats
- **Filesystems**: Local directories and projects
- **SBOMs**: Syft-generated SBOMs and other formats (CycloneDX, SPDX)

## SBOM Scanning
Scan an SBOM for vulnerability detection. Supports piping SBOMs directly into Grype.

```
grype sbom:./sbom.json
cat ./sbom.json | grype
```

Container and filesystem examples:
```
grype alpine:latest
grype ./my-project
```

## SBOM Format Support
Topics list includes "cyclonedx" and "openvex", confirming integration with CycloneDX and SPDX standards. OpenVEX support for filtering and augmenting scan results.

## Vulnerability Detection Features
Incorporates threat prioritization through EPSS, KEV, and risk scoring mechanisms. Databases: NVD, GitHub Security Advisories, and vendor-specific feeds.

## Nix-Related Information
No Nix-specific mentions in README. Grype is Nix-agnostic but can consume Nix-generated SBOMs if they use standard CycloneDX/SPDX formats with recognized PURLs/CPEs.
