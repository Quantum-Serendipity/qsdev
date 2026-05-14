<!-- Compiled research document -->
<!-- Retrieved: 2026-05-14 -->
<!-- Sources listed inline throughout -->

# Sigstore Ecosystem Research: Detailed Findings

## 1. Sigstore Architecture

Sigstore is a CNCF-graduated project (March 2024) providing software artifact signing, verification, and transparency logging. It comprises three core components:

### Cosign (Client Tool)
- CLI tool for signing and verifying artifacts
- Supports container images, blobs (arbitrary files), SBOMs, binaries, release files
- Available in nixpkgs: `nix-env -iA nixpkgs.cosign` or `nixos.cosign`
- Written in Go; future versions will be based on sigstore-go
- Source: https://docs.sigstore.dev/about/overview/

### Fulcio (Certificate Authority)
- Code-signing CA that issues short-lived certificates (typically 10-minute validity)
- Validates OIDC identity tokens from providers (Google, GitHub, Microsoft)
- Binds an ephemeral public key to a verified identity (email, CI workflow URL)
- Certificates contain the signer's identity in a machine-readable format
- No long-lived keys to manage -- the private key is discarded after single use
- Source: https://docs.sigstore.dev/about/overview/

### Rekor (Transparency Log)
- Immutable, append-only ledger recording all signing events
- Records: artifact digest, signature, certificate, timestamp
- Enables public audit: anyone can verify when a signature was created
- Provides "signed entry timestamps" proving an entry existed at a given time
- Privacy concern: signer identity (email) is publicly visible in the log
- Source: https://docs.sigstore.dev/about/overview/

### TUF Integration
- Sigstore uses The Update Framework (TUF) internally to protect its own root of trust
- TUF secures Fulcio CA keys, Rekor log keys via a "TUF sandwich" architecture
- Root of trust established via public key ceremony with 5 hardware keys held by community members
- TUF delegation allows other projects to leverage Sigstore's root
- TUF and Sigstore are complementary, not competing: TUF handles key lifecycle, Sigstore handles identity-based signing
- Source: https://dlorenc.medium.com/using-the-update-framework-in-sigstore-dc393cfe6b52

## 2. Cosign Blob Signing (Non-Container Artifacts)

### How It Works

`cosign sign-blob` signs arbitrary files using the same infrastructure as container signing:

**Keyless (OIDC-based):**
```bash
cosign sign-blob <file> --bundle bundle.sigstore.json
```
- Opens browser for OIDC authentication (Google/GitHub/Microsoft)
- Generates ephemeral keypair, gets Fulcio certificate, signs digest, logs to Rekor
- Produces a `.sigstore.json` bundle containing signature + certificate + Rekor inclusion proof

**Key-based:**
```bash
cosign sign-blob --key cosign.key --bundle bundle.sigstore.json myfile.bin
```
- Uses a static ECDSA-P256 key pair (generated via `cosign generate-key-pair`)
- Supports KMS backends (AWS KMS, GCP KMS, Azure Key Vault, HashiCorp Vault)
- Supports hardware tokens (Yubikey via PIV)

**CI/CD automated:**
```bash
cosign sign-blob --yes --key cosign.key --bundle bundle.sigstore.json artifact.tar.gz
```
- `--yes` skips interactive confirmation
- GitHub Actions provides ambient OIDC tokens for keyless signing without browser

### Verification

```bash
cosign verify-blob <file> --bundle bundle.sigstore.json \
  --certificate-identity user@example.com \
  --certificate-oidc-issuer https://accounts.google.com
```
- Must specify expected identity and OIDC issuer (security requirement to prevent impersonation)
- Verifies: signature over digest, certificate chain to Sigstore root, Rekor inclusion proof
- Source: https://docs.sigstore.dev/cosign/signing/signing_with_blobs/

### What Gets Signed

Cosign signs the SHA-256 **digest** of the file, not the file content itself:
- `Sign(sha256(payload))` -- only the hash is signed and uploaded
- The full file never leaves the local machine or gets sent to Rekor
- This means signing a 50 GB ZIM file is feasible: only the hash computation takes time
- SHA-256 hashing of a 50 GB file takes ~2-5 minutes on modern hardware
- Alternative hash algorithms supported: sha224, sha256, sha384, sha512
- Source: https://github.com/sigstore/cosign/blob/main/specs/SIGNATURE_SPEC.md

### Output Artifacts

The `.sigstore.json` bundle contains:
- **Signature**: base64-encoded ECDSA-P256 signature over the artifact digest
- **Certificate**: Fulcio-issued X.509 certificate with signer identity
- **SignedEntryTimestamp**: Rekor's signature over the log entry metadata
- **Payload**: integratedTime (UNIX timestamp), logIndex, logID

