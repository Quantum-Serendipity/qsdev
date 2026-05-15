<!-- Source: https://containerinfra.nl/blog/2025/01/26/using-goreleaser-with-gitlab-multi-arch-builds-cosign-and-sbom-generation/ -->
<!-- Retrieved: 2026-05-15 -->

# GoReleaser with GitLab: Multi-Arch Builds, Cosign, and SBOM Generation

## Cosign Configuration

Signing Docker images with Cosign:

```yaml
docker_signs:
  - artifacts: all
    args: ["sign", "--key=${COSIGN_KEY}", "--tlog-upload=false", 
           "-a", "builder=gitlab-promote", "${artifact}@${digest}"]
```

This ensures every image deployed can be verified for authenticity by passing the signing key through environment variables.

## SBOM Generation

Software Bill of Materials creation using Syft:

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
        - syft registry.gitlab.com/yourproject/{{ .ProjectName }}:{{ .Version }} 
          -o spdx-json > ./sbom.json
```

Generates SPDX-formatted documentation of all dependencies and licenses.

## GitLab CI Pipeline Setup

```yaml
.go:goreleaser:
  extends:
    - .go:release
    - .docker
  image:
    name: goreleaser/goreleaser:v2.3.2
    entrypoint: ['']
  services:
    - name: docker:26-dind
  script:
    - docker run --privileged --rm tonistiigi/binfmt:qemu-v6.2.0 --install all
    - docker login -u gitlab-ci-token -p $CI_JOB_TOKEN $CI_REGISTRY
    - goreleaser release --clean --skip-validate
```

The pipeline installs QEMU for cross-architecture compilation and authenticates with the container registry automatically.
