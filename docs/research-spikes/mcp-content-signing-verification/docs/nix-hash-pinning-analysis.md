# Nix Hash Pinning: Mechanisms, Gaps, and Signature Verification Precedent

- **Author**: Sub-agent research for P1-T4
- **Date**: 2026-05-14
- **Scope**: What Nix hashes protect, what they don't, and nixpkgs precedent for layering cryptographic signature verification on top

## 1. Nix Integrity Mechanisms

### 1.1 SRI Hashes in fetchurl/fetchzip

Every `fetchurl` call in nixpkgs requires a cryptographic hash of the expected output, specified in SRI (Subresource Integrity) format: `sha256-<base64>`. The hash resolution priority is: SRI hash > explicit `outputHash`/`outputHashAlgo` > legacy `sha256`/`sha512` parameters. If no hash is provided, the derivation fails with an error.

**When verification happens**: After the download completes and any `postFetch` hooks run, Nix computes the hash of the output file (or NAR serialization for recursive hashes) and compares it to the declared hash. A mismatch fails the build. This happens at the Nix store level, not in the builder script -- the builder.sh for fetchurl delegates hash checking entirely to the Nix daemon.

**What it verifies**: The downloaded content is bit-for-bit identical to what was recorded when the hash was first captured. Any modification -- corruption, MITM, mirror substitution -- produces a hash mismatch.

**What it does NOT verify**: Who produced the content, whether the content is authentic (from the claimed author), or whether the recorded hash was correct in the first place.

### 1.2 Fixed-Output Derivations: The Trust Model

Fixed-output derivations (FODs) are the mechanism underlying fetchurl. They represent a "carefully controlled impurity" in Nix's otherwise deterministic build model:

- **Normal derivations**: Fully sandboxed, no network access. Output path derived from all inputs (source hash, build script, dependencies).
- **Fixed-output derivations**: Granted network access, but must declare the expected output hash in advance. Output path derived from the hash alone, not from the URL or build process.

The security property: "Regardless of what the builder does during the build, it cannot influence downstream builds in unanticipated ways because all information it passed downstream flows through the outputs whose content-addresses are fixed." (Nix Reference Manual)

**The hash IS the trust anchor.** There is no secondary verification. If the hash is correct, the content is accepted. If the hash is wrong (because the person who recorded it was deceived), the wrong content is accepted.

### 1.3 Binary Cache Signatures

Nix binary caches add a separate signing layer for pre-built store paths:

- **Key type**: Ed25519 keypairs, generated via `nix-store --generate-binary-cache-key`
- **What gets signed**: A fingerprint of format `1;<StorePath>;<NarHash>;<NarSize>;<refs>`, signed with Ed25519 and included as a `Sig:` line in `.narinfo` files
- **Verification**: Nix checks narinfo signatures against `trusted-public-keys` in nix.conf before accepting substitutions
- **Granularity**: Trust is per-cache, not per-package. Trusting a cache key means trusting everything it serves.

This protects against MITM on binary cache downloads and unauthorized store manipulation. It does NOT protect against a compromised cache operator or a compromised signing key.

### 1.4 Content-Addressable Store Model

In the content-addressed store, paths are derived from content:
- Identical files get identical store paths regardless of origin
- Changing a URL doesn't trigger rebuilds if content is unchanged
- The path encodes WHAT the content is, not WHERE it came from

This is powerful for deduplication and reproducibility but provides no provenance information.

## 2. Gaps That Signing Would Close

### 2.1 First-Download Trust (TOFU)

When a nixpkgs maintainer first packages a piece of software:
1. They set the hash to `lib.fakeHash` (a known-wrong placeholder)
2. They run the build, which downloads the content and fails with the real hash
3. They copy the real hash into the derivation

This is Trust On First Use (TOFU). The maintainer trusts whatever the URL served at that moment. If the server was compromised, if DNS was hijacked, or if a CDN served modified content, the wrong hash gets recorded permanently.

**What signing closes**: If the content is signed by the upstream author, the maintainer (or an automated tool) can verify the signature before recording the hash. The hash then inherits trust from the signature verification.

### 2.2 Hash Update Trust

When upstream releases a new version:
1. An automated bot (e.g., nixpkgs-update) or a maintainer updates the URL and hash
2. The new hash is typically obtained the same TOFU way -- run with fakeHash, get real hash, update
3. A reviewer may check the diff, but rarely downloads and independently verifies the content

**What signing closes**: If the update pipeline verifies the upstream signature on new content before recording the new hash, each hash update carries the same trust as the initial recording. Without signing, each update is a fresh TOFU event.

### 2.3 Mirror Trust

