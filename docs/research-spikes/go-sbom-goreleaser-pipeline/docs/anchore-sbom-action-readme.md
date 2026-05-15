# Anchore SBOM Action (anchore/sbom-action)

- **Source URL**: https://github.com/anchore/sbom-action
- **Retrieved**: 2026-05-15

---

## Overview

The anchore/sbom-action is a GitHub Action that automates software bill of materials (SBOM) generation using Syft, an open-source tool for generating SBOMs from container images and filesystems.

## Core Functionality

- Automatic SBOM generation and workflow artifact uploads
- Release asset attachment during GitHub release events
- Dependency submission API integration
- Support for filesystem paths, individual files, and container images
- Configurable output formats and naming conventions

## Supported Input Scenarios

### 1. Scan Filesystem Directory
```yaml
- uses: anchore/sbom-action@v0
  with:
    path: ./build/
```

### 2. Scan Individual File
```yaml
- uses: anchore/sbom-action@v0
  with:
    file: ./build/file
```

### 3. Scan Container Image
```yaml
- uses: anchore/sbom-action@v0
  with:
    image: ghcr.io/example/image_name:tag
```

### 4. Basic Usage (Default)
```yaml
- uses: anchore/sbom-action@v0
```

## Output Formats

- `spdx` -- SPDX tag-value format
- `spdx-json` -- SPDX JSON format (default)
- `cyclonedx` -- CycloneDX XML format
- `cyclonedx-json` -- CycloneDX JSON format

## Release Asset Integration

The action auto-detects GitHub release events and automatically uploads generated SBOMs as release assets, eliminating manual attachment steps.

**Required permissions for release uploads:**
```yaml
jobs:
  build:
    permissions:
      actions: read
      contents: write
```

## Configuration Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `path` | Filesystem path to scan | Current directory |
| `file` | Single file to scan | -- |
| `image` | Container image URI | -- |
| `artifact-name` | Custom artifact name | Auto-generated |
| `output-file` | Output file location | -- |
| `format` | SBOM format | `spdx-json` |
| `dependency-snapshot` | Upload to dependency API | `false` |
| `upload-artifact` | Create workflow artifact | `true` |
| `upload-artifact-retention` | Artifact retention (days) | -- |
| `upload-release-assets` | Attach to releases | `true` |
| `syft-version` | Syft binary version | Latest |
| `github-token` | GitHub authentication token | `github.token` |
| `config` | Syft configuration file path | -- |

## Dependency Snapshot Submission

Enable GitHub's dependency submission API integration:
```yaml
- uses: anchore/sbom-action@v0
  with:
    dependency-snapshot: true
```

This uploads SBOM data to GitHub's vulnerability tracking system.

## Sub-actions

### publish-sbom
```yaml
- uses: anchore/sbom-action/publish-sbom@v0
  with:
    sbom-artifact-match: ".*\\.spdx$"
```
Enables uploading externally-generated or pre-existing SBOMs to releases via regex pattern matching.

### download-syft
Downloads standalone Syft binary for independent use within workflows.
