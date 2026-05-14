# DevDocs Architecture, Self-Hosting, and MCP Integration

## Executive Summary

DevDocs is a mature, open-source documentation browser maintained by freeCodeCamp that aggregates 100+ documentation sets into a unified, searchable interface. Its architecture is surprisingly simple: a Ruby scraper generates normalized HTML partials and JSON index files, which a client-side JavaScript app serves via a thin Sinatra layer. The data format — just three JSON files per doc set — is well-suited for programmatic access. Multiple MCP server implementations already exist, with varying maturity levels. The strongest candidate for gdev integration is a direct-file-access approach using DevDocs' generated JSON data, avoiding the need to run a full DevDocs web server. This approach provides genuine prompt injection risk reduction by serving documentation from a curated, locally-cached corpus rather than fetching from arbitrary web sources at query time.

## 1. DevDocs Internals

### Architecture

DevDocs is a two-part system:

1. **Scraper** (Ruby): Generates documentation files and metadata. Written under the `Docs` module, uses Typhoeus for HTTP, Nokogiri for HTML parsing, and HTML::Pipeline for filter processing.
2. **Web App** (JavaScript + Sinatra): All client-side JS, powered by a minimal Sinatra/Sprockets backend. Uses XHR to load content into the main frame. Service workers and localStorage enable offline functionality.

The design is driven by a key constraint: content is loaded via XHR into the main frame, so original HTML is stripped of most markup (to avoid polluting the frame) and all CSS classes are prefixed with underscores (to prevent conflicts).

### Scraper System

Two scraper types exist:

- **UrlScraper**: Downloads documentation via HTTP from a `base_url`, recursively following internal links
- **FileScraper**: Reads from the local filesystem, substituting `base_url` with a local `dir` path

Both pass each page through a filter pipeline:

**HTML Filters** (operate on Nokogiri DOM):
1. `ContainerFilter` — isolates content within a CSS selector (strips everything outside)
2. `CleanHtmlFilter` — removes scripts, styles, comments
3. `NormalizeUrlsFilter` — converts all URLs to fully qualified form
4. `InternalUrlsFilter` — identifies internal links, converts to relative paths
5. `NormalizePathsFilter` — ensures path consistency
6. `CleanLocalUrlsFilter` — removes localhost references (FileScraper only)

**Text Filters** (operate on HTML strings):
1. `InnerHtmlFilter` — converts DOM to string
2. `CleanTextFilter` — removes empty nodes
3. `AttributionFilter` — appends license/copyright info

Each scraper must implement `EntriesFilter`, which extracts page metadata by overriding `get_name`, `get_type`, `include_default_entry?`, and `additional_entries`. This is the filter that builds the searchable index.

Scrapers are located in `lib/docs/scrapers/` and can be extended with custom filters inserted before/after/replacing default ones. Abstract scrapers (e.g., `Mdn`) share behavior across related documentation sets.

### Data Format

Each documentation set generates three files in a directory named by slug (e.g., `python~3.12/`):

**`index.json`** — The searchable index:
```json
{
  "entries": [
    { "name": "list", "path": "library/functions#list", "type": "Built-in Functions" },
    { "name": "dict.items", "path": "library/stdtypes#dict.items", "type": "Mapping Types" }
  ],
  "types": [
    { "name": "Built-in Functions", "slug": "built-in-functions", "count": 69 }
  ]
}
```

**`db.json`** — Page content database:
```json
{
  "library/functions": "<article>...normalized HTML...</article>",
  "library/stdtypes": "<article>...normalized HTML...</article>"
}
```

**`meta.json`** — Documentation metadata:
```json
{
  "name": "Python",
  "slug": "python~3.12",
  "type": "python",
  "version": "3.12",
  "release": "3.12.4",
  "mtime": 1719849600,
  "db_size": 4521890
}
```

A global **`docs.json`** manifest lists all available documentation sets with their metadata plus attribution strings.

Version slugs are sanitized: lowercase, `+` becomes `p`, `#` becomes `s`, non-alphanumeric becomes `_`. Versioned slugs use the format `base~version` (e.g., `react~18`).

