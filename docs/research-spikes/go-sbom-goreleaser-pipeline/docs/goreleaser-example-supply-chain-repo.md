<!-- Source: https://github.com/goreleaser/goreleaser-example-supply-chain -->
<!-- Retrieved: 2026-05-15 -->

# GoReleaser Supply Chain Example Repository

## Core Purpose
This repository demonstrates "GoReleaser + Go Mod proxying + Cosign keyless signing + Syft SBOM generation." The project automates the entire software supply chain workflow.

## Workflow Process
GoReleaser orchestrates several key steps:
- Builds using the Go Mod Proxy as the source of truth
- Generates Software Bill of Materials (SBOMs) via Syft
- Creates checksum files
- Signs artifacts with Cosign
- Builds Docker images from the compiled binaries
- Signs container images

## Verification Instructions

**Getting the Latest Release:**
Users retrieve the current version and download verification files (`checksums.txt` and `checksums.txt.sigstore.json`), then validate them using Cosign with OIDC issuer verification.

**Binary Verification:**
Downloaded artifacts are verified against checksums using standard tools like `sha256sum`.

**SBOM Inspection:**
Dependency trees can be examined by downloading `.sbom.json` files and analyzing them with Grype for vulnerability detection.

**Docker Image Validation:**
Container images are verified using Cosign with certificate identity checks and scanned with Grype. Attestations are verified via the `gh` CLI tool.

## Key Technologies
- **GoReleaser**: Release automation
- **Cosign & Sigstore**: Keyless signing and verification
- **Syft**: SBOM generation
- **Grype**: Vulnerability scanning
- **GitHub Actions**: CI/CD pipeline
- **SLSA**: Provenance attestations

**Repository Stats:** 60 stars, 11 forks, 19 releases, primarily Go (70.6%) with Dockerfile components (29.4%)
