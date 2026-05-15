# GitHub SBOM & Supply Chain Security Integration Research

## Executive Summary

GitHub provides a layered supply chain security ecosystem for Go projects spanning dependency tracking, SBOM generation/export, artifact attestation, and SLSA provenance. For qsdev, the recommended approach combines GoReleaser's native SBOM generation (via Syft) with GitHub's `actions/attest@v4` for build provenance and SBOM attestation, automatic dependency graph population (now Dependabot-powered for Go as of Dec 2025), and the SBOM export REST API for compliance. This avoids the complexity of `slsa-github-generator` while still achieving SLSA Build Level 2 with a path to Level 3 via reusable workflows.

---

## 1. GitHub Dependency Graph for Go

### How It Works

As of December 2025, GitHub uses **Dependabot-based dynamic dependency resolution** for Go projects, replacing the previous static `go.mod` parsing approach.

**Mechanism**: When a commit modifies `go.mod`, GitHub triggers a specialized Dependabot job that:
1. Performs full Go module resolution (equivalent to `go mod graph`)
2. Constructs a dependency snapshot including transitive dependencies
3. Submits the snapshot via the Dependency Submission API

**Why dynamic resolution matters**: Go resolves dependency versions dynamically -- the same `go.mod` can produce different resolved dependency trees depending on the Go version, build tags, and target platform. Static parsing of `go.mod` only captures direct `require` directives and misses the full transitive closure.

### What It Captures

- Direct dependencies from `go.mod`
- Transitive dependencies (the full resolved tree)
- Version information for each dependency
- Package URL (PURL) identifiers for vulnerability matching

### Limitations

- Only triggered on changes to `go.mod` (not on every push)
- Build-tag-specific dependencies may not all be captured (resolution uses default build context)
- Replace directives and local module replacements may not be fully represented
- Does not capture build-time tool dependencies (e.g., `go generate` tools)

### Key Advantage for qsdev

The Dependabot-based approach **does not consume GitHub Actions minutes** and supports organization-level configurations for private package registries. This means qsdev gets accurate dependency tracking for free, with no workflow configuration needed.

**Source**: `docs/github-dependabot-go-dependency-graphs.md`

---

## 2. Dependency Submission API

### Purpose

The Dependency Submission API allows programmatic submission of dependency data beyond what GitHub's automatic detection captures. This is essential for:
- Build-time resolved dependencies (different from static `go.mod` parsing)
- Dependencies from non-standard build systems
- Richer SBOM data than auto-detection provides

### API Endpoint

```
POST /repos/{owner}/{repo}/dependency-graph/snapshots
```

### Go-Specific Action: `actions/go-dependency-submission@v2`

GitHub provides a first-party action specifically for Go:

```yaml
name: Go Dependency Submission
on:
  push:
    branches: [main]

permissions:
  contents: write

jobs:
  go-action-detection:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - uses: actions/go-dependency-submission@v2
        with:
          go-mod-path: go.mod
          go-build-target: ./cmd/qsdev/main.go
```

**Key inputs:**
- `go-mod-path` (required): Path to `go.mod`
- `go-build-target` (optional): Path to `main()` file. When omitted, collects all build targets' dependencies.

**Private module handling**: Requires `GOPRIVATE` env var and git credential configuration for private repos.

### SBOM-Based Submission

You can also submit pre-generated SBOMs to the dependency graph:

```yaml
- uses: anchore/sbom-action@v0
  with:
    dependency-snapshot: true  # Submits to dependency graph API
```

Or use the SPDX Dependency Submission Action:

```yaml
- uses: advanced-security/spdx-dependency-submission-action@v0.2.0
  with:
    filePath: "_manifest/spdx_2.2/"
```

### Recommendation for qsdev

Since the Dependabot-based auto-detection (Dec 2025) now handles Go dynamically, **qsdev likely does not need a manual Dependency Submission workflow** unless:
1. Build targets use unusual build tags that change the dependency tree
2. Private modules require special authentication
3. You want to submit richer SBOM data (e.g., with license info) beyond what auto-detection provides

