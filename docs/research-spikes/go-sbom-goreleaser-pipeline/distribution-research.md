# SBOM Distribution Mechanisms for Go CLI Binaries

## Executive Summary

SBOM distribution for a Go CLI tool like qsdev must serve multiple channels simultaneously: GitHub Releases (primary), Homebrew taps, `go install`, and potentially Nix. No single distribution mechanism covers all channels. The practical approach is a layered strategy: (1) GitHub Release assets as the canonical SBOM source using OpenSSF naming conventions, (2) GitHub Attestations for cryptographic binding, (3) embedded build metadata via Go's native `debug.BuildInfo` for `go install` users, and (4) channel-specific accommodations for Homebrew and Nix. The Transparency Exchange API (TEA) is an emerging standard for automated SBOM discovery but is not yet production-ready.

---

## 1. GitHub Release Assets

### The Primary Distribution Channel

For CLI tools distributed via GitHub Releases, attaching SBOMs as release assets alongside binaries is the most widely adopted and immediately practical approach. GoReleaser automates this natively via its `sboms:` configuration, using Syft to generate SBOMs that are automatically uploaded as release assets.

### Naming Conventions

The **OpenSSF SBOM-Everywhere** working group has published authoritative naming guidance ([source](docs/openssf-sbom-naming-conventions.md)):

- **CycloneDX JSON**: `artifact-name.cdx.json` (e.g., `qsdev_1.2.0_linux_amd64.tar.gz.cdx.json`)
- **SPDX JSON**: `artifact-name.spdx.json` (e.g., `qsdev_1.2.0_linux_amd64.tar.gz.spdx.json`)
- **Mandatory**: JSON format must always be provided
- **Structure**: Flat file list, no directories (matches GitHub/GitLab Release constraints)

The key principle is that SBOM filenames are the artifact filename with an SBOM-format extension appended. This creates a predictable, discoverable mapping between artifacts and their SBOMs.

### GoReleaser Default Naming

GoReleaser's default SBOM naming uses `.sbom.json` as the extension ([source](docs/goreleaser-sbom-configuration.md)):

- **For archives**: `{{ .ArtifactName }}.sbom.json`
- **For binaries**: `{{ .Binary }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}.sbom.json`

**Recommendation**: Override the GoReleaser default to use OpenSSF-compliant extensions. For CycloneDX: `{{ .ArtifactName }}.cdx.json`. For SPDX: `{{ .ArtifactName }}.spdx.json`. The `.sbom.json` extension is GoReleaser-specific and not recognized by the broader ecosystem's tooling conventions.

### Practical Configuration

```yaml
sboms:
  - id: archive
    artifacts: archive
    documents:
      - "${artifact}.cdx.json"
    cmd: syft
    args: ["$artifact", "--output", "cyclonedx-json=$document"]
  - id: source
    artifacts: source
    documents:
      - "${artifact}.cdx.json"
    cmd: syft
    args: ["$artifact", "--output", "cyclonedx-json=$document"]
```

### Problems with GitHub Release SBOM Distribution

A critical analysis from SBOM Insights ([source](docs/github-releases-sbom-distribution-problems.md)) identifies GitHub Releases as "where SBOMs go to die" due to:

- **No automated discovery**: Consumers must manually search releases for SBOM files
- **No standardized API**: No convention for programmatically finding the SBOM for a given artifact
- **Scale problems**: Security teams cannot keep up with the manual download/upload cycle across repositories

The article highlights that tools like `sbommv` exist to automate transfer from GitHub Releases to platforms like Dependency-Track, but this is a workaround for a fundamentally broken discovery model.

### GitHub Attestations (Recommended Complement)

GitHub Artifact Attestations ([source](docs/github-actions-attest-sbom.md)) provide a superior complement to plain release assets:

