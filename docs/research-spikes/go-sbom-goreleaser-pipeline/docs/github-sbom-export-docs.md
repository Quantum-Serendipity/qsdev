# Exporting a Software Bill of Materials for Your Repository

- **Source URL**: https://docs.github.com/en/code-security/how-tos/secure-your-supply-chain/establish-provenance-and-integrity/exporting-a-software-bill-of-materials-for-your-repository
- **Retrieved**: 2026-05-15

---

## UI Method

Navigate to your repository's Insights tab, select Dependency graph from the sidebar, then click "Export SBOM" in the Dependencies tab to download the file.

## REST API

REST API endpoints available at the dependency-graph/sboms endpoint for programmatic SBOM exports.

## SBOM Contents

The exported file includes an inventory of a project's dependencies and associated information such as versions, package identifiers, licenses, transitive paths, and copyright information. However, SBOMs do not include dependents (other projects that rely on your project).

## Format

SBOMs are exported in the industry-standard SPDX format (version 2.3 specified for UI exports).

## GitHub Actions Alternatives

Three marketplace actions can generate SBOMs:
- **SPDX Dependency Submission Action** -- Creates SPDX 2.2 compatible files
- **Anchore SBOM Action** -- Uses Syft for SPDX 2.2 compatible generation
- **SBOM Dependency Submission Action** -- Uploads CycloneDX format SBOMs