**Sources**: `docs/github-dependency-submission-api-docs.md`, `docs/go-dependency-submission-action-readme.md`

---

## 3. GitHub SBOM Export (Dependabot SBOM)

### REST API Endpoints

GitHub provides three SBOM-related REST API endpoints:

#### Export SBOM (immediate)
```bash
GET /repos/{owner}/{repo}/dependency-graph/sbom
```
Returns the dependency graph as an SPDX 2.3 JSON document. Requires read access.

#### Trigger SBOM Generation (async)
```bash
GET /repos/{owner}/{repo}/dependency-graph/sbom/generate-report
```
Returns a `sbom_url` for later retrieval. Reports retained up to one week.

#### Fetch Generated Report
```bash
GET /repos/{owner}/{repo}/dependency-graph/sbom/fetch-report/{sbom_uuid}
```
Returns 302 redirect to temporary download URL.

### CLI Access

The `gh sbom` CLI extension provides command-line access:

```bash
# Install
gh ext install advanced-security/gh-sbom

# Export SPDX (fast, server-side generation)
gh sbom -l

# Export CycloneDX (slower, client-side assembly)
gh sbom -l -c
```

**SPDX output** uses the Dependency Graph SBOM API (fast, works for large repos, includes license info). **CycloneDX output** uses the GraphQL API and ClearlyDefined for licenses (slower, may fail on large repos).

### UI Export

Navigate to: Repository > Insights > Dependency graph > Export SBOM. Generates SPDX 2.3 format.

### What's Included

- All dependencies from the dependency graph (direct + transitive)
- Version information
- Package identifiers (PURLs)
- License information (concluded and declared)
- Copyright text
- External references
- Relationship types (DEPENDS_ON, etc.)

### What's NOT Included

- Dependents (other projects depending on yours)
- Build-time tool dependencies not in `go.mod`
- Embedded/vendored code not tracked by the module system

### Relationship to Dependency Graph

The exported SBOM is a **snapshot of the dependency graph** at export time. It reflects whatever data has been submitted via auto-detection or the Dependency Submission API. This means:
- If the dependency graph is incomplete, the SBOM will be too
- Submitting richer data via the API improves the exported SBOM quality
- The SBOM export is only as good as the data in the dependency graph

**Sources**: `docs/github-sbom-rest-api-endpoints.md`, `docs/github-sbom-export-docs.md`

---

## 4. GitHub Artifact Attestations

### Overview

GitHub Artifact Attestations (GA since June 2024) create **cryptographically signed claims** about build artifacts. They establish provenance (where and how software was built) and can optionally include SBOMs.

### How They Work

1. **During build**: The `actions/attest@v4` action generates an attestation in the in-toto format
2. **Signing**: A short-lived Sigstore certificate is obtained via OIDC token exchange
3. **Storage**: The signed attestation (Sigstore bundle) is uploaded to GitHub's attestations API
4. **Transparency**: For public repos, attestations are also written to Sigstore's public transparency log (Rekor)

### Attestation Types

| Type | Trigger | Use Case |
|------|---------|----------|
| **Build Provenance** | Default (no sbom-path) | Proves where/how the binary was built |
| **SBOM** | `sbom-path` provided | Binds an SBOM to a specific artifact |
| **Custom** | `predicate-type`/`predicate` | User-defined attestation predicates |

### Sigstore Integration Details

**Public repositories**: Use the Sigstore Public Good Instance. Attestations are:
- Signed with a keyless certificate (OIDC-bound)
- Stored in GitHub's attestations API
- Written to the public Rekor transparency log (immutable, publicly auditable)

**Private repositories**: Use GitHub's private Sigstore instance. Attestations are:
- Signed with the same keyless mechanism
- Stored in GitHub's attestations API
- NOT written to any public transparency log
- Only federate with GitHub Actions (not third-party verifiers)

### SLSA Levels Achieved

