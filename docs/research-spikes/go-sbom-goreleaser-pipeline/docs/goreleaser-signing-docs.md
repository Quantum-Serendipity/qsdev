<!-- Source: https://goreleaser.com/customization/sign/sign/ -->
<!-- Retrieved: 2026-05-15 -->

# GoReleaser Signing Configuration Documentation

## Overview

GoReleaser's signing feature ensures artifact authenticity by generating cryptographic signatures that users can verify against your public key. Signing typically targets checksum files rather than individual artifacts.

## Core Configuration

### Basic Setup

To enable signing with default settings:

```yaml
signs:
  - artifacts: checksum
```

This creates detached signatures for checksum files using GnuPG with your default key.

## Configuration Options

### Key Fields

| Field | Purpose | Default |
|-------|---------|---------|
| `id` | Unique identifier for the signing config | 'default' |
| `signature` | Output signature filename | '${artifact}.sig' |
| `cmd` | Path to signing command | 'gpg' |
| `args` | Command-line arguments | `["--output", "${signature}", "--detach-sign", "${artifact}"]` |
| `artifacts` | What to sign (see types below) | 'none' |
| `ids` | Specific artifact IDs to sign | -- |
| `if` | Template condition for conditional signing | -- |
| `stdin` | Data passed to command via stdin | -- |
| `stdin_file` | File containing stdin data | -- |
| `certificate` | Certificate filename for keyless signing | -- |
| `env` | Environment variables for signing command | -- |
| `output` | Display command stdout/stderr | false |

### Artifact Types for Signing

The `artifacts` field accepts:
- `none` -- no signing
- `all` -- all artifacts
- `checksum` -- checksum files only
- `source` -- source archives
- `package` -- Linux packages (deb, rpm, apk)
- `installer` -- MSI, NSIS, macOS Pkgs
- `diskimage` -- macOS DMG images
- `archive` -- archives from archive pipeline
- **`sbom` -- generated SBOMs** (can sign SBOMs directly!)
- `binary` -- binaries (when format is 'binary')

## Template Variables

Available in templated fields:
- `${artifact}` -- path to artifact being signed
- `${artifactID}` -- artifact identifier
- `${certificate}` -- certificate filename
- `${signature}` -- signature filename

## Integration Examples

### Cosign Keyless Signing (Recommended)

```yaml
signs:
  - cmd: cosign
    signature: "${artifact}.sigstore.json"
    args:
      - "sign-blob"
      - "--bundle=${signature}"
      - "${artifact}"
      - "--yes"
    artifacts: checksum
```

Users verify with: `cosign verify-blob --bundle file.tar.gz.sigstore.json file.tar.gz`

### GnuPG Signing (Default)

```yaml
signs:
  - artifacts: checksum
```

### Custom Signing Command

```yaml
signs:
  - cmd: sh
    args:
      - "-c"
      - 'echo "${artifact} is signed" | tee ${signature}'
    artifacts: all
```

## Key Interactions with SBOMs

1. **Checksums + Signing Workflow:** GoReleaser recommends signing checksum files as the primary security measure. The checksums.txt file covers ALL artifacts including SBOMs.

2. **Direct SBOM Signing:** The `sbom` artifact type allows signing SBOMs directly if you need per-SBOM signatures rather than relying on checksum-level signing.

3. **Pipeline Order:** Build -> SBOM -> Checksum -> Sign. This means SBOMs are included in checksums, and signing the checksum transitively covers everything.

## Limitations & Notes

- The signing command must either write to a file or modify the signed artifact
- Commands writing only to stdout require wrapping in `sh -c`
- For macOS notarization, refer to separate notarization documentation
- Docker image signing uses distinct configuration (docker_signs section)
