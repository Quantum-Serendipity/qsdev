# Nix Hash Pinning: Coverage, Gaps, and Signing Integration for gdev

## What Nix Hashes Protect

Every `fetchurl` in nixpkgs requires an SRI hash (e.g., `sha256-<base64>`) that Nix verifies after download. This is implemented through fixed-output derivations (FODs): builders get network access but must produce output matching the declared hash. If the hash mismatches, the build fails. The Nix daemon handles verification -- it is not bypassable by the builder script.

Binary caches add a separate layer: Ed25519 signatures on `.narinfo` metadata, verified against `trusted-public-keys` in nix.conf. This prevents MITM on cache downloads but operates at the cache-operator level, not the upstream-author level.

Together, these mechanisms guarantee: once a correct hash is recorded, all downstream consumers get bit-identical content. The content-addressed store means identical outputs get identical paths regardless of source URL, providing deduplication and reproducibility.

## What Nix Hashes Do NOT Protect

The hash is the sole trust anchor, creating four gaps:

1. **First-download trust (TOFU)**: Maintainers record hashes by building with `lib.fakeHash`, getting the real hash from the error output, and pasting it in. If the URL served malicious content at that moment, the wrong hash gets recorded permanently.

2. **Hash update trust**: Each version bump repeats the TOFU process. Automated bots like nixpkgs-update record whatever the URL serves. A compromised upstream server at update time poisons the hash.

3. **Compromised nixpkgs commits**: If a malicious commit changes the hash, all consumers faithfully verify against the attacker's hash. Nix's integrity machinery works perfectly -- against the wrong reference value.

4. **No provenance**: Hashes prove content matches what was recorded but say nothing about who produced it or whether it was the legitimate author.

Signing closes all four gaps by providing an independent trust root: even if the hash was recorded from compromised content, a forged upstream signature would fail verification.

## Nixpkgs Precedent for Signature Verification

**The Nix community has considered and rejected build-time signature verification.** The key precedents:

- **PR #43233** (2018, closed 2020 without merge): Proposed `fetchpgpkey` and `verifySignatureHook` for GPG verification during builds. Demonstrated with 1password, tor-browser-bundle-bin, and samba4. Rejected because reviewers believed verification belongs in tooling, not the build process. Mic92's summary: "Once we have our own checksum we no longer need to rely on public key cryptography."

- **Scott Worley's pattern** (2022): A standalone derivation that fetches key + artifact + signature separately (all hash-pinned), then runs `gpg --verify` in a `runCommand`. Works today with no nixpkgs changes. Proves per-artifact GPG verification is technically feasible in Nix.

- **Discourse discussion**: Community split between "signatures add defense-in-depth" and "hashes already handle this." No RFC was filed. The prevailing philosophy: hash pinning IS the security model; signature verification is tooling's job.

- **Sigstore/cosign**: No nixpkgs precedent found. Sigstore has been adopted by npm, PyPI, Maven Central, and Homebrew (2024-2025), but not by the Nix ecosystem.

- **Binary cache signing**: The only deployed signature verification in Nix. Ed25519 keys sign narinfo fingerprints (`1;<StorePath>;<NarHash>;<NarSize>;<refs>`). Trust is per-cache, not per-package. Not applicable for verifying upstream content artifacts.

## Options for Layering Signing on Nix in gdev

**Option A -- Fetch-time verification**: Custom Nix fetcher or `postFetch` hook runs `gpg --verify` / `cosign verify` during build. Prevents unverified content from entering the store. Adds gnupg/cosign as build dependencies. Follows the Scott Worley pattern. Requires upstream to provide signatures.

**Option B -- Runtime verification**: gdev's MCP server verifies signatures before serving content. Can tag responses with provenance metadata for differential trust. Content enters the store before verification (failed check = untrusted content already present). Adds per-query latency.

**Option C -- CI pipeline verification**: Update script downloads content, verifies signature, records hash. Human reviews the PR. Normal Nix hash pinning handles deployment. Zero build/runtime overhead. Aligns with nixpkgs philosophy. Works even without upstream signatures (human verification).

## Recommendation

**Option C as the primary approach, with Option B as a future enhancement.**

Option C matches the nixpkgs community's established philosophy and gdev's operational reality: content updates are infrequent (monthly ZIM files, periodic DevDocs scrapes), so pipeline-time verification is practical. The CI step closes the TOFU gap at the exact moment it matters -- when the hash is first recorded. Hash pinning then carries that trust to every consumer at zero ongoing cost.

Option B can be layered later to give MCP responses provenance metadata (verification status, signer identity), enabling Claude Code to apply differential trust. This is valuable but not blocking -- the integrity guarantee from Option C is the foundation.

Option A is technically sound but adds build complexity the nixpkgs community has already rejected as unnecessary. gdev should not diverge from Nix idiom without a compelling reason.

---

*Detailed analysis with source citations: `docs/nix-hash-pinning-analysis.md`*
*Raw sources: `docs/nix-fetchurl-default-nix-analysis.md`, `docs/nix-binary-cache-signing-mechanism.md`, `docs/nix-fixed-output-derivations-trust-model.md`, `docs/nixpkgs-pr-43233-gpg-verification-helpers.md`, `docs/nixpkgs-signature-verification-discourse.md`, `docs/openpgp-signatures-nix-builds-scottworley.md`*
