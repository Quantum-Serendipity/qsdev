<!-- Source: https://goreleaser.com/blog/supply-chain-security/ -->
<!-- Retrieved: 2026-05-15 -->

# GoReleaser and Software Supply Chain Security

## Overview

The blog post examines how GoReleaser addresses software supply chain vulnerabilities through integrated security tooling. The article emphasizes that "software supply chains are anything that's needed to deliver your product -- including all the components you use."

## Key Threat Landscape

Supply chain attacks have accelerated significantly, with trends showing "exponential rate of 4-5x per year" growth. Common attack vectors include dependency confusion, typosquatting, and malicious source code injection.

## Three Primary Security Mechanisms

### 1. Software Bill of Materials (SBOM)

**Purpose**: Creating an inventory of all software components used during build and deployment cycles.

The post defines SBOM as "a structured list of components, modules, and libraries that are included in a given piece of software." Multiple formats exist: SPDX, SWID Tags, and CycloneDX.

**GoReleaser Configuration**:
```yaml
sboms:
  - id: archive
    artifacts: archive
  - id: source
    artifacts: source
```

GoReleaser uses Syft by default when no command is specified. Installation requires the Anchore GitHub Action:
```yaml
- uses: anchore/sbom-action/[email protected]
```

### 2. Artifact Signing with Cosign

**Concept**: Ensuring integrity by verifying artifacts haven't been tampered with since creation.

Cosign supports multiple key types: text-based keys, cloud KMS-based keys, hardware tokens, and Kubernetes Secrets.

**Keyless Signing**: A revolutionary approach eliminating manual key pair generation. The article notes that "Actions runs can get OIDC tokens from GitHub for use with cloud providers," enabling signing without provisioning private keys.

**Installation**:
```yaml
- uses: sigstore/[email protected]
```

**Container Image Signing Configuration**:
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

**Checksum File Signing Configuration**:
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

### 3. Verification

**Container Image Verification**:
```bash
$ COSIGN_EXPERIMENTAL=1 cosign verify ghcr.io/goreleaser/supply-chain-example:v1.2.0
```

**Blob Verification**:
```bash
$ COSIGN_EXPERIMENTAL=1 cosign verify-blob \
  --cert checksums.txt.pem \
  --signature checksums.txt.sig \
  checksums.txt
```

## Underlying Technologies

**Sigstore** (Open Software Security Foundation project) provides:
- **Fulcio**: Root CA issuing signing certificates from OIDC tokens
- **Rekor**: Transparency log for issued certificates

These enable "zero-friction keyless signing" without manual key management.

## Implementation Impact

The article demonstrates that GoReleaser integrates these security layers transparently. Configuration additions enable SBOM generation, signing, and verification "effortlessly" through configuration file modifications rather than complex workflow changes.

## Additional Considerations

The post references reproducible builds as complementary security practice -- "a set of software development practices that create an independently-verifiable path from source to binary code."

## Conclusion

GoReleaser mitigates supply chain risks through thoughtfully integrated open-source security tools, making institutional-grade artifact integrity verification accessible to projects of all sizes.
