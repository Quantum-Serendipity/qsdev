# GoReleaser SBOM Integration: Deep Research Report

## Executive Summary

GoReleaser provides first-class, tool-agnostic SBOM generation through its `sboms:` configuration block, available in the free/OSS edition since v1.2.0 (December 2021). The default configuration uses Anchore's Syft to generate SPDX-JSON format SBOMs for each archive artifact, but the design supports any SBOM generator (cyclonedx-gomod, trivy, custom scripts) via the `cmd` field. SBOMs are placed in the dist directory, included in the checksums.txt file, and uploaded as GitHub Release assets alongside binaries. For Docker images, the newer `dockers_v2:` block generates and attaches SBOMs to OCI images by default (since v2.12). The recommended qsdev configuration is minimal: `sboms: [{artifacts: archive}]` plus cosign signing of the checksum file for full supply chain coverage.

---

## 1. Configuration Reference

### The `sboms:` Block

Source: [GoReleaser SBOM Documentation](https://goreleaser.com/customization/sbom/)

The `sboms:` block accepts a list of SBOM configurations. Each entry has:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `id` | string | `'default'` | Unique identifier (required when multiple configs) |
| `cmd` | string | `'syft'` | Path to SBOM generator binary. CWD is set to dist dir |
| `documents` | []string | varies by artifact type | Output filenames (templates allowed) |
| `args` | []string | `["$artifact", "--output", "spdx-json=$document", "--enrich", "all"]` | Arguments passed to cmd |
| `env` | []string | `["SYFT_FILE_METADATA_CATALOGER_ENABLED=true"]` | Environment variables |
| `artifacts` | string | `'archive'` | Which artifacts to catalog (see below) |
| `ids` | []string | all | Filter to specific artifact IDs |
| `disable` | bool | `true` | Disable this config (since v2.10, templates allowed) |

### Artifact Types

| Value | Description | Pro Required? |
|-------|-------------|---------------|
| `archive` | Archives from the archive pipe | No |
| `binary` | Binaries from build stage | No |
| `source` | Source archive | No |
| `package` | Linux packages (deb, rpm, apk) | No |
| `any` | Tool determines what to catalog | No |
| `installer` | MSI, NSIS, macOS pkg | **Yes (Pro)** |
| `diskimage` | macOS DMG disk images | **Yes (Pro)** |

### Document Naming Defaults

The default `documents` template varies by artifact type:
- **Binary**: `["{{ .Binary }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}.sbom.json"]`
- **Any**: `[]` (empty -- tool determines output)
- **All others**: `["{{ .ArtifactName }}.sbom.json"]`

### Template Variables

Available in `args`, `documents`, `env`, and `disable`:
- `${artifact}` -- Path to the artifact being cataloged (unavailable for `any`)
- `${artifactID}` -- ID of the artifact being cataloged
- `${document}` -- Generated SBOM filename (alias for `${document0}`)
- `${document0}`, `${document1}`, ... -- Indexed SBOM filenames (for multi-document)

---

## 2. Supported Generators

GoReleaser's SBOM support is **tool-agnostic**: it runs an external command and captures output files. Any tool that can write an SBOM to a file path works.

### Syft (Default)

- **Maintainer**: Anchore
- **Default args**: `["$artifact", "--output", "spdx-json=$document", "--enrich", "all"]`
- **Formats**: SPDX-JSON (default), CycloneDX-JSON, SPDX-tag-value, and many others
- **Strengths**: Multi-format support, file metadata cataloging, enrichment from external sources
- **Installation**: `anchore/sbom-action/download-syft@v0` GitHub Action or direct binary

To switch Syft's output to CycloneDX:
```yaml
sboms:
  - args: ["$artifact", "--output", "cyclonedx-json=$document"]
```

### cyclonedx-gomod

- **Maintainer**: CycloneDX project
- **Usage**: Set `cmd: cyclonedx-gomod`
- **Strengths**: Go-native, includes license information, understands Go module structure deeply
- **Example** (from OWASP Amass):
```yaml
sboms:
  - documents:
      - "{{ .Binary }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}.bom.json"
    artifacts: binary
    cmd: cyclonedx-gomod
    args: ["app", "-licenses", "-json", "-output", "$document", "../"]
    env:
      - GOARCH={{ .Arch }}
      - GOOS={{ .Os }}
```

### Trivy

- **Maintainer**: Aqua Security
- **Usage**: Set `cmd: trivy` with appropriate args
- **Note**: Trivy is primarily a scanner that can also generate SBOMs; less common in GoReleaser configs

### Custom Scripts

Any script or binary that writes an SBOM file can be used:
```yaml
sboms:
  - cmd: ./scripts/generate-sbom.sh
    args: ["$artifact", "$document"]
```

---

## 3. How SBOMs Attach to GitHub Releases

### Pipeline Execution Order

GoReleaser's pipeline runs in this order:

1. **Build** -- compile binaries
2. **Archive** -- create tar.gz/zip archives
3. **SBOM** -- generate SBOMs for specified artifacts
4. **Checksum** -- compute checksums.txt covering ALL artifacts including SBOMs
5. **Sign** -- sign the checksum file (and/or individual artifacts)
6. **Release** -- upload everything to GitHub

This ordering is critical: SBOMs are generated BEFORE checksums, so checksums.txt includes SBOM file hashes. Signing the checksum file transitively covers all artifacts and SBOMs.

### Naming Conventions

With default configuration (`artifacts: archive`), a release with archives for linux/amd64 and darwin/arm64 would produce:

```
qsdev_1.0.0_Linux_x86_64.tar.gz
qsdev_1.0.0_Linux_x86_64.tar.gz.sbom.json     # SBOM for this archive
qsdev_1.0.0_Darwin_arm64.tar.gz
qsdev_1.0.0_Darwin_arm64.tar.gz.sbom.json      # SBOM for this archive
checksums.txt                                     # covers all above
checksums.txt.sigstore.json                       # cosign signature
```

### Per-Artifact vs Per-Release SBOMs

- **Per-artifact** (default): One SBOM per archive/binary. Each SBOM catalogs the specific artifact.
- **Per-release** (`artifacts: any`): The SBOM tool decides scope. With `artifacts: any`, the `documents` list must be empty (tool determines output), and `${artifact}` template variable is unavailable.
- **Source SBOM**: Using `artifacts: source` generates an SBOM for the source tarball, which captures the full module dependency tree at source level.

Multiple configs can coexist:
```yaml
sboms:
  - artifacts: archive        # one SBOM per archive
  - id: source                # separate ID required
    artifacts: source         # one SBOM for source tarball
```

### Checksums File Integration

All SBOM files are listed in `checksums.txt` alongside binary archives:
```
sha256:abc123  qsdev_1.0.0_Linux_x86_64.tar.gz
sha256:def456  qsdev_1.0.0_Linux_x86_64.tar.gz.sbom.json
sha256:ghi789  qsdev_1.0.0_Darwin_arm64.tar.gz
sha256:jkl012  qsdev_1.0.0_Darwin_arm64.tar.gz.sbom.json
```

This means signing only the checksum file is sufficient to verify integrity of everything.

---

## 4. GoReleaser Pro vs OSS

### Core SBOM features: ALL FREE/OSS

The `sboms:` configuration block and all common artifact types (`archive`, `binary`, `source`, `package`, `any`) are fully available in the free/OSS GoReleaser.

### Pro-only SBOM-related features

Only two artifact types are gated behind Pro, because the artifact types themselves are Pro features:
- **`installer`** -- SBOMs for MSI, NSIS, macOS pkg installers (added v2.15)
- **`diskimage`** -- SBOMs for macOS DMG disk images

### Pro-only features NOT related to SBOMs

Nightly builds, monorepo support, macOS notarization, Windows MSI/NSIS installers, NPM publishing, configuration includes, and ~25 other features require Pro. See `docs/goreleaser-pro-features-list.md` for the complete list.

**Bottom line for qsdev**: No Pro license needed for SBOM generation. All required features are in the free OSS version.

---

## 5. Docker Image SBOM Attachment

### Docker v2 (Recommended, since v2.12)

The `dockers_v2:` block uses `docker buildx` and produces multi-architecture manifests. It has a dedicated SBOM field:

```yaml
dockers_v2:
  - images:
      - "ghcr.io/org/qsdev"
    tags:
      - "{{ .Tag }}"
      - latest
    sbom: true    # DEFAULT -- SBOMs are ON by default
```

- **`sbom`** field (default: `'true'`): Creates and attaches an SBOM to the OCI image via `docker buildx --sbom=true`
- The SBOM is stored as an OCI artifact associated with the image in the registry
- Templates are supported (since v2.12), so you can conditionally disable it

### Important Limitation: `sboms:` Block Cannot Catalog Container Images

From the official docs: **"Container images generated by GoReleaser are not available to be cataloged by the SBOM tool."**

This means the `sboms:` configuration block (which runs syft/cyclonedx-gomod/etc.) cannot generate SBOMs for Docker images. Docker image SBOMs come ONLY from the `dockers_v2:` block's built-in `sbom` field (which delegates to `docker buildx`).

### Workaround for Docker v1 or Custom Image SBOMs

If using the older `dockers:` block or needing custom container SBOMs, use a build hook:
```yaml
builds:
  - hooks:
      post:
        - syft ghcr.io/org/qsdev:{{ .Version }} -o spdx-json > sbom-container.json
```

### Docker Image Signing

Docker image signing is separate from SBOM generation, configured via `docker_signs:`:
```yaml
docker_signs:
  - cmd: cosign
    artifacts: manifests    # or "images" or "all"
    args:
      - "sign"
      - "${artifact}@${digest}"
      - "--yes"
```

---

## 6. Package Manager Distribution and SBOMs

### Do SBOMs Flow to Package Managers?

**No.** SBOMs do NOT automatically flow through to Homebrew, Scoop, Nix, Winget, APT/RPM, or any other package manager distribution channel.

Package manager integrations (Homebrew taps, Scoop buckets, Nix derivations) reference the binary archives on GitHub Releases. They do not include SBOM files. The SBOMs exist only as GitHub Release assets.

### How Consumers Access SBOMs

1. **GitHub Releases page**: SBOMs appear as downloadable assets alongside binaries
2. **Direct URL**: `https://github.com/org/qsdev/releases/download/v1.0.0/qsdev_1.0.0_Linux_x86_64.tar.gz.sbom.json`
3. **gh CLI**: `gh release download v1.0.0 --pattern '*.sbom.json'`
4. **OCI registries**: For Docker images using `dockers_v2:`, SBOMs are attached to the OCI manifest

### Nix-Specific Consideration

For Nix users who build from source, the GoReleaser-generated SBOM documents the upstream release. A Nix-native SBOM would need to be generated separately from the Nix derivation's build inputs, which is a different problem space.

---

## 7. Signing Integration

### Pipeline Order Enables Transitive Trust

```
Build -> Archive -> SBOM -> Checksum -> Sign -> Release
```

Because SBOMs are generated before checksums, and checksums are generated before signing:
- Signing the checksum file provides integrity verification for ALL artifacts including SBOMs
- No need to sign each SBOM individually (though you can)

### Recommended Configuration (Cosign Keyless)

```yaml
signs:
  - cmd: cosign
    signature: "${artifact}.sigstore.json"
    args:
      - sign-blob
      - "--bundle=${signature}"
      - "${artifact}"
      - "--yes"
    artifacts: checksum
```

This uses Sigstore's keyless signing via GitHub Actions OIDC tokens. No keys to manage.

### Direct SBOM Signing (Optional)

If you need per-SBOM signatures (e.g., for compliance):
```yaml
signs:
  - id: sbom-sign
    cmd: cosign
    signature: "${artifact}.sigstore.json"
    args:
      - sign-blob
      - "--bundle=${signature}"
      - "${artifact}"
      - "--yes"
    artifacts: sbom    # sign SBOM files directly
```

### Verification by Consumers

```bash
# Verify checksum signature
cosign verify-blob --bundle checksums.txt.sigstore.json checksums.txt

# Verify individual file against checksums
sha256sum --check checksums.txt --ignore-missing

# Inspect SBOM
cat qsdev_1.0.0_Linux_x86_64.tar.gz.sbom.json | jq .

# Scan SBOM for vulnerabilities
grype sbom:qsdev_1.0.0_Linux_x86_64.tar.gz.sbom.json
```

---

## 8. Version History and Evolution

| Version | Date | SBOM-Related Changes |
|---------|------|---------------------|
| v1.2.0 | Dec 2021 | Initial SBOM support added (PR #2648 by wagoodman/Anchore) |
| v1.x | 2022-2024 | Gradual refinement, bug fixes |
| v2.0.0 | Jun 2024 | v2 release (same as v1.26.2 with deprecated options removed) |
| v2.10 | Late 2024 | `disable` field added (templateable) |
| v2.12 | Early 2025 | Docker v2 with built-in SBOM attachment; `sbom` field defaults to `true` |
| v2.13 | 2025 | Default args changed to include `--enrich all` for richer SBOMs |
| v2.15 | 2025 | SBOM pipe now covers `installer` artifact type (Pro) |

### Default Args Evolution

The default `args` have changed over time:
- **Original**: `["$artifact", "--output", "spdx-json=$document"]`
- **Current** (v2.13+): `["$artifact", "--output", "spdx-json=$document", "--enrich", "all"]`

The `--enrich all` flag was added to enable Syft's enrichment feature, which augments SBOM data with information from external sources (e.g., matching CPEs, license databases).

---

## 9. Real-World Examples

### Minimal (GoReleaser itself, Syft, k8sgpt)

```yaml
sboms:
  - artifacts: archive
```

Used by: [goreleaser/goreleaser](https://github.com/goreleaser/goreleaser), [anchore/syft](https://github.com/anchore/syft), [k8sgpt-ai/k8sgpt](https://github.com/k8sgpt-ai/k8sgpt)

### Archive + Source (GoReleaser supply-chain example, FluxCD)

```yaml
sboms:
  - artifacts: archive
  - id: source
    artifacts: source
```

Used by: [goreleaser/example-supply-chain](https://github.com/goreleaser/goreleaser-example-supply-chain), [fluxcd/source-watcher](https://github.com/fluxcd/source-watcher)

### Per-Binary with Custom Naming (Defense Unicorns UDS CLI)

```yaml
sboms:
  - artifacts: binary
    documents:
      - "sbom_{{ .ProjectName }}_{{ .Tag }}_{{- title .Os }}_{{ .Arch }}.sbom"
```

Used by: [defenseunicorns/uds-cli](https://github.com/defenseunicorns/uds-cli)

### CycloneDX with cyclonedx-gomod (OWASP Amass)

```yaml
sboms:
  - documents:
      - "{{ .Binary }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}.bom.json"
    artifacts: binary
    cmd: cyclonedx-gomod
    args: ["app", "-licenses", "-json", "-output", "$document", "../"]
    env:
      - GOARCH={{ .Arch }}
      - GOOS={{ .Os }}
```

Used by: [owasp-amass/amass](https://github.com/OWASP/Amass)

### Full Supply Chain (signing + Docker + SBOM)

```yaml
sboms:
  - artifacts: archive
  - id: source
    artifacts: source

signs:
  - cmd: cosign
    signature: "${artifact}.sigstore.json"
    args:
      - sign-blob
      - "--bundle=${signature}"
      - "${artifact}"
      - "--yes"
    artifacts: checksum
    output: true

dockers_v2:
  - images:
      - "ghcr.io/org/project"
    tags:
      - "{{ .Tag }}"
      - latest
    # sbom: true is the default

docker_signs:
  - cmd: cosign
    output: true
    args:
      - "sign"
      - "${artifact}"
      - "--yes"
```

Used by: [goreleaser/example-supply-chain](https://github.com/goreleaser/goreleaser-example-supply-chain)

---

## 10. Limitations and Gotchas

### Critical Limitations

1. **Container images cannot be cataloged by the `sboms:` block.** The official docs explicitly state this. Container SBOMs come only from `dockers_v2:`'s built-in `sbom` field or from manual hooks.

2. **Syft must be pre-installed.** GoReleaser calls syft as an external binary. It is NOT bundled. In GitHub Actions, use `anchore/sbom-action/download-syft@v0` to install it. In other CI systems, install it manually.

3. **SBOMs do not flow to package managers.** Homebrew, Scoop, Nix, APT, RPM installations do not include SBOMs. They exist only as GitHub Release assets (or OCI artifacts for Docker images).

### Common Pitfalls

4. **Forgetting to set unique IDs.** Multiple `sboms:` entries require unique `id` values. The first entry defaults to `'default'`; subsequent ones need explicit IDs. GoReleaser will error if IDs collide.

5. **Default `disable: true` confusion.** The `disable` field defaults to `true` in the schema documentation, but this is misleading -- when you add an `sboms:` entry, it is enabled. The `disable` field is for conditional disabling via templates (e.g., `disable: "{{ if .IsSnapshot }}true{{ end }}"`).

6. **Archive vs Binary artifact choice.** `artifacts: archive` catalogs the tar.gz/zip file (seeing what's inside the archive). `artifacts: binary` catalogs the raw binary. For Go binaries, both produce similar results since syft can analyze Go binaries directly. The archive option is preferred because it matches what users actually download.

7. **cyclonedx-gomod requires GOOS/GOARCH env vars.** When using cyclonedx-gomod for per-binary SBOMs, you must pass the target OS and architecture as environment variables (see OWASP Amass example), or the tool will catalog for the host platform only.

8. **SBOM format in args vs filename.** The default produces SPDX-JSON but the default filename is `.sbom.json`. If you switch to CycloneDX format via args, consider updating the document naming template to reflect this (e.g., `.cdx.json` or `.bom.json`).

9. **Ko integration.** When using Ko for Docker images (instead of Dockerfile-based builds), Ko handles its own SBOM generation and uploads to the OCI registry. GoReleaser's `sboms:` block is not involved for Ko-built images.

### Edge Cases

10. **Snapshot/development builds.** SBOMs are generated for `--snapshot` builds too. Use the `disable` field with a template to skip SBOM generation for non-release builds if desired.

11. **Cross-compilation and SBOM accuracy.** Syft catalogs the archive/binary it's given, not the build environment. For Go binaries, this works well because `go version -m` metadata is embedded. For non-Go content in archives (config files, scripts), syft will catalog what it finds.

---

## 11. Recommended Configuration for qsdev

### Minimal Viable SBOM Setup

```yaml
version: 2

sboms:
  - artifacts: archive

signs:
  - cmd: cosign
    signature: "${artifact}.sigstore.json"
    args:
      - sign-blob
      - "--bundle=${signature}"
      - "${artifact}"
      - "--yes"
    artifacts: checksum
```

### Full Supply Chain Setup (if Docker images are needed)

```yaml
version: 2

sboms:
  - artifacts: archive
  - id: source
    artifacts: source

signs:
  - cmd: cosign
    signature: "${artifact}.sigstore.json"
    args:
      - sign-blob
      - "--bundle=${signature}"
      - "${artifact}"
      - "--yes"
    artifacts: checksum

dockers_v2:
  - images:
      - "ghcr.io/org/qsdev"
    tags:
      - "{{ .Tag }}"
      - latest
    # sbom: true (default)

docker_signs:
  - cmd: cosign
    args:
      - "sign"
      - "${artifact}@${digest}"
      - "--yes"
```

### GitHub Actions Workflow Requirements

```yaml
- uses: anchore/sbom-action/download-syft@v0  # Install syft
- uses: sigstore/cosign-installer@v3            # Install cosign
- uses: goreleaser/goreleaser-action@v6
  with:
    args: release --clean
  env:
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

The `GITHUB_TOKEN` automatically provides OIDC identity for cosign keyless signing in GitHub Actions.

---

## Sources

All source material saved to `docs/`:

| File | Source |
|------|--------|
| `goreleaser-sbom-configuration-docs.md` | [GoReleaser SBOM docs](https://goreleaser.com/customization/sbom/) |
| `goreleaser-supply-chain-security-blog.md` | [GoReleaser supply chain blog](https://goreleaser.com/blog/supply-chain-security/) |
| `goreleaser-example-supply-chain-repo.md` | [Example supply chain repo](https://github.com/goreleaser/goreleaser-example-supply-chain) |
| `goreleaser-example-supply-chain-yaml.md` | [Example .goreleaser.yaml](https://raw.githubusercontent.com/goreleaser/goreleaser-example-supply-chain/main/.goreleaser.yaml) |
| `goreleaser-sbom-pr-2648.md` | [PR #2648 adding SBOM support](https://github.com/goreleaser/goreleaser/pull/2648) |
| `goreleaser-own-goreleaser-yaml.md` | [GoReleaser's own config](https://github.com/goreleaser/goreleaser/blob/main/.goreleaser.yaml) |
| `goreleaser-full-goreleaser-yaml.md` | [Full .goreleaser.yaml raw content](https://raw.githubusercontent.com/goreleaser/goreleaser/main/.goreleaser.yaml) |
| `goreleaser-docker-v2-docs.md` | [Docker v2 documentation](https://goreleaser.com/customization/package/dockers_v2/) |
| `goreleaser-signing-docs.md` | [Signing documentation](https://goreleaser.com/customization/sign/sign/) |
| `goreleaser-pro-features-list.md` | [Pro features list](https://goreleaser.com/pro/) |
| `goreleaser-sbom-proposal-issue-2597.md` | [SBOM proposal issue #2597](https://github.com/goreleaser/goreleaser/issues/2597) |
| `goreleaser-cyclonedx-sbom-issue-2808.md` | [CycloneDX format request #2808](https://github.com/goreleaser/goreleaser/issues/2808) |
| `syft-goreleaser-yaml.md` | [Syft's own .goreleaser.yaml](https://raw.githubusercontent.com/anchore/syft/main/.goreleaser.yaml) |
| `k8sgpt-goreleaser-yaml.md` | [k8sgpt config](https://raw.githubusercontent.com/k8sgpt-ai/k8sgpt/main/.goreleaser.yaml) |
| `uds-cli-goreleaser-yaml.md` | [UDS CLI config](https://raw.githubusercontent.com/defenseunicorns/uds-cli/main/.goreleaser.yaml) |
| `owasp-amass-goreleaser-yaml.md` | [OWASP Amass config](https://raw.githubusercontent.com/OWASP/Amass/master/.goreleaser.yaml) |
| `containerinfra-goreleaser-gitlab-sbom-blog.md` | [ContainerInfra GitLab blog](https://containerinfra.nl/blog/2025/01/26/using-goreleaser-with-gitlab-multi-arch-builds-cosign-and-sbom-generation/) |
