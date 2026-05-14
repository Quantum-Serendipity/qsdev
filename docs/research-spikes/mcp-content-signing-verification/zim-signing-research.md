# ZIM Signing Research — Summary

## Current State of ZIM Integrity

ZIM files have two integrity mechanisms, both providing corruption detection only:

**In-file MD5 checksum**: The final 16 bytes of every ZIM file contain an MD5 hash covering all preceding bytes. The header field `checksumPos` (offset 72, uint64) points to where these 16 bytes begin. Creation: after all content and the header are written, libzim seeks back to offset 0, re-reads the entire file in 1024-byte chunks, computes MD5, and appends it. Verification: `Archive::check()` in libzim and the `zimcheck` CLI tool recompute the hash and compare. The `IntegrityCheck` enum also validates structural invariants (dirent ordering, cluster pointer ranges, MIME types).

**SHA-256 sidecar files**: For every ZIM published at download.kiwix.org, a `.zim.sha256` file is served at the same path (e.g., `.../wikipedia_en_all_maxi_2026-02.zim.sha256`). Standard sha256sum format. Available through the library browser and OPDS catalog metadata. Verification is manual (`sha256sum -c`) or automated by tools like kiwix-zim-updater.

**Known gap — write-time corruption** ([libzim#614](https://github.com/openzim/libzim/issues/614), open since 2021, milestone 10.0.0): Because the MD5 is computed by re-reading data already on disk, corruption during the initial write is silently checksummed as correct. Proposed fix: incremental checksumming during write. Status: stalled — a volunteer offered to implement it but has not delivered. Not on any roadmap.

Neither mechanism provides authentication. MD5 is cryptographically broken (practical collision attacks since 2004). The SHA-256 sidecars are unsigned plain text — HTTPS protects them during download but not after.

## Upstream Signing Discussion

Exactly **one issue** exists across all Kiwix and openZIM repositories: [libzim#40](https://github.com/openzim/libzim/issues/40) — "Add spec to allow content signing" — opened 2017-07-31, still open, no assignee, no milestone, empty body.

The discussion reveals the project's stance:
- **@kelson42** (project lead) acknowledges the need "at middle term" but considers it non-urgent because HTTPS secures downloads from library.kiwix.org.
- A community member raised the sneakernet scenario (sharing ZIM files offline in censored or low-connectivity regions). The project lead was unconvinced this justified the effort.
- A cross-reference in issue #614 notes that signing would require linking to OpenSSL, connecting the two issues architecturally.

No RFC, specification draft, implementation branch, or concrete proposal exists. The issue has been open for nearly 9 years. Searching all openzim/* and kiwix/* repositories for "signing", "signature", "GPG", "sigstore", "cosign", and "authenticity" returned no other results. Content signing is not on any published Kiwix roadmap.

## Comparison to Other Offline Formats

**WARC** (ISO 28500): No built-in signing. The third-party [warcsigner](https://github.com/ikreymer/warcsigner) tool stores RSA signatures in gzip extension fields, but the project is abandoned (14 commits, no releases).

**WACZ** (Web Archive Collection Zipped): The most mature approach. The [WACZ Auth Specification](https://specs.webrecorder.net/wacz-auth/0.1.0/) by Webrecorder defines a hash chain from individual WARC records through `datapackage.json` to a signed `datapackage-digest.json`. Supports two signature types: anonymous (ECDSA + embedded public key) and domain-ownership (ECDSA + TLS certificate + RFC 3161 timestamp). Implemented in ReplayWeb.page and js-wacz. This is the gold standard for signed offline archives, but its ZIP-based architecture doesn't transfer directly to ZIM's single-file format.

**Signed Exchanges (SXG)**: Signs HTTP exchanges using special TLS certificates with 90-day validity. Designed for CDN prefetch, not archival. Chromium-only. Not applicable.

**MHTML**: No signing. Legacy format with no active development.

## Where a Signature Could Architecturally Fit in ZIM

Four options, in order of pragmatism:

1. **Detached sidecar** (`.zim.sig`): Sign the file's SHA-256 hash externally. Zero format changes. Works with existing files. Sidecar can be lost during transfer, but gdev controls distribution.

2. **In-archive metadata entry**: Store signature as a ZIM directory entry (e.g., `M/signature`, MIME `application/x-zim-signature`). Self-contained but creates a chicken-and-egg problem — the signature must exclude itself from what it signs, requiring careful definition of the signed content boundary.

3. **Extended footer**: Add signature data between the last cluster and the MD5 checksum, with a new header field pointing to it. Cleaner than option 2 but requires a ZIM format version bump and libzim changes.

4. **Hybrid manifest + detached signature**: Store a per-cluster hash manifest inside the archive, sign only the manifest hash externally. Enables partial verification but is the most complex option.

## Assessment: Is Upstream Signing Likely?

**No.** The evidence is unambiguous:
- Single issue, open 9 years, no assignee, no milestone, no spec, no implementation
- Project lead's mental model is HTTPS-centric; offline provenance is not a priority
- The project's bandwidth is consumed by format evolution (v5->v6), compression upgrades (LZMA2->Zstd), and scraper maintenance
- The write-corruption fix (a simpler problem) has also stalled for 4+ years

**gdev must solve this independently.** The recommended approach is a detached sidecar signature (option 1) implemented entirely in the gdev/MCP layer, using modern cryptography (Ed25519 or ECDSA P-256 with SHA-256). The WACZ Auth spec provides a well-designed reference architecture for the signing metadata format, particularly its domain-ownership model and RFC 3161 timestamping. These concepts can be adapted to sign ZIM files without requiring any upstream changes.
