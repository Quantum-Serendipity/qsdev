# Homebrew 4.3.0: SBOM Support and Bottle Attestation

- **Source**: https://brew.sh/2024/05/14/homebrew-4.3.0/
- **Retrieved**: 2026-05-15

## SBOM Support

Homebrew 4.3.0 introduces Software Bill of Materials (SBOM) functionality through two mechanisms:

1. **Basic SPDX file in bottles**: The `brew bottle` command now includes "a basic SPDX file inside the bottle"
2. **Comprehensive SPDX file post-installation**: After installation, "a more comprehensive one" is generated

This implementation provides support for the widely-adopted SBOM standard.

## Bottle Attestation Verification

The release introduces artifact attestation verification with the following details:

- **Activation**: Users must set the `HOMEBREW_VERIFY_ATTESTATIONS` environment variable to enable this feature
- **Functionality**: When enabled, `brew install` verifies bottle artifact attestations during the pouring process
- **Tool requirement**: Currently relies on GitHub's `gh` CLI tool
- **Status**: Still in beta phase

The developers plan to remove the `gh` dependency and improve performance before making attestation verification the default behavior.

## Build Provenance

homebrew-core cryptographically attests to all bottles built in the official Homebrew CI. Each bottle comes with:
- A cryptographically verifiable statement binding the bottle's content to the specific workflow
- The git commit and GitHub Actions run ID for the workflow that produced the bottle
- SLSA Build L2-compatible attestation

## Formula Metadata Security Chain

Once provenance on homebrew-core is fully deployed:
- Formula metadata used to install packages is authenticated through Homebrew's signed JSON API
- The bottle has not been tampered with in transit thanks to digests in the formula metadata
- The bottle was built in a public, auditable, controlled CI/CD environment against a specific source revision
