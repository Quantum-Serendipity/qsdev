# Kiwix Architecture, ZIM Libraries, and Stack Overflow Alternatives

## Executive Summary

The ZIM file format and its ecosystem offer a mature, well-supported path for local-first documentation access in an MCP server. The critical finding is that **multiple existing MCP servers already wrap ZIM files for AI model access**, with `openzim-mcp` being the most mature (57 stars, 21 tools, active development as of May 2026). Direct ZIM file access without kiwix-serve is entirely feasible through official bindings in Python (`python-libzim`), Node.js (`@openzim/libzim`), and pure implementations in Go (`akhenakh/gozim`) and Rust (`dignifiedquire/zim`). The main practical limitation is Stack Overflow's ZIM file size: **74-75 GB** for the full dump, last updated November 2023. Smaller Stack Exchange sites (Unix & Linux at 1.2 GB, Server Fault at 1.5 GB) are much more practical.

---

## 1. Kiwix Architecture

### Overview

Kiwix is an offline content reader built around the ZIM file format. The ecosystem consists of:

- **libzim** (C++) — Reference implementation for reading/writing ZIM files (236 stars, GPLv2+, latest release v9.7.0 May 2026)
- **kiwix-serve** — HTTP server that serves ZIM content with search, OPDS catalog, and viewer endpoints
- **kiwix-tools** — CLI utilities including kiwix-serve and kiwix-manage
- **Kiwix apps** — Desktop/mobile readers (Android, iOS, Windows, macOS, Linux)
- **openZIM scrapers** — Tools that convert web content to ZIM (sotoki for StackExchange, zimit for arbitrary sites)

### kiwix-serve HTTP API

kiwix-serve exposes a comprehensive HTTP API:

- **Content**: `/raw/ZIMNAME/content/PATH` — raw entry data; `/content/ZIMNAME/PATH` — processed with viewer
- **Search**: `/search?pattern=QUERY&books.name=ZIMNAME&pageLength=25&format=xml` — full-text search with Xapian
- **Suggestions**: `/suggest?content=ZIMNAME&term=QUERY` — title-based autocomplete
- **OPDS Catalog**: `/catalog/v2/entries` — filtered, paginated ZIM file listing
- **Multi-ZIM**: supports searching across multiple ZIM files (limited by `--searchLimit`)

**For an MCP server, kiwix-serve adds operational complexity (another HTTP service to manage) but provides a battle-tested search and content-serving layer. The alternative — direct ZIM access via libzim bindings — eliminates this dependency.**

Source: `docs/kiwix-serve-api-docs.md`

---

## 2. ZIM File Format Deep Dive

### Header Structure

The ZIM header is a fixed-size structure at byte 0:

| Field | Type | Size | Purpose |
|-------|------|------|---------|
| Magic Number | uint32 | 4 | `0x44D495A` |
| Major Version | uint16 | 2 | Format version (5 or 6) |
| Minor Version | uint16 | 2 | 0=old namespace, 1+=new namespace |
| UUID | bytes | 16 | Unique identifier |
| Article Count | uint32 | 4 | Number of entries |
| Title Index Pos | uint64 | 8 | Offset to title index |
| Path Pointer Pos | uint64 | 8 | Offset to path pointers |
| MIME List Pos | uint64 | 8 | Offset to MIME type list |
| Cluster Count | uint32 | 4 | Number of clusters |
| Cluster Pointer Pos | uint64 | 8 | Offset to cluster pointers |
| Main Page | uint32 | 4 | Entry index of main page |
| Layout Page | uint32 | 4 | Entry index of layout page |
| Checksum Pos | uint64 | 8 | Offset to MD5 checksum |

### Cluster/Blob Architecture

Content is organized into **clusters** (typically 1-2 MiB each) containing multiple **blobs**:

1. Each cluster starts with a compression type byte (1=none, 4=LZMA [legacy read-only], 5=Zstd)
2. Followed by an offset table (4-byte for v5, 8-byte for v6 extended) pointing to blob boundaries
3. Blobs contain raw content (HTML, images, CSS, etc.)
4. Multiple entries share a cluster, enabling 3x compression ratios

