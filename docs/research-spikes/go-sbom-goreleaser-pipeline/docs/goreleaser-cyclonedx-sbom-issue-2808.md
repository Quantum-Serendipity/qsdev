<!-- Source: https://github.com/goreleaser/goreleaser/issues/2808 -->
<!-- Retrieved: 2026-05-15 -->

# GoReleaser CycloneDX SBOM Support Request - Issue #2808

## Original Request

On January 5, 2022, developer-guy opened issue #2808 requesting the addition of CycloneDX format support for SBOM generation in GoReleaser.

## Proposed Solutions

**Option 1:** Integrate the `cyclonedx-gomod` tool directly, similar to how Syft is implemented. Referenced a GitHub Action for downloading the tool.

**Option 2:** Leverage Syft's existing capability to generate CycloneDX-formatted SBOMs by adding an additional flag to override the `--output` argument of Syft.

## Resolution

The issue was closed. GoReleaser's design already supports this because:
1. The `cmd` field can be set to any SBOM generation tool (e.g., `cyclonedx-gomod`)
2. The `args` field can override Syft's output format (e.g., `--output cyclonedx-json=$document`)

Both approaches work without any GoReleaser code changes -- the existing configuration is flexible enough to support any SBOM format through any tool.

## Key Insight
GoReleaser's SBOM support is tool-agnostic by design. It doesn't have built-in knowledge of SBOM formats -- it just runs a command and captures the output file. This means any SBOM generator that can write to a file path works out of the box.