- SBOMs are cryptographically bound to artifacts using in-toto format with Sigstore signing
- Verification via `gh attestation verify <binary> --owner <org>`
- SBOM-specific verification: `gh attestation verify <binary> --predicate-type https://cyclonedx.org/bom`
- Attestations are stored in GitHub's attestation API, providing a structured query mechanism
- Public repos use Sigstore's public-good instance (free); private repos need GitHub Enterprise Cloud
- **Format requirements**: JSON format, SPDX or CycloneDX, max 16 MB

**This is the strongest recommendation for qsdev**: generate SBOMs as release assets AND create attestations binding them to the built artifacts.

---

## 2. OCI Artifact Attachment

### How It Works

The OCI v1.1 specification introduced the Referrers API, which allows artifacts (SBOMs, signatures, attestations) to be attached to container images in registries via a `subject` field in the manifest ([source](docs/oci-reference-types-attached-artifacts.md)).

### Discovery Mechanism

```
GET /v2/<name>/referrers/<digest>?artifactType=application/vnd.cyclonedx+json
```

The Referrers API returns all artifacts linked to a given digest, filterable by artifact type. This is a structured, automatable discovery mechanism -- far superior to searching GitHub Release asset names.

### Cosign Integration

Cosign can sign SBOMs stored in OCI registries and attach SBOM metadata to container images ([source](docs/cosign-signing-other-types-sbom.md)):

```bash
# Push SBOM to registry via ORAS
oras push ghcr.io/org/qsdev-sbom:v1.2.0 sbom.cdx.json

# Sign the SBOM
cosign sign --key cosign.key ghcr.io/org/qsdev-sbom@sha256:...

# Or attach as attestation to a container image
cosign attest --type custom --predicate sbom.cdx.json $IMAGE
```

### Relevance for qsdev

**Low priority for a CLI tool.** OCI artifact attachment is designed for container image ecosystems. If qsdev is distributed as a container image (e.g., for CI pipelines), this becomes relevant. For binary distribution via GitHub Releases + Homebrew + `go install`, this adds complexity without proportional benefit. The Referrers API requires consumers to interact with OCI registries, which is not a natural workflow for CLI tool users.

**Exception**: If qsdev publishes Docker images, attaching SBOMs via ORAS/cosign to GHCR is best practice.

---

## 3. Homebrew Considerations

### Homebrew's Native SBOM Support

Homebrew 4.3.0 (May 2024) introduced SBOM support ([source](docs/homebrew-4.3-sbom-attestation.md)):

1. **Basic SPDX file inside bottles**: `brew bottle` includes a basic SPDX document
2. **Comprehensive SPDX file post-installation**: A more complete SBOM is generated after install

These SBOMs are generated by Homebrew itself, describing the bottle's contents -- they are NOT the upstream project's SBOM.

### Do Upstream SBOMs Survive Homebrew Distribution?

**No.** When a user installs via `brew install`, Homebrew downloads a pre-built bottle and installs it. The upstream project's SBOM from the GitHub Release is not downloaded, embedded, or referenced. Homebrew generates its own SBOM for the bottle.

### Bottle Attestation

Homebrew has introduced build provenance attestations (SLSA Build L2-compatible):
- Each bottle has a cryptographically verifiable statement linking it to the specific CI workflow that built it
- Verification is opt-in via `HOMEBREW_VERIFY_ATTESTATIONS` environment variable
- Currently requires the `gh` CLI tool

### What Can Be Done in a Tap Formula?

For a custom tap formula:
- **Cannot embed upstream SBOMs** in the bottle itself (bottles are pre-built and signed by Homebrew CI)
- **Could add an `sbom` resource** pointing to the GitHub Release SBOM URL, but this is non-standard
- **Best approach**: Document in the tap's README that SBOMs are available as GitHub Release assets and via `gh attestation verify`

### Practical Recommendation

Accept that Homebrew is a separate distribution channel with its own SBOM story. The upstream SBOM's value is at the GitHub Release level. Users who need the SBOM for compliance purposes should obtain it from the GitHub Release, not through Homebrew.

