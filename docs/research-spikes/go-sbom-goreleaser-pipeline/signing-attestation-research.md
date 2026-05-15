# SBOM Signing, Attestation, and Cryptographic Verification for Go Binaries

## Executive Summary

For a Go CLI tool like qsdev, the recommended approach is **cosign keyless signing via GitHub Actions OIDC** for both the binary checksums and SBOMs, **SLSA L3 provenance** via `slsa-github-generator`'s Go builder or GoReleaser's built-in signing, and **GitHub artifact attestations** for SBOMs. This eliminates all key management, provides cryptographic non-repudiation through Rekor transparency logs, and gives consumers three independent verification paths: `cosign verify-blob`, `slsa-verifier verify-artifact`, and `gh attestation verify`. The entire pipeline can be configured in GoReleaser's `signs:` block plus a few GitHub Actions steps.

---

## 1. The Sigstore Ecosystem

### 1.1 Architecture Overview

Sigstore is a suite of four components that together provide keyless code signing:

| Component | Role | Analogy |
|-----------|------|---------|
| **Cosign** | CLI for signing and verifying containers, blobs, attestations | `gpg sign`/`gpg verify` |
| **Fulcio** | Certificate Authority issuing short-lived (10-minute) X.509 certificates bound to OIDC identities | DigiCert, Let's Encrypt |
| **Rekor** | Append-only transparency log recording all signing events with timestamps | Certificate Transparency logs |
| **TUF** | Secure distribution of Fulcio root CA cert and Rekor public key | Manual root cert installation |

The fundamental shift from traditional PKI: instead of asking "do you trust this key?", Sigstore asks "do you trust this identity, at this moment?" All signatures are publicly auditable in Rekor.

### 1.2 How the Pieces Fit Together

The three-phase signing workflow:

**Phase 1 - Identity Verification**: An ephemeral keypair is generated in memory. An OIDC identity token is obtained (from GitHub Actions, Google, or Microsoft). Fulcio validates the token and issues a short-lived certificate binding the identity to the public key. The private key is destroyed after use.

**Phase 2 - Transparency Log Entry**: A timestamped record containing the artifact hash, public key, and signature is written to Rekor. This creates an immutable audit trail with cryptographic timestamps.

**Phase 3 - Verification**: Consumers verify by checking: (a) the certificate chain up to Fulcio's root CA, (b) that Rekor's `integratedTime` falls within the certificate's validity window, (c) that the certificate's Subject Alternative Name (SAN) matches the expected signer identity, (d) that the signature over the artifact hash is valid.

### 1.3 Trust Root Distribution (TUF)

Cosign ships with a foundational `root.json`. On first run, it fetches the latest trust root via TUF's chained verification protocol: each new `root.json` must be signed by the previous version's keys, enabling key rotation without binary updates. The `trusted_root.json` file contains the Fulcio CA chain and Rekor public key needed for offline verification.

---

## 2. Keyless Signing (Recommended for qsdev)

### 2.1 OIDC-Based Signing in CI

In GitHub Actions, keyless signing is the default and most practical approach. The workflow needs one permission:

```yaml
permissions:
  id-token: write  # Required for OIDC token minting
```

GitHub's OIDC provider (`https://token.actions.githubusercontent.com`) issues a JWT that Fulcio accepts. The resulting certificate's SAN encodes the exact workflow path:

```
https://github.com/org/repo/.github/workflows/release.yml@refs/tags/v1.0.0
```

Custom OID extensions (Enterprise Number `1.3.6.1.4.1.57264`) encode additional CI/CD provenance: OIDC issuer URL, source repository URI, commit digest, workflow trigger type, and runner environment type.

### 2.2 Why Keyless Eliminates Key Management

| Concern | Long-lived keys | Keyless |
|---------|----------------|---------|
| Key generation | Must generate and distribute | Ephemeral, auto-created |
| Key storage | Needs HSM, Vault, or encrypted secret | No keys to store |
| Key rotation | Manual process, coordination needed | N/A - each signing uses a fresh key |
| Key revocation | Complex CRL/OCSP infrastructure | Obsolete - 10-minute cert validity |
| Key compromise | Attacker has unlimited signing ability | 10-minute window, Rekor provides audit trail |
| Cost | $200-500+/year for commercial CAs | Free (Sigstore public instance) |

