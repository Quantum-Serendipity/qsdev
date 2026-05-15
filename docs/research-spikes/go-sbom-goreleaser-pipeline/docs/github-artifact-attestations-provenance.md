<!-- Source: https://docs.github.com/en/actions/security-for-github-actions/using-artifact-attestations/using-artifact-attestations-to-establish-provenance-for-builds -->
<!-- Retrieved: 2026-05-15 -->

# GitHub Artifact Attestations: Establishing Provenance for Builds

## Purpose and Scope

Artifact attestations enable you to increase the supply chain security of your builds by establishing where and how your software was built. They work with binaries, container images, and software bill of materials (SBOMs).

## Supported Artifact Types

1. **Binaries** - standalone executable files
2. **Container images** - Docker and similar formats
3. **SBOMs** - signed software bill of materials in SPDX or CycloneDX formats

## Generating Attestations: Core Requirements

All attestation workflows require:
- The `actions/attest@v4` action
- Three permissions: `id-token: write`, `contents: read`, and `attestations: write`
- Container images need an additional `packages: write` permission

### Binary Attestations

```yaml
- name: Generate artifact attestation
  uses: actions/attest@v4
  with:
    subject-path: 'PATH/TO/ARTIFACT'
```

### Container Image Attestations

```yaml
- name: Generate artifact attestation
  uses: actions/attest@v4
  with:
    subject-name: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}
    subject-digest: 'sha256:fedcba0...'
    push-to-registry: true
```

### SBOM Attestations

**For binaries:**
```yaml
- name: Generate SBOM attestation
  uses: actions/attest@v4
  with:
    subject-path: 'PATH/TO/ARTIFACT'
    sbom-path: 'PATH/TO/SBOM'
```

**For container images:**
```yaml
- name: Generate SBOM attestation
  uses: actions/attest@v4
  with:
    subject-name: ${{ env.REGISTRY }}/PATH/TO/IMAGE
    subject-digest: 'sha256:fedcba0...'
    sbom-path: 'sbom.json'
    push-to-registry: true
```

## Verification with GitHub CLI

### Binary Verification
```bash
gh attestation verify PATH/TO/YOUR/BUILD/ARTIFACT-BINARY -R ORGANIZATION_NAME/REPOSITORY_NAME
```

### Container Image Verification
```bash
docker login ghcr.io
gh attestation verify oci://ghcr.io/ORGANIZATION_NAME/IMAGE_NAME:test -R ORGANIZATION_NAME/REPOSITORY_NAME
```

### SBOM Verification
```bash
gh attestation verify PATH/TO/YOUR/BUILD/ARTIFACT-BINARY \
  -R ORGANIZATION_NAME/REPOSITORY_NAME \
  --predicate-type https://spdx.dev/Document/v2.3
```

To view detailed attestation data in JSON:
```bash
gh attestation verify PATH/TO/YOUR/BUILD/ARTIFACT-BINARY \
  -R ORGANIZATION_NAME/REPOSITORY_NAME \
  --predicate-type https://spdx.dev/Document/v2.3 \
  --format json \
  --jq '.[].verificationResult.statement.predicate'
```

## Important Notes

- Attestations appear in your repository's Actions tab
- Uses Sigstore standards for SBOM predicates and in-toto specifications
- Offline verification options exist for air-gapped environments
- Public repos use Sigstore's public-good instance; private repos use GitHub's private Sigstore instance
- Lifecycle management available for deleting obsolete attestations
