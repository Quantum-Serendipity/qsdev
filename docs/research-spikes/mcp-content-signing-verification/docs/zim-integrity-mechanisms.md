# ZIM File Integrity Mechanisms — Detailed Findings

- **Sources**: See individual citations below
- **Retrieved**: 2026-05-14
- **Scope**: ZIM in-file checksums, SHA-256 sidecars, write-corruption gap, upstream signing discussions, comparison to other offline formats, architectural options for adding signatures

---

## 1. In-File MD5 Checksum

### Location in the Format

The ZIM file header is 80 bytes. The last field is `checksumPos` (uint64 at offset 72), which stores the byte offset where the MD5 checksum begins. The checksum is always the final 16 bytes of the file. For a file of size N bytes, the checksum occupies bytes [N-16, N).

**Source**: [ZIM File Format Spec](https://docs.fileformat.com/compression/zim/) — header table shows checksumPos at offset 72, 8 bytes.

### What It Covers

The MD5 hash covers all bytes from the start of the file (offset 0) up to but not including the checksum itself — i.e., bytes [0, checksumPos). This means the header, MIME type list, directory entries, all pointer lists, all clusters (compressed content), and all metadata are included in the hash.

### How It Is Computed (Creation)

From `creator.cpp` in libzim ([source](https://github.com/openzim/libzim/blob/main/src/writer/creator.cpp)):

```cpp
// In writeLastParts(), after all content and header are written:
lseek(out_fd, 0, SEEK_SET);   // Seek to beginning
zim_MD5Init(&md5ctx);
while (true) {
   auto r = read(out_fd, batch_read, 1024);  // Read 1024-byte chunks
   if (r == 0) break;
   zim_MD5Update(&md5ctx, batch_read, r);
}
zim_MD5Final(digest, &md5ctx);
_write(out_fd, digest, 16);   // Append 16-byte MD5 to end
```

The process is:
1. Write all ZIM content (clusters, directory entries, pointers)
2. Write the header (some fields like checksumPos are only known after content is written)
3. Seek back to offset 0 and re-read the entire file
4. Compute MD5 over all bytes read
5. Append the 16-byte digest to the file

Uses a custom `zim_MD5` implementation (not OpenSSL). The 1024-byte read buffer is noted as inefficient by contributors.

### How It Is Verified (Reading)

From `fileimpl.cpp` in libzim ([source](https://github.com/openzim/libzim/blob/main/src/fileimpl.cpp)):

```cpp
bool FileImpl::verify() {
  // Read all bytes [0, checksumPos) in chunks
  // Compute MD5
  // Compare against stored 16 bytes at checksumPos
  return (computed == stored);
}
```

Supports multi-part ZIM files (split across physical files). This is exposed via `Archive::check()` in the public API and called by `zimcheck`.

### Limitations of the MD5 Checksum

1. **MD5 is cryptographically broken** — collision attacks are practical since 2004 (Wang et al.). MD5 provides corruption detection but zero resistance to intentional tampering.
2. **No authentication** — the checksum proves the file is internally consistent, not who created it or whether the content is what it claims to be.
3. **Write-corruption gap** (issue #614) — see section 3 below.

---

## 2. SHA-256 Sidecar Files at download.kiwix.org

### URL Pattern

For each ZIM file at `https://download.kiwix.org/zim/<category>/<filename>.zim`, a corresponding SHA-256 checksum is available at:
```
https://download.kiwix.org/zim/<category>/<filename>.zim.sha256
```

**Source**: [kiwix-zim-updater](https://github.com/jojo2357/kiwix-zim-updater/blob/main/kiwix-zim-updater.sh)

### File Format

Standard sha256sum format:
```
<64-char-hex-hash>  <filename.zim>
```

### Discovery

- Available through [browse.library.kiwix.org](https://browse.library.kiwix.org) alongside download links and torrent links
- Also embedded in Kiwix catalog metadata as `<hash type="sha-256">` XML elements (OPDS feed)
- The .sha256 files are **not listed** in directory indexes at download.kiwix.org — they are served by the web server but hidden from browsing

### User Verification

Standard command-line verification:
```bash
# Linux
sha256sum -c filename.zim.sha256

# macOS
shasum -a 256 -c filename.zim.sha256

# Windows PowerShell
Get-FileHash -Algorithm SHA256 -Path .\filename.zim
```

### Automation Support

The kiwix-zim-updater script automates verification:
```bash
cd "$ZIMPath" && sha256sum --status -c "$NewZIM.sha256"
```

kiwix-android issue [#4466](https://github.com/kiwix/kiwix-android/issues/4466) (closed) built a user-facing integrity checker using libzim's `Archive::check()`.

### Limitations

- **Transit integrity only** — SHA-256 sidecars verify the download matches what the server has. They do not prove who created the content.
- **No signatures** — the .sha256 files are unsigned plain text. HTTPS provides channel integrity during download, but the checksum file itself has no authentication after download.
- **Sneakernet gap** — when ZIM files are shared offline (USB drives, local networks, mesh), neither the in-file MD5 nor the sidecar SHA-256 proves provenance.

---

## 3. libzim Issue #614 — Write-Corruption Gap

**Source**: [openzim/libzim#614](https://github.com/openzim/libzim/issues/614) (opened 2021-08-22, still open, milestone 10.0.0)

### The Problem

The ZIM creator writes all content to disk first, then seeks back to position 0 and re-reads the entire file to compute the MD5 checksum. Any data corruption occurring during the initial write (disk errors, filesystem bugs, interrupted I/O) is silently incorporated into the checksum. The checksum then "validates" the corrupted file as correct.

### Performance Impact

Re-reading the entire file nearly doubles I/O time for large ZIM files (which can be 100+ GB for full Wikipedia).

### Proposed Solutions

1. **Incremental checksumming** — compute the checksum as data is written, chunk by chunk. Challenge: the header is written non-linearly (some fields only known after all content is written).
2. **Cluster-level checksumming** — hash clusters as they are written, then fold in header data afterward. Drawback: breaks backward compatibility because the checksum would no longer equal `md5(file[0:checksumPos])`.
3. **Faster hash algorithm** — contributor suggested xxHash or OpenSSL's MD5 (3x faster than libzim's custom implementation). This doesn't fix the integrity gap but reduces the performance cost.

### Current Status

- Issue remains open (last activity 2024-11)
- A volunteer (@juuz0) offered to implement a fix but appears stalled
- Maintainer @kelson42 noted it's "not on any roadmap" but welcomed volunteer contributions
- A comment from @veloman-yunkan noted that the bottleneck is CPU (MD5 computation), not I/O — custom MD5 takes 14-16s vs 3s for OpenSSL's implementation on the same data

### Cross-Reference to Signing

In the discussion, @veloman-yunkan noted: "if at some point we need to sign content, see #40, linking to OpenSSL will be necessary." This is the only concrete connection between the checksum improvement and signing in the codebase.

---

## 4. Upstream Signing Discussions

### libzim Issue #40 — "Add spec to allow content signing"

**Source**: [openzim/libzim#40](https://github.com/openzim/libzim/issues/40) (opened 2017-07-31, still open)

This is the **only** issue in any Kiwix or openZIM repository that addresses content signing. Key points from the discussion:

- **@kelson42** (Kiwix project lead): "this is not urgent, but at middle term I suppose we should have a solution to sign digitally the ZIM files."
- **@mofosyne** proposed a stepping-stone: embed hashes of well-known ZIM files into the libzim binary itself, providing a trust anchor tied to the software distribution. This was rejected by @kelson42 as "a very imperfect hack not adding much value" because HTTPS already secures downloads.
- **@mofosyne** rebutted with the **sneakernet use case**: "what if they were sharing in an offline manner via sneakernet? Downloading such a large file like library.kiwix.org is not always a certainty (e.g. blocked by government, very slow or expensive internet in a developing country)."

### GitHub-Wide Search Results

Searching across all openzim/* and kiwix/* repositories for "signing", "signature", "GPG", "sigstore", "cosign", and "authenticity" returned **only issue #40** as relevant. No other issues, PRs, RFCs, or proposals exist.

### No Roadmap Presence

- Issue #40 has been open for nearly 9 years with no assignee
- No milestone assigned
- No linked PRs or implementation branches
- The issue body is empty (title only)
- The most recent substantive comment is from 2020 (@mofosyne's sneakernet argument)

### Assessment

The Kiwix project acknowledges the need for signing but treats it as low priority. The maintainer's mental model is HTTPS-centric — if you download from library.kiwix.org, HTTPS provides authenticity. The offline/sneakernet scenario (which is exactly gdev's use case for MCP-served documentation) is acknowledged but not prioritized.

---

## 5. Comparison to Other Offline Content Formats

### WARC (Web ARChive) — ISO 28500

**Signing**: The [warcsigner](https://github.com/ikreymer/warcsigner) tool (Python, RSA) stores signatures in extra gzip chunks using custom fields. Technically clever but the project is effectively abandoned (14 commits, no releases, unclear maintenance).

**Assessment**: WARC itself (ISO standard) has no built-in signing. warcsigner is a third-party hack using gzip metadata extension points. Not a production-grade solution.

### WACZ (Web Archive Collection Zipped)

**Signing**: The most mature solution in the offline-content space. The [WACZ Auth Specification](https://specs.webrecorder.net/wacz-auth/0.1.0/) (by Webrecorder) defines:

- **Hash chain**: Individual WARC records hashed -> indexed in CDXJ -> hashed into `datapackage.json` -> hashed into `datapackage-digest.json`
- **Two signature types**:
  1. **Anonymous**: ECDSA signature with embedded public key (external key validation required)
  2. **Domain-ownership**: ECDSA signature + domain TLS certificate (proves the signer controls a domain) + RFC 3161 timestamp
- **Algorithms**: ECDSA (P-256), SHA-256, RFC 3161 timestamps
- **Verification tools**: ReplayWeb.page displays integrity badges; js-wacz (Harvard LIL) provides CLI signing/verification

**Assessment**: WACZ Auth is the gold standard for offline archive signing. It's a real specification with implementations, uses modern cryptography, and supports both anonymous and domain-bound identity. However, WACZ is a ZIP-based format — its signing architecture doesn't directly transfer to ZIM's single-file format.

### Signed Exchanges (SXG) / Web Bundles

**Signing**: SXG signs individual HTTP request/response pairs using certificates with the `CanSignHttpExchanges` extension. Certificates have 90-day max validity and require DNS CAA records.

**Assessment**: SXG is designed for CDN prefetch optimization, not offline archives. Certificates must be obtained from specific CAs (Google's ACME), 90-day rotation is impractical for archival content, and browser support is Chromium-only. Web Bundles can carry signed or unsigned exchanges but don't define their own signing mechanism. Not applicable to our use case.

### MHTML

**Signing**: No signing mechanism exists. MHTML is a legacy format (RFC 2557) with no active development.

---

## 6. Architectural Options for Adding Signatures to ZIM

Given ZIM's architecture (single-file archive, 80-byte header, content in compressed clusters, MD5 at EOF), there are three places a signature could live:

### Option A: Detached Signature File (Sidecar)

Place a `.zim.sig` file alongside the `.zim` file containing a cryptographic signature of the file's SHA-256 hash.

**Pros**: Zero changes to ZIM format. Works today. Can sign existing ZIM files retroactively. Compatible with GPG, sigstore, or any signing scheme.
**Cons**: Two files to distribute. Sidecar can be lost, separated, or replaced independently. Doesn't survive format-unaware transfers (e.g., copy just the .zim).

### Option B: In-Archive Metadata Entry

ZIM files contain directory entries with MIME types. A signature could be stored as a special entry (e.g., path `M/signature`, MIME type `application/x-zim-signature`).

**Pros**: Self-contained — signature travels with the file. Uses existing ZIM structure. Readers that don't understand signatures simply ignore the entry.
**Cons**: Requires format-level support in libzim. The signature must be computed over the file contents excluding itself — creates a chicken-and-egg problem similar to the checksum. Would need to define what exactly is signed (all content entries? the whole file minus the signature entry?). Modifying the archive to add a signature changes the file, invalidating any whole-file hash.

### Option C: Wrapper Format / Extended Header

Extend the ZIM header (or add a footer section before the MD5) to include signature data.

**Pros**: Clean integration. The signature could cover [0, signaturePos) similar to the checksum.
**Cons**: Breaking format change — all existing tools would need updates. The 80-byte header is a fixed structure in the spec. A footer approach is more feasible but still requires libzim changes. Would need a major version bump (v7).

### Option D: Hybrid — Detached Signature with In-Archive Hash Manifest

Create a content-addressable manifest (SHA-256 of each cluster or entry) stored as an in-archive metadata entry. Then sign only the manifest hash externally.

**Pros**: Enables partial verification (check individual entries without reading entire file). Detached signature is simple. Manifest is self-describing.
**Cons**: Most complex approach. Still needs libzim support for the manifest entry. Two-artifact problem for the signature itself.

### Recommendation for gdev

**Option A (detached sidecar)** is the pragmatic choice. It requires zero upstream cooperation, works with existing ZIM files, and can be implemented entirely in the gdev layer. The sidecar loss problem is manageable when gdev controls distribution (both files can be bundled or the signature embedded in MCP server metadata).