---

## 4. Nix Derivation Considerations

### The Nix SBOM Landscape

Three tools exist for generating SBOMs from Nix derivations ([source](docs/nix-state-of-the-sbom.md)):

- **sbomnix**: Works at the `.drv` level, best for runtime dependency discovery, enriches metadata via nixpkgs attributes ([source](docs/sbomnix-nix-sbom-generation.md))
- **bombon**: Works at the `.nix` level, generates CycloneDX 1.5 SBOMs, supports `passthru` attributes for vendored dependency SBOMs ([source](docs/bombon-nix-cyclonedx-sbom.md))
- **genealogos**: Works at the `.nix` level using nixtract

### Passthru Attributes

Bombon uses `passthru` attributes to attach SBOM data to Nix packages. The `bombonVendoredSbom` passthru attribute allows a package to provide SBOM data for vendored dependencies (notably Rust and Go packages where dependencies are vendored during build):

```nix
bombon.lib.${system}.buildBom pkgs.qsdev {
  includeBuildtimeDependencies = true;
}
```

### Key Challenges for Go in Nix

- Go packages in nixpkgs use `buildGoModule` which vendors dependencies via `go mod vendor` -- the individual Go modules don't appear as separate Nix derivations
- sbomnix can identify the Go binary and its Nix-level dependencies but not the Go module dependency tree within the vendored blob
- bombon's `passthruVendoredSbom` mechanism addresses this for Rust but does not yet have a Go equivalent

### Practical Recommendation

For qsdev in Nix:

1. **Include the upstream SBOM in the Nix package** via `postInstall`:
   ```nix
   postInstall = ''
     install -Dm644 ${sbom} $out/share/sbom/qsdev.cdx.json
   '';
   ```
   Where `sbom` is either a pre-generated file fetched from the GitHub Release or generated during the Nix build.

2. **Use sbomnix or bombon** to generate a Nix-level SBOM that captures the full closure (build tools, libraries, etc.), complementing the Go-specific SBOM.

3. **Add a passthru attribute** pointing to the SBOM:
   ```nix
   passthru.sbom = ./sbom.cdx.json;
   ```

This is not yet standardized across nixpkgs. The Nix SBOM ecosystem is immature compared to container-focused tooling.

---

## 5. Embedded SBOMs

### Go's `//go:embed` Directive

Go's embed package allows compiling arbitrary files into the binary at build time ([source](docs/go-sbom-generation-comprehensive-guide.md)):

```go
import _ "embed"

//go:embed sbom.cdx.json
var sbomData []byte
```

This makes the SBOM accessible at runtime without external file dependencies.

### Implementation Pattern

1. Generate SBOM during CI (pre-build or as a separate step)
2. Place the SBOM file where the Go embed directive expects it
3. Expose via a CLI subcommand: `qsdev sbom` or `qsdev version --sbom`
4. Optionally expose via an HTTP endpoint if qsdev runs a server

### Tradeoffs

**Advantages:**
- SBOM is always available, regardless of distribution channel
- Works for `go install` users who never see GitHub Release assets
- Self-contained -- no external file dependencies
- Enables runtime self-inspection and vulnerability checking

**Disadvantages:**
- Increases binary size (a typical Go SBOM is 50-200 KB in CycloneDX JSON -- minimal impact)
- Circular dependency: the SBOM must be generated before the build, so it describes the source dependencies, not the final binary. This is a source SBOM, not a build SBOM
- The embedded SBOM cannot include the binary's own hash (chicken-and-egg problem)
- Updating the SBOM requires rebuilding the binary

### Two-Phase Build Pattern

The circular dependency can be mitigated with a two-phase approach:

1. **Phase 1**: Generate source SBOM from `go.mod`/`go.sum` (pre-build)
2. **Phase 2**: Build binary with embedded source SBOM
3. **Phase 3**: Generate build SBOM from the compiled binary (post-build, for release assets)

