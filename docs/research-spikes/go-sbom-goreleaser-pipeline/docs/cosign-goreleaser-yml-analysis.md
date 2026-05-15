<!-- Source: https://raw.githubusercontent.com/sigstore/cosign/main/.goreleaser.yml -->
<!-- Retrieved: 2026-05-15 -->

# Cosign GoReleaser Configuration - SBOM Analysis

## SBOM Configuration
```yaml
sboms:
  - artifacts: binary
```

Cosign generates SBOMs for binary artifacts only, without explicitly specifying a format. This means GoReleaser uses its default: **SPDX JSON via Syft**.

## Signing & Attestation Strategy

Multi-layered signing approach:

**KMS-Based Signing:**
- Uses GCP KMS for key management
- Produces signatures in `-kms.sigstore.json` format

**Keyless Signing (Three Variants):**
1. Binary artifacts: `.sigstore.json` bundles via `sign-blob`
2. Checksums: Signs checksum files independently
3. Packages: Signs package artifacts (APK, DEB, RPM)

## Key Takeaway
Cosign -- Sigstore's own signing tool -- uses GoReleaser's default SBOM configuration, which produces SPDX JSON format SBOMs via Syft. This is a strong signal that SPDX JSON is the de facto default for Go projects using GoReleaser.
