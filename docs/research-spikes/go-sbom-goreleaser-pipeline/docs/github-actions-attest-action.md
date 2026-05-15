# GitHub Actions: actions/attest

- **Source URL**: https://github.com/actions/attest
- **Retrieved**: 2026-05-15

---

## Overview

The `actions/attest` action generates signed attestations for workflow artifacts using the in-toto format. It creates verifiable signatures through Sigstore-issued certificates, with public repositories using the public-good Sigstore instance and private repositories using GitHub's private instance.

## Required Permissions

```yaml
permissions:
  id-token: write        # OIDC token for Sigstore certificate
  attestations: write    # Persist attestations
  artifact-metadata: write # Create artifact storage records
```

## Attestation Modes

| Mode | Trigger | Purpose |
|------|---------|---------|
| **Provenance** | No sbom-path or predicate inputs | Auto-generates SLSA build provenance |
| **SBOM** | sbom-path provided | Creates attestation from SPDX or CycloneDX |
| **Custom** | predicate-type/predicate inputs | User-supplied predicate content |

## Key Inputs

- **subject-path**: Artifact path (supports globs; max 1024 subjects)
- **subject-digest**: SHA256 digest in format "sha256:hex_digest"
- **subject-name**: Artifact name for digest-based identification
- **subject-checksums**: Path to checksums file (shasum format)
- **sbom-path**: SPDX/CycloneDX JSON file (max 16MB)
- **predicate-type**: URI identifying predicate type
- **predicate**: String containing predicate value (max 16MB)
- **predicate-path**: File containing predicate content
- **push-to-registry**: Boolean for container image registry publishing
- **create-storage-record**: Boolean for artifact metadata records
- **show-summary**: Boolean for workflow run summary attachment
- **github-token**: Authentication token (defaults to github.token)

## Outputs

- **attestation-id**: GitHub attestation identifier
- **attestation-url**: URL to attestation summary
- **bundle-path**: Local filesystem path to attestation JSON
- **storage-record-ids**: GitHub IDs for storage records

## Usage Examples

### Basic Provenance Attestation

```yaml
name: build-attest-provenance

on:
  workflow_dispatch:

jobs:
  build:
    permissions:
      id-token: write
      contents: read
      attestations: write

    steps:
      - name: Checkout
        uses: actions/checkout@v4
      - name: Build artifact
        run: make my-app
      - name: Attest
        uses: actions/attest@v4
        with:
          subject-path: '${{ github.workspace }}/my-app'
```

### SBOM Attestation

```yaml
- name: Generate SBOM
  run: syft . -o spdx-json > sbom.spdx.json

- uses: actions/attest@v4
  with:
    subject-path: '${{ github.workspace }}/my-app'
    sbom-path: '${{ github.workspace }}/sbom.spdx.json'
```

### Custom Attestation

```yaml
- uses: actions/attest@v4
  with:
    subject-path: '${{ github.workspace }}/my-app'
    predicate-type: 'https://example.com/predicate/v1'
    predicate: '{}'
```

### Multiple Subjects (Wildcard)

```yaml
- uses: actions/attest@v4
  with:
    subject-path: 'dist/**/my-bin-*'
    predicate-type: 'https://example.com/predicate/v1'
    predicate: '{}'
```

### Checksums File

```yaml
- name: Calculate artifact digests
  run: |
    shasum -a 256 foo_0.0.1_* > subject.checksums.txt
- uses: actions/attest@v4
  with:
    subject-checksums: subject.checksums.txt
```

### Container Image with Registry Push

```yaml
name: build-attested-image

on:
  push:
    branches: [main]

jobs:
  build:
    runs-on: ubuntu-latest
    permissions:
      id-token: write
      packages: write
      contents: read
      attestations: write
      artifact-metadata: write
    env:
      REGISTRY: ghcr.io
      IMAGE_NAME: ${{ github.repository }}

    steps:
      - name: Checkout
        uses: actions/checkout@v4
      - name: Login to GitHub Container Registry
        uses: docker/login-action@v3
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - name: Build and push image
        id: push
        uses: docker/build-push-action@v5.0.0
        with:
          context: .
          push: true
          tags: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:latest
      - name: Attest
        uses: actions/attest@v4
        id: attest
        with:
          subject-name: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}
          subject-digest: ${{ steps.push.outputs.digest }}
          push-to-registry: true
```

## Verification

Attestations can be verified using the GitHub CLI: `gh attestation verify`

## Limits

- Maximum 1024 subjects per attestation
- Predicate content: 16MB maximum
- SBOM file size: 16MB maximum
