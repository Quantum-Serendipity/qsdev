# DevDocs Integrity Mechanisms — Raw Research

> **Source**: Multiple sources (GitHub repo, issues, source code, web searches)
> **Retrieved**: 2026-05-14
> **Primary repo**: https://github.com/freeCodeCamp/devdocs

---

## 1. Content Pipeline Architecture

### Scraper System

DevDocs uses a Ruby-based scraper system built on three core libraries:
- **Typhoeus** — HTTP requests (parallel/async)
- **HTML::Pipeline** — filter chain processing
- **Nokogiri** — HTML parsing

Two scraper types exist:
- **UrlScraper** — downloads files via HTTP from upstream documentation sites
- **FileScraper** — reads from local filesystem (same processing pipeline, different input)

Source: `lib/docs/core/scraper.rb`, `lib/docs/core/scrapers/url_scraper.rb`, `lib/docs/core/scrapers/file_scraper.rb`

### Processing Pipeline

Each page goes through:
1. Response validation (200 status, HTML content type, URL within base_url boundary)
2. HTML parsing via Nokogiri
3. HTML filter chain: ContainerFilter, CleanHtmlFilter, URL normalization, internal URL detection, path consistency, localhost URL removal
4. Text filter chain: string conversion, empty node removal, attribution appending
5. Per-scraper custom filters (metadata extraction)

URL deduplication is string-based Set operations — no cryptographic verification.

Source: https://raw.githubusercontent.com/freeCodeCamp/devdocs/main/lib/docs/core/scraper.rb
Source: https://raw.githubusercontent.com/freeCodeCamp/devdocs/main/docs/scraper-reference.md

### Output Format

The scraper produces:
- **Normalized HTML partials** — cleaned, processed documentation pages
- **Two JSON files per doc**: index (metadata) + offline data (full content)
- **docs.json manifest** — lists all available documentations with name, version, update date, aliases

Source: README.md and scraper-reference.md

### PageDb (In-Memory Storage)

`PageDb` class (`lib/docs/core/page_db.rb`) uses a simple in-memory hash dictionary:
- `add(path, content)` stores content keyed by path
- `to_json` serializes entire hash via `JSON.generate`
- No persistent storage, no hashing, no integrity checks
- Data exists only during scraper runtime

### Manifest Generation

`lib/docs/core/manifest.rb` generates `docs.json` by:
1. Iterating documentation objects
2. Checking if metadata file exists in store
3. Parsing existing metadata JSON
4. Enriching with attribution and aliases
5. Outputting formatted JSON

**No checksums, hashes, or signatures in the manifest.** It is a plain JSON serialization of metadata.

---

## 2. Distribution Mechanism

### Web App (devdocs.io)

- Sinatra/Sprockets Ruby app serving client-side JavaScript
- Catch-all regex route matches doc slugs: `get %r{/([\w~\.%]+)(\-[\w\-]+)?(/.*)?}`
- Content-Security-Policy headers set but no SRI attributes on served assets
- Static assets use Sprockets with digest-based cache busting (fingerprinted filenames) but this is for cache invalidation, not integrity
- Production CSP restricts script sources to specific domains (Google Analytics, Gauge.es, jQuery CDN)

Source: https://raw.githubusercontent.com/freeCodeCamp/devdocs/main/lib/app.rb

### Offline Download (Browser)

Client-side IndexedDB storage (`assets/javascripts/app/db.js`):
- Primary: loads from IndexedDB cache via `loadWithIDB()`
- Fallback: XHR requests via `loadWithXHR()` when cache unavailable
- Stores docs with `mtime` (modification time) for staleness comparison
- `checkForCorruptedDocs()` — detects missing "index" entries and orphaned refs, deletes them
- **No hash or checksum verification on downloaded content**
- Version tracking combines `DB.VERSION` (15) with user schema version

### Update Mechanism

`assets/javascripts/app/update_checker.js`:
- Polls the application script URL; 404 triggers update notification
- Auto-checks every ~6 hours on window focus
- Can trigger `docs.checkForUpdates()` or `docs.updateInBackground()`
- **No hash validation, no signature checks, no version number comparison**

### Service Worker

`assets/javascripts/app/serviceworker.js`:
- Handles registration and lifecycle management only
- Delegates actual caching to the service worker file at `app.config.service_worker_path`
- No integrity checks in the registration scaffolding

### Docker Images

Pre-generated documentation packages available via Docker:
- `thor docs:download` command for bulk download
- Images auto-built and updated monthly
- No documented integrity verification for Docker-distributed content beyond Docker's own image signing

---

## 3. GitHub Issue #1113 — Integrity Check Request

**Title**: "Add integrity check to assets files served by CDN"
**Status**: Open (since October 17, 2019 — over 6 years)
**Label**: improvement
**Author**: @eloyesp