Bundle is a small JSON file (~2-5 KB) regardless of signed artifact size.

## 3. Ecosystem Precedent (Non-Container Usage)

### npm (GA since September 2023)
- Sigstore integrated directly into the npm CLI via sigstore-js and tuf-js libraries
- All packages published from GitHub Actions get Sigstore provenance attestations by default
- SLSA-compliant provenance linking package to source repo + commit
- 3,800+ projects adopted during beta; 500M+ downloads of provenance-enabled packages
- Verification happens transparently on `npm install`
- Source: https://blog.sigstore.dev/npm-provenance-ga/

### PyPI (GA since November 2024)
- PEP 740 standard for Sigstore-based attestations on PyPI
- Leverages Trusted Publishing (existing OIDC integration with GitHub)
- Zero config: projects using GitHub Actions + Trusted Publishing get attestations automatically
- 20,000+ attestations uploaded; ~5% of top 360 projects attested
- Source: https://blog.sigstore.dev/pypi-attestations-ga/

### Kubernetes
- All release artifacts signed with cosign since Kubernetes 1.24
- Includes binaries, container images, SBOMs

### GitHub Artifact Attestations
- GitHub's own artifact attestation feature built on Sigstore
- Available for any GitHub Actions workflow

### Linux Distributions
- No major distro has adopted Sigstore for package signing yet
- Fedora, Arch, Debian still use GPG-based package signing
- Cosign available as .deb and .rpm packages for easy installation