This architecture means that accessing a single article requires decompressing an entire cluster (~1-2 MiB), which is efficient for sequential reads but has per-article overhead for random access.

### Namespace System

**Old scheme** (minor version 0): Entries have single-character namespace prefixes:
- `A/` — Articles (main content)
- `I/` — Images and media
- `M/` — Metadata
- `-/` — Various auxiliary data
- `X/` — Full-text search index (Xapian databases)

**New scheme** (minor version >= 1): All user content under `C/` namespace; metadata via `archive.metadata_keys`.

### Full-Text Search (Xapian)

ZIM files embed Xapian databases for full-text search:
- Custom inverted index with term dictionary and postings lists
- BM25 relevance scoring (term frequency, document length, inverse document frequency)
- Phrase search support via positional information
- If libzim is compiled without Xapian, all search APIs are removed
- The `has_fulltext_index` property indicates whether a ZIM file contains a search index

### Content Entry Structure

Each directory entry has a 16-byte fixed header:
- 2 bytes: MIME type index (into global MIME list)
- 1 byte: parameter length (always 0)
- 1 byte: namespace character
- 4 bytes: revision (always 0)
- 4 bytes: cluster number
- 4 bytes: blob number within cluster

Followed by null-terminated URL and title strings.

Redirect entries replace cluster/blob numbers with a 4-byte redirect target index.

Sources: `docs/zim-format-spec-from-libzim-source.md`, `docs/zim-format-wikipedia.md`

---

## 3. ZIM Parsing Libraries — Comprehensive Survey

This is the highest-value section. For each library: maturity, API quality, dependencies, search support, and MCP-server suitability.

### C/C++: libzim (Reference Implementation)

