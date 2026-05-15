# Best Practices for Naming and Directory Conventions for SBOMs in Open Source Projects

- **Source**: https://sbom-catalog.openssf.org/sbom-naming.html
- **Retrieved**: 2026-05-15

## Scope

This guidance applies to "SBOMs of Type Source and Build" only, excluding deployment and runtime SBOMs. The recommendations target open source projects that distribute artifacts directly rather than through ecosystems like Maven or NPM.

## Key Directory Principles

The document emphasizes a critical constraint: "no directory structures should be used." Instead, release files should follow "a flat list of files without directories (think GitHub or GitLab Release artifacts)." This aligns with typical release platform requirements.

## Naming Convention Framework

The guidance adopts SLSA provenance attestation principles by appending specific extensions corresponding to the SBOM standard and format used. Here are the prescribed patterns:

**CycloneDX:**
- JSON format: `artifact-1.0.0.tar.gz.cdx.json`
- XML format: `artifact-1.0.0.tar.gz.cdx.xml`

**SPDX:**
- TAG:VALUE: `artifact-1.0.0.tar.gz.spdx`
- JSON: `artifact-1.0.0.tar.gz.spdx.json`
- XML: `artifact-1.0.0.tar.gz.spdx.xml`
- YAML: `artifact-1.0.0.tar.gz.spdx.yml` (or `.yaml`)
- RDF XML: `artifact-1.0.0.tar.gz.spdx.rdf` (or `.rdf.xml`)

## Critical Requirement

The guidance mandates that "JSON format files should be considered a mandatory requirement." While other formats may be provided, JSON must always accompany any SBOM distribution.