### Nix Ecosystem
- No existing Sigstore integration in Nix's build/fetch infrastructure
- nixpkgs has precedent for GPG verification via fetchurl + runCommand pattern (PR #43233)
- GPG approach: fetch key + signature as separate derivations, verify in a runCommand step
- cosign and minisign both packaged in nixpkgs

### Documentation/Content Signing
- No known precedent for signing documentation packages (ZIM, DevDocs) with Sigstore or any other tool
- This would be novel usage

## 4. Nix Integration Feasibility

### cosign Binary Availability
- `cosign` is packaged in nixpkgs and installable via `nix-env -iA nixpkgs.cosign`
- Can be included as a build dependency in a Nix derivation

### Verification at Build Time
Following the GPG precedent from nixpkgs PR #43233, cosign verification could work as:

```nix
# Conceptual -- not tested
let
  zimFile = fetchurl {
    url = "https://download.kiwix.org/zim/wikipedia_en_all.zim";
    sha256 = "...";
  };
  zimBundle = fetchurl {
    url = "https://download.kiwix.org/zim/wikipedia_en_all.zim.sigstore.json";
    sha256 = "...";
  };
in runCommand "verified-zim" { nativeBuildInputs = [ cosign ]; } ''
  cosign verify-blob ${zimFile} \
    --bundle ${zimBundle} \
    --certificate-identity "kiwix-release@kiwix.org" \
    --certificate-oidc-issuer "https://accounts.google.com"
  ln -s ${zimFile} $out
''
```

**Challenges:**
- Nix sandbox restricts network access during builds -- verification requiring Rekor online checks would fail
- Offline verification is possible with the bundle (contains inclusion proof) but needs `--offline` flag or equivalent
- cosign pulls in significant Go dependencies, increasing build closure size
- The Nix SRI hash already pins the exact content -- signing adds provenance but not additional integrity

### Verification at Runtime
- gdev could run `cosign verify-blob` as a post-download check before serving content via MCP
- Runtime verification avoids Nix sandbox issues
- Could be a health check or one-time verification on first use
- Requires cosign in the gdev runtime closure

### Key Management for gdev
If gdev signs its own content packages (wrapping upstream ZIM/DevDocs):
- **Keyless OIDC**: Requires CI/CD environment (GitHub Actions) -- good for automated releases
- **Static key**: Simpler, works offline, but requires key distribution and rotation
- **KMS-backed**: Production-grade but adds infrastructure dependency

## 5. Alternatives Comparison

### GPG/PGP Detached Signatures
- **How it works**: `gpg --detach-sign file` produces `.sig` file; verify with `gpg --verify file.sig file`
- **Key management**: Manual, complex. Web of Trust or manual key exchange. Key rotation is painful.
- **Nix precedent**: Yes -- nixpkgs PR #43233 demonstrates fetchurl + GPG verification
- **Large files**: No issue -- signs SHA digest
- **Pros**: Universal tooling, well-understood, offline-native, no infrastructure dependency
- **Cons**: Key distribution is the unsolved problem. GPG UX is notoriously poor. No transparency log.
- **Nixpkgs**: gnupg packaged

### Minisign
- **How it works**: `minisign -Sm file` produces `.minisig` file; verify with `minisign -Vm file -P <pubkey>`
- **Key management**: Simple Ed25519 keypair. Public key is a short base64 string (can be embedded in source code).
- **Nix precedent**: None specific, but minisign is packaged in nixpkgs
- **Large files**: Pre-hashing mode (`-H`) uses Blake2b-512, handles large files efficiently
- **Pros**: Dead simple. Tiny binary. Public key fits in a CLI argument or config file. Ed25519 is strong. Compatible with OpenBSD signify.
- **Cons**: No transparency log. No identity binding. Key rotation requires manual distribution. No revocation mechanism.
- **Nixpkgs**: minisign packaged (v0.11)

### TUF (The Update Framework)
- **How it works**: Framework for securing software update systems. Defines roles (root, targets, snapshot, timestamp) with separate keys. Metadata includes hashes, signatures, expiration dates.
- **Key management**: Sophisticated multi-key model with delegation, threshold signatures, and built-in key rotation.
- **Nix precedent**: None
- **Large files**: Designed for software distribution -- handles any size
- **Pros**: Specifically designed for the software distribution threat model. Handles key compromise, rollback attacks, freeze attacks, mix-and-match attacks. Battle-tested (PyPI, Docker, Rust/crates.io).
- **Cons**: Complex to implement. Requires running repository infrastructure. Overkill for a single-developer project. python-tuf or go-tuf needed as dependencies.
- **Nixpkgs**: python-tuf available via pip; no native nixpkgs package for the framework itself

### SSH Signing
- **How it works**: `ssh-keygen -Y sign -f key -n file myfile` produces `.sig` file; verify with `ssh-keygen -Y verify`
- **Key management**: Leverages existing SSH key infrastructure. Keys already on GitHub. Allowed_signers file maps identities to keys.
- **Nix precedent**: None specific, but OpenSSH is always available on NixOS
- **Large files**: Hashes file content before signing -- no size limit
- **Pros**: Zero additional tooling (OpenSSH is ubiquitous). Keys already distributed via GitHub. Simple allowed_signers format. Git already supports SSH signing.
- **Cons**: No transparency log. No certificate authority. Allowed_signers file must be manually maintained and distributed. No standard for artifact-signing metadata beyond the signature itself.
- **Nixpkgs**: openssh always available

### Sigstore/Cosign
- **How it works**: `cosign sign-blob file --bundle out.sigstore.json`; verify with `cosign verify-blob`
- **Key management**: Keyless via OIDC (best for CI/CD), or static keys, or KMS-backed
- **Nix precedent**: None specific; cosign packaged in nixpkgs
- **Large files**: Signs SHA-256 digest only -- works for any size
- **Pros**: No key management with keyless mode. Transparency log provides audit trail. Identity-based trust. Strong ecosystem momentum (npm, PyPI, Kubernetes). Bundle format is self-contained.
- **Cons**: Keyless requires internet + OIDC provider at signing time. Public transparency log exposes signer identity. Verification can require online access (or offline with bundle). Heavier dependency than minisign/SSH. Privacy concerns with email in Rekor.
- **Nixpkgs**: cosign packaged

## Sources Index

1. Sigstore Overview: https://docs.sigstore.dev/about/overview/
2. Cosign Blob Signing: https://docs.sigstore.dev/cosign/signing/signing_with_blobs/
3. Cosign Signature Spec: https://github.com/sigstore/cosign/blob/main/specs/SIGNATURE_SPEC.md
4. Cosign Beginner Experience: https://code.mendhak.com/understanding-sigstore-cosign-as-a-beginner/
5. npm Sigstore Provenance: https://blog.sigstore.dev/npm-provenance-ga/
6. PyPI Sigstore Attestations: https://blog.sigstore.dev/pypi-attestations-ga/
7. TUF + Sigstore: https://dlorenc.medium.com/using-the-update-framework-in-sigstore-dc393cfe6b52
8. Nix GPG Verification: https://scottworley.com/blog/2022-09-20-checking-openpgp-signatures-in-nix-builds.html
9. nixpkgs GPG helpers PR: https://github.com/NixOS/nixpkgs/pull/43233/files
10. Minisign: https://jedisct1.github.io/minisign/
11. SSH File Signing: https://www.agwa.name/blog/post/ssh_signatures
12. Sigstore Graduation: https://openssf.org/blog/2024/03/20/sigstore-graduates-a-monumental-step-towards-secure-software-supply-chains/
13. TUF Website: https://theupdateframework.io/
14. python-tuf: https://github.com/theupdateframework/python-tuf