- **`actions/attest@v4` in a normal workflow**: SLSA Build Level 2 (provenance generated by a hosted build service, signed)
- **`actions/attest@v4` in a reusable workflow**: SLSA Build Level 3 (build runs in an isolated, shared workflow that callers cannot modify)
- **`slsa-github-generator` reusable workflows**: SLSA Build Level 3 (purpose-built for L3 isolation)

### Required Permissions

```yaml
permissions:
  id-token: write      # For Sigstore OIDC certificate
  contents: read       # For checking out code
  attestations: write  # For uploading attestations
```

### Availability

- Public repos: All GitHub plans
- Private repos: **GitHub Enterprise Cloud only**

### Verification

```bash
# Verify build provenance
gh attestation verify ./qsdev-linux-amd64 -R owner/qsdev

# Verify SBOM attestation specifically
gh attestation verify ./qsdev-linux-amd64 \
  -R owner/qsdev \
  --predicate-type https://spdx.dev/Document/v2.3

# View SBOM content from attestation
gh attestation verify ./qsdev-linux-amd64 \
  -R owner/qsdev \
  --predicate-type https://spdx.dev/Document/v2.3 \
  --format json \
  --jq '.[].verificationResult.statement.predicate'
```

### Offline Verification

For air-gapped environments:
```bash
# On connected machine:
gh attestation download ./qsdev-linux-amd64 -R owner/qsdev
gh attestation trusted-root > trusted_root.jsonl

# On air-gapped machine:
gh attestation verify ./qsdev-linux-amd64 \
  -R owner/qsdev \
  --bundle sha256:DIGEST.jsonl \
  --custom-trusted-root trusted_root.jsonl
```

Key rotation: Sigstore rotates keys several times per year. Trusted root files have no built-in expiration but won't detect post-creation revocations.

**Sources**: `docs/github-artifact-attestations-docs.md`, `docs/github-actions-attest-action.md`, `docs/github-using-artifact-attestations-guide.md`, `docs/github-offline-attestation-verification.md`

---

## 5. SLSA Provenance via GitHub Actions

### Two Paths to SLSA Provenance

#### Path A: `actions/attest@v4` (Recommended for qsdev)

The simpler approach. Generates SLSA v1.0 build provenance as an in-toto attestation.

**Achieves**: SLSA Build Level 2 (normal workflow) or Level 3 (reusable workflow)

```yaml
- uses: actions/attest@v4
  with:
    subject-path: './dist/qsdev-linux-amd64'
```

**Advantages**:
- Single action, minimal configuration
- Integrates directly with GoReleaser via checksums file
- Uses GitHub's native attestation storage
- Verified with `gh attestation verify`
- Actively maintained by GitHub

#### Path B: `slsa-framework/slsa-github-generator` (Maximum SLSA rigor)

A more complex reusable workflow approach with stronger isolation guarantees.

**Achieves**: SLSA Build Level 3 (by design -- the reusable workflow IS the isolated builder)

The Go builder (`builder_go_slsa3.yml`) compiles your Go binary in an isolated workflow:

```yaml
jobs:
  build:
    permissions:
      id-token: write
      contents: read
      actions: read
    uses: slsa-framework/slsa-github-generator/.github/workflows/builder_go_slsa3.yml@v1.10.0
    with:
      go-version-file: go.mod
      config-file: .slsa-goreleaser.yml
```

Requires a `.slsa-goreleaser.yml` config:
```yaml
version: 1
env:
  - CGO_ENABLED=0
flags:
  - -trimpath
goos: linux
goarch: amd64
binary: qsdev-{{ .Os }}-{{ .Arch }}
ldflags:
  - "-s -w -X main.version={{ .Tag }}"
```

**Advantages**:
- Strongest SLSA L3 guarantees (provenance content cannot be affected by caller)
- Verification via `slsa-verifier` (independent of GitHub CLI)
- Designed specifically for supply chain security certification

**Disadvantages**:
- Cannot use GoReleaser for the build step (it IS the builder)
- Requires separate config file
- Multi-platform builds need matrix strategy with separate configs per platform
- `actions/download-artifact@v3` compatibility issues with v4
- Less flexible than GoReleaser for cross-compilation, archives, Docker images
- Must be referenced by exact version tag (`@v1.10.0`)

