<!-- Source: https://goreleaser.com/customization/package/dockers_v2/ -->
<!-- Retrieved: 2026-05-15 -->

# GoReleaser Docker (v2) Configuration Documentation

## Overview

Docker v2 is an experimental feature (since v2.12) that uses `docker buildx` to build multi-architecture manifests, reusing previously compiled binaries and packages.

## Core Configuration Fields

### Basic Image Setup

**Images & Tags:**
- `images`: List of image names (templates allowed). Empty names are ignored.
- `tags`: Version tags (templates allowed). Empty tags are ignored.
- `id`: Image identifier for filtering later in custom publishers

**Example:**
```yaml
dockers_v2:
  - id: myimg
    images:
      - "myuser/myimage"
      - "gcr.io/myuser/myimage"
    tags:
      - "v{{ .Version }}"
      - "{{ if .IsNightly }}nightly{{ end }}"
```

### Build Configuration

- `dockerfile`: Path to Dockerfile (default: `'Dockerfile'`, templates allowed)
- `ids`: Filter binaries/packages to include (match against `builds` and `nfpms` sections)
- `extra_files`: Additional source files to copy into build context (relative paths, no wildcards)
- `build_args`: Additional `--build-arg` values (templates allowed)
- `flags`: Arbitrary build command flags (must use `=` syntax, templates allowed)
- `platforms`: Target architectures (default: `[linux/amd64, linux/arm64]`, templates allowed since v2.14)

## SBOM & Attestation Configuration

**SBOM Attachment:**
- `sbom`: Whether to create and attach a SBOM to the image (default: `'true'`, templates allowed, since v2.12)

**Key Insight:** SBOMs are ON by default for Docker v2 images. The documentation states "create and attach a SBOM to the image," indicating SBOMs are generated and associated with the resulting Docker/OCI artifact automatically.

**Single-Architecture Image Workaround:**

If building single-arch images and preferring Images over Manifests, disable SBOMs and attestations:

```yaml
dockers_v2:
  - images:
      - foo
    tags:
      - latest
    platforms:
      - linux/amd64
    sbom: false
    flags:
      - "--provenance=false"
```

### Metadata Configuration

**Labels & Annotations:**
- `labels`: OCI-compliant metadata (empty keys/values ignored, templates allowed)
- `annotations`: Additional image annotations (empty keys/values ignored, templates allowed)

## Conditional & Control Options

- `disable`: Conditionally disable configuration (templates allowed, since v2.12)
- `retry`: Configure retry behavior for failed builds
  - `attempts`: Retry count (default: 10)
  - `delay`: Delay between attempts (default: 10s)
  - `max_delay`: Maximum delay (default: 5m)

## Build Context Structure

GoReleaser creates a temporary directory containing:
```
temp-context-dir/
├── Dockerfile
├── linux/arm64/myprogram
├── linux/arm64/myprogram.rpm
├── linux/amd64/myprogram
└── linux/amd64/myprogram.deb
```

**Critical Design Principle:** "Reuse the previously built binaries instead of building them again when creating the Docker image."

## Testing with Snapshots

Running `goreleaser release --snapshot` builds separate platform-specific images with suffixes (e.g., `user/repo:1.2.4-amd64`, `user/repo:1.2.4-arm64`) instead of a manifest, enabling local verification without pushing.