### 2.3 Verification Workflow for Consumers

For blob/binary signing, consumers run:

```bash
# Download the artifact and its .sigstore.json bundle
cosign verify-blob \
  --bundle qsdev_checksums.txt.sigstore.json \
  --certificate-identity "https://github.com/org/qsdev/.github/workflows/release.yml@refs/tags/v1.0.0" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  qsdev_checksums.txt
```

The `--certificate-identity` parameter pins verification to the exact workflow and tag. The `--certificate-oidc-issuer` ensures the token came from GitHub Actions. Both are required for keyless verification.

### 2.4 GoReleaser Keyless Signing Configuration

Modern cosign v3+ with GoReleaser:

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

This signs the checksum file (which covers all release artifacts) and produces a single `.sigstore.json` bundle containing the signature, Fulcio certificate, and Rekor log entry. The `--yes` flag skips interactive confirmation in CI.

For signing SBOMs as well:

```yaml
signs:
  - id: checksums
    cmd: cosign
    signature: "${artifact}.sigstore.json"
    args:
      - "sign-blob"
      - "--bundle=${signature}"
      - "${artifact}"
      - "--yes"
    artifacts: checksum
  - id: sboms
    cmd: cosign
    signature: "${artifact}.sigstore.json"
    args:
      - "sign-blob"
      - "--bundle=${signature}"
      - "${artifact}"
      - "--yes"
    artifacts: sbom
```

---

## 3. Key-Based Signing (When You'd Want It)

### 3.1 Use Cases for Long-Lived Keys

Keyless signing depends on Sigstore's public infrastructure (Fulcio, Rekor). Long-lived keys are appropriate when:

- **Air-gapped environments**: No network access during signing or verification
- **Regulatory requirements**: Some compliance frameworks mandate specific key management controls
- **Self-hosted infrastructure**: Organizations running their own signing infrastructure
- **Offline verification**: Consumers who cannot reach Rekor (though cosign supports `--offline=true` with a trusted root)

### 3.2 GPG vs Cosign Key Pairs

**GPG**: Traditional approach. GoReleaser defaults to `gpg` as the signing command. Widely understood but has terrible UX for key distribution (keyservers, manual fingerprint verification). Configuration:

```yaml
signs:
  - cmd: gpg
    args: ["-u", "<key-id>", "--output", "${signature}", "--detach-sign", "${artifact}"]
    artifacts: checksum
```

**Cosign key pairs**: Better UX than GPG. Generate with `cosign generate-key-pair`, produces encrypted PEM files. Still records signatures in Rekor for transparency. The public key can be embedded in documentation or distributed via TUF.

```yaml
signs:
  - cmd: cosign
    args:
      - "sign-blob"
      - "--key=cosign.key"
      - "--bundle=${signature}"
      - "${artifact}"
      - "--yes"
    artifacts: checksum
    env:
      - COSIGN_PASSWORD={{ .Env.COSIGN_PASSWORD }}
```

### 3.3 Key Distribution

For key-based signing, you must solve key distribution. Options:
- Embed the public key in the project's README and repository
- Distribute via a KMS (AWS KMS, GCP KMS, Azure Key Vault, HashiCorp Vault)
- Use hardware tokens for signing, distribute public key normally
- Host a TUF repository for automated trust root updates

### 3.4 Recommendation for qsdev

**Use keyless signing.** qsdev is built on GitHub Actions, the Sigstore public instance is free and highly available (99.5% SLO), and keyless eliminates all operational overhead. The only scenario requiring keys would be if consumers are in strictly air-gapped environments - and even then, cosign's `--offline=true` mode with a saved trusted root handles most cases.

---

## 4. in-toto Attestation Framework

### 4.1 What in-toto Is

in-toto is a CNCF framework for making verifiable claims about software supply chains. It defines a standard **Statement** structure:

```json
{
  "_type": "https://in-toto.io/Statement/v1",
  "subject": [
    {
      "name": "my-artifact",
      "digest": { "sha256": "abc123..." }
    }
  ],
  "predicateType": "https://example.com/predicate/v1",
  "predicate": { ... }
}
```

