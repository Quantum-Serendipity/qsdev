<!-- Source: https://goreleaser.com/customization/sign/sign/ -->
<!-- Retrieved: 2026-05-15 -->

# GoReleaser Signing Configuration Guide

## Overview

GoReleaser enables artifact signing to verify that releases come from you. The signing system works primarily with checksum files and supports multiple signing methods.

## Basic Configuration

To enable signing, add this minimal configuration:

```yaml
signs:
  - artifacts: checksum
```

This creates detached signatures for checksum files using GnuPG with your default key.

## Complete Configuration Options

The full signs block supports these parameters:

```yaml
signs:
  - id: foo                          # Unique identifier (default: 'default')
    signature: "${artifact}_sig"     # Output filename (default: '${artifact}.sig')
    cmd: gpg2                        # Signing command (default: 'gpg')
    args: ["--output", "${signature}", "--detach-sign", "${artifact}"]
    artifacts: all                   # What to sign (see types below)
    ids: [foo, bar]                  # Specific artifact IDs to sign
    if: '{{ eq .Os "linux" }}'       # Conditional signing
    stdin: "{{ .Env.GPG_PASSWORD }}" # Password via stdin
    stdin_file: ./.password          # Or from file
    certificate: '{{ trimsuffix .Env.artifact ".tar.gz" }}.pem'
    env:
      - FOO=bar
    output: true                     # Show command output in logs
```

## Artifact Types

The `artifacts` parameter accepts these values:

- **none** - No signing
- **checksum** - Checksum files only
- **all** - All artifacts
- **source** - Source archives
- **package** - Linux packages (deb, rpm, apk)
- **installer** - MSI, NSIS, macOS Pkg files
- **archive** - Tar/zip archives
- **sbom** - Generated SBOMs
- **binary** - Raw binaries (when `archives.format` is 'binary')

## Template Variables

Available in template-supporting fields:

- `${artifact}` - Path to the artifact being signed
- `${artifactID}` - Artifact identifier
- `${certificate}` - Certificate filename
- `${signature}` - Signature filename

## Cosign Integration

Sign artifacts with cosign using keyless signing:

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

## Custom Signing Commands

For commands that output to stdout, wrap them in a shell:

```yaml
signs:
  - cmd: sh
    args:
      - "-c"
      - 'echo "${artifact} is signed" | tee ${signature}'
    artifacts: all
```

Always use `${signature}` for the output filename and `${artifact}` for the source file.

## GPG-Specific Example

To sign with a specific GPG key:

```yaml
signs:
  - cmd: gpg
    args: ["-u", "<key-id>", "--output", "${signature}", "--detach-sign", "${artifact}"]
    artifacts: checksum
```

## macOS and Docker Signing

- **macOS executables**: Use the separate Notarization configuration
- **Docker images/manifests**: Refer to Docker-specific signing documentation