### SLSA Level Comparison

| Feature | `actions/attest` | `slsa-github-generator` |
|---------|-----------------|------------------------|
| SLSA Level (normal workflow) | Level 2 | N/A (always reusable) |
| SLSA Level (reusable workflow) | Level 3 | Level 3 |
| GoReleaser compatible | Yes (post-build step) | No (replaces build) |
| Verification tool | `gh attestation verify` | `slsa-verifier` |
| Provenance isolation | Caller can modify workflow | Caller cannot modify builder |
| Setup complexity | Low (one action) | Medium (reusable workflow + config) |
| Ecosystem maturity | GitHub-native, actively developed | SLSA-framework maintained, stable |

### Recommendation for qsdev

**Use `actions/attest@v4`** as the primary attestation mechanism:
- It integrates cleanly with GoReleaser
- Achieves SLSA Level 2 immediately, Level 3 with a reusable workflow wrapper
- The verification story (`gh attestation verify`) is simpler for consumers
- GitHub is clearly converging on this as the canonical path (both `attest-build-provenance` and `attest-sbom` are now wrappers around `actions/attest`)

If certification requirements demand formally audited SLSA Level 3, wrap the GoReleaser release workflow in a reusable workflow or consider running `slsa-github-generator` alongside GoReleaser for just the binary builds.

**Sources**: `docs/slsa-github-generator-readme.md`, `docs/slsa-go-builder-readme.md`, `docs/github-blog-slsa3-go-compliance.md`

---

## 6. GitHub Actions Workflow Patterns for qsdev

### Complete GoReleaser + Attestation Workflow

Based on the goreleaser/example-supply-chain reference implementation, adapted for qsdev:

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write       # Create releases, upload assets
  id-token: write       # Sigstore OIDC certificates
  packages: write       # Push Docker images to GHCR
  attestations: write   # Upload attestations

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      # Install Syft for SBOM generation
      - uses: anchore/sbom-action/download-syft@v0

      # Run GoReleaser (builds, archives, SBOMs, checksums, Docker, signing)
      - uses: goreleaser/goreleaser-action@v7
        with:
          version: '~> v2'
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

      # Attest build provenance for all release artifacts (via checksums)
      - uses: actions/attest@v4
        with:
          subject-checksums: ./dist/checksums.txt

      # Attest build provenance for Docker images (via digests)
      - uses: actions/attest@v4
        if: startsWith(github.ref, 'refs/tags/v')
        with:
          subject-checksums: ./dist/digests.txt
```

### GoReleaser SBOM Configuration (`.goreleaser.yaml`)

```yaml
# Generate SBOMs for archives and source tarballs
sboms:
  - artifacts: archive    # SBOM for each platform archive
  - id: source
    artifacts: source     # SBOM for source tarball

# Stable checksum filename for attestation
checksum:
  name_template: "checksums.txt"

# Docker digest filename for container attestation
docker_digest:
  name_template: "digests.txt"