**Description**: Proposes implementing the `integrity` attribute (Subresource Integrity / SRI) on `<link>` elements for CDN-served assets. Notes that Sprockets already provides built-in support via DigestUtils, requiring only modifications to the stylesheet_tag helper.

**Scope**: This issue is specifically about SRI for CSS/JS assets served via CDN — NOT about documentation content integrity. It addresses whether the web app's own code is tampered with in transit, not whether documentation content is authentic.

**Status**: No assignees, no PRs linked, no development branches. Effectively stalled for 6+ years.

Source: https://github.com/freeCodeCamp/devdocs/issues/1113

### Other Integrity-Related Issues

GitHub issue search for terms "signing", "signature", "integrity", "checksum", "hash", "verify", "security", "tamper" yielded only:
- #1113 (above)
- #2660, #2642 — Hashgraph docs scraper (unrelated, name collision)
- #2322 — "@" sign in filenames (unrelated, character encoding)

**No issues requesting or discussing documentation content signing or verification.**

---

## 4. devdocs-desktop (Electron App)

Third-party Electron wrapper by @egoist (https://github.com/egoist/devdocs-desktop):
- Wraps devdocs.io in a webview — no independent content fetching
- Mac app is NOT code-signed (developer program expired)
- No ASAR integrity checking enabled
- No content integrity verification beyond what the web app provides (which is none)
- AppImage distribution (Linux) — AppImages are generally not independently verified

Source: https://github.com/egoist/devdocs-desktop

---

## 5. Gemfile Dependency Analysis

Key dependencies from `Gemfile`:
- **rack-ssl-enforcer** — enforces HTTPS (transport security only)
- **No cryptographic libraries** — no gpg, openssl wrappers, sigstore, or hashing gems beyond Ruby stdlib
- Standard web stack: sinatra, sprockets, nokogiri, typhoeus, yajl-ruby

Source: https://raw.githubusercontent.com/freeCodeCamp/devdocs/main/Gemfile

---

## 6. Comparison: Other Documentation Aggregators

### Dash (macOS, Kapeli)

**Docset format**: Folder bundles containing HTML docs + SQLite index database + Info.plist metadata + optional icon.

**Feed XML format** (e.g., NodeJS.xml):
```xml
<entry>
  <version>25.9.0</version>
  <ios_version>1</ios_version>
  <url>[mirror1]</url>
  <url>[mirror2]</url>
  ...
  <other-versions>...</other-versions>
</entry>
```

**No integrity fields in feed XML** — no checksums, hashes, or signatures. Multiple geographic mirrors for availability, not verification.

**Download mechanism** (from Dash-iOS source `DHDocsetDownloader.m`):
- Downloads from feeds, selects best mirror by latency
- HTTP URLs converted to HTTPS
- Archives extracted via `DHUnarchiver`
- **No checksum verification** (no MD5, SHA-1, SHA-256)
- **No certificate pinning** — standard NSURLConnection
- TLS is the only transport protection

Source: https://raw.githubusercontent.com/Kapeli/Dash-iOS/master/Dash/DHDocsetDownloader.m
Source: https://kapeli.com/docsets

### Zeal (Cross-platform, Qt-based)

- Uses Dash-compatible docset format
- Downloads docsets from Dash feeds
- Built with Qt 6.4.2+, uses SQLite and libarchive
- **No documented integrity verification, signing, or checksums**
- Directs users to Dash's docset generation guide for custom docsets

Source: https://github.com/zealdocs/zeal

### Velocity (Windows)

- Windows-only Dash alternative
- Uses same docset format
- No publicly documented integrity mechanisms found

### Summary: Documentation Aggregator Integrity

| Aggregator | Content Signing | Hash Verification | Transport Security | Feed Integrity |
|------------|----------------|-------------------|-------------------|----------------|
| DevDocs | None | None | HTTPS | None |
| Dash | None | None | HTTPS | None |
| Zeal | None | None | HTTPS (via Dash) | None |
| Velocity | None | None | HTTPS | None |

**No documentation aggregator implements content signing or hash verification.** The entire ecosystem relies solely on HTTPS transport security and trust in the distribution channel.

---

## 7. Relevant Standards and Tools

### Subresource Integrity (SRI)
- W3C standard for browser verification of CDN-served resources
- Works via `integrity` attribute on `<script>` and `<link>` tags
- Only applies to browser-loaded assets, not downloaded documentation content
- DevDocs issue #1113 requested this for app assets (not doc content)

### Sigstore / Cosign
- Modern keyless signing infrastructure
- Could sign documentation bundles as blob artifacts
- Used by npm, PyPI for package provenance
- No documentation ecosystem has adopted it

### SLSA Framework
- Supply-chain security framework with graduated verification levels
- Relevant conceptual model but no documentation aggregator implements any SLSA level
