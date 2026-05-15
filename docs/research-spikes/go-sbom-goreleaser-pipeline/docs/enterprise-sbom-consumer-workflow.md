<!-- Source: Multiple search results from web search on enterprise SBOM consumption workflows -->
<!-- Retrieved: 2026-05-15 -->

# Enterprise SBOM Consumer Workflow

## Pipeline Stages

A modern enterprise SBOM pipeline includes:

1. **Ingestion** — Internal builds and supplier SBOMs
2. **Normalization & Validation** — Identifiers, signatures, schemas
3. **Enrichment** — Licenses, VEX/CSAF, vulnerabilities, provenance
4. **Policy Evaluation** — Automated compliance checks
5. **Continuous Monitoring** — Ongoing vulnerability correlation

## Scanning and Analysis

SBOM scanning works by using parsers to read SBOM files, extract component identifiers, and query vulnerability databases (OSV for ecosystem-specific advisories, NVD for CVE records, and vendor-specific feeds from GitHub, npm, PyPI) to return findings.

Effective SBOM scanning prioritizes findings based on exploitability, reachability, and environmental context — not just severity scores.

## Policy Gates and Enforcement

Governance-driven SBOM programs reduce time-to-decision during vulnerability disclosures by replacing manual reviews with policy-driven gates. Policy-as-code provides the rules engine; SBOMs provide the structured data the policy engine evaluates.

When a container image or binary is added, policy compliance checks are automatically applied against the SBOM.

## Common Enterprise Patterns

### Build-Time Scanning + Continuous Monitoring
Many teams run both: scan in CI with Grype or Trivy, then upload the resulting SBOM to Dependency-Track so components stay under watch after the build.

### Dual-Tool Approach
- **Grype/Trivy**: Fast, point-in-time scanning in CI/CD
- **Dependency-Track**: Persistent monitoring, portfolio-level visibility

### Key Compliance Drivers (2025-2026)
- US: Executive Order 14028, NTIA minimum elements, CISA 2025 draft (hash + license fields)
- EU: Cyber Resilience Act (legally requires SBOMs), NIS2 (supply chain security mandate)

## Best Practices for Go SBOM Producers

- Ship SBOMs in both CycloneDX and SPDX formats for maximum consumer compatibility
- Include component hashes for integrity verification
- Produce VEX documents alongside SBOMs (especially using govulncheck for reachability)
- Use Package URLs (PURLs) consistently for component identification
- Sign SBOMs with cosign/Sigstore for provenance
