<!-- Source: https://github.com/goreleaser/goreleaser/issues/2808 -->
<!-- Retrieved: 2026-05-15 -->

# CycloneDX SBOM Support in GoReleaser - Issue #2808

## Issue Status
**Closed** (assigned to @caarlos0)

## Original Request
Developer-guy opened this feature request on January 5, 2022, asking for CycloneDX format SBOM generation support in GoReleaser. The request states: "We have many tools that can allow us to generate an SBOM file based on CycloneDX format."

## Proposed Solutions
The issue outlined two approaches:

1. **Direct Tool Integration**: Support `cyclonedx-gomod` as an alternative to syft, passing arguments similarly to how syft is handled in the codebase.

2. **Syft Output Format Override**: "Syft can generate CycloneDX format-based SBOM files" by accepting a flag to override syft's output argument, enabling CycloneDX format generation.

## Resolution
GoReleaser's SBOM configuration is now fully flexible -- you can specify any `cmd` and `args`, meaning both approaches work:
- Use syft with `--output cyclonedx-json=$document`
- Use cyclonedx-gomod directly as the cmd
- Generate multiple SBOMs by defining multiple sboms entries in .goreleaser.yaml
