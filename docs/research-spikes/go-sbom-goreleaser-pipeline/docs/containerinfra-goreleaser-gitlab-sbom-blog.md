<!-- Source: https://containerinfra.nl/blog/2025/01/26/using-goreleaser-with-gitlab-multi-arch-builds-cosign-and-sbom-generation/ -->
<!-- Retrieved: 2026-05-15 -->

# Using GoReleaser with GitLab: Multi-Arch Builds, Cosign, and SBOM Generation

## Overview
This article from ContainerInfra details automating multi-architecture builds for AMD64 and ARM64 using GoReleaser and GitLab CI, with container signing via Cosign and SBOM generation.

## Key Configuration Examples

### Basic GoReleaser Setup
```yaml
version: 2

gitlab_urls:
  api: https://gitlab.com/api/v4/
  download: https://gitlab.com
```

### Multi-Architecture Builds
```yaml
builds:
  - env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
    goarch:
      - amd64
      - arm64
```

### Docker Image Templates (v1 style with buildx)
```yaml
dockers:
  - image_templates: ["registry.gitlab.com/yourgroup/{{ .ProjectName }}:{{ .Version }}-amd64"]
    use: buildx
    dockerfile: Dockerfile
    goos: linux
    goarch: amd64
    build_flag_templates:
      - "--platform=linux/amd64"
  - image_templates: ["registry.gitlab.com/yourgroup/{{ .ProjectName }}:{{ .Version }}-arm64v8"]
    use: buildx
    dockerfile: Dockerfile
    goos: linux
    goarch: arm64
    build_flag_templates:
      - "--platform=linux/arm64/v8"
```

### Docker Manifest Configuration
```yaml
docker_manifests:
  - name_template: "registry.gitlab.com/yourgroup/{{ .ProjectName }}:{{ .Version }}"
    image_templates:
      - "registry.gitlab.com/yourgroup/{{ .ProjectName }}:{{ .Version }}-amd64"
      - "registry.gitlab.com/yourgroup/{{ .ProjectName }}:{{ .Version }}-arm64v8"
```

### Cosign Image Signing
```yaml
docker_signs:
  - artifacts: all
    args: ["sign", "--key=${COSIGN_KEY}", "--tlog-upload=false", "-a",
           "builder=gitlab-promote", "${artifact}@${digest}"]
```

### SBOM Generation with Syft (hook-based approach)
This blog uses a build hook rather than the native `sboms:` block:
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

Note: This hook-based approach is less idiomatic than using the native `sboms:` configuration block. It's shown here for GitLab-specific contexts where the native block may not work as expected with container images.

## GitLab CI Pipeline

### CI Variables Setup
```yaml
variables:
  GOPATH: $CI_PROJECT_DIR/.go
  REGISTRY_USERNAME: gitlab-ci-token
  REGISTRY_PASSWORD: $CI_JOB_TOKEN
  REGISTRY_NAME: $CI_REGISTRY
  GITLAB_TOKEN: $YOUR_GITLAB_ENV_TOKEN
  NETRC: machine gitlab.com login gitlab-ci-token password $GITLAB_TOKEN
  GOPRIVATE: gitlab.com
```

### GoReleaser CI Template
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
  only:
    - tags
  script:
    - echo $NETRC > ~/.netrc
    - docker run --privileged --rm tonistiigi/binfmt:qemu-v6.2.0 --install all
    - docker login -u gitlab-ci-token -p $CI_JOB_TOKEN $CI_REGISTRY
    - goreleaser release --clean --skip-validate
```

## Key Insight
This blog demonstrates that for container image SBOMs in non-GitHub contexts (GitLab), a hook-based approach may be necessary since GoReleaser's native `sboms:` block cannot catalog container images directly. The workaround is to run syft against the registry image in a build hook.
