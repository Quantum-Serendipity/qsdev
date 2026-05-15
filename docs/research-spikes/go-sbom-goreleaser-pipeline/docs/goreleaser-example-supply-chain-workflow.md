# GoReleaser Example Supply Chain - Release Workflow

- **Source URL**: https://raw.githubusercontent.com/goreleaser/goreleaser-example-supply-chain/main/.github/workflows/release.yml
- **Retrieved**: 2026-05-15
- **Note**: The WebFetch returned a summary rather than the raw file. Key details extracted below.

---

## Workflow Structure

**Trigger**: Activates exclusively on version tags (e.g., `v1.0.0`)

**Permissions Granted**:
- Write access to repository contents for publishing releases
- ID token generation for keyless signing operations
- Package registry write permissions for container image distribution
- Attestation generation capabilities

**Workflow Steps**:

1. Source code checkout with full history retrieval
2. Go runtime setup from `go.mod` specification
3. QEMU and Docker Buildx configuration for multi-platform builds
4. Cosign installation for container signing
5. SBOM (Software Bill of Materials) tool procurement via Syft
6. Container registry authentication against ghcr.io
7. GoReleaser execution with `--clean` flag
8. Build provenance attestation generation from checksums
9. Build provenance attestation generation from digests

**Notable Security Features**: The workflow implements keyless signing through Cosign, uses pinned action versions with SHA-256 hashes, generates software composition attestations, and leverages GitHub's native provenance mechanisms for supply chain integrity verification.
