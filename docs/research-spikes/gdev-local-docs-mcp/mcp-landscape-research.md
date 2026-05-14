# Documentation MCP Servers Landscape Research

## Executive Summary

The documentation MCP server ecosystem is large and growing rapidly. As of May 2026, there are hundreds of documentation-related MCP servers across registries like Glama (23,000+ total servers), Smithery, mcp.so, and PulseMCP. This report catalogs the servers most relevant to gdev's local-first documentation strategy, evaluates their security properties (particularly prompt injection surface), and identifies gaps that a custom DevDocs or Kiwix MCP server would fill.

**Key finding:** The ecosystem splits cleanly into two architectural camps -- **cloud-hosted servers** that fetch/proxy documentation from remote APIs (Context7, GitMCP, AWS Docs, Microsoft Learn) and **local-first servers** that operate entirely on-machine after initial setup (OpenZIM MCP, DevDocs-MCP variants, man-mcp-server, godoc-mcp). For gdev's security goals, only the local-first camp eliminates the prompt injection surface from web-fetched content.

---

## 1. Context7 MCP Server (Deep Dive)

**Repository:** [upstash/context7](https://github.com/upstash/context7) | **Stars:** 55.3k | **License:** MIT | **Language:** TypeScript

### How It Works

Context7 is a **centralized, cloud-backed documentation service** operated by Upstash. Despite its popularity, the architecture has significant implications for security:

**Backend pipeline (proprietary, not open-source):**
1. **Crawler** fetches documentation from GitHub repositories, guided by `context7.json` configuration files with exclusion rules
2. **Parser Engine** processes raw documentation into structured chunks
3. **Embedding Engine** converts chunks into vector embeddings stored in a DiskANN vector database
4. **Quality Assurance Pipeline** validates documentation from 33,000+ libraries (but verification is post-publication via community reports, not pre-indexing validation)
5. **Upstash Vector** stores embeddings for semantic search
6. **c7score Algorithm** (proprietary) scores and reranks search results for relevance
7. **Upstash Redis** (multi-region Global Database) caches top results for fast delivery

**MCP tools exposed:**
- `resolve-library-id`: Maps library names to Context7 IDs
- `get-library-docs` (aka `query-docs`): Retrieves version-specific documentation

**Key clarification:** Despite marketing language about "real-time" docs, Context7 serves **pre-indexed content** from its vector database and Redis cache, not live web fetches at query time. Content freshness is "typically within days" of upstream changes.

### Performance

- **Latency:** Reduced from 24s to 15s (38% improvement) via server-side reranking
- **Token efficiency:** Reduced from 9,700 to 3,300 tokens per response (65% reduction)
- **Edge infrastructure** for low latency (Upstash's global network)
- Cross-library queries scored as low as 3.5/10 in quality evaluations; single-library queries reached 9.4/10

### Rate Limits

- **Free tier (no API key):** 1,000 requests/month + 20 bonus daily requests after cap (quietly reduced from ~6,000 in January 2026 -- an 83% cut)
- **Pro tier:** 5,000 requests/seat/month at $10/month
- **Enterprise:** Custom limits

### Security Properties -- Critical Concerns

**ContextCrush Vulnerability (February 2026):**
A context poisoning vulnerability was discovered, reported February 18, and patched by February 23. The attack exploited Context7's dual role as both an **open registry** (anyone with a GitHub account can register a library) and a **trusted delivery mechanism**:

1. Attacker registers a library on Context7 and injects malicious prompt injection instructions into "Custom Rules" / "AI Instructions" field
2. Attacker manufactures credibility through repeated API requests to earn "trending" badges
3. When developers query the library, Context7 delivers the malicious rules **verbatim** into the agent's context, indistinguishable from legitimate documentation
4. The AI agent executes the embedded instructions using its own tool access (Bash, file operations, network)

**Demonstrated attack:** Credential theft (.env exfiltration), data exfiltration (via GitHub issue creation), and destructive cleanup (file deletion).

**Post-patch mitigations:**
- Custom Rules sanitization added
- `researchMode` parameter removed from `query-docs`
- Stacklok recommends outbound network filtering

**Remaining structural risk:** The dual registry/delivery architecture persists. Any server aggregating user-generated content and delivering it through a trusted MCP channel creates inherent supply-chain attack potential. Context7's minimal scope (read-only docs) is irrelevant -- it only needs to deliver content into an agent's context; the agent's own tools do the damage.

### Assessment for gdev

Context7 is **high utility but architecturally incompatible** with gdev's local-first security goals. It requires internet access at query time, sends queries to Upstash's servers, delivers content from a centralized registry with a demonstrated prompt injection history, and has opaque rate limits that have been reduced without notice. It could serve as a **fallback** behind local documentation sources, but should never be the primary documentation source for a security-conscious developer environment.

---

## 2. DevDocs MCP Servers

### 2a. madhan-g-p/DevDocs-MCP

**Repository:** [madhan-g-p/DevDocs-MCP](https://github.com/madhan-g-p/DevDocs-MCP) | **Stars:** 11 | **License:** MIT | **Language:** TypeScript (NestJS)

**Architecture:** This is the most promising DevDocs MCP for gdev. It uses DevDocs.io's structured JSON datasets (the same format devdocs.io uses internally for offline access). On first `ingest`, it downloads documentation JSONs for specified dependencies to a local `./data` directory. After ingestion, it operates **100% offline** with no internet required.

**Key features:**
- `ingest` tool: Downloads docs for project dependencies (reads `package.json` for version detection)
- `search` tool: Ranked fuzzy search across cached entries
- SQLite metadata via `sql.js` (zero native dependencies)
- Version-pinned documentation matching project dependencies
- STDIO and HTTP/SSE transport

**Security:** Excellent for gdev. After initial ingestion, no network access needed. Content sourced from DevDocs.io's curated, official documentation. No user-generated content in the pipeline.

**Maturity concern:** Only 11 stars, "under active and heavy development" -- may be unstable.

### 2b. katvito/devdocs-mcp

**Repository:** [katvito/devdocs-mcp](https://github.com/katvito/devdocs-mcp) | **Stars:** 3 | **License:** MIT | **Language:** JS/TS

**Architecture:** Runs a **full self-hosted DevDocs instance** via Docker on port 9292, then wraps it with an MCP server. This is the heaviest-weight approach but provides the most complete DevDocs experience.

**Key features:**
- `view_available_docs`: Lists available documentation
- `search_specific_docs`: Searches within specific documentation by slug
- Full Docker Compose setup with multi-container orchestration
- 10+ minutes initial build (downloads DevDocs image)
- Fully offline after setup

**Security:** Good -- self-hosted DevDocs instance with no external API calls. But heavier than necessary for MCP use case.

**Maturity concern:** Only 3 stars, 46 commits, last release October 2025.

### 2c. llmian-space/devdocs-mcp

**Repository:** [llmian-space/devdocs-mcp](https://github.com/llmian-space/devdocs-mcp) | **Stars:** 9 | **License:** MIT | **Language:** Python

**Assessment:** Very early stage (3 commits). "Inspired by devdocs.io" but does not appear to actually integrate with DevDocs data. Search implementation, caching, and core functionality are listed as unimplemented. **Not viable.**

### 2d. cyberagiinc/DevDocs

**Repository:** [cyberagiinc/DevDocs](https://github.com/cyberagiinc/DevDocs) | **Stars:** 2.1k | **License:** Apache 2.0 | **Language:** TS/Python

**Architecture:** This is NOT a DevDocs.io wrapper -- it is a **general-purpose web crawler** with MCP integration. Uses Crawl4AI + Playwright to crawl arbitrary websites and make them queryable. Despite the name, it has no special integration with devdocs.io.

**Security concern:** Crawls live websites at query time, which is the exact opposite of gdev's local-first requirement. The README warns it "is not publicly maintained."

**Assessment:** Wrong tool for gdev despite high star count. Name is misleading.

---

## 3. Other Documentation MCP Servers

### 3a. Grounded Docs MCP Server (docs-mcp-server)

**Repository:** [arabold/docs-mcp-server](https://github.com/arabold/docs-mcp-server) | **Stars:** 1,300+ | **License:** MIT | **Language:** TypeScript

**Architecture:** Self-hosted documentation indexing system. Fetches docs from URLs, GitHub repos, npm/PyPI packages, or local files, then creates a searchable local index. Supports semantic search via optional embedding models (OpenAI, Ollama, Gemini).

**Key features:**
- CLI, Web UI (localhost:6280), and MCP server modes
- Supports 90+ document formats (PDF, Word, Markdown, HTML, source code, etc.)
- Version-specific documentation targeting
- Local indexing with persistent cache
- Optional Playwright for SPA documentation sites

**Security:** Mixed. **After indexing**, content is served locally -- no web fetch at query time. But the indexing step fetches from the web, and content is not sanitized for prompt injection. If optional semantic search is enabled via OpenAI, queries leave the machine.

**Assessment:** Good architecture for gdev if Ollama is used for embeddings instead of OpenAI. The scrape-then-index model is similar to what a custom DevDocs MCP would do. Self-describes as "open-source alternative to Context7, Nia, and Ref.Tools."

### 3b. MDN MCP Server (Official)

**Repository:** [mdn/mcp](https://github.com/mdn/mcp) | **Stars:** 33 | **License:** MPL-2.0 | **Language:** JavaScript

**Architecture:** Official Mozilla experimental MCP server. Operates as a **remote HTTP service** at `https://mcp.mdn.mozilla.net/` or locally on port 3002. Queries backend MDN services in real-time.

**Tools:** `get-doc` (full MDN article as markdown), `get-compat` (Browser Compatibility Data as JSON)

**Security:** Content is first-party (Mozilla-controlled), reducing prompt injection risk vs. user-generated registries. But queries go to Mozilla's servers and telemetry data is collected during the experimental phase. **Not local-first.**

**Assessment:** Useful for web development docs but requires internet. Could complement a local DevDocs instance that already includes MDN content.

### 3c. man-mcp-server

**Repository:** [guyru/man-mcp-server](https://github.com/guyru/man-mcp-server) | **Stars:** 13 | **License:** MIT | **Language:** Python

**Architecture:** **Purely local.** Wraps the system `man` and `apropos` commands. Three tools: `search_man_pages`, `get_man_page`, `list_man_sections`. Also exposes `man://` URI resources.

**Security:** Excellent. No network access whatsoever. Reads only from locally installed man pages. Includes timeout protection for subprocess calls. Content is from official package man pages installed via the system package manager.

**Assessment:** Perfect fit for gdev on NixOS. Man pages are already installed by Nix; this server just exposes them to Claude Code. Lightweight, zero dependencies beyond Python.

### 3d. tldr-mcp-server

**Repository:** [onatm/tldr-mcp-server](https://github.com/onatm/tldr-mcp-server) | **Stars:** 1 | **License:** MIT | **Language:** TypeScript

**Architecture:** Provides access to tldr-pages (simplified man pages). Two tools: "Get Page" and "Search". Likely maintains a local copy of English tldr content.

**Security:** Presumably local-only (tldr pages are small static files). Very early stage.

**Assessment:** Low maturity. The man-mcp-server or a combined solution would be more practical.

### 3e. godoc-mcp

**Repository:** [mrjoshuak/godoc-mcp](https://github.com/mrjoshuak/godoc-mcp) | **Stars:** 115 | **License:** MIT | **Language:** Go

**Architecture:** Wraps the local `go doc` command. **No internet connection required.** Queries Go packages from the local Go installation, handles both stdlib and third-party packages via temporary module contexts. Built-in response caching.

**Tools:** `get_doc` (with pagination, flags), `list_packages`

**Security:** Excellent. Purely local operation. Token-efficient (returns only documentation, not source files). Well-maintained (Docker support, multiple transports).

**Assessment:** Model example of how language-specific doc servers should work for gdev. The pattern (wrap the language's native doc tool) is replicable for other languages.

### 3f. Rust Docs MCP Server

**Repository:** [Govcraft/rust-docs-mcp-server](https://github.com/Govcraft/rust-docs-mcp-server) | **Stars:** 275 | **License:** MIT | **Language:** Rust (with Nix support)

**Architecture:** More complex than godoc-mcp. Creates a temporary Rust project, runs `cargo doc` to generate HTML, parses HTML with `scraper`, generates embeddings via OpenAI's API, caches everything locally in XDG data directories.

**Security:** Mixed. Local `cargo doc` generation is safe, but **requires OpenAI API key** for embeddings and query answering (sends data to OpenAI). The actual documentation generation is local, but the search/answer pipeline is cloud-dependent.

**Assessment:** The `cargo doc` generation approach is clever, but the OpenAI dependency is a dealbreaker for gdev's security model. Could be forked to use Ollama instead.

### 3g. Microsoft Learn MCP Server

**Repository:** [MicrosoftDocs/mcp](https://github.com/MicrosoftDocs/mcp) | **Stars:** 1.6k | **License:** CC-BY-4.0 + MIT

**Architecture:** Remote HTTP endpoint at `https://learn.microsoft.com/api/mcp`. No authentication required. Three tools: semantic search, page fetch (as markdown), code sample search.

**Security:** First-party content (Microsoft-controlled), reducing prompt injection risk. But **online-only**, no local caching. All queries go to Microsoft's servers.

**Assessment:** Useful for Azure/.NET teams but not local-first. Similar pattern to MDN MCP.

### 3h. AWS Documentation MCP Server

**Repository:** [awslabs/mcp](https://awslabs.github.io/mcp/servers/aws-documentation-mcp-server) | **License:** AWS open-source

**Architecture:** Fetches AWS docs via HTTP in real-time. Five tools including search, page reading, and recommendations. **No offline capability, no caching, no local storage.**

**Security:** First-party content but online-dependent. User-Agent customization for corporate proxies suggests awareness of restrictive environments.

**Assessment:** Not suitable for local-first. Same pattern as Microsoft Learn and MDN.

### 3i. MCP-NixOS

**Repository:** [utensils/mcp-nixos](https://github.com/utensils/mcp-nixos) | **Stars:** 638 | **License:** MIT | **Language:** Python

**Architecture:** Unified MCP server for NixOS ecosystem. Two consolidated tools (`nix`, `nix_versions`) covering 130,000+ packages, 23,000+ options, Home Manager, nix-darwin, Nixvim, FlakeHub, and noogle.dev. Queries **remote APIs** (search.nixos.org, FlakeHub, noogle.dev, wiki.nixos.org, NixHub.io, cache.nixos.org).

**Security:** Online-dependent -- queries multiple external APIs. No local caching mentioned. But content sources are all first-party NixOS infrastructure, reducing prompt injection risk.

**Assessment:** Highly relevant to gdev (NixOS is the target platform) but not local-first. A local Nix documentation source (e.g., ZIM file of wiki.nixos.org + local `nix` CLI integration) would be more secure.

### 3j. OpenZIM MCP Server

**Repository:** [cameronrye/openzim-mcp](https://github.com/cameronrye/openzim-mcp) | **Stars:** 57 | **License:** MIT | **Language:** Python

**Architecture:** **The most security-aligned server for gdev.** Fully offline MCP server for ZIM (Zeno IMproved) archives. Simple Mode exposes one `zim_query` tool; Advanced Mode exposes 21 specialized tools covering search, navigation, content retrieval, structure analysis, and metadata.

**Key technical details:**
- Full-text search via libzim's indexed search (when ZIM file includes an index)
- Suggestion-based fallback, typo-tolerance, cursor-based pagination
- Compact mode reducing response size 3-6x
- LRU cache with configurable TTL (default 3,600s)
- Path traversal prevention, input sanitization, timing-safe auth, ReDoS protection
- Multi-architecture Docker images (amd64, arm64)
- Three transports: stdio, HTTP (with bearer auth), SSE

**Security:** Excellent. No network access after ZIM files are downloaded. Content is from Kiwix's curated archives (Wikipedia, Stack Overflow, documentation sites). The server itself has production-grade security features (path traversal prevention, timing-safe auth, input sanitization).

**Assessment:** This is the ZIM/Kiwix server gdev should integrate. It handles the "offline Stack Overflow" use case directly. ZIM files for Stack Overflow, MDN, and other documentation sites are available from `download.kiwix.org`.

### 3k. GitMCP

**Repository:** [idosal/git-mcp](https://github.com/idosal/git-mcp) | **Stars:** 8.1k | **License:** Apache 2.0 | **Language:** TypeScript

**Architecture:** Cloud-based remote service. Transforms any GitHub repo into a documentation hub by prioritizing llms.txt, then README. Tools include documentation fetch, documentation search, URL content fetch, and code search.

**Security:** Cloud-hosted, no local operation. Accesses only public content. Self-hostable but designed for cloud use. No authentication, no personal data collection.

**Assessment:** Useful for accessing project-specific documentation but cloud-first. Not suitable as primary local docs source.

### 3l. Nia and Ref.Tools (Commercial)

**Nia** (trynia.ai): Commercial MCP server for "agentic search" providing documentation, codebases, and package search to coding agents. Claimed 27% improvement in Cursor performance. Cloud-hosted, proprietary.

**Ref.Tools** (ref.tools): Commercial MCP server providing `ref_search_documentation` and `ref_read_url` tools. Streamable HTTP (recommended) or local stdio. Indexed documentation resources. Limited public architecture details.

**Assessment:** Both are cloud-hosted commercial alternatives to Context7. Neither serves gdev's local-first requirement.

---

## 4. MCP Server Registries/Catalogs

| Registry | Scale | Documentation Servers | Notes |
|----------|-------|-----------------------|-------|
| **Glama.ai** | 23,000+ total servers | Context7, docs-mcp-server, many others | Largest registry. Auto-indexes from GitHub. Free. |
| **Smithery.ai** | Thousands | Context7 featured prominently | CLI tool for installation. Zero OAuth config. |
| **mcp.so** | 20,000+ | Multiple doc servers listed | Third-party marketplace. |
| **PulseMCP** | Hundreds tracked | Tracks weekly visitor counts | Good for popularity metrics. |
| **Official MCP Registry** | Curated | Reference servers only (Fetch, Filesystem, Git) | No documentation-specific servers in official list. |
| **mcpservers.org** | Curated list | Community-maintained awesome list | Lower scale but higher curation. |

**Key observation:** No registry provides security metadata about MCP servers (local vs. cloud, data access patterns, prompt injection risk). Discovery is by functionality alone.

---

## 5. Security Properties Comparison

### Prompt Injection Risk Matrix

| Server | Fetch at Query Time? | Pre-indexed/Cached? | Official/Trusted Source? | Content Tamper Risk | Sanitization? |
|--------|----------------------|----------------------|--------------------------|---------------------|---------------|
| **Context7** | No (pre-indexed) | Yes (DiskANN + Redis) | Mixed (open registry) | **HIGH** (ContextCrush demonstrated) | Post-patch: partial |
| **DevDocs-MCP (madhan)** | No (after ingest) | Yes (local SQLite + JSON) | Yes (DevDocs.io curated) | Low | None needed (trusted source) |
| **DevDocs-MCP (katvito)** | No (after setup) | Yes (local Docker instance) | Yes (DevDocs.io curated) | Low | None needed |
| **OpenZIM MCP** | No (ZIM files) | Yes (local ZIM) | Yes (Kiwix curated archives) | Low | Input sanitization, path traversal prevention |
| **Grounded Docs** | At index time only | Yes (local index) | Depends on source URLs | Medium (fetched URLs could be poisoned) | None mentioned |
| **man-mcp-server** | No | Yes (system man pages) | Yes (package manager) | Negligible | Timeout protection |
| **godoc-mcp** | No | Yes (local Go docs) | Yes (Go toolchain) | Negligible | Built-in caching |
| **MDN MCP** | Yes | No | Yes (Mozilla first-party) | Low (first-party) | N/A |
| **MCP-NixOS** | Yes (remote APIs) | No | Yes (NixOS infra) | Low (first-party) | N/A |
| **AWS Docs MCP** | Yes | No | Yes (AWS first-party) | Low (first-party) | N/A |
| **Microsoft Learn** | Yes | No | Yes (MS first-party) | Low (first-party) | N/A |
| **GitMCP** | Yes | No | Mixed (any public repo) | Medium (public repos) | robots.txt respected |
| **Rust Docs MCP** | Yes (OpenAI API) | Partial (cargo doc cached) | Yes (crates.io) | Medium (OpenAI dependency) | N/A |

### Security Architecture Categories

**Tier 1 -- Local-only, trusted sources (ideal for gdev):**
- man-mcp-server: System man pages only
- godoc-mcp: Local `go doc` only
- OpenZIM MCP: Local ZIM files only
- DevDocs-MCP (madhan): Local cache after ingest from trusted DevDocs.io

**Tier 2 -- Local after setup, but initial fetch from web:**
- Grounded Docs MCP: Scrapes then indexes locally
- DevDocs-MCP (katvito): Self-hosted DevDocs Docker instance
- Rust Docs MCP: Local cargo doc but OpenAI for search

**Tier 3 -- Cloud-hosted, first-party sources (moderate risk):**
- MDN MCP, Microsoft Learn, AWS Docs, MCP-NixOS: All query official APIs
- Context7: Pre-indexed but open registry with demonstrated vulnerability

**Tier 4 -- Cloud-hosted, mixed sources (higher risk):**
- GitMCP: Any public GitHub repo
- CyberAGI DevDocs: Crawls arbitrary websites
- Nia, Ref.Tools: Commercial cloud services

---

## 6. Gap Analysis

### What existing MCP servers cover well:
- **Individual library docs** (Context7 for 9,000+ libraries, Grounded Docs for arbitrary URLs)
- **Language-specific toolchain docs** (godoc-mcp for Go, Rust Docs MCP for Rust)
- **Vendor platform docs** (AWS, Microsoft, NixOS)
- **System documentation** (man pages)
- **Offline knowledge bases** (OpenZIM for Wikipedia/Stack Overflow ZIM files)

### What is NOT well served:

1. **DevDocs.io as a unified documentation source via MCP:** The existing DevDocs MCP servers are all low-maturity (3-11 stars). None properly leverage DevDocs.io's excellent curated documentation covering 600+ libraries with consistent formatting and search. A production-quality DevDocs MCP that ingests DevDocs JSON datasets, indexes them locally with fuzzy search, and serves them offline would be high-value.

2. **Kiwix/ZIM for Stack Overflow specifically:** OpenZIM MCP exists and is well-built, but it is a general ZIM reader. A purpose-built integration for gdev would pre-configure it with relevant ZIM files (Stack Overflow for specific tags, MDN, NixOS wiki) and handle ZIM file lifecycle (download, update, storage management).

3. **Multi-source documentation routing:** No existing MCP server provides local-first documentation with automatic fallback to web search. Each server is a silo. gdev needs an orchestration layer: query local DevDocs first, then local ZIM/Kiwix, then Context7 as a last resort.

4. **Project-aware documentation:** Context7 and DevDocs-MCP (madhan) attempt version-specific docs, but none deeply integrate with project dependency files across ecosystems (package.json, Cargo.toml, go.mod, flake.nix) to automatically ingest the right documentation.

5. **NixOS documentation locally:** MCP-NixOS (638 stars) is excellent but queries remote APIs. For gdev on NixOS, local access to NixOS options, Home Manager options, and package metadata would be valuable. A ZIM file of wiki.nixos.org + local `nix` CLI queries would close this gap.

6. **Content sanitization for prompt injection:** No existing documentation MCP server sanitizes documentation content for embedded prompt injection attacks. Even trusted sources like DevDocs could theoretically contain malicious content in code examples. This is an unsolved problem across the entire MCP ecosystem.

### Where a custom solution adds value:

A gdev-integrated documentation MCP strategy should:
1. **Use OpenZIM MCP** for Stack Overflow, Wikipedia, and reference documentation ZIM files -- it is mature, well-secured, and fully offline
2. **Build or adopt DevDocs-MCP (madhan)** for language/framework API documentation -- the architecture is right but maturity needs improvement
3. **Include man-mcp-server** for system documentation on NixOS -- zero-cost, zero-risk
4. **Add MCP-NixOS** for NixOS-specific queries (accepting the online dependency since NixOS APIs are first-party)
5. **Relegate Context7** to a clearly-labeled fallback with network filtering, not a primary source
6. **Add a routing/orchestration skill** that queries local sources first and falls back to web-based servers only when local results are insufficient

---

## Sources

All source documents saved to `docs/`:
- `context7-github-readme.md` -- Context7 GitHub repository
- `context7-blog-upstash.md` -- Context7 Upstash blog post
- `context7-faq.md` -- Context7 FAQ with rate limits and library count
- `context7-architecture-chatforest.md` -- Context7 architecture deep dive
- `contextcrush-vulnerability-noma-security.md` -- ContextCrush vulnerability analysis
- `llmian-devdocs-mcp-github.md` -- llmian-space/devdocs-mcp
- `cyberagiinc-devdocs-github.md` -- CyberAGI DevDocs crawler
- `madhan-devdocs-mcp-github.md` -- madhan-g-p/DevDocs-MCP (best DevDocs MCP)
- `katvito-devdocs-mcp-github.md` -- katvito/devdocs-mcp (Docker-based)
- `grounded-docs-mcp-server-github.md` -- arabold/docs-mcp-server
- `mdn-mcp-server-github.md` -- Official MDN MCP server
- `man-mcp-server-github.md` -- man pages MCP server
- `tldr-mcp-server-github.md` -- tldr-pages MCP server
- `godoc-mcp-github.md` -- Go documentation MCP server
- `rust-docs-mcp-server-github.md` -- Rust documentation MCP server
- `microsoft-learn-mcp-server-github.md` -- Microsoft Learn MCP server
- `aws-documentation-mcp-server.md` -- AWS documentation MCP server
- `openzim-mcp-github.md` -- OpenZIM/Kiwix MCP server
- `gitmcp-github.md` -- GitMCP
- `mcp-nixos-github.md` -- MCP-NixOS
- `ref-tools-mcp.md` -- Ref.Tools MCP