This gives you an embedded source SBOM in the binary AND a build SBOM as a release asset.

### Practical Recommendation

**Embed a source SBOM** in qsdev. The binary size cost is negligible, and it ensures every distribution channel (including `go install`) has SBOM access. Complement with a build SBOM as a GitHub Release asset for consumers who need the most accurate post-compilation picture.

---

## 6. `go install` Path

### What Users Get with `go version -m`

When a user installs via `go install github.com/org/qsdev@latest`, Go embeds build metadata in the binary ([source](docs/go-binary-build-information.md)):

```
$ go version -m $(which qsdev)
qsdev: go1.22.0
    path    github.com/org/qsdev
    mod     github.com/org/qsdev  v1.2.0  h1:abc123...
    dep     github.com/spf13/cobra  v1.8.0  h1:def456...
    dep     github.com/spf13/viper  v1.18.0 h1:ghi789...
    build   -buildmode=exe
    build   CGO_ENABLED=0
    build   GOARCH=amd64
    build   GOOS=linux
```

This provides:
- Go compiler version
- Module path and version (with hash)
- All dependency modules with versions and hashes
- Build settings (GOARCH, GOOS, CGO_ENABLED, etc.)

### Is This an SBOM?

`go version -m` output is **SBOM-adjacent but not a formal SBOM**. It lacks:
- Standardized format (not SPDX or CycloneDX)
- License information
- Supplier metadata
- Vulnerability correlation identifiers (PURLs, CPEs)

However, tools like Syft can analyze a Go binary and generate a proper SBOM from this embedded metadata:

```bash
syft $(which qsdev) -o cyclonedx-json > qsdev-binary.cdx.json
```

### Practical Implication

