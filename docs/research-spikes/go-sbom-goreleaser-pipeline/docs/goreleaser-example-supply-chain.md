# GoReleaser Example Supply Chain Repository

- **Source URL**: https://github.com/goreleaser/goreleaser-example-supply-chain
- **Retrieved**: 2026-05-15

---

## What This Example Demonstrates

This repository showcases a complete software supply chain implementation combining several security tools. The project illustrates how to integrate GoReleaser with GitHub Actions to create a secure release process that includes keyless code signing, software bill of materials (SBOM) generation, and provenance attestations.

## Key Components

**Build and Release Process:**
The workflow automates building binaries using Go's module proxy as the source of truth, generating SBOMs via Syft, creating checksums, signing artifacts with Cosign, building Docker images from the compiled binaries, and signing container images.

**Verification Methods:**
Users can verify artifacts through multiple approaches:
- Downloading `checksums.txt` and its signature bundle (`checksums.txt.sigstore.json`)
- Using Cosign to verify blob signatures with keyless authentication
- Validating individual artifact checksums
- Inspecting SBOM files with Grype to check dependencies
- Verifying Docker image signatures and scanning for vulnerabilities
- Confirming attestations via GitHub's CLI

## Technology Stack

The repository uses Go (70.6% of codebase) and Dockerfile (29.4%), implementing standards like SLSA provenance through tools including Cosign, Syft, and Sigstore for keyless signing via OIDC tokens from GitHub Actions.