Nix supports hashed mirrors (content addressed by hash). When fetching from a mirror:
- The content is verified against the declared hash -- so a corrupted mirror is caught
- BUT: this only works if the hash itself is correct

If both the hash in nixpkgs AND the mirror content are compromised (e.g., via a coordinated attack on the nixpkgs repo and a mirror), the hash catches nothing because it's verifying against the wrong expected value.

**What signing closes**: An independent signature check provides a second trust root that doesn't depend on the hash being correct. Even if the hash was recorded from compromised content, a signature from the legitimate author would fail to verify.

### 2.4 Supply Chain Compromise

If a nixpkgs commit is compromised (malicious contributor, compromised maintainer account, or CI injection):
- The hash in the commit is wrong -- it matches the attacker's payload
- Every downstream consumer faithfully verifies against the wrong hash
- This is undetectable by Nix's integrity mechanisms alone

**What signing closes**: If signature verification is part of the build or update process, a compromised hash alone isn't sufficient -- the attacker also needs to forge the upstream signature or replace the trusted public key. This raises the attack bar significantly.

## 3. Nixpkgs Precedent for Signature Verification

### 3.1 PR #43233: fetchpgpkey and verifySignatureHook (CLOSED, NOT MERGED)

The most significant attempt was [PR #43233](https://github.com/NixOS/nixpkgs/pull/43233) (2018, closed 2020), which proposed:

- **`fetchpgpkey`**: A fetchurl extension that downloads a PGP public key, verifies its fingerprint, and returns it for use in signature verification. Required both `sha256` (content hash) and `fingerprint` (key identity).
- **`verifySignatureHook`**: A setup hook that imports a public key and verifies a detached signature against the source before unpacking. Functions: `_importPublicKey()`, `verifySignature()`, `verifySrcSignature()`.
- **Example packages**: 1password, tor-browser-bundle-bin, samba4.

**Why it was rejected**:
1. Security concern: placing verification inside `fetchurl` could enable trojan-horse attacks
2. Philosophical disagreement: Mic92 argued "we should make this part of our tooling rather than the build process. Once we have our own checksum we no longer need to rely on public key cryptography"
3. Practical concerns: handling expired keys, key rotation over time

### 3.2 Scott Worley's Pattern: Standalone Verification Derivation

[Blog post](https://scottworley.com/blog/2022-09-20-checking-openpgp-signatures-in-nix-builds.html) demonstrating GPG verification without any nixpkgs infrastructure:

1. Fetch the signing key with `fetchurl` (hash-pinned)
2. Fetch the artifact with `fetchurl` (hash-pinned)
3. Fetch the detached signature with `fetchurl` (hash-pinned)
4. Create a `runCommand` derivation that imports the key, runs `gpg --verify`, and symlinks the verified output

This works TODAY with no nixpkgs changes. The key, artifact, and signature are all individually hash-pinned, and the verification derivation runs GPG as a standard build step. If the signature doesn't verify, the build fails.

**Key insight**: The signing key hash rarely changes (long-lived keys), so version updates only require updating the artifact hash and signature hash. The key itself is a stable trust anchor.

### 3.3 Discourse Discussion: Community Philosophy

The [NixOS Discourse discussion](https://discourse.nixos.org/t/any-interest-in-checkings-signatures-while-building-packages/8918) on signature verification revealed the prevailing nixpkgs philosophy:

- Hash pinning IS the security mechanism for nixpkgs
- Signature verification, if done, should happen in the update/review pipeline (tooling), not at build time
- The complexity of per-package key management is seen as too high for the benefit
- No RFC was ever filed; the topic remains unresolved

### 3.4 Sigstore/Cosign in Nixpkgs

**No precedent found.** Web searches found no nixpkgs packages using Sigstore/cosign for verification. Sigstore adoption has been strong in container ecosystems (npm, PyPI, Maven Central, Homebrew as of 2024-2025), but Nix's content-addressed model with hash pinning has meant there's been no push to adopt Sigstore within nixpkgs itself.

### 3.5 trusted-public-keys for Binary Caches

This is the only widely-deployed signature verification in the Nix ecosystem. It uses Ed25519 keys to sign narinfo metadata for pre-built store paths. However, it operates at the binary cache level (trust a cache operator), not at the source artifact level (trust an upstream author). It's not applicable for verifying upstream content like ZIM files or DevDocs data.

## 4. Options for Layering Signing on Nix in gdev

### Option A: Verify Signature at Nix Fetch Time (Custom Fetcher or postFetch Hook)

**How it works**: Write a custom Nix fetcher (e.g., `fetchVerifiedUrl`) that downloads the artifact, downloads the detached signature, imports the expected public key, and runs `gpg --verify` or `cosign verify` before producing the output. Alternatively, use fetchurl's `postFetch` hook for verification. The derivation fails if verification fails.

**Advantages**:
- Verification is embedded in the Nix build -- it's reproducible and auditable
- Cannot produce a store path with unverified content
- Follows the Scott Worley pattern (proven to work)
- No runtime dependency on verification infrastructure

**Disadvantages**:
- Adds gnupg or cosign as a build dependency
- Signature file must be fetchable at build time (adds a URL dependency)
- postFetch is skipped for hashed mirror fetches (verification only runs on direct downloads)
- If upstream doesn't provide signatures, this option is unavailable
- Key management complexity: key rotation, expiry, revocation must be handled

**gdev applicability**: Moderate. Works well for ZIM files IF Kiwix starts providing GPG/cosign signatures. Does not work for DevDocs (no signatures exist). Requires per-content-source key management.

### Option B: Verify Signature at gdev Runtime (After Nix Places Content)

**How it works**: After `nix build` produces the content in the store, gdev's runtime (e.g., the MCP server or a health-check module) verifies signatures before serving content. Signatures and public keys are distributed alongside or embedded in the content packages.

**Advantages**:
- Decoupled from Nix build -- works even if content is already in the store
- Can verify at query time, providing per-request trust signals
- Can be added incrementally without changing Nix packaging
- Can provide richer metadata to MCP consumers (verification status, signer identity)

**Disadvantages**:
- Content is already in the store before verification -- a failed check means the content exists but shouldn't be trusted
- Runtime verification adds latency to MCP responses
- Must manage signature distribution separately from content
- The Nix store path gives no indication of verification status

**gdev applicability**: High for the MCP provenance use case (P1-T6). The MCP server can tag responses with verification metadata. Low for preventing bad content from entering the system in the first place.

### Option C: Verify Signature in CI/Update Pipeline, Then Pin Hash (Human-in-the-Loop)

**How it works**: When content is updated (new ZIM file, new DevDocs scrape), a CI pipeline or update script:
1. Downloads the new content
2. Downloads and verifies the signature (if available)
3. Records the verified hash in a Nix expression
4. A human reviews the update PR (seeing that signature verification passed)
5. After merge, all consumers get the verified hash via normal Nix mechanisms

**Advantages**:
- Aligns with nixpkgs philosophy ("make this part of our tooling rather than the build process")
- Hash pinning does the heavy lifting at build/deploy time -- zero runtime overhead
- Human review provides an additional trust check
- Works even when upstream doesn't provide signatures (human manually verifies)
- No additional build or runtime dependencies

**Disadvantages**:
- Verification happens once (at update time), not continuously
- Trust depends on the CI pipeline and reviewer not being compromised
- If the update pipeline is automated without human review, it's just TOFU with extra steps
- No verification signal reaches the MCP consumer

**gdev applicability**: High. This is the most practical option for gdev because:
- gdev already packages content through Nix (hash pinning is already in place)
- Content updates are infrequent (ZIM files update monthly at most)
- A CI-based verification step fits naturally into gdev's update workflow
- The human-in-the-loop review provides the provenance guarantee that hash pinning alone lacks

### Comparison Matrix

| Aspect | Option A (Fetch-time) | Option B (Runtime) | Option C (CI Pipeline) |
|--------|----------------------|--------------------|-----------------------|
| When verification runs | At nix build | At MCP query time | At content update time |
| Bad content enters store? | No | Yes (flagged later) | No |
| Build dependency added? | Yes (gnupg/cosign) | No | No (CI only) |
| Runtime overhead | None | Per-query | None |
| Upstream signatures needed? | Yes | Yes | Preferred but optional |
| MCP provenance metadata? | No | Yes | No |
| Nixpkgs precedent? | Scott Worley pattern | None | Matches nixpkgs philosophy |
| Implementation complexity | Medium | Medium-High | Low |

## 5. Recommendation

**Option C (CI pipeline verification) as the primary approach, with Option B (runtime metadata) as an enhancement.**

Rationale:
1. Option C matches the nixpkgs community's established philosophy that signature verification belongs in tooling, not builds
2. gdev's content update cadence (monthly ZIM updates, periodic DevDocs scrapes) makes pipeline-time verification practical
3. Hash pinning already provides the integrity guarantee at build/deploy time -- signing closes the TOFU gap at the one point where it matters (when the hash is recorded)
4. Option B can be layered on later to provide MCP-level provenance metadata, giving Claude Code visibility into content trust status
5. Option A is viable but adds unnecessary build complexity for a problem that's better solved upstream in the pipeline

The key insight from the nixpkgs precedent is that the Nix community tried and rejected build-time signature verification (PR #43233) in favor of keeping verification in tooling. gdev should learn from this and place verification where it has the most leverage: the moment content enters the system.