For `go install` users:
- They always have `go version -m` metadata (it's built into Go)
- Syft can convert this to a proper SBOM on demand
- An embedded SBOM (via `//go:embed`) would give them a formal SBOM without requiring Syft
- There is no mechanism to attach a separate SBOM file to a `go install` binary

**Recommendation**: The embedded SBOM approach (Section 5) is the only way to provide formal SBOM data to `go install` users. For users with Syft, `go version -m` metadata is sufficient for SBOM generation.

---

## 7. SBOM Discovery Conventions

### Current State: No Universal Standard

There is no single, universally adopted standard for SBOM discovery across all distribution channels. Discovery mechanisms are fragmented by ecosystem.

### OpenSSF Naming Convention (Best Available)

The OpenSSF SBOM-Everywhere guidance ([source](docs/openssf-sbom-naming-conventions.md)) provides the closest thing to a cross-ecosystem standard for GitHub/GitLab Releases:

- Append `.cdx.json` (CycloneDX) or `.spdx.json` (SPDX) to the artifact filename
- Always provide JSON format
- Use a flat file list (no directories)

This makes discovery predictable: given `qsdev_1.2.0_linux_amd64.tar.gz`, the SBOM is at `qsdev_1.2.0_linux_amd64.tar.gz.cdx.json`.

### Transparency Exchange API (TEA) -- Emerging Standard

TEA ([source](docs/transparency-exchange-api-tea.md)) is the most promising universal SBOM discovery mechanism:

- Uses **Transparency Exchange Identifiers (TEI)** -- URN-based identifiers resolved via DNS
- Discovery flow: TEI -> DNS resolution -> `.well-known` endpoint -> API -> SBOM artifacts
- Format-agnostic (supports both SPDX and CycloneDX)
- Being standardized by ECMA TC54 Task Group 1 (same group as CycloneDX and PURL)
- Currently in Beta 2

**Timeline**: TEA is not yet production-ready. Monitor for GA release, likely late 2026 or 2027. When ready, it would allow qsdev to publish a TEI that resolves to its SBOM automatically.

### GitHub-Specific Discovery

- `gh attestation verify` provides structured SBOM discovery for GitHub-hosted projects
- GitHub's dependency graph API can export repository-level SBOMs
- GitHub's SBOM REST API exports dependency graph as SPDX

### Package Manager Conventions

Different ecosystems are developing their own conventions:
- **Python/PyPI**: PEP 770 standardizes `.dist-info/sboms/` as the SBOM location in wheel packages
- **npm**: `npm sbom` command generates SBOMs; provenance attestations use Sigstore
- **OCI/Container**: Referrers API (`/v2/<name>/referrers/<digest>`) with artifactType filtering
- **Go**: `go version -m` provides raw metadata; no formal SBOM location convention

---

## 8. Package Manager SBOM Standards (Cross-Ecosystem Comparison)

### npm

- `npm sbom` command generates SBOM from package-lock.json
- Provenance attestations via Sigstore (published to public transparency log)
- `npm audit signatures` verifies provenance
- No convention for distributing pre-built SBOMs alongside packages

### PyPI (Python)

- **PEP 770** (2026): Standardizes `.dist-info/sboms/` directory in wheel packages
- Adoption remains very low: 1.58% of packages, 8.46% of wheels ([source](docs/pypi-sbom-adoption-pep770.md))
- 100% CycloneDX JSON format among adopters (zero SPDX)
- Most are CycloneDX 1.4/1.5 -- tooling has not caught up to 1.6/1.7

### Homebrew

- Generates its own SPDX SBOMs for bottles (since 4.3.0)
- Build provenance attestations for bottles (SLSA Build L2)
- No mechanism for upstream SBOMs to flow through

### Cargo (Rust)

- `cargo-cyclonedx` generates CycloneDX SBOMs
- Bombon integrates Rust SBOMs into Nix via `passthruVendoredSbom`

### Key Lesson for qsdev

The ecosystem is converging on CycloneDX JSON as the dominant format for package-level SBOMs. SPDX dominates at the organizational/compliance level. For a CLI tool, **CycloneDX JSON is the pragmatic choice** -- it has better tooling support for Go (cyclonedx-gomod, Syft default output), is the universal choice in PyPI, and is more compact than SPDX.

---

## Recommended Distribution Strategy for qsdev

### Tier 1: GitHub Release Assets (Must Have)

1. Generate per-archive CycloneDX JSON SBOMs via GoReleaser + Syft
2. Use OpenSSF naming: `qsdev_<version>_<os>_<arch>.tar.gz.cdx.json`
3. Generate a source SBOM: `qsdev_<version>_source.tar.gz.cdx.json`
4. Create GitHub Attestations binding SBOMs to artifacts
5. Sign checksums and SBOMs via cosign (keyless/Sigstore)

### Tier 2: Embedded SBOM (Should Have)

1. Generate source SBOM from `go.mod` pre-build
2. Embed via `//go:embed sbom.cdx.json`
3. Expose via `qsdev version --sbom` subcommand
4. Ensures `go install` users and all channels have SBOM access

### Tier 3: Channel-Specific (Nice to Have)

- **Homebrew**: Document SBOM availability in tap README; accept Homebrew's own bottle SBOMs as complementary
- **Nix**: Include upstream SBOM in `$out/share/sbom/` via `postInstall`; add `passthru.sbom` attribute
- **Docker** (if applicable): Attach SBOM to GHCR images via ORAS + cosign

### Tier 4: Future (Monitor)

- **TEA**: When GA, publish a TEI for automated discovery
- **OCI Referrers**: If qsdev container images become a primary distribution path

### Format Choice

**CycloneDX JSON** as primary format. Rationale:
- Better Go ecosystem tooling (cyclonedx-gomod, Syft)
- Dominant in practice (100% of PyPI adopters chose it)
- More compact than SPDX
- Native VEX support for vulnerability status communication
- ECMA-424 standardization provides institutional backing

Optionally generate SPDX JSON as a secondary format for consumers with SPDX-only tooling, but this doubles the maintenance burden for minimal practical benefit in 2026.
