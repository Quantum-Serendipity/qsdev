<!-- Source: https://raw.githubusercontent.com/anchore/syft/main/.goreleaser.yaml -->
<!-- Retrieved: 2026-05-15 -->
<!-- Note: WebFetch returned a summary rather than the raw file content. Key details extracted below. -->

# Syft's Own .goreleaser.yaml Configuration

Syft (the SBOM generation tool by Anchore) uses GoReleaser for its own releases.

## Key Components

**Build Targets:**
- Linux (5 architectures: amd64, arm64, ppc64le, riscv64, s390x)
- Darwin/macOS (amd64, arm64) with notarization via Quill
- Windows (amd64, arm64)

**Distribution Methods:**
- Archives and compressed formats
- RPM/DEB packages via NFPM
- Homebrew via anchore/homebrew-syft repository
- Docker images across multiple registries (Docker Hub and GitHub Container Registry)

**Container Images:**
Three image variants are produced:
1. Production (standard)
2. Nonroot (security-hardened)
3. Debug (development)

Each supports the five Linux architectures listed above.

**SBOM Generation Configuration:**
- `artifacts: archive` -- produces SBOMs using Syft itself
- Naming: `{binary}_{version}_{os}_{arch}.sbom`

**Artifact Signing:**
- Uses Cosign to cryptographically sign release checksums
- Generates both `.sig` signatures and `.pem` certificates via GitHub's OIDC token system

**Release Strategy:**
Automatic prerelease detection with non-draft releases. Docker manifests aggregate multi-architecture images under unified tags (latest, version-specific, and variant-specific).

## Key Insight
Syft (itself the default SBOM generator for GoReleaser) uses GoReleaser with `artifacts: archive` for its own SBOM generation -- a self-referential pattern where the tool generates its own SBOMs.
