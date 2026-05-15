# GoReleaser Supply Chain Example Repository

- **Source**: https://github.com/goreleaser/goreleaser-example-supply-chain
- **Retrieved**: 2026-05-15

## Repository Purpose

Demonstrates "GoReleaser manages the entire thing, basically" by automating build, signing, and verification workflows for Go binaries and container images.

## Core Workflow Steps

The automated process includes:
- Building binaries via Go Mod Proxy
- Generating SBOMs using `syft`
- Creating checksum files
- Signing artifacts with `cosign` (keyless)
- Building and signing Docker images

## Asset Naming Conventions

SBOM files follow this pattern: `{artifact-name}.sbom.json`

Signature bundles use: `{file-name}.sigstore.json`

## Verification Process

Users verify artifacts by:
1. Downloading `checksums.txt` and `checksums.txt.sigstore.json`
2. Running cosign verification with OIDC issuer authentication
3. Validating individual artifacts against the checksum file
4. Inspecting SBOM files using vulnerability scanning tools like `grype`

## Key Technologies

- **GoReleaser**: Release automation
- **Cosign**: Keyless signing via Sigstore
- **Syft**: SBOM generation
- **GitHub Attestations**: Supply chain provenance
