<!-- Source: https://github.com/goreleaser/example-supply-chain -->
<!-- Retrieved: 2026-05-15 -->

# GoReleaser Supply Chain Security Example Repository

## Overview
Official example demonstrating complete supply chain security with GoReleaser: keyless signing, SBOM generation, and attestations.

## Key Security Features

**Signing & Verification:**
Keyless signing via Cosign using GitHub Actions' OIDC token:
```
cosign verify-blob \
    --certificate-identity "https://github.com/goreleaser/example-supply-chain/.github/workflows/release.yml@refs/tags/$VERSION" \
    --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
    --bundle "checksums.txt.sigstore.json" \
    ./checksums.txt
```

**SBOM Generation:**
Uses Syft to generate JSON-formatted SBOMs for each artifact. These can be scanned with Grype.

**Attestations:**
SLSA provenance attestations verified via GitHub CLI:
```
gh attestation verify --owner goreleaser *.tar.gz
```

## Tools Used
- **GoReleaser**: Release automation and orchestration
- **Cosign/Sigstore**: Keyless cryptographic signing
- **Syft**: SBOM generation (default SPDX JSON format)
- **Grype**: Vulnerability scanning
- **GitHub Actions**: CI/CD automation with OIDC integration
