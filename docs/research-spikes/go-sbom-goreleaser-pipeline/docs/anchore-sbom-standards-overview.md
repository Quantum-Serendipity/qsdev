<!-- Source: https://anchore.com/sbom/key-things-to-know-about-sboms-and-sbom-standards/ -->
<!-- Retrieved: 2026-05-15 -->

# SBOM Standards: SPDX and CycloneDX Overview (Anchore)

## Core SBOM Standards

### SPDX (Software Package Data Exchange)

SPDX is described as "a machine-readable international open standard (ISO/IEC 5962:2021) format for communicating the components, licenses, and copyrights associated with a software package." The standard is maintained by a Linux Foundation grassroots project with representatives from vendors, foundations, and system integrators.

**Technical Details:**
- Supports multiple serialization formats: Tag-Value and JSON formats
- Includes comprehensive metadata such as package names, Package URLs, and license information
- Version referenced in examples: SPDX-2.3
- Provides location annotations identifying file associations within packages

### CycloneDX

CycloneDX operates as "a lightweight, machine-readable SBOM standard useful for application security contexts and supply chain component analysis." It originated within the OWASP community and is guided by a Core Team for strategic oversight.

**Technical Details:**
- Supports multiple serialization formats: XML and JSON variants
- Includes component references, publisher information, and external references
- Implements property-based extensions for additional metadata capture
- Can output structured component data with CPE and PURL identifiers

## Format Comparison

Historical distinctions between formats have diminished. Previously, "arguments for using one format" favored SPDX for open source dependencies or CycloneDX for licensing concerns. However, the industry has "converge[d] on SBOM formats containing complete and accurate dependency and license details."

Current consensus suggests the formats are functionally equivalent for most use cases, with "the data contained within them" being largely interchangeable. Both support identical core information: component names, versions, identifiers, dependencies, authors, and timestamps.

## Tooling: Syft

Anchore's open source SBOM generator, Syft, demonstrates practical format implementation. Users can generate output in both standards using command-line arguments (`-o spdx-json`, `-o cyclonedx-xml`, etc.), enabling teams to produce multiple format variants from identical source analysis.

## Selection Recommendations

"At Anchore we support both SPDX and CycloneDX because there is equally strong demand for the two formats." Organizations should evaluate tooling support and downstream consumer requirements rather than inherent format superiority.
