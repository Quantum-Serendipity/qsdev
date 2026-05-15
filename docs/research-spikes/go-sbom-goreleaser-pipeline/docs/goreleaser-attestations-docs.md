# GoReleaser Attestations Documentation

- **Source URL**: https://goreleaser.com/customization/publish/attestations/
- **Retrieved**: 2026-05-15

---

## Overview

GoReleaser supports GitHub Actions attestations for build artifacts, enabling users to verify the authenticity of released files and container images.

## GitHub Actions Integration

### Required Permissions

Your GitHub Actions workflow must include specific permissions:

```yaml
permissions:
  id-token: write
  attestations: write
```

These permissions allow the workflow to generate and write attestations for your artifacts.

## Workflow Configuration

### Basic Setup

After running GoReleaser, add attestation steps using `actions/attest@v4`:

```yaml
- uses: goreleaser/goreleaser-action@v7
  with:
    # your configuration
    
- uses: actions/attest@v4
  with:
    subject-checksums: ./dist/checksums.txt
    
- uses: actions/attest@v4
  if: startsWith(github.ref, 'refs/tags/v')
  with:
    subject-checksums: ./dist/digests.txt
```

The conditional check prevents attestation of snapshot builds that don't push Docker images.

## GoReleaser Configuration

Configure predictable filenames for checksums and digests:

```yaml
checksum:
  name_template: "checksums.txt"

docker_digest:
  name_template: "digests.txt"
```

## Verification

Users can verify attestations using the GitHub CLI:

```bash
gh attestation verify --owner <user-or-org> <filename>
gh attestation verify --owner <user-or-org> <image>
```

## Attestable Artifacts

- **Binaries and archives**: Attested via checksums file
- **Container images**: Attested via Docker digests file

## Additional Resources

Refer to the example-supply-chain repository for complete examples including signing and SBOMs.