```

### Adding SBOM Attestation (Optional Enhancement)

To also attest the SBOMs themselves (not just build provenance):

```yaml
      # After GoReleaser, attest each SBOM
      - name: Attest SBOMs
        run: |
          for sbom in ./dist/*.sbom.json; do
            artifact="${sbom%.sbom.json}"
            if [ -f "$artifact" ]; then
              gh attestation attest "$artifact" \
                --sbom "$sbom" \
                -R ${{ github.repository }}
            fi
          done
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

Or using `actions/attest@v4`:

```yaml
      - uses: actions/attest@v4
        with:
          subject-path: './dist/qsdev_linux_amd64.tar.gz'
          sbom-path: './dist/qsdev_linux_amd64.tar.gz.sbom.json'
```

### Dependency Graph Submission Workflow (Separate)

```yaml
name: Dependency Submission

on:
  push:
    branches: [main]
    paths: ['go.mod', 'go.sum']

permissions:
  contents: write

jobs:
  submit-deps:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - uses: actions/go-dependency-submission@v2
        with:
          go-mod-path: go.mod
          go-build-target: ./cmd/qsdev/main.go
```

**Note**: This may be redundant with the Dec 2025 Dependabot auto-submission for Go, but provides a guarantee that the dependency graph is populated immediately after go.mod changes.

### Key Actions Summary

| Action | Purpose | Status |
|--------|---------|--------|
| `actions/attest@v4` | Build provenance + SBOM attestation | **Current canonical action** |
| `actions/attest-build-provenance@v4` | Build provenance (wrapper around attest) | Deprecated, use attest |
| `actions/attest-sbom@v4` | SBOM attestation (wrapper around attest) | Deprecated, use attest |
| `anchore/sbom-action@v0` | Generate SBOM via Syft | Active, mature |
| `actions/go-dependency-submission@v2` | Submit Go deps to dependency graph | Active |
| `slsa-framework/slsa-github-generator` | SLSA L3 reusable workflow builders | Active, stable for Go |

**Sources**: `docs/goreleaser-example-supply-chain-config.md`, `docs/goreleaser-attestations-docs.md`, `docs/goreleaser-sbom-customization-docs.md`, `docs/anchore-sbom-action-readme.md`

---

## 7. How Uploaded SBOMs Trigger Vulnerability Alerts

### The Alert Chain

1. **Dependency data enters the graph** via:
   - Automatic parsing (Dependabot for Go, Dec 2025+)
   - Dependency Submission API
   - Manual SBOM upload actions

2. **GitHub cross-references** the dependency graph against the GitHub Advisory Database:
   - Maintained by GitHub Security Lab + community
   - Includes CVEs, GitHub Security Advisories (GHSAs)
   - Ecosystem-specific identifiers (Go module paths)

3. **Dependabot alerts fire** when:
   - A new advisory is published matching a dependency
   - The dependency graph changes (new dependencies added)
   - An existing advisory is updated with new affected versions

4. **Alert contents include**:
   - Link to the advisory
   - Affected dependency and version range
   - Fixed version (if available)
   - CVSS score and severity
   - Remediation suggestions

### Implications for qsdev's SBOM Strategy

Submitting accurate dependency data (either via auto-detection or the Dependency Submission API) directly improves Dependabot alert accuracy. If the dependency graph is incomplete, vulnerabilities in missing dependencies will go undetected.

The **GitHub-exported SBOM** (via API or UI) reflects whatever is in the dependency graph. Downstream consumers who import this SBOM into their own vulnerability management tools (e.g., Dependency-Track, Grype) will see the same dependency set.

**Sources**: `docs/github-artifact-attestations-docs.md`

---

## 8. Attestation Storage and Consumer Verification

### Where Attestations Live

**GitHub API**: All attestations (provenance, SBOM, custom) are stored in GitHub's attestations API, associated with the source repository. They are accessible via:

```bash
# List attestations for a specific artifact digest
GET /repos/{owner}/{repo}/attestations/{subject_digest}

# Also available at org and user scope
GET /orgs/{org}/attestations/{subject_digest}
GET /users/{username}/attestations/{subject_digest}
```

**Sigstore Transparency Log (Rekor)**: For public repos only. Provides an immutable, publicly auditable record. The log entry includes the signing certificate, signature, and artifact digest.

**GitHub Release Assets**: SBOMs generated by GoReleaser/Syft are uploaded as release assets (e.g., `qsdev_linux_amd64.tar.gz.sbom.json`). These are the raw SBOM files, separate from attestations.

### How Consumers Verify

**Simple verification** (most common):
```bash
# Download the binary
# Verify it was built by the expected repo
gh attestation verify ./qsdev-linux-amd64 --owner quantumserendipitysoftware
```

**Specific predicate verification**:
```bash
# Verify the SBOM attestation exists
gh attestation verify ./qsdev-linux-amd64 \
  -R quantumserendipitysoftware/qsdev \
  --predicate-type https://spdx.dev/Document/v2.3

# Extract and inspect the SBOM content
gh attestation verify ./qsdev-linux-amd64 \
  -R quantumserendipitysoftware/qsdev \
  --predicate-type https://spdx.dev/Document/v2.3 \
  --format json \
  --jq '.[].verificationResult.statement.predicate'
```

**Programmatic verification** (via REST API):
```bash
# Compute artifact digest
DIGEST="sha256:$(shasum -a 256 ./qsdev-linux-amd64 | cut -d' ' -f1)"

# Fetch attestations
curl -L \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer $TOKEN" \
  "https://api.github.com/repos/owner/qsdev/attestations/$DIGEST"
```

**Cosign verification** (for Sigstore bundles):
```bash
cosign verify-blob \
  --bundle checksums.txt.sigstore.json \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --certificate-identity-regexp '^https://github.com/owner/qsdev/' \
  checksums.txt
```

**Sources**: `docs/github-attestation-rest-api.md`, `docs/github-offline-attestation-verification.md`

---

## 9. Concrete Reference Implementations

### goreleaser/example-supply-chain (Official)
The canonical example combining GoReleaser + Cosign + Syft + GitHub Attestations.
- **Config**: `sboms:` section generates SBOMs for archives and source tarballs
- **Signing**: Cosign keyless signing of checksums file
- **Attestation**: `actions/attest@v4` with `subject-checksums`
- **Docker**: Image signing with Cosign, GHCR publishing
- **Repo**: https://github.com/goreleaser/goreleaser-example-supply-chain

### goreleaser/example-slsa-provenance
Example using `slsa-github-generator` Generic Generator with GoReleaser.
- **Repo**: https://github.com/goreleaser/goreleaser-example-slsa-provenance

### mchmarny/s3cme
Template Go app with full pipeline: test/lint/build/vuln check + image build/release with ko SBOM, cosign attestation, and SLSA build provenance.
- **Repo**: https://github.com/mchmarny/s3cme

---

## 10. Recommendations for qsdev

### Tier 1: Essential (Implement First)

1. **GoReleaser `sboms:` section** generating SPDX SBOMs for archives and source via Syft
2. **`actions/attest@v4`** for build provenance attestation using `subject-checksums: ./dist/checksums.txt`
3. **Stable checksum/digest filenames** in `.goreleaser.yaml` for attestation compatibility

### Tier 2: Valuable (Implement Second)

4. **SBOM attestation** via `actions/attest@v4` with `sbom-path` binding SBOMs to specific artifacts
5. **Cosign signing** of checksums file (keyless, via GoReleaser `signs:` section)
6. **`actions/go-dependency-submission@v2`** workflow if Dependabot auto-detection proves insufficient

### Tier 3: Advanced (Implement If Needed)

7. **Reusable workflow wrapper** around the release workflow for SLSA Level 3
8. **Docker image attestation** via `subject-checksums: ./dist/digests.txt` when container distribution is added
9. **Offline verification documentation** for consumers in air-gapped environments

### Anti-Patterns to Avoid

- **Don't use both `slsa-github-generator` Go builder AND GoReleaser** -- they are competing build systems. Pick one.
- **Don't use `attest-build-provenance` or `attest-sbom`** -- these are deprecated wrappers. Use `actions/attest@v4` directly.
- **Don't skip the dependency graph** -- even with GoReleaser SBOMs as release assets, the dependency graph drives Dependabot alerts.
- **Don't submit SBOMs to the dependency graph AND use auto-detection** without understanding that one may overwrite the other.

---

## Open Questions

1. **GoReleaser + `actions/attest` checksum attestation**: Does the `subject-checksums` input correctly resolve all artifacts listed in checksums.txt, including SBOMs? Need to verify with a test release.
2. **Dependabot auto-submission vs `go-dependency-submission`**: How do they interact when both are active? Does one take precedence?
3. **Private module handling**: If qsdev depends on private modules, both the Dependabot auto-submission and `go-dependency-submission` will need authentication configuration.
4. **SBOM format choice**: GoReleaser defaults to SPDX via Syft. GitHub's attestation system accepts both SPDX and CycloneDX (JSON). The dependency graph export only outputs SPDX. Standardizing on SPDX simplifies the story.
