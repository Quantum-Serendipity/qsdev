<!-- Source: https://goreleaser.com/blog/supply-chain-security/ -->
<!-- Retrieved: 2026-05-15 -->
<!-- Note: This is a signing-focused extract; the broader post was previously saved as goreleaser-supply-chain-security-blog.md -->

# GoReleaser and Software Supply Chain Security - Signing Details

## Overview

The blog post explains how GoReleaser helps secure software supply chains through integration with specialized tools. It addresses the exponential growth of supply chain attacks (increasing at a 4-5x per year rate) by providing practical security mechanisms.

## SBOM Generation

GoReleaser uses Syft to generate SBOMs. Configuration:

```yaml
sboms:
  - id: archive
    artifacts: archive
  - id: source
    artifacts: source
```

## Code Signing with Cosign

GoReleaser integrates Cosign (from Sigstore) to verify artifact integrity. The platform supports keyless signing, eliminating the need to manage private keys manually.

**Container image signing:**
```yaml
docker_signs:
  - cmd: cosign
    env:
      - COSIGN_EXPERIMENTAL=1
    artifacts: images
    output: true
    args:
      - "sign"
      - "${artifact}"
```

**Checksum file signing:**
```yaml
signs:
  - cmd: cosign
    env:
      - COSIGN_EXPERIMENTAL=1
    certificate: "${artifact}.pem"
    args:
      - sign-blob
      - "--output-certificate=${certificate}"
      - "--output-signature=${signature}"
      - "${artifact}"
    artifacts: checksum
    output: true
```

**Note**: The above uses the older cosign v1/v2 flag style with separate `--output-certificate` and `--output-signature`. Modern cosign v3+ uses the `--bundle` flag instead, producing a single `.sigstore.json` file.

## Implementation in CI/CD

For GitHub Actions, install required tools:
- Syft: `anchore/sbom-action/download-syft@v0`
- Cosign: `sigstore/cosign-installer@v4`

## Verification

Users can verify signed artifacts:
- **Container images:** `cosign verify ghcr.io/user/image:tag`
- **Blobs:** `cosign verify-blob --cert file.pem --signature file.sig file`

Modern verification with bundles: `cosign verify-blob --bundle file.sigstore.json file`

## Conclusion

GoReleaser streamlines supply chain security by integrating SBOM generation and cryptographic signing without adding complexity - configuration occurs through simple YAML additions to `.goreleaser.yml`.
