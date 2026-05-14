# DevDocs Content Signing — Research Summary

## Current State of DevDocs Integrity

DevDocs has **zero integrity mechanisms** at every layer of its pipeline:

**Scraper pipeline**: The Ruby scraper fetches upstream HTML via Typhoeus, processes it through Nokogiri/HTML::Pipeline filter chains (cleaning, normalizing, syntax highlighting), and outputs normalized HTML partials plus JSON index/data files. The `PageDb` class is an in-memory hash with no checksumming. The manifest (`docs.json`) is plain JSON metadata — name, version, update date — with no hashes or signatures.

**Distribution**: The Sinatra web app serves content over HTTPS with Content-Security-Policy headers, but applies no Subresource Integrity (SRI) attributes to assets. Sprockets digest-based filenames provide cache busting, not integrity verification. Offline download stores content in IndexedDB via XHR, with a `checkForCorruptedDocs()` function that only detects structural corruption (missing index entries, orphaned refs) — not content tampering.

**Update checking**: Polls the application script URL; a 404 triggers update notification. No hash comparison, no signature verification, no version metadata validation.

**Desktop app**: `devdocs-desktop` (egoist/devdocs-desktop) is an unsigned Electron wrapper around the web app. It adds no integrity layer; the Mac build is explicitly not code-signed.

**Gemfile**: No cryptographic dependencies beyond Ruby's stdlib. The only security-related gem is `rack-ssl-enforcer` for HTTPS enforcement.

## Upstream Discussion of Signing

Effectively none. A search of the freeCodeCamp/devdocs issue tracker for terms including "signing", "signature", "integrity", "checksum", "hash", "verify", "security", and "tamper" returned exactly one relevant result:

**Issue #1113** (opened October 2019, still open): Requests SRI `integrity` attributes on CSS/JS assets served via CDN. This is about the app's own asset integrity, not documentation content. The fix is trivial (Sprockets already supports it), yet the issue has been open for over 6 years with no assignee, no linked PRs, and no activity. This signals low upstream priority for integrity concerns.

No issues exist requesting documentation content signing, hash verification, or tamper detection. The project has never discussed content provenance or supply-chain security for the documentation it aggregates.

## Comparison to Other Documentation Aggregators

Every major documentation aggregator follows the same pattern — HTTPS transport is the sole integrity mechanism:

- **Dash** (macOS, Kapeli): Docset feed XML contains version and mirror URLs but no checksums or signatures. The iOS downloader (`DHDocsetDownloader.m`) uses standard NSURLConnection without certificate pinning or hash verification. HTTP URLs are upgraded to HTTPS, but that is the extent of security.
- **Zeal** (cross-platform, Qt): Uses Dash-compatible docset format and feeds. No additional integrity layer. No documented verification mechanisms.
- **Velocity** (Windows): Same docset format, no documented integrity mechanisms.

No documentation aggregator in the ecosystem implements content signing, hash verification, or any form of authenticated distribution. This is an industry-wide gap, not a DevDocs-specific oversight.

## Feasibility Assessment for Adding Signing

### What could be signed?

DevDocs content is well-structured for signing. Each documentation set consists of:
1. A JSON index file (page metadata, search entries)
2. A JSON data file (full content of all pages, concatenated)
3. An entry in the `docs.json` manifest

Three practical approaches:

**Per-docset manifest with hashes** (recommended): Generate SHA-256 hashes of the index and data JSON files for each documentation version. Publish a signed manifest mapping `{slug, version, index_hash, data_hash}`. Consumers verify content against the manifest, then verify the manifest signature. This mirrors how package managers (npm, PyPI) handle content integrity.

**Bundle signing**: Sign the tarball/archive of each docset as a blob using cosign. Simple but requires downloading the full archive before verification — incompatible with DevDocs' per-page XHR loading pattern.

**Per-page hashing**: Hash each HTML partial individually. Provides granular verification but explodes the manifest size (hundreds of entries per doc) and adds verification overhead on every page load.

### Who would sign?

- **freeCodeCamp as publisher**: Most practical. They control the scraper pipeline and could sign at build time. This attests "freeCodeCamp's scraper produced this content from upstream source X at time T" — a provenance claim, not an authenticity claim about the upstream docs themselves.
- **Original documentation authors**: Ideal but impractical. Would require every upstream project (MDN, Node.js, Python, etc.) to sign their documentation, and DevDocs to verify those signatures survive the scraping/normalization pipeline. The transformation pipeline fundamentally changes the content, breaking any upstream signature.
- **gdev as consumer-side verifier**: gdev could pin known-good hashes after initial verification (similar to Nix's approach), creating a local trust-on-first-use model without requiring upstream changes.

### Update cadence impact

DevDocs documentation updates vary from weekly (fast-moving projects) to monthly. Docker images rebuild monthly. A signing workflow would need to:
- Run as a post-scrape step in the build pipeline
- Generate hashes of all output files
- Sign the hash manifest (ideally keyless via Sigstore for CI integration)
- Publish the signed manifest alongside the documentation

This fits naturally into the existing `thor` task pipeline and could be automated in CI.

## Most Practical Path for gdev

Given that upstream signing adoption is unlikely (6+ years of inaction on even SRI for assets), gdev should implement **consumer-side integrity verification**:

1. **Hash-on-first-download**: When gdev first downloads a DevDocs docset, compute SHA-256 hashes of the index and data files. Store these in a local manifest alongside the content.

2. **Pin in Nix derivation**: For docsets bundled into the gdev Nix package, pin the expected hashes in the derivation. Nix's fixed-output derivation mechanism already provides this — the content hash is verified at build time and any change breaks the build.

3. **Detect drift on update**: When checking for documentation updates, compare new content hashes against pinned values. Flag unexpected changes rather than silently accepting them.

4. **Optional: Publish a community manifest**: gdev could publish a signed manifest of known-good DevDocs content hashes, allowing other consumers to verify content independently. This creates a third-party attestation layer without requiring upstream changes.

This approach mirrors how Nix handles integrity for all fetched content — trust the hash, not the channel — and requires zero upstream cooperation. It also applies identically to Dash/Zeal docsets, making it a universal solution for the documentation aggregator ecosystem's integrity gap.
