# Using GoReleaser with GitLab: Multi-Arch Builds, Cosign, and SBOM Generation

- **Source**: https://containerinfra.nl/blog/2025/01/26/using-goreleaser-with-gitlab-multi-arch-builds-cosign-and-sbom-generation/
- **Retrieved**: 2026-05-15

## SBOM Integration Overview

"SBOMs provide a detailed list of all components and dependencies included in the software, which is essential for understanding potential vulnerabilities."

## Syft Configuration

SBOM generation utilizes Syft within the builds section:

```yaml
builds:
  - id: "platform"
    env:
      - CGO_ENABLED=0
    goos:
      - linux
    goarch:
      - amd64
    hooks:
      before:
        - syft registry.gitlab.com/yourproject/{{ .ProjectName }}:{{ .Version }} -o spdx-json > ./sbom.json
```

## Key Configuration Details

- **SBOM Format**: SPDX JSON format via the `-o spdx-json` flag
- **Execution Timing**: Syft command runs in the `before` hook, prior to the main build
- **Output Location**: `./sbom.json` alongside other release artifacts

## Practical Benefit

Provides "a detailed breakdown of dependencies and their licenses," supporting both compliance requirements and security auditing across multi-architecture releases.
