# Using Artifact Attestations to Establish Provenance for Builds

- **Source URL**: https://docs.github.com/en/actions/how-tos/secure-your-work/use-artifact-attestations/use-artifact-attestations
- **Retrieved**: 2026-05-15

---

## Overview

Artifact attestations enable you to increase the supply chain security of your builds by establishing where and how your software was built. You can attest binaries, container images, and software bills of materials (SBOMs).

## Binary Attestation Workflow

**Required Permissions:**
```yaml
permissions:
  id-token: write
  contents: read
  attestations: write
```

**Attestation Step:**
```yaml
- name: Generate artifact attestation
  uses: actions/attest@v4
  with:
    subject-path: 'PATH/TO/ARTIFACT'
```

## Container Image Attestation Workflow

**Required Permissions:**
```yaml
permissions:
  id-token: write
  contents: read
  attestations: write
  packages: write
```

**Attestation Step:**
```yaml
- name: Generate artifact attestation
  uses: actions/attest@v4
  with:
    subject-name: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}
    subject-digest: 'sha256:fedcba0...'
    push-to-registry: true
```

## SBOM Attestation for Binaries

```yaml
- name: Generate SBOM attestation
  uses: actions/attest@v4
  with:
    subject-path: 'PATH/TO/ARTIFACT'
    sbom-path: 'PATH/TO/SBOM'
```

## SBOM Attestation for Container Images

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

**Verify binary attestations:**
```bash
gh attestation verify PATH/TO/YOUR/BUILD/ARTIFACT-BINARY -R ORGANIZATION_NAME/REPOSITORY_NAME
```

**Verify container image attestations:**
```bash
docker login ghcr.io
gh attestation verify oci://ghcr.io/ORGANIZATION_NAME/IMAGE_NAME:test -R ORGANIZATION_NAME/REPOSITORY_NAME
```

**Verify SPDX SBOM attestations:**
```bash
gh attestation verify PATH/TO/YOUR/BUILD/ARTIFACT-BINARY \
  -R ORGANIZATION_NAME/REPOSITORY_NAME \
  --predicate-type https://spdx.dev/Document/v2.3
```

**View detailed JSON output:**
```bash
gh attestation verify PATH/TO/YOUR/BUILD/ARTIFACT-BINARY \
  -R ORGANIZATION_NAME/REPOSITORY_NAME \
  --predicate-type https://spdx.dev/Document/v2.3 \
  --format json \
  --jq '.[].verificationResult.statement.predicate'
```

## Linked Artifacts Integration

The attest action automatically creates storage records on your organization's linked artifacts page when both conditions are met:
- `push-to-registry` is set to `true`
- Workflow has the `artifact-metadata: write` permission

This enables tracking build history, deployment records, and storage details for vulnerability prioritization and team attribution.