### Documentation Coverage

DevDocs includes 100+ documentation sets with multiple versions. Coverage spans:

- **Languages**: Bash, C, C++, Clojure, Crystal, D, Dart, Elixir, Erlang, Go, Groovy, Haskell, Julia, Kotlin, Lua, Perl, PHP, Python, Ruby, Rust, Scala, Swift, TypeScript, Zig
- **Web Frameworks**: Angular, Django, Express, FastAPI, Flask, Laravel, Next.js, Nuxt, Phoenix, Rails, React, Svelte, Vue
- **Frontend**: Bootstrap, D3, Ember, HTMX, jQuery, Knockout, Backbone
- **Runtime/Tools**: Bun, Deno, Node.js, Docker, Git, CMake, Webpack, ESBuild, ESLint
- **Databases**: CouchDB, DuckDB, Elasticsearch, MariaDB, MongoDB, PostgreSQL, Redis, SQLite
- **DevOps**: Ansible, Chef, HAProxy, Nginx, Packer, Puppet, Salt, Terraform, Vagrant
- **Web Standards (via MDN)**: HTML, CSS, JavaScript, Web APIs, HTTP, SVG

The MDN scraper alone covers a massive surface area of web standards documentation.

## 2. Self-Hosting DevDocs

### Manual Installation

Requirements:
- Ruby 3.4.1+
- libcurl
- Node.js (JavaScript runtime)

Process:
```bash
git clone https://github.com/freeCodeCamp/devdocs.git
cd devdocs
gem install bundler
bundle install
thor docs:download --default   # popular docs only
# or: thor docs:download --all  # ALL docs
bundle exec rackup              # starts on localhost:9292
```

### Docker

Two official images from GitHub Container Registry:
- `ghcr.io/freecodecamp/devdocs:latest` — standard
- `ghcr.io/freecodecamp/devdocs:latest-alpine` — smaller base

**Image size with ALL docs: ~3.5 GB compressed** (the documentation content layer is ~3.48 GB; the application itself is only ~43 MB). Images update monthly.

### Key Thor Commands
```bash
thor docs:list                    # list available documentations
thor docs:download --default      # download popular docs
thor docs:download --installed    # update installed docs
thor docs:download --all          # download everything
thor docs:generate <name>         # scrape documentation from source
```

### Resource Requirements

Not formally documented, but derived from architecture:
- **CPU**: Minimal at runtime (static file serving)
- **Memory**: Low — Sinatra is lightweight; the main cost is if generating/scraping docs
- **Disk**: Depends on documentation selection. Full set is ~3.5 GB compressed, likely 5-8 GB uncompressed. A curated subset (e.g., 20 doc sets) would be hundreds of MB.
- **Network**: Only needed for initial download and updates

### Update Mechanism

No built-in auto-update. Options:
- `git pull origin main` for code updates
- `thor docs:download --installed` to refresh documentation
- Docker image re-pull (monthly updates)
- The `get_latest_version` method on each scraper enables version checking, reported via periodic "Documentation versions report" issues

### Selective Documentation

Documentation sets can be individually selected. For gdev, this is critical — instead of pulling all 3.5 GB, a team can define a documentation profile (e.g., TypeScript + React + Node + PostgreSQL) and pull only those sets.

## 3. DevDocs API / Query Interface

### No Formal API Exists

