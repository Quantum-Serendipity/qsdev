<!-- Source: https://sbomify.com/2026/01/15/sbom-formats-cyclonedx-vs-spdx/ -->
<!-- Retrieved: 2026-05-15 -->

# CycloneDX vs SPDX: SBOM Formats Compared

## Overview

Two industry-standard formats dominate the SBOM landscape. CycloneDX, created by OWASP, prioritizes application security, while SPDX, maintained by the Linux Foundation, emphasizes license compliance and transparency.

## Version Histories

**SPDX Timeline:**
- 2010: Originated as Linux Foundation project for open source license standardization
- 2015: SPDX 2.0 released
- 2021: Achieved ISO/IEC standard status (5962:2021) with version 2.2.1
- 2023: SPDX 2.3 added relationship types and improved package identification
- 2024: SPDX 3.0 introduced profile-based architecture with specialized profiles

**CycloneDX Timeline:**
- 2017: Created by OWASP for supply chain security
- 2024: CycloneDX 1.6 added attestation and cryptography bill of materials support; achieved Ecma International standardization (ECMA-424)
- 2025: CycloneDX 1.7 introduced patent/IP metadata and enhanced cryptographic transparency

## Governance & Standards

| Aspect | CycloneDX | SPDX |
|--------|-----------|------|
| **Governing Body** | OWASP Foundation | Linux Foundation |
| **Standards Body** | Ecma International (ECMA-424) | ISO/IEC (5962:2021) |
| **Primary Focus** | Application security & supply chain risk | License compliance & software transparency |
| **Release Cadence** | Frequent (~annual major versions) | Less frequent (2-3 years between majors) |
| **Current Version** | 1.7 (2025) | 3.0.1 (2024); 2.3 widely deployed |

## Serialization Formats

**CycloneDX supports:** JSON, XML, Protocol Buffers

**SPDX 2.3 supports:** JSON, XML, RDF, Tag-Value

**SPDX 3.0 supports:** JSON-LD alongside traditional formats

## Structural Differences

### Document Models

CycloneDX employs a "flat, component-centric model" where dependencies are expressed through a dedicated array mapping component references. SPDX 2.3 uses package-and-relationship structures with broader relationship type support. SPDX 3.0 shifts to an "element-based model with a linked-data approach," offering greater flexibility for complex scenarios.

### Component Identification

Both formats support Package URL (purl) and CPE identifiers, with purl preferred for precision. CycloneDX references purl as `components[].purl`; SPDX uses `packages[].externalRefs[]` for both formats.

### License Data

SPDX maintains stronger historical advantages in license documentation, defining the standard SPDX License List referenced industry-wide. Both formats support SPDX license expressions (e.g., "MIT OR Apache-2.0"). CycloneDX incorporates license data through `components[].licenses[]`.

### Vulnerability Support

**CycloneDX** includes dedicated `vulnerabilities` array at document level, enabling direct vulnerability and VEX document creation.

**SPDX 2.3** lacks native vulnerability structures; requires external references.

**SPDX 3.0** addresses this gap with Security profile support for vulnerability data.

### Lifecycle & Build Information

CycloneDX 1.7 includes `metadata.lifecycles[].phase` tracking (design, pre-build, build, post-build, operations, discovery, decommission), aligning with CISA taxonomy. SPDX 3.0's Build profile documents build systems, commands, and environments.

## Compliance Framework Coverage

| Framework | Preference |
|-----------|-----------|
| Executive Order 14028 (US) | Format-agnostic |
| CISA Minimum Elements | References both formats |
| EU Cyber Resilience Act (BSI TR-03183-2) | CycloneDX 1.6+ or SPDX 3.0.1+ required (JSON/XML) |
| NTIA Minimum Elements | Format-agnostic |
| FDA Medical Device | Both referenced in guidance |
| NIST SP 800-53 | Format-agnostic (controls-focused) |
| NIST SP 800-171 | Format-agnostic |

EU CRA implementing guidance represents the most prescriptive requirement, specifying minimum format versions.

## Tooling Ecosystem

### Generation Tools
- **sbomify GitHub Action:** Both formats, CI/CD integration
- **Syft:** Multi-ecosystem, container support, both formats
- **Trivy:** Vulnerability scanning with SBOM generation (note: security concerns noted post-March 2026)
- **cdxgen:** CycloneDX-native, broad language coverage
- **Microsoft SBOM Tool:** SPDX-native focus

### Analysis & Management
- **sbomify:** Integrated management, monitoring, distribution
- **Grype:** Vulnerability scanning (both formats)
- **OWASP Dependency-Track:** Standalone monitoring
- **OSV-Scanner:** Google's vulnerability scanner

## When to Use Each Format

**Choose CycloneDX when:**
- Application security and vulnerability management are priorities
- VEX data inclusion in SBOMs is required
- EU CRA compliance demands compact formatting
- Simpler document models with fewer mandatory fields are preferred

**Choose SPDX when:**
- License compliance drives primary objectives
- File-level and snippet-level analysis needed
- Ecosystems standardize on SPDX (automotive, embedded Linux)
- ISO/IEC standard certification matters (ISO/IEC 5962:2021)
- SPDX 3.0 specialized profiles (AI, Build, Dataset) address use cases

**Support both when:**
- Software distributes to customers with varying preferences
- Compliance obligations span multiple frameworks
- Tools like Syft enable simultaneous generation

## Key Takeaway

"The formats are converging in capability, and the practical differences are narrowing with each release." Most organizations generate SBOMs in both formats or convert using tools like the CycloneDX CLI, though native format generation prevents data loss in critical compliance scenarios.
