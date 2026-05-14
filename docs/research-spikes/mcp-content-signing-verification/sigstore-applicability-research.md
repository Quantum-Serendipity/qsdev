# Sigstore Applicability for gdev Content Signing

## How Sigstore Works for Non-Container Artifacts

Sigstore's `cosign sign-blob` command signs arbitrary files -- not just containers. The process signs the SHA-256 digest of the file (never uploading the file itself), making it feasible for large artifacts like 50+ GB ZIM files. The signing produces a small (~2-5 KB) `.sigstore.json` bundle containing the signature, a Fulcio-issued certificate binding the signer's identity, and a Rekor transparency log inclusion proof.

**Keyless signing** (the flagship mode) works through OIDC: cosign generates an ephemeral key pair, authenticates the signer via Google/GitHub/Microsoft, gets a short-lived certificate from Fulcio, signs the digest, logs everything to Rekor, then discards the private key. No long-lived keys to manage. In CI/CD environments (GitHub Actions), this happens automatically via ambient OIDC tokens.

**Key-based signing** is also supported: static ECDSA-P256 keys, KMS-backed keys (AWS/GCP/Azure/Vault), or hardware tokens. This mode still optionally logs to Rekor for auditability.

**Verification** requires the bundle file and the original artifact. The verifier checks: (1) signature matches the artifact's digest, (2) certificate was issued by Fulcio and matches expected identity/issuer, (3) Rekor inclusion proof is valid. Offline verification is possible when the bundle contains the inclusion proof -- no network call to Rekor needed.

## Precedent in Other Ecosystems

Sigstore has strong adoption momentum for non-container artifacts:

- **npm** (GA Sept 2023): All packages published from GitHub Actions get Sigstore provenance attestations by default. Integrated directly into the npm CLI. 500M+ downloads of provenance-enabled packages during beta.
- **PyPI** (GA Nov 2024): PEP 740 standard for Sigstore attestations. Zero-config for projects using GitHub Actions + Trusted Publishing. 20,000+ attestations uploaded.
- **Kubernetes**: All release artifacts (binaries, SBOMs, images) signed with cosign since v1.24.
- **GitHub**: Artifact attestation feature built on Sigstore for any Actions workflow.

No Linux distribution has adopted Sigstore for package signing -- Fedora/Arch/Debian still use GPG. No known precedent exists for signing documentation or content packages (ZIM files, DevDocs bundles) with any tool. This would be novel usage.

In the Nix ecosystem, there is no Sigstore integration. The closest precedent is nixpkgs PR #43233, which provides helpers for GPG signature verification during builds using `fetchurl` + `runCommand` + `gnupg`.

## Nix Integration Feasibility

Both `cosign` and `minisign` are packaged in nixpkgs and available for NixOS. Integration has two viable paths:

**Build-time verification** (in a Nix derivation): Fetch the artifact and its signature bundle via `fetchurl`, then verify in a `runCommand` step before linking the output. This follows the existing GPG verification pattern. Challenge: the Nix sandbox blocks network access during builds, so Rekor online verification would fail. Offline verification with the self-contained bundle should work but needs testing. The Nix SRI hash already pins exact content, so signing adds provenance attestation (who signed it and when) rather than additional integrity protection.

**Runtime verification** (in gdev): Run verification as a post-download health check before serving content via MCP. This avoids sandbox issues and is architecturally simpler. cosign would need to be in gdev's runtime closure.

For gdev-signed content (wrapping upstream ZIM/DevDocs), keyless OIDC signing in GitHub Actions CI is the natural fit -- it requires no key management and produces strong provenance attestations automatically.

## Comparison of Signing Approaches