The Statement says: "for this subject (artifact), I am stating this predicate (claim)." The predicate is typed and can contain anything: SLSA provenance, an SBOM, vulnerability scan results, test results, etc.

### 4.2 How It Differs from Simple Signing

| Aspect | Simple Signing (cosign sign-blob) | in-toto Attestation (cosign attest) |
|--------|-----------------------------------|-------------------------------------|
| What's signed | The artifact's hash | A Statement *about* the artifact |
| Metadata | None beyond the signature itself | Typed predicate with structured metadata |
| Use case | "This artifact came from this identity" | "This artifact has these properties (SBOM, provenance, scan results)" |
| Format | Signature + certificate | DSSE envelope wrapping an in-toto Statement |
| Composability | Single assertion | Multiple attestations can be stacked for one artifact |

Simple signing proves **identity** (who signed it). Attestations prove **properties** (what's in it, how it was built, what vulnerabilities were found).

### 4.3 DSSE (Dead Simple Signing Envelope)

in-toto Statements are wrapped in DSSE envelopes for signing:

```json
{
  "payloadType": "application/vnd.in-toto+json",
  "payload": "<base64-encoded Statement>",
  "signatures": [
    {
      "keyid": "...",
      "sig": "<base64-encoded signature>"
    }
  ]
}
```

The payload is base64-encoded inside, the signature wraps the outside. This prevents signature-stripping attacks and supports multiple signatures.

### 4.4 Go Ecosystem Support

in-toto-golang is the Go implementation, used by many cloud-native tools. The protobuf definitions support Go as the most mature language binding. Cosign, slsa-github-generator, and the GitHub Actions `attest` action all produce in-toto format attestations.

### 4.5 Predicate Types Relevant to qsdev

| Predicate Type | URI | Purpose |
|---------------|-----|---------|
| SLSA Provenance v1 | `https://slsa.dev/provenance/v1` | How the artifact was built |
| CycloneDX SBOM | `https://cyclonedx.org/bom` | What's inside the artifact |
| SPDX SBOM | `https://spdx.dev/Document/v2.3` | What's inside (alternative format) |
| Vulnerability scan | `https://cosign.sigstore.dev/attestation/vuln/v1` | Known vulnerabilities |

---

## 5. SLSA Provenance

### 5.1 SLSA Levels

SLSA (Supply-chain Levels for Software Artifacts) defines four Build levels:

| Level | Name | Requirements | What it stops |
|-------|------|-------------|---------------|
| L0 | No guarantees | Nothing | Nothing |
| L1 | Provenance exists | Build process generates provenance (even self-attested) | Mistakes, accidental corruption |
| L2 | Hosted build platform | Signed provenance from a hosted platform (e.g., GitHub Actions) | Explicit forgery requires attacking the platform |
| L3 | Hardened builds | Build platform isolates user code from provenance generation; no cross-run influence | Insider threats, compromised credentials, SUNSPOT-style attacks |

**L3 is the target for qsdev.** GitHub Actions + `slsa-github-generator` achieves this because the reusable workflow that generates provenance runs in a separate, isolated execution environment from user code. Even if the user's build steps are compromised, they cannot tamper with the provenance.

### 5.2 How Go Builds Achieve L3

The `slsa-framework/slsa-github-generator` provides a dedicated Go builder:

```yaml
jobs:
  build:
    permissions:
      id-token: write
      contents: write
      actions: read
    uses: slsa-framework/slsa-github-generator/.github/workflows/builder_go_slsa3.yml@v2.1.0
    with:
      go-version: "1.22"
      config-file: .slsa-goreleaser.yml
      evaluated-envs: "VERSION:${{ github.ref_name }}"
      upload-assets: true
```

The `.slsa-goreleaser.yml` configuration:

```yaml
version: 1
env:
  - GO111MODULE=on
  - CGO_ENABLED=0
flags:
  - -trimpath
ldflags:
  - "-X main.Version={{ .Env.VERSION }}"
  - "-s -w"
goos: linux
goarch: amd64
binary: qsdev-{{ .Os }}-{{ .Arch }}
```

The build job compiles user code only. It never participates in provenance assembly or signing. Since `builder_go_slsa3.yml` runs as a **separate reusable workflow**, user code cannot reach the signing key. This structural separation satisfies SLSA L3.

### 5.3 Provenance Content

The generated `.intoto.jsonl` file contains a DSSE-wrapped in-toto Statement with SLSA provenance:

```json
{
  "_type": "https://in-toto.io/Statement/v0.1",
  "predicateType": "https://slsa.dev/provenance/v0.2",
  "subject": [
    {
      "name": "qsdev-linux-amd64",
      "digest": { "sha256": "abc123..." }
    }
  ],
  "predicate": {
    "builder": {
      "id": "https://github.com/slsa-framework/slsa-github-generator/.github/workflows/builder_go_slsa3.yml@refs/tags/v2.1.0"
    },
    "buildType": "https://github.com/slsa-framework/slsa-github-generator/go@v1",
    "invocation": {
      "configSource": {
        "uri": "git+https://github.com/org/qsdev@refs/tags/v1.0.0",
        "digest": { "sha1": "abc123..." },
        "entryPoint": ".github/workflows/release.yml"
      },
      "environment": {
        "github_event_name": "push",
        "github_ref": "refs/tags/v1.0.0",
        "github_repository_owner": "org",
        "os": "ubuntu24"
      }
    },
    "materials": [
      {
        "uri": "git+https://github.com/org/qsdev@refs/tags/v1.0.0",
        "digest": { "sha1": "abc123..." }
      }
    ]
  }
}
```

### 5.4 SLSA vs SBOM: Complementary, Not Competing

| Aspect | SLSA Provenance | SBOM |
|--------|----------------|------|
| Question answered | "How was this built? By whom? From what source?" | "What components are inside this artifact?" |
| Abstraction level | Coarse - build parameters, source, builder | Fine-grained - every dependency, version, license |
| Threat model | Build tampering, source substitution | Known vulnerabilities, license compliance |
| Format | in-toto Statement with SLSA predicate | SPDX or CycloneDX document |

Both are needed. SLSA proves the build wasn't tampered with. The SBOM proves what's inside. Together they answer "was this built correctly?" and "what was built?"

### 5.5 SLSA L3 Limitation: GoReleaser Compatibility

**Important tradeoff**: The `slsa-github-generator` Go builder uses its own build configuration (`.slsa-goreleaser.yml`), which is a subset of GoReleaser's config. If qsdev uses GoReleaser for its full release pipeline (archives, Homebrew, Docker images, changelogs), you cannot use the SLSA Go builder directly because GoReleaser needs to control the build.

**Two paths forward:**

1. **GoReleaser + Generic SLSA Generator**: Use GoReleaser for the build, then use `slsa-github-generator`'s generic generator to create provenance for the resulting artifacts. This achieves SLSA L2 (not L3, because GoReleaser runs in the same job as user code).

2. **GoReleaser + GitHub Attestations**: Use `actions/attest@v4` after GoReleaser runs to create provenance attestations. This is simpler but also L2.

3. **SLSA Go Builder + manual distribution**: Use the SLSA Go builder for L3 provenance but handle distribution (Homebrew, Docker, etc.) separately. More work but achieves L3.

For qsdev, **Option 1 (GoReleaser + cosign signing + GitHub attestations)** is the pragmatic choice. SLSA L2 with cosign-signed checksums and SBOMs provides strong security with minimal pipeline complexity. L3 is achievable later if needed by splitting the build step.

---

## 6. SBOM-Specific Signing

### 6.1 Two Things to Sign

There are two distinct signing targets:

1. **Sign the artifact** (the binary): Proves identity - "this binary came from this CI pipeline." Done via cosign signing the checksum file.

2. **Sign the SBOM document**: Proves the SBOM's integrity and provenance - "this SBOM was generated by this CI pipeline for this artifact and hasn't been tampered with." Done via cosign signing the SBOM file, or wrapping it in an in-toto attestation.

**Both are needed.** Signing only the binary proves who built it but doesn't protect the SBOM from tampering. Signing only the SBOM doesn't prove the binary itself is authentic.

### 6.2 SBOM as Signed Attestation

The strongest approach wraps the SBOM in an in-toto attestation. This binds the SBOM to a specific artifact digest:

```json
{
  "_type": "https://in-toto.io/Statement/v1",
  "subject": [
    {
      "name": "qsdev-linux-amd64.tar.gz",
      "digest": { "sha256": "binary-hash..." }
    }
  ],
  "predicateType": "https://cyclonedx.org/bom",
  "predicate": {
    "... full CycloneDX SBOM content ..."
  }
}
```

This creates a cryptographic binding: "this SBOM describes this exact artifact." A tampered SBOM or a mismatched artifact both fail verification.

### 6.3 Three Approaches for qsdev

**Approach A: Sign SBOM files as blobs (simplest)**
- GoReleaser generates SBOMs with syft
- Cosign signs each SBOM file as a blob
- Consumer verifies with `cosign verify-blob`
- Limitation: No cryptographic binding between SBOM and the artifact it describes

**Approach B: GitHub Attestations (recommended for binary releases)**
- GoReleaser generates SBOMs and binaries
- `actions/attest@v4` creates SBOM attestations binding each SBOM to its artifact
- Consumer verifies with `gh attestation verify --predicate-type https://spdx.dev/Document/v2.3`
- Advantage: SBOM is cryptographically bound to the artifact digest

**Approach C: Cosign attestations on OCI images (for container releases)**
- `syft attest --key cosign.key <image> -o cyclonedx-json` creates an SBOM attestation attached to the image
- Or keyless: `cosign attest --predicate sbom.json --type cyclonedx <image>`
- Consumer verifies with `cosign verify-attestation --type cyclonedx <image>`

**For qsdev: Use Approach A (cosign sign-blob for SBOM files) + Approach B (GitHub attestations for SBOM binding).** This covers both GitHub-native consumers (who use `gh attestation verify`) and Sigstore-native consumers (who use `cosign verify-blob`).

### 6.4 How They Compose

The full signing stack for a release:

```
Binary artifacts
  |-- checksums.txt (SHA256 of all files)
  |     |-- checksums.txt.sigstore.json (cosign keyless signature)
  |-- qsdev_0.1.0_linux_amd64.tar.gz.sbom.cdx.json (CycloneDX SBOM)
  |     |-- qsdev_0.1.0_linux_amd64.tar.gz.sbom.cdx.json.sigstore.json (cosign keyless signature)
  |-- GitHub Attestation: build provenance (binds binary to build workflow)
  |-- GitHub Attestation: SBOM (binds SBOM to binary digest)
```

---

## 7. Verification Workflows for Consumers

### 7.1 Cosign Verify-Blob (for signed checksums and SBOMs)

```bash
# Download release artifacts
curl -LO https://github.com/org/qsdev/releases/download/v1.0.0/qsdev_1.0.0_linux_amd64.tar.gz
curl -LO https://github.com/org/qsdev/releases/download/v1.0.0/checksums.txt
curl -LO https://github.com/org/qsdev/releases/download/v1.0.0/checksums.txt.sigstore.json

# Verify the checksum file's signature
cosign verify-blob \
  --bundle checksums.txt.sigstore.json \
  --certificate-identity "https://github.com/org/qsdev/.github/workflows/release.yml@refs/tags/v1.0.0" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  checksums.txt

# Verify the binary against the checksums
sha256sum -c checksums.txt --ignore-missing

# Verify SBOM signature
cosign verify-blob \
  --bundle qsdev_1.0.0_linux_amd64.tar.gz.sbom.cdx.json.sigstore.json \
  --certificate-identity "https://github.com/org/qsdev/.github/workflows/release.yml@refs/tags/v1.0.0" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  qsdev_1.0.0_linux_amd64.tar.gz.sbom.cdx.json
```

### 7.2 GitHub Attestation Verify (simplest for GitHub-hosted projects)

```bash
# Verify build provenance
gh attestation verify qsdev_1.0.0_linux_amd64.tar.gz \
  -R org/qsdev

# Verify SBOM attestation
gh attestation verify qsdev_1.0.0_linux_amd64.tar.gz \
  -R org/qsdev \
  --predicate-type https://spdx.dev/Document/v2.3

# View SBOM content from attestation
gh attestation verify qsdev_1.0.0_linux_amd64.tar.gz \
  -R org/qsdev \
  --predicate-type https://spdx.dev/Document/v2.3 \
  --format json \
  --jq '.[].verificationResult.statement.predicate'
```

### 7.3 SLSA Verifier (for SLSA provenance)

```bash
# Download binary and provenance
curl -LO https://github.com/org/qsdev/releases/download/v1.0.0/qsdev-linux-amd64
curl -LO https://github.com/org/qsdev/releases/download/v1.0.0/qsdev-linux-amd64.intoto.jsonl

# Verify SLSA provenance
slsa-verifier verify-artifact qsdev-linux-amd64 \
  --provenance-path qsdev-linux-amd64.intoto.jsonl \
  --source-uri github.com/org/qsdev \
  --source-tag v1.0.0
```

### 7.4 Comparison of Verification Tools

| Tool | What it verifies | Requires | Best for |
|------|-----------------|----------|----------|
| `cosign verify-blob` | Keyless signature on any file | cosign CLI | Signing checksums, SBOMs, any blob |
| `gh attestation verify` | GitHub-generated attestations | GitHub CLI | GitHub-native consumers, simplest UX |
| `slsa-verifier` | SLSA provenance from trusted builders | slsa-verifier CLI | Build provenance, SLSA compliance |
| `cosign verify-attestation` | in-toto attestations on OCI images | cosign CLI | Container image attestations |

---

## 8. GoReleaser Signing Integration

### 8.1 Signs Block

GoReleaser's `signs:` block supports multiple signing configurations:

```yaml
signs:
  - id: cosign-checksums
    cmd: cosign
    signature: "${artifact}.sigstore.json"
    args:
      - "sign-blob"
      - "--bundle=${signature}"
      - "${artifact}"
      - "--yes"
    artifacts: checksum

  - id: cosign-sboms
    cmd: cosign
    signature: "${artifact}.sigstore.json"
    args:
      - "sign-blob"
      - "--bundle=${signature}"
      - "${artifact}"
      - "--yes"
    artifacts: sbom
```

### 8.2 Docker Signs Block

For container images:

```yaml
docker_signs:
  - cmd: cosign
    artifacts: images
    args:
      - "sign"
      - "${artifact}"
      - "--yes"
    output: true
```

### 8.3 Artifact Types Available for Signing

- `checksum` - Checksum files (recommended: sign this, it covers everything)
- `sbom` - Generated SBOMs
- `all` - All artifacts
- `archive` - Tar/zip archives
- `binary` - Raw binaries
- `source` - Source archives
- `package` - Linux packages (deb, rpm, apk)
- `installer` - MSI, NSIS, macOS Pkg

### 8.4 Cosign v3 Bundle Format

Cosign v3 introduced the `--bundle` flag, replacing the older `--output-certificate` + `--output-signature` approach. The `.sigstore.json` bundle contains everything needed for verification in a single file: signature, Fulcio certificate, and Rekor log entry. This simplifies both the GoReleaser config and the consumer verification workflow.

### 8.5 CI Workflow Setup

```yaml
# .github/workflows/release.yml
name: Release
on:
  push:
    tags: ["v*"]

permissions:
  contents: write      # Upload release assets
  id-token: write      # OIDC for keyless signing
  attestations: write  # GitHub attestations
  packages: write      # Container registry (if applicable)

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: actions/setup-go@v5
        with:
          go-version: stable

      - uses: sigstore/cosign-installer@v3

      - uses: anchore/sbom-action/download-syft@v0

      - uses: goreleaser/goreleaser-action@v6
        with:
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

      # Optional: GitHub attestations for SBOM binding
      - name: Generate SBOM attestation
        uses: actions/attest@v4
        with:
          subject-path: 'dist/qsdev_*_checksums.txt'
          sbom-path: 'dist/*.sbom.cdx.json'
```

---

## 9. Transparency Logs (Rekor)

### 9.1 How Rekor Provides Non-Repudiation

Rekor is an immutable, tamper-resistant ledger. Every signing event is recorded with:
- The artifact hash
- The signature
- The Fulcio certificate (including signer identity)
- A cryptographically signed timestamp (Signed Entry Timestamp, SET)
- An inclusion proof (Merkle tree path)

Because Rekor is append-only and publicly auditable, a signer cannot deny having signed an artifact. The signed timestamp proves the signing occurred at a specific moment. The inclusion proof proves the entry exists in the log without requiring the full log to be downloaded.

### 9.2 Merkle Tree Structure

Rekor uses a Merkle tree where:
- Leaf nodes contain individual entry hashes
- Parent nodes contain `Hash(left_child + right_child)`
- The root hash encompasses the entire log and is signed by Rekor's key

Verifying an entry's existence requires only O(log n) hashes. A million entries need ~20 hashes for an inclusion proof. This makes verification efficient even as the log grows.

### 9.3 Rekor v2

Rekor v2 went GA in October 2025, replacing the Trillian backend with Trillian-Tessera. Cosign v2.6.0+ automatically uses v2 based on SigningConfig and TrustedRoot distributed via TUF. The API and user experience are unchanged.

### 9.4 Querying Rekor

```bash
# Search by artifact hash
rekor-cli search --sha sha256:abc123...

# Search by email identity
rekor-cli search --email user@example.com

# Get a specific entry
rekor-cli get --log-index 12345678

# Verify inclusion
rekor-cli verify --entry-uuid <uuid>
```

### 9.5 Monitoring

- **Rekor Monitor**: GitHub Actions-based tool that performs consistency checks
- **Omniwitness**: Created by Trillian's team for independent log auditing
- Organizations can monitor for unexpected signing events under their identities

---

## 10. Real-World Examples

### 10.1 goreleaser/example-supply-chain

The canonical example. GoReleaser manages the entire pipeline:
1. Build binaries using Go Mod Proxy as source of truth
2. Generate SBOMs with syft
3. Generate checksums
4. Sign checksums with cosign (keyless)
5. Build Docker images from the same binaries
6. Sign container images with cosign

Repository: https://github.com/goreleaser/example-supply-chain (60 stars, 19 releases)

### 10.2 mchmarny/s3cme

Template Go app demonstrating full supply chain security:
- Container images built with ko (automatic SBOM generation in SPDX v2.3)
- Images signed and attested via cosign
- SLSA provenance via slsa-github-generator
- Provenance verification using slsa-verifier and CUE policies
- In-cluster enforcement via Sigstore admission controller

Four OCI artifacts per release: image, `.sig` (signature), `.att` (attestation), `.sbom` (SBOM).

### 10.3 slsa-framework/slsa-verifier (dogfooding)

The SLSA verifier itself uses the SLSA Go builder for L3 provenance. Every release includes:
- Binary per platform
- `.intoto.jsonl` provenance file per binary
- Verification with: `slsa-verifier verify-artifact <binary> --provenance-path <provenance> --source-uri github.com/slsa-framework/slsa-verifier --source-tag <tag>`

### 10.4 Kubernetes ecosystem

Multiple Kubernetes components (kubectl, kustomize) use slsa-github-generator. Sigstore's policy controller can enforce that only images with valid SLSA provenance and cosign signatures are admitted to clusters.

---

## 11. Recommended Architecture for qsdev

### 11.1 Complete Pipeline

```
Tag push (v1.0.0)
    |
    v
GoReleaser runs:
    |- Build Go binaries (cross-platform)
    |- Generate SBOMs with syft (CycloneDX JSON)
    |- Generate checksums (SHA256)
    |- Sign checksums with cosign (keyless, .sigstore.json bundle)
    |- Sign SBOMs with cosign (keyless, .sigstore.json bundle)
    |- Build Docker images (optional)
    |- Sign Docker images with cosign (optional)
    |- Upload all artifacts to GitHub Release
    |
    v
GitHub Actions post-GoReleaser:
    |- actions/attest@v4: build provenance attestation
    |- actions/attest@v4: SBOM attestation (binds SBOM to binary digest)
```

### 11.2 Release Artifacts

Each release includes:

| File | Purpose |
|------|---------|
| `qsdev_1.0.0_linux_amd64.tar.gz` | Binary archive |
| `qsdev_1.0.0_darwin_arm64.tar.gz` | Binary archive |
| `qsdev_1.0.0_checksums.txt` | SHA256 checksums |
| `qsdev_1.0.0_checksums.txt.sigstore.json` | Cosign signature bundle for checksums |
| `qsdev_1.0.0_linux_amd64.tar.gz.sbom.cdx.json` | CycloneDX SBOM |
| `qsdev_1.0.0_linux_amd64.tar.gz.sbom.cdx.json.sigstore.json` | Cosign signature bundle for SBOM |
| GitHub Attestation (stored in GitHub API) | Build provenance + SBOM attestation |

### 11.3 Consumer Verification Options

| Consumer Sophistication | Verification Method | Effort |
|------------------------|-------------------|--------|
| Basic | `sha256sum -c checksums.txt` | No tooling needed |
| Standard | `gh attestation verify <binary> -R org/qsdev` | GitHub CLI only |
| Security-conscious | `cosign verify-blob --bundle checksums.txt.sigstore.json ...` | Cosign CLI |
| Compliance | All three: cosign verify-blob + gh attestation verify + SBOM verification | Full verification |

### 11.4 What This Achieves

- **SLSA L2** (L3 achievable by splitting build into SLSA Go builder)
- **Signed checksums**: Proves all artifacts came from the CI pipeline
- **Signed SBOMs**: Proves SBOM integrity and provenance
- **GitHub attestations**: Binds SBOMs to artifact digests with in-toto format
- **Transparency**: All signatures recorded in Rekor for audit
- **Zero key management**: Keyless signing via GitHub Actions OIDC
- **Multiple verification paths**: cosign, gh CLI, or manual checksum

---

## 12. Open Questions

1. **SLSA L3 vs GoReleaser**: If L3 is required, should qsdev use the SLSA Go builder for binaries and GoReleaser only for packaging/distribution? This splits the pipeline but achieves the highest provenance level.

2. **VEX integration**: Should vulnerability scan results also be signed and attached as attestations alongside SBOMs? Grype can produce VEX documents that complement SBOM data.

3. **Nix derivation signing**: How should SBOMs and provenance work for Nix-packaged builds? Nix has its own reproducibility guarantees but the SBOM/signing story is different.

4. **Offline verification**: Should qsdev ship a `verify.sh` script or Makefile target that consumers can run locally?

---

## Sources

All raw source material is saved in `docs/`:

- `cosign-signing-overview-sigstore.md` - Sigstore cosign signing overview
- `sigstore-keyless-verification-deep-dive.md` - Technical deep dive on keyless verification internals
- `cosign-readme-github.md` - Cosign GitHub README with all commands
- `goreleaser-signing-configuration.md` - GoReleaser signs: block configuration
- `goreleaser-supply-chain-security-blog-signing.md` - GoReleaser supply chain blog post
- `slsa-3-compliance-github-actions-go.md` - GitHub blog on SLSA L3 for Go
- `slsa-github-generator-go-builder.md` - SLSA GitHub Generator README
- `slsa-provenance-hands-on-github-actions.md` - Hands-on SLSA provenance tutorial
- `slsa-levels-spec-v1.1.md` - SLSA levels specification
- `slsa-faq-spec-v1.1.md` - SLSA FAQ (SLSA vs SBOM, in-toto relationship)
- `in-toto-attestation-framework-readme.md` - in-toto attestation framework
- `github-actions-attest-sbom.md` - GitHub Actions attest-sbom action
- `github-artifact-attestations-provenance.md` - GitHub artifact attestations documentation
- `rekor-transparency-log-overview.md` - Rekor transparency log overview
- `syft-sigstore-sbom-attestations.md` - Creating SBOM attestations with Syft and Sigstore
- `s3cme-template-go-supply-chain.md` - s3cme template Go repo with full supply chain
- `goreleaser-gitlab-cosign-sbom.md` - GoReleaser + GitLab + cosign + SBOM
