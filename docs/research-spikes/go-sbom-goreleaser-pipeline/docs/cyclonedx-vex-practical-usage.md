<!-- Source: Multiple (cyclonedx.org/capabilities/vex/, endorlabs.com, vulncheck.com) -->
<!-- Retrieved: 2026-05-15 -->

# CycloneDX VEX (Vulnerability Exploitability Exchange) - Practical Usage

## What is VEX?
VEX focuses on whether a vulnerability in a component can actually be exploited in its specific context. It helps organizations prioritize responses and reduces unnecessary mitigation efforts.

## CycloneDX VEX Implementation
CycloneDX embeds VEX information directly within the SBOM structure -- a single CycloneDX document can contain both the component inventory and exploitability status for known vulnerabilities.

## Practical Workflow
1. **Scan phase**: SBOM enriched with vulnerability data via scanners (Grype, Snyk, Trivy)
2. **VEX phase**: Vulnerabilities filtered/marked as exploitable or non-exploitable
3. **Automated exploitability analysis**: Examines if vulnerable functions are actually called
4. **Output**: VEX document in CycloneDX format

## Tool Support
- **Grype**: Initial support for CycloneDX VEX documents
- **osv-scanner**: Container image scanning (Debian-based)
- **Trivy, Grype, DepScan**: Can scan SBOMs from various tools

## Key Statistic
Up to 85% of vulnerabilities flagged in open-source libraries aren't reachable in production environments. VEX addresses this by providing context for vulnerability prioritization.

## Relevance to Go Projects
For Go binaries, VEX is particularly valuable because Go's dead code elimination means many dependencies in go.mod may not actually be compiled into the binary. A VEX document can communicate that a vulnerable dependency exists in go.mod but the vulnerable code path is not reachable in the compiled binary.