| Criterion | Sigstore/Cosign | Minisign | GPG/PGP | SSH Signing | TUF |
|-----------|----------------|----------|---------|-------------|-----|
| **Setup complexity** | Medium (OIDC flow or key gen) | Very low (one command) | High (keyring, WoT) | Very low (existing keys) | High (role hierarchy) |
| **Key management** | None (keyless) or static/KMS | Manual static key | Manual, complex rotation | Manual, GitHub-distributed | Built-in rotation/delegation |
| **Transparency log** | Yes (Rekor) | No | No | No | No (but auditable metadata) |
| **Identity binding** | Yes (OIDC email/workflow) | No | Yes (but WoT-dependent) | Yes (allowed_signers) | Yes (role-based) |
| **Offline verification** | Yes (with bundle) | Yes | Yes | Yes | Yes |
| **Large file support** | Yes (signs digest) | Yes (pre-hash mode) | Yes (signs digest) | Yes (signs digest) | Yes (hashes in metadata) |
| **Nix availability** | cosign in nixpkgs | minisign in nixpkgs | gnupg in nixpkgs | openssh always present | python-tuf via pip only |
| **Binary size** | ~70 MB (Go binary) | ~200 KB | ~10 MB | Already installed | N/A (framework) |
| **Ecosystem momentum** | Strong (npm, PyPI, K8s) | Niche (WireGuard, VyOS) | Declining for new projects | Growing (git signing) | Strong (PyPI, Docker) |
| **Privacy** | Email visible in Rekor | N/A | Depends on keyserver use | N/A | N/A |

## Recommendation for gdev

**Primary: Minisign for content package signing.**

For gdev's specific needs -- signing ZIM files and DevDocs bundles that gdev packages and distributes -- minisign is the best fit. The reasoning:

1. **Simplicity matches the threat model.** gdev is a single-organization tool. The signer is always the gdev CI pipeline or maintainer. There is no need for decentralized identity verification or transparency logging. The question is simply "did this content come from the gdev project?"

2. **Minimal dependency footprint.** Minisign is a ~200 KB binary versus cosign's ~70 MB. In a Nix closure where every dependency matters, this is significant. Minisign has zero runtime dependencies.

3. **Public key embeds trivially.** A minisign public key is a single short base64 string that can be hardcoded in gdev's Nix configuration. No key distribution infrastructure needed. Compare to GPG (keyring import) or Sigstore (OIDC issuer configuration).

4. **Offline-native.** No network calls during verification. No dependency on external infrastructure (Fulcio, Rekor). Works in air-gapped environments and inside the Nix sandbox without special handling.

5. **Pre-hashing handles large files.** Minisign's `-H` flag uses Blake2b-512 pre-hashing, making 50+ GB ZIM files practical.

**Secondary: Cosign keyless signing in CI for provenance attestation.**

If gdev later needs to prove *when* content was signed and provide a public audit trail (e.g., for compliance or multi-organization trust), add cosign keyless signing in GitHub Actions as a supplementary layer. The `.sigstore.json` bundle would be distributed alongside the minisign `.minisig` signature. Consumers could verify either or both.

**Not recommended:**
- **GPG**: Worse UX, declining ecosystem, complex key management -- no advantage over minisign for this use case.
- **TUF**: Designed for large-scale software distribution with multiple signing roles and key compromise recovery. Massively over-engineered for gdev's single-signer content distribution. Would be appropriate if gdev became a multi-team platform with delegated signing authority.
- **SSH signing**: Viable but awkward. No standard metadata format for artifact signing. Allowed_signers file distribution is manual. Solves a problem gdev doesn't have (leveraging existing SSH keys).

**Implementation sketch:**

1. Generate a minisign keypair for the gdev project. Store the secret key in CI secrets.
2. In the gdev release pipeline, sign each content artifact: `minisign -Sm wikipedia.zim -t "gdev-content v2026.05 sha256:abc123"`
3. Distribute `.minisig` files alongside content artifacts (same URL path with `.minisig` suffix).
4. Embed the minisign public key in gdev's Nix configuration (a single string).
5. At download/update time, gdev runs `minisign -Vm file.zim -P <embedded-pubkey>` before accepting content.
6. Nix SRI hashes continue to provide integrity pinning; minisign adds provenance verification.
