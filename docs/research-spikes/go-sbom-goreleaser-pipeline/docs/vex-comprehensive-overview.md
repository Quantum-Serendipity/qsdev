<!-- Source: https://www.aquasec.com/cloud-native-academy/vulnerability-management/vulnerability-exploitability-exchange/ -->
<!-- Retrieved: 2026-05-15 -->

# VEX: Vulnerability Exploitability eXchange - Comprehensive Overview

## Definition & Core Purpose

VEX is "a standardized format used to convey information about the exploitability of vulnerabilities in software products." Introduced in 2021 by the NTIA, it addresses a critical gap: vulnerability scanners often identify vulnerabilities that aren't actually exploitable in specific configurations.

## Four Status Categories

- **not_affected**: Vulnerability exists in component but is not exploitable in this context
- **affected**: Vulnerability is exploitable
- **fixed**: Vulnerability has been remediated
- **under_investigation**: Assessment is in progress

## Producer-Side VEX Creation

**Step 1: Generate SBOM**
```bash
trivy image --format cyclonedx --output debian11.sbom.cdx debian:11
```

**Step 2: Write VEX Documents**
Producers create VEX attestations specifying:
- CVE identifiers
- Affected package references (bom-ref)
- Exploitability analysis with justifications (e.g., "code_not_reachable")
- Remediation responses (e.g., "will_not_fix", "update")

## Consumer-Side Usage

Security teams consume VEX to filter scanner results:
```bash
trivy sbom debian11.sbom.cdx --vex trivy.vex.cdx
```

This enables teams to "zero in on critical vulnerabilities faster" by eliminating non-exploitable findings.

## Implementation Comparison

Three major VEX implementations:

| Implementation | Key Feature |
|---|---|
| **OpenVEX** | Lightweight standalone JSON-LD, format-agnostic |
| **CSAF VEX** | Full advisory framework from OASIS |
| **CycloneDX VEX** | Embedded directly into CycloneDX SBOM documents |

"Each implementation uses different formatting and syntax to describe the exploitability of vulnerabilities. But all implementations make it possible to share the same basic information."

## VEX vs. SBOM Distinction

- **SBOM**: Tracks "software supply chain components and contextual information"
- **VEX**: Focuses exclusively on "describing vulnerabilities"
- VEX can be integrated into SBOM but remains distinct

## Enterprise Adoption Patterns

- Trivy was "one of the first scanners to support VEX"
- **VEX Hub**: Centralized repository where "software vendors can share VEX attestations" and security teams can download them
- Addresses gap between standardized format and practical discoverability

## Practical Workflow Benefits

VEX enables organizations to:
- Determine which vulnerabilities are actually exploitable
- Reduce noise from scanners by filtering non-exploitable findings
- Identify conditions for exploitability to inform avoidance strategies
- Prioritize remediation efforts based on actual risk

## Kubernetes-Specific Adoption

VEX gained "notable momentum" in Kubernetes environments for managing "millions of public container images" and their associated vulnerabilities.