DevDocs has **no REST API** for programmatic queries. This is a long-standing feature request (GitHub Issue #133, opened 2014, never implemented). The OpenSearch URL exists but requires JavaScript execution, making it incompatible with curl/HTTP clients.

### Programmatic Access Approaches

Since there's no API, MCP servers use one of three approaches:

1. **Direct file access** (recommended): Read `index.json` for search, `db.json` for content. No server needed.
2. **HTTP access to static files**: DevDocs serves its JSON files at predictable URLs (`/docs/<slug>/index.json`, `/docs/<slug>/db.json`), but devdocs.io blocks automated access (403). A self-hosted instance would work.
3. **Docker extraction**: Extract documentation files from the DevDocs Docker image without running the server.

### Client-Side Search

DevDocs' search is entirely client-side JavaScript. It loads `index.json` into memory and performs fuzzy matching on entry names. This is fast because the index is pre-built and relatively small. An MCP server would replicate this logic.

## 4. Existing DevDocs MCP Servers

Six implementations were found on GitHub. They vary significantly in maturity and approach:

### Tier 1: Production-Ready

**[jiegec/devdocs-mcp-server](https://github.com/jiegec/devdocs-mcp-server)** — Python
- **Approach**: Extracts docs from DevDocs Docker image into local files. No running DevDocs server needed.
- **Tools**: `search_devdocs` (fuzzy matching), `read_devdocs` (file retrieval), `list_doc_sets`
- **Transport**: stdio and HTTP
- **HTML-to-Markdown conversion**: Automatic
- **CLI**: `devdocs search`, `devdocs read`, `devdocs list-sets`
- **Assessment**: Clean, minimal, does one thing well. Best model for gdev.

**[madhan-g-p/DevDocs-MCP](https://github.com/madhan-g-p/DevDocs-MCP)** — NestJS/TypeScript
- **Approach**: Lazy-ingestion engine that downloads DevDocs structured datasets on-demand. SQLite metadata with cached JSON.
- **Tools**: Ingest, Search (ranked fuzzy), Explain (version-aware retrieval)
- **Key differentiator**: Version-pinning from `package.json` — serves docs matching your project's actual dependency versions
- **Docker**: Available (`madhandock1/devdocs-mcp:latest`)
- **Assessment**: Most sophisticated version-awareness. Node-only (no Python/C++ deps). Good for version-sensitive work.

### Tier 2: Functional but Different Goals

**[katvito/devdocs-mcp](https://github.com/katvito/devdocs-mcp)** — JavaScript/TypeScript
- **Approach**: Docker Compose running DevDocs + MCP server. Queries live DevDocs instance via HTTP.
- **Tools**: `view_available_docs`, `search_specific_docs`
- **Limitation**: Requires running a full DevDocs Docker container. 10+ minute initial download.
- **Assessment**: Heaviest approach. Clean architecture but over-engineered for what's needed.

**[cyberagiinc/DevDocs](https://github.com/cyberagiinc/DevDocs)** — Python/TypeScript (confusingly named)
- **Approach**: General-purpose documentation crawler (Crawl4AI) with MCP server. NOT specifically for DevDocs data.
- **Tools**: Table of Contents, Section Access
- **Key differentiator**: Crawls ANY documentation URL, not just DevDocs. UI for managing crawled docs.
- **Status**: "Not publicly maintained. Enhanced internal version at CyberAGI."
- **Assessment**: Different tool entirely — it's a web scraper with MCP, not a DevDocs accessor.

### Tier 3: Early Stage / Incomplete

**[llmian-space/devdocs-mcp](https://github.com/llmian-space/devdocs-mcp)** — Python
- Skeleton project with URI template system. Not production-ready.

**[kelvinzer0/DevDocsMCP](https://github.com/kelvinzer0/DevDocsMCP)** — Accepts language slugs. Minimal documentation.

### Comparison with Context7

**[Context7](https://github.com/upstash/context7)** (55k+ GitHub stars) is the dominant MCP documentation server but takes a fundamentally different approach:

| Aspect | DevDocs MCP | Context7 |
|--------|------------|----------|
| Data location | Local filesystem | Cloud (context7.com) |
| Internet required | No (after setup) | Yes (always) |
| Documentation source | Official DevDocs scrapers | Proprietary crawling engine |
| Version awareness | Via slug (e.g., `python~3.12`) | Automated detection |
| Security model | Local files, no network | Cloud API, API keys |
| Prompt injection surface | Minimal (curated, local) | Higher (cloud, community-contributed) |
| Coverage | 100+ doc sets | Broader (any library) |
| Backend | Open source | Proprietary/closed |

**For gdev's security goals, Context7 is unsuitable**: it's cloud-first, proprietary backend, and explicitly warns "cannot guarantee the accuracy, completeness, or security of all library documentation." Local DevDocs is the right approach.

## 5. Alternative Local Documentation Tools

### Zeal

[Zeal](https://zealdocs.org/) is an open-source offline documentation browser (C++/Qt) inspired by Dash. Key characteristics:

- **Docset format**: Uses the Dash docset standard (different from DevDocs format)
- **Storage**: SQLite database with HTML files in a `.docset` bundle
- **Search**: Desktop UI only, command-line launch with query (`zeal python:pprint`)
- **No server mode**: Purely a desktop application. No API, no headless mode, no MCP integration
- **Platform**: Linux, Windows (Qt 6.4.2+, requires Qt WebEngine)

**Dash Docset Format** (used by Zeal, Dash, Velocity):
```
example.docset/
├── Contents/
│   ├── Info.plist          # XML metadata
│   └── Resources/
│       ├── Documents/      # HTML files
│       └── docSet.dsidx    # SQLite index
```

SQLite schema: `CREATE TABLE searchIndex(id INTEGER PRIMARY KEY, name TEXT, type TEXT, path TEXT)`

The Dash format supports 90+ entry types (Class, Function, Method, Property, etc.) and has a richer type taxonomy than DevDocs.

### doc2dash

[doc2dash](https://doc2dash.hynek.me/) converts Sphinx/intersphinx documentation to Dash docsets. Stand-alone binaries available. Supports Sphinx, pydoctor, MkDocs (with mkdocstrings). Could generate docsets for custom internal documentation.

### Comparison: DevDocs vs Dash/Zeal Format

| Aspect | DevDocs | Dash/Zeal |
|--------|---------|-----------|
| Index format | JSON (`index.json`) | SQLite (`docSet.dsidx`) |
| Content format | JSON blob (`db.json`) | Individual HTML files |
| Type taxonomy | Free-form strings | 90+ standardized types |
| Programmatic access | Read JSON files | Query SQLite DB |
| Server mode | Yes (Sinatra) | No |
| MCP servers exist | Yes (multiple) | No |
| Update mechanism | Thor commands / Docker | Feed XML / manual |
| Custom doc generation | Ruby scraper framework | doc2dash, custom tools |

**For MCP integration, DevDocs format is superior**: JSON is trivially parseable, no SQLite dependency needed, and multiple MCP servers already exist. The Dash format would require building an MCP server from scratch and adds a SQLite dependency.

## 6. Security Properties

### Documentation Sourcing

DevDocs documentation is sourced from official upstream projects:
- Scrapers target official documentation URLs (e.g., `docs.python.org`, `developer.mozilla.org`)
- The `AttributionFilter` appends copyright and license information from the original source
- Each scraper explicitly defines its `base_url`, limiting crawl scope to official domains
- Content processing strips scripts, styles, and comments (`CleanHtmlFilter`)
- Only HTML content type with 200 status codes is accepted

However, there is no content verification or signing mechanism. DevDocs trusts that the upstream documentation sites serve legitimate content.

### Content Sanitization

The filter pipeline provides meaningful sanitization:
1. Scripts and styles are removed
2. External resources (iframes, images pointing to localhost) are cleaned
3. HTML is normalized to partials (no full page structure)
4. Content is stripped to the container element only
5. Attribution is appended (provides provenance)

This is primarily for display hygiene, not security. There is no explicit check for prompt injection payloads in documentation content.

### Prompt Injection Risk Analysis

**Web fetch (current approach) — HIGH RISK:**
- Every web fetch exposes the LLM to arbitrary web content
- Documentation sites may contain user-generated content (comments, examples)
- CDN/proxy compromise could serve modified content
- Bot detection pages inject unexpected content into the context
- Attackers can embed invisible instructions in web pages (Perplexity Comet incident)

**Local DevDocs — REDUCED RISK:**
- Documentation is fetched once, at setup time, not at query time
- Content comes from official upstream sources via curated scrapers
- The scraper's HTML sanitization strips scripts and styles
- No user-generated content in the corpus (official docs only)
- No dynamic content fetching during LLM interaction
- Content is versioned and auditable (diff between updates)

**Residual risks with local DevDocs:**
- Upstream documentation could be compromised at scrape time
- DevDocs Docker images are pre-built by freeCodeCamp (supply chain trust)
- HTML content in `db.json` could theoretically contain prompt injection strings, though official documentation is unlikely to contain them
- Attribution strings include HTML, which could be a vector if crafted
- No content signing or integrity verification beyond HTTPS during scrape

### Mitigation Recommendations for gdev

1. **Pin documentation versions**: Use specific doc versions rather than `latest`
2. **Strip HTML before LLM ingestion**: Convert HTML from `db.json` to plain text/markdown before sending to the LLM
3. **Selective documentation**: Only install documentation sets the team actually uses
4. **Audit updates**: Diff documentation content between version updates
5. **Prefer official Docker images**: Use `ghcr.io/freecodecamp/devdocs` rather than building from scratch
6. **Consider content hashing**: Store hashes of `db.json` content to detect unexpected changes

## 7. Recommended Architecture for gdev

Based on this research, the optimal approach for gdev is:

### Direct File Access (No DevDocs Server)

1. **Extract documentation** from the DevDocs Docker image or use `thor docs:download` to get JSON files
2. **Build a lightweight MCP server** (TypeScript, matching gdev's stack) that:
   - Reads `index.json` for search (fuzzy matching on entry names)
   - Reads `db.json` for content retrieval (convert HTML to markdown for LLM consumption)
   - Supports `list_doc_sets`, `search_docs`, `read_doc` tools
3. **Profile-driven documentation selection**: Team `.gdev` config specifies which doc sets to install
4. **Nix packaging**: Package the MCP server and documentation download as a Nix derivation
5. **Update mechanism**: Periodic `thor docs:download --installed` or Docker image re-pull

### Why Not Run DevDocs Server?

- Unnecessary overhead — the Sinatra server adds Ruby runtime dependency for what's just JSON file reading
- The server has no API anyway — MCP servers that wrap it are just HTTP-scraping the web interface
- Direct file access is simpler, faster, and has fewer failure modes
- gdev is TypeScript-based; adding a Ruby dependency for the server is undesirable

### Model: jiegec/devdocs-mcp-server

The `jiegec/devdocs-mcp-server` approach is the closest to what gdev needs:
- Extracts docs from Docker image (no running server)
- Provides search, read, and list tools
- Converts HTML to Markdown automatically
- Supports both stdio and HTTP transport

A TypeScript port of this approach, with version-pinning from `madhan-g-p/DevDocs-MCP` added, would be ideal.

## Sources

All source documents are saved in `docs/`:
- `devdocs-github-readme.md` — Main project README
- `devdocs-scraper-reference.md` — Complete scraper documentation
- `devdocs-filter-reference.md` — Filter system documentation
- `devdocs-source-code-data-format.md` — Data format from source code analysis
- `devdocs-doc-class.md` — Doc class structure
- `devdocs-scraper-coverage.md` — Full list of available documentation
- `devdocs-dockerfile-analysis.md` — Docker build process
- `devdocs-docker-image-size.md` — Container image sizing
- `devdocs-api-issue-133.md` — API/programmatic access discussion
- `katvito-devdocs-mcp.md` — katvito MCP server
- `cyberagiinc-devdocs-mcp.md` — CyberAGI MCP server
- `jiegec-devdocs-mcp-server.md` — jiegec MCP server
- `llmian-devdocs-mcp.md` — llmian MCP server
- `madhan-devdocs-mcp.md` — madhan MCP server (version-pinning)
- `context7-platform.md` — Context7 comparison
- `zeal-offline-docs-browser.md` — Zeal documentation browser
- `dash-docset-format-spec.md` — Dash docset format specification
- `indirect-prompt-injection-lakera.md` — Indirect prompt injection analysis
- `owasp-prompt-injection-llm01.md` — OWASP LLM security guidance
