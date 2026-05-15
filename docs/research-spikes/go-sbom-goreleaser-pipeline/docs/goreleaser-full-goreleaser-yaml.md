<!-- Source: https://raw.githubusercontent.com/goreleaser/goreleaser/main/.goreleaser.yaml -->
<!-- Retrieved: 2026-05-15 -->

# GoReleaser's Own Complete .goreleaser.yaml

```yaml
# yaml-language-server: $schema=https://goreleaser.com/static/schema-pro.json
# vim: set ts=2 sw=2 tw=0 fo=jcroql
version: 2
pro: true

env:
  - GO111MODULE=on

before:
  hooks:
    - go mod tidy
    - ./scripts/completions_and_manpages.sh

snapshot:
  version_template: "{{ incpatch .Version }}-next"

gomod:
  proxy: true

report_sizes: true

git:
  ignore_tags:
    - nightly
    - "*-nightly"

metadata:
  mod_timestamp: "{{ .CommitTimestamp }}"

builds:
  - env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - "386"
      - amd64
      - arm
      - arm64
      - loong64
      - ppc64
      - riscv64
    goarm:
      - "7"
    ignore:
      - goos: windows
        goarch: arm
    mod_timestamp: "{{ .CommitTimestamp }}"
    flags:
      - -trimpath
    ldflags:
      - -s -w -X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{ .CommitDate }} -X main.builtBy=goreleaser -X main.treeState={{ .IsGitDirty }}

universal_binaries:
  - replace: false

notarize:
  macos:
    - enabled: '{{ isEnvSet "MACOS_SIGN_P12" }}'
      sign:
        certificate: "{{.Env.MACOS_SIGN_P12}}"
        password: "{{.Env.MACOS_SIGN_PASSWORD}}"
      notarize:
        issuer_id: "{{.Env.MACOS_NOTARY_ISSUER_ID}}"
        key_id: "{{.Env.MACOS_NOTARY_KEY_ID}}"
        key: "{{.Env.MACOS_NOTARY_KEY}}"

checksum:
  name_template: "checksums.txt"

changelog:
  sort: asc
  use: github
  # ... (changelog config omitted for brevity)

dockers_v2:
  - images:
      - "goreleaser/goreleaser"
      - "ghcr.io/goreleaser/goreleaser"
    tags:
      - "v{{ .Version }}"
      - "{{ if .IsNightly }}nightly{{ end }}"
      - "{{ if not .IsNightly }}latest{{ end }}"
    extra_files:
      - scripts/entrypoint.sh
    labels:
      # ... OCI labels
    annotations:
      "org.opencontainers.image.description": "Release engineering, simplified"

archives:
  - name_template: >-
      {{- .ProjectName }}_
      {{- title .Os }}_
      {{- if eq .Arch "amd64" }}x86_64
      {{- else if eq .Arch "386" }}i386
      {{- else }}{{ .Arch }}{{ end }}
      {{- if .Arm }}v{{ .Arm }}{{ end -}}
    format_overrides:
      - goos: windows
        formats: [zip]
    builds_info:
      group: root
      owner: root
      mtime: "{{ .CommitDate }}"

# Package manager distributions
homebrew_casks:
  - repository:
      owner: goreleaser
      name: homebrew-tap
      token: "{{ .Env.GH_PAT }}"

nix:
  - name: goreleaser
    repository:
      owner: goreleaser
      name: nur
      token: "{{ .Env.GH_PAT }}"
    path: pkgs/goreleaser/default.nix
    license: mit
    extra_install: |-
      installManPage ./manpages/goreleaser.1.gz
      installShellCompletion ./completions/*

winget:
  - name: goreleaser
    publisher: goreleaser
    license: MIT

aurs:
  - homepage: https://goreleaser.com
    description: Release engineering, simplified

scoops:
  - repository:
      owner: goreleaser
      name: scoop-bucket
      token: "{{ .Env.GH_PAT }}"

npms:
  - name: "@goreleaser/goreleaser"

nfpms:
  - file_name_template: "{{ .ConventionalFileName }}"
    id: packages
    formats:
      - apk
      - deb
      - rpm
      - archlinux

snapcrafts:
  - name_template: "{{ .ProjectName }}_{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}"

flatpak:
  - app_id: com.goreleaser.GoReleaser

# SBOM - minimal config, defaults to syft
sboms:
  - artifacts: archive

# Signing - cosign keyless via sigstore
signs:
  - cmd: cosign
    signature: "${artifact}.sigstore.json"
    output: '{{ not (isEnvSet "CI" )}}'
    artifacts: checksum
    args:
      - sign-blob
      - "--bundle=${signature}"
      - "${artifact}"
      - --yes

docker_signs:
  - cmd: cosign
    artifacts: manifests
    output: '{{ not (isEnvSet "CI" )}}'
    args:
      - "sign"
      - "${artifact}@${digest}"
      - --yes

milestones:
  - close: true

nightly:
  publish_release: true
  tag_name: "{{ incminor .Tag }}-{{ .ShortCommit }}-nightly"
  version_template: "{{ incminor .Version }}-{{ .ShortCommit }}-nightly"

release:
  name_template: "{{ .Tag }}"
  prerelease: auto
```
