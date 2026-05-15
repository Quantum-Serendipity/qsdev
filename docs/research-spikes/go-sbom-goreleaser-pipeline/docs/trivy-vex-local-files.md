<!-- Source: https://trivy.dev/docs/latest/supply-chain/vex/file/ -->
<!-- Retrieved: 2026-05-15 -->

# Trivy's Local VEX File Support

## Supported Formats

1. **CycloneDX** — Works with container images and SBOMs only
2. **OpenVEX** — Compatible with all Trivy targets (images, filesystems, repositories, VMs, Kubernetes, SBOMs)
3. **CSAF** — Supports all targets; format-agnostic for input SBOMs

## Using the --vex Flag

```bash
trivy image debian:11 --vex document.vex.json
trivy sbom sbom.cdx --vex vulnerability-exceptions.vex.cdx
```

## CycloneDX VEX Workflow

Generate an SBOM first:
```bash
trivy image --format cyclonedx --output image.sbom.cdx debian:11
```

Create a VEX document referencing the SBOM's BOM-Links (format: `urn:cdx:serialNumber/version#bom-ref`). Vulnerabilities marked with `"state": "not_affected"` are filtered out during scanning.

## OpenVEX Format

OpenVEX uses Package URLs (PURLs) to identify components. Minimal example:

```json
{
  "@context": "https://openvex.dev/ns/v0.2.0",
  "statements": [{
    "vulnerability": {"name": "CVE-2019-8457"},
    "products": [{"@id": "pkg:deb/debian/libdb5.3@5.3.28+dfsg1-0.8"}],
    "status": "not_affected",
    "justification": "vulnerable_code_not_in_execute_path"
  }]
}
```

OpenVEX supports subcomponents for precise scoping.

## CSAF Support

CSAF documents use product IDs and PURL identifiers within a `product_tree`. Vulnerabilities are mapped to products via `product_status` fields. "CSAF aims to be SBOM format agnostic."

## PURL Matching Rules

- **No version specified** — Matches all versions of a package
- **No qualifiers** — Matches any architectural or platform variations
- **Specific qualifiers** — Matches only packages with identical qualifiers

## How VEX Filtering Works

Trivy constructs a dependency graph internally and applies VEX statements across parent-child relationships. When a vulnerability is marked "not_affected" for a component, Trivy suppresses it throughout that branch of the dependency tree, but may still flag it if other unexcused paths exist to the vulnerable package.