| Attribute | Value |
|-----------|-------|
| Repository | [openzim/libzim](https://github.com/openzim/libzim) |
| Stars | 236 |
| Latest Release | v9.7.0 (May 9, 2026) |
| License | GPLv2+ |
| Language | C++ (95.4%) |
| Dependencies | LZMA, ICU, Zstd, Xapian (optional) |

**API**: `Archive` (open/iterate/access), `Entry`/`Item` (content), `Searcher`/`Query` (full-text), `SuggestionSearcher` (autocomplete). Thread-safe reads.

**Verdict**: The gold standard. All other bindings wrap this. Excellent for C++ projects but heavy dependency for embedding.

### Python: python-libzim (Official Binding)

| Attribute | Value |
|-----------|-------|
| Repository | [openzim/python-libzim](https://github.com/openzim/python-libzim) |
| Stars | 104 |
| Latest Release | v3.9.0 (March 2026) |
| License | GPLv3 |
| PyPI | `pip install libzim` |
| Platforms | Linux (x86_64/armhf/aarch64), macOS (x86_64/arm64), Windows (x64) |

**API**: Mirrors the C++ API closely:
```python
from libzim.reader import Archive
from libzim.search import Query, Searcher
from libzim.suggestion import SuggestionSearcher

archive = Archive("file.zim")
entry = archive.get_entry_by_path("path/to/article")
content = entry.get_item().content  # memoryview of raw bytes

# Full-text search
searcher = Searcher(archive)
query = Query().set_query("python decorators")
search = searcher.search(query)
results = search.getResults(0, 10)

# Suggestions
suggester = SuggestionSearcher(archive)
suggestions = suggester.suggest("pyth", 5)
```

**Key properties**: `has_fulltext_index`, `has_title_index`, `article_count`, `entry_count`, `metadata_keys`.

**Search**: Full Xapian-powered full-text search and title-based suggestions. This is the only Python option with real search support.

**Verdict**: **Best option for MCP server development.** Pre-built wheels include native libzim — no separate C library installation needed. All three existing MCP servers use this. GIL released during C++ calls for good performance.

### Python: ZIMply (Pure Python)

| Attribute | Value |
|-----------|-------|
| Repository | [kimbauters/ZIMply](https://github.com/kimbauters/ZIMply) |
| PyPI | `pip install zimply` |
| License | Unknown |

**API**: Primarily an offline reader with web UI, not a library for programmatic access. Pure Python but much slower than python-libzim.

**Verdict**: Not suitable. No search API, slow, designed as an end-user reader not a library.

### Node.js: @openzim/libzim (Official Binding)

| Attribute | Value |
|-----------|-------|
| Repository | [openzim/node-libzim](https://github.com/openzim/node-libzim) |
| Stars | 32 |
| Latest Release | v4.1.0 (March 2026) |
| License | GPLv3 |
| NPM | `@openzim/libzim` |
| Platforms | Linux, macOS (auto-downloads libzim binary) |

**API**:
```javascript
const { Archive, Query, Searcher, SuggestionSearcher } = require('@openzim/libzim');
const archive = new Archive('file.zim');
// Iteration, search, suggestions — mirrors C++ API
```

**Search**: Full-text search and suggestions via Xapian (same as C++ API).

**Verdict**: **Strong option for TypeScript MCP servers.** Native addon (N-API), auto-downloads libzim on Linux/macOS. The `zicojiao/zim-mcp-server` uses this approach.

### JavaScript: javascript-libzim (WASM)

| Attribute | Value |
|-----------|-------|
| Repository | [openzim/javascript-libzim](https://github.com/openzim/javascript-libzim) |
| Stars | 4 |
| License | GPL-3.0 |
| API Stability | Pre-1.0, unstable |

**API**: `Module.loadArchive()`, `Module.search()`, `Module.searchWithSnippets()`, `Module.suggest()`. Full search support via WASM-compiled Xapian.

**Verdict**: Interesting for browser-based use but overkill for server-side MCP. The Node.js native binding is better for server use cases.

### Go: akhenakh/gozim (Most Mature Go Option)

| Attribute | Value |
|-----------|-------|
| Repository | [akhenakh/gozim](https://github.com/akhenakh/gozim) |
| Stars | 216 |
| License | MIT |
| Search | Bleve (pure Go full-text search) |

**Features**: Native Go ZIM parsing, HTTP server, full-text search via Bleve (not Xapian). XZ decompression available in both CGO (fast) and pure Go (2.5x slower) modes. Zstd support.

**Key limitation**: Search requires pre-building a Bleve index (separate from ZIM's built-in Xapian index). The `gozimindex` tool builds this, but it means you can't use the ZIM file's embedded search index directly.

**Verdict**: Best Go option. Pure Go build possible (slower decompression). Active community (216 stars). But search requires separate index building, which is a significant operational burden for large ZIM files.

### Go: Bornholm/go-zim (Newer, Simpler)

| Attribute | Value |
|-----------|-------|
| Repository | [Bornholm/go-zim](https://github.com/Bornholm/go-zim) |
| Stars | 3 |
| License | MIT |
| Pure Go | Yes |

**Features**: Pure Go, XZ + Zstd support, http.FS compatibility. No search support — read-only access to entries.

**Verdict**: Too immature (3 stars, 5 commits). Useful as reference for understanding Go ZIM parsing but not production-ready.

### Rust: dignifiedquire/zim (Pure Rust)

| Attribute | Value |
|-----------|-------|
| Repository | [dignifiedquire/zim](https://github.com/dignifiedquire/zim) |
| Stars | 32 |
| Crate | `zim` v0.4.0 |
| License | Apache-2.0 / MIT |
| Pure Rust | Yes |

**Features**: Pure Rust ZIM reading, XZ + Zstd decompression, memory-mapped file access (memmap), parallel extraction via Rayon. CLI tools for extraction and IPFS linking.

**Limitations**: No full-text search (no Xapian). Only 31.48% API documentation coverage. No releases since the fork.

**Verdict**: Viable for read-only access if you build your own search layer. Pure Rust is attractive for deployment but the lack of search means you'd need to build a full-text index externally (e.g., with Tantivy).

### Rust: rndlabs/zim-rs (libzim Bindings)

| Attribute | Value |
|-----------|-------|
| Repository | [rndlabs/zim-rs](https://github.com/rndlabs/zim-rs) |
| Stars | 0 |
| License | Apache-2.0 / MIT |

**Verdict**: Dead project (0 stars, 47 commits). Not viable.

### Library Comparison Matrix

| Library | Language | Pure? | Search | Stars | Last Release | MCP Suitability |
|---------|----------|-------|--------|-------|-------------|-----------------|
| libzim | C++ | N/A | Xapian | 236 | May 2026 | Reference only |
| python-libzim | Python | No (Cython) | Xapian | 104 | Mar 2026 | **Excellent** |
| @openzim/libzim | Node.js | No (N-API) | Xapian | 32 | Mar 2026 | **Excellent** |
| akhenakh/gozim | Go | Optional | Bleve | 216 | 2015 | Good (needs index) |
| dignifiedquire/zim | Rust | Yes | None | 32 | Stale | Fair (no search) |
| javascript-libzim | JS/WASM | Yes | Xapian | 4 | Active | Niche |
| ZIMply | Python | Yes | None | — | — | Poor |
| Bornholm/go-zim | Go | Yes | None | 3 | — | Poor |

**Recommendation**: For an MCP server, **python-libzim is the clear winner** — it has full search support via Xapian, pre-built wheels with no system dependencies, and all three existing MCP servers validate this choice. Node.js via `@openzim/libzim` is the second-best option. Going pure Go or Rust means losing the embedded Xapian search, which is the whole point of ZIM files.

---

## 4. Self-Hosted Stack Overflow

### Stack Exchange Data Dumps

Stack Exchange releases quarterly data dumps to archive.org:

- **Format**: XML files in 7Z archives (bzip2 compressed)
- **Total size**: 92.3 GB (April 2024 dump)
- **Files per site**: Posts.xml, Users.xml, Votes.xml, Comments.xml, Badges.xml, Tags.xml, PostHistory.xml, PostLinks.xml
- **License**: CC-BY-SA 4.0
- **Latest dump**: April 2, 2024
- **371 separate site archives**

### Stack Overflow as ZIM

Kiwix packages Stack Exchange sites into ZIM files via **sotoki** (241 stars, Python, GPL v3):

- **Full Stack Overflow**: 74-75 GB (last updated November 2023 — significantly stale)
- **Unix & Linux**: 1.2 GB (February 2026)
- **Server Fault**: 1.5 GB (February 2026)
- **Super User**: 3.7 GB (February 2026)
- **Ask Ubuntu**: 2.6 GB (December 2025)

**Critical finding**: The full Stack Overflow ZIM has not been updated since November 2023, likely because of its enormous size. Smaller SE sites are kept current on a ~2-3 month cadence.

### Alternative Self-Hosting Approaches

1. **offstack** — Docker-based: imports XML dumps into a database, provides Node.js web interface with search
2. **seekoff** — Uses Elasticsearch for indexing, web interface for searching, supports tag-based filtering
3. **sodata** — Imports XML into PostgreSQL or SQLite3
4. **SODDI** — Stack Overflow Data Dump Importer for SQL Server

**For gdev's use case**: The ZIM approach (via kiwix/sotoki) is more practical than raw XML import because ZIM files are self-contained, pre-indexed, and don't require database infrastructure. The 74 GB Stack Overflow ZIM is impractical for developer workstations, but the smaller SE sites (1-4 GB) are very manageable.

Sources: `docs/stack-exchange-zim-file-sizes.md`, `docs/stack-exchange-data-dump-archive-org.md`, `docs/sotoki-stackexchange-to-zim-github.md`

---

## 5. Alternative Q&A Corpora

### Developer-Relevant Stack Exchange Sites Available as ZIM

| Site | Focus | ZIM Size | Updated |
|------|-------|----------|---------|
| Unix & Linux | Shell, CLI, system admin | 1.2 GB | Feb 2026 |
| Server Fault | Enterprise sysadmin | 1.5 GB | Feb 2026 |
| Super User | General computing | 3.7 GB | Feb 2026 |
| Ask Ubuntu | Ubuntu-specific | 2.6 GB | Dec 2025 |
| DevOps | CI/CD, infrastructure | ~100-500 MB | Recent |
| Software Engineering | Design, architecture | ~500 MB | Recent |
| Code Review | Code quality | ~200-500 MB | Recent |
| Database Administrators | SQL, databases | ~800 MB | Recent |

A curated selection of 3-5 smaller SE sites (Unix & Linux, Server Fault, DevOps, Software Engineering) would total roughly 4-5 GB — very practical for developer workstations.

### Beyond Stack Exchange

Other offline corpora available as ZIM:
- **Wikipedia** (various sizes, up to 97 GB for full English with images)
- **Wiktionary** — Dictionary/language reference
- **WikiBooks** — Technical books
- **MDN Web Docs** — Available as ZIM via Kiwix
- **Arch Wiki** — Available as ZIM
- **Various programming documentation** — Some available via zimit (any-website-to-ZIM tool)

### Tag-Filtered Stack Overflow Subsets

Instead of the full 74 GB Stack Overflow, a custom sotoki run could create a **tag-filtered subset**. For example, filtering to only `python`, `javascript`, `typescript`, `rust`, `go`, `nix`, `docker`, `kubernetes` tags would dramatically reduce size while keeping developer-relevant content. This would require running sotoki with custom parameters (or modifying the scraper).

---

## 6. Existing Kiwix/ZIM MCP Servers

Three existing MCP servers were found, spanning the maturity spectrum:

### cameronrye/openzim-mcp (Best Option)

| Attribute | Value |
|-----------|-------|
| Stars | 57 |
| Language | Python (96.9%) |
| License | MIT |
| Latest Release | v2.0.0a12 (May 14, 2026) |
| Install | `pip install openzim-mcp` or `uv tool install openzim-mcp` |

**Architecture**: Uses python-libzim directly (no kiwix-serve dependency). Dual mode: simple (1 `zim_query` tool with natural language) or advanced (21 specialized tools). Supports stdio, HTTP, and SSE transports.

**Key features**:
- Full-text search with Xapian via libzim
- Title-based suggestions with fuzzy matching
- Multi-archive search across all loaded ZIM files
- Batch entry retrieval (up to 50 per call)
- LRU caching with TTL, rate limiting, pagination
- Bearer-token auth for HTTP transport
- Response metadata with token estimates
- 80%+ test coverage

**Tools (advanced mode)**: list_zim_files, search_zim_file, search_with_filters, search_all, get_zim_entry, get_zim_entries, get_entry_summary, get_article_structure, get_table_of_contents, extract_article_links, get_related_articles, find_entry_by_title, get_search_suggestions, list_namespaces, browse_namespace, walk_namespace, get_zim_metadata, get_main_page, get_binary_entry, get_server_health, get_server_configuration.

**Assessment**: This is a serious, production-quality project. Actively maintained (literally released the same day as this research). The simple mode `zim_query` tool is particularly well-designed for LLM consumption with natural language intent parsing.

### ThinkInAI-Hackathon/zim-mcp-server

| Attribute | Value |
|-----------|-------|
| Stars | 17 |
| Language | Python |
| License | MIT |
| Tools | 3 (list, search, get_entry) |

Simple hackathon project. Uses python-libzim. Functional but minimal.

### zicojiao/zim-mcp-server

| Attribute | Value |
|-----------|-------|
| Stars | 12 |
| Language | TypeScript |
| License | MIT |
| Tools | 3 (list, search, get_entry) |

TypeScript implementation. Experimental, WSL2-tested only. Uses `@openzim/libzim` Node.js binding.

### jeffreyrampineda/kiwix-wiki-mcp-server

| Attribute | Value |
|-----------|-------|
| Stars | 14 |
| Language | TypeScript |
| License | ISC |
| Tools | 3 (search_wiki, get_article, list_libraries) |

**Different architecture**: This is a thin wrapper around a running kiwix-serve instance (HTTP client), NOT a direct ZIM reader. Requires kiwix-serve to be running separately.

**Assessment**: The kiwix-serve dependency makes this less suitable for gdev's local-first approach — it adds another service to manage.

### Implications for gdev

**Option A: Adopt openzim-mcp directly.** It's MIT-licensed, actively maintained, feature-rich, and already does what we need. Configuration via environment variables. Can be installed via pip/uv and added to `.mcp.json`.

**Option B: Build a lightweight custom MCP server** using python-libzim or @openzim/libzim, with only the tools gdev needs (search, get_entry, list_files). Simpler surface area, tighter control, but reinvents tested code.

**Option C: Fork/extend openzim-mcp** with gdev-specific features (e.g., tag-filtered search for SE sites, integration with DevDocs MCP, failover logic).

Sources: `docs/openzim-mcp-server-github.md`, `docs/thinkinai-zim-mcp-server-github.md`, `docs/zim-mcp-server-zicojiao-github.md`, `docs/kiwix-wiki-mcp-server-github.md`

---

## 7. Architectural Recommendations for gdev

### Recommended Stack

1. **MCP Server**: `openzim-mcp` in simple mode (single `zim_query` tool) — install via Nix wrapping `uv tool install openzim-mcp`
2. **ZIM Files**: Curated selection of developer-relevant SE sites (Unix & Linux, Server Fault, DevOps, Software Engineering) totaling ~4-5 GB, plus optionally MDN Web Docs and Arch Wiki
3. **Full Stack Overflow**: Offer as optional download (74 GB) with clear size warning
4. **No kiwix-serve required**: openzim-mcp reads ZIM files directly via python-libzim

### Deployment Model

```
~/.local/share/gdev/zim/          # ZIM file storage
├── unix.stackexchange.com_en_all_2026-02.zim
├── serverfault.com_en_all_2026-02.zim
├── devops.stackexchange.com_en_all_2026-02.zim
└── softwareengineering.stackexchange.com_en_all_2026-02.zim

~/.config/claude/.mcp.json        # MCP configuration
{
  "mcpServers": {
    "local-docs": {
      "command": "openzim-mcp",
      "env": {
        "OPENZIM_MCP_ZIM_DIR": "~/.local/share/gdev/zim",
        "OPENZIM_MCP_TOOL_MODE": "simple"
      }
    }
  }
}
```

### Security Properties

- **No web access required**: All content served from local ZIM files
- **No prompt injection surface**: Content is pre-packaged, static, and from trusted sources (Stack Exchange CC-BY-SA dumps)
- **No bot blocks**: No HTTP requests to documentation sites
- **Content integrity**: ZIM files include MD5 checksums
- **Residual risk**: ZIM content could theoretically contain adversarial text in user-contributed Q&A, but the attack surface is dramatically smaller than arbitrary web fetches

### Open Questions

1. **ZIM file updates**: How to automate periodic downloads of updated ZIM files? Kiwix has an OPDS catalog API that could be polled.
2. **Tag-filtered SO**: Can sotoki be configured to build a smaller, tag-filtered Stack Overflow ZIM? This would make the full SO corpus practical (~5-10 GB instead of 74 GB).
3. **Nix packaging**: openzim-mcp depends on python-libzim which includes native code. Nix packaging needs to handle the libzim C++ dependency correctly.
4. **Combined search**: How to search across both ZIM-based Q&A and DevDocs-based API documentation in a single query? May need a meta-MCP or skill-level routing.

---

## Sources

All raw sources saved to `docs/`:
- `libzim-github-readme.md` — libzim reference implementation
- `zim-format-spec-from-libzim-source.md` — ZIM format from C++ headers
- `zim-format-wikipedia.md` — ZIM format Wikipedia article
- `libzim-cpp-api-usage.md` — C++ API usage guide
- `python-libzim-github.md` — python-libzim overview
- `python-libzim-reader-api.md` — python-libzim reader API reference
- `node-libzim-github.md` — Node.js libzim binding
- `javascript-libzim-wasm-github.md` — WASM libzim
- `gozim-akhenakh-github.md` — Go ZIM (most mature)
- `go-zim-bornholm-github.md` — Go ZIM (newer)
- `rust-zim-crate-docs.md` — Rust zim crate
- `rust-zim-dignifiedquire-github.md` — Rust ZIM library
- `openzim-mcp-server-github.md` — Best existing MCP server
- `zim-mcp-server-zicojiao-github.md` — TypeScript MCP server
- `kiwix-wiki-mcp-server-github.md` — kiwix-serve wrapper MCP
- `thinkinai-zim-mcp-server-github.md` — Hackathon MCP server
- `kiwix-serve-api-docs.md` — kiwix-serve HTTP API
- `sotoki-stackexchange-to-zim-github.md` — SE-to-ZIM scraper
- `stack-exchange-zim-file-sizes.md` — ZIM file sizes
- `stack-exchange-data-dump-archive-org.md` — Raw SE data dumps
