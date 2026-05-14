# Research Summary: gdev Local Documentation MCP Servers

## Overview

Deep dive from first principles into the utility of using or creating MCP servers that integrate with DevDocs and Kiwix (or similar self-hosted Stack Overflow alternatives), providing Claude Code with skills to query local documentation sources first, with automatic failover to regular web search/fetch when local results are insufficient. The primary motivations are: (1) reducing prompt injection surface by avoiding untrusted web content, (2) avoiding bot blocks and rate limits on documentation sites, and (3) improving research reliability and speed. This spike is part of the gdev-secure-devenv-bootstrap implementation plan's security and Claude Code integration story.

## Topics

### Documentation MCP Servers Landscape
- **Status**: Complete
- **Report**: [mcp-landscape-research.md](mcp-landscape-research.md)
- **Summary**: The documentation MCP server ecosystem splits into cloud-hosted servers (Context7, GitMCP, AWS/Microsoft/MDN MCP) that fetch from remote APIs and local-first servers (OpenZIM MCP, DevDocs-MCP, man-mcp-server, godoc-mcp) that operate offline after setup. Context7 (55k stars) is pre-indexed via DiskANN + Redis but suffered the ContextCrush prompt injection vulnerability in Feb 2026 due to its open registry model; free tier rate limits were quietly cut 83%. For gdev's security goals, the recommended stack is: OpenZIM MCP for Stack Overflow/reference ZIM files, madhan-g-p/DevDocs-MCP for language/framework API docs (offline after ingest, 11 stars but right architecture), man-mcp-server for system docs, and MCP-NixOS for NixOS queries. Context7 should be a clearly-labeled fallback only. The key unserved gap is a multi-source routing/orchestration layer that queries local sources first with automatic web fallback.

### DevDocs Architecture, Self-Hosting & MCP Integration
- **Status**: Complete
- **Report**: [devdocs-research.md](devdocs-research.md)
- **Summary**: DevDocs is a two-part system (Ruby scraper + JS/Sinatra web app) that generates documentation as three simple JSON files per doc set: `index.json` (name/path/type triples for search), `db.json` (path-to-HTML-content map), and `meta.json` (version/slug metadata). The format requires no database -- just filesystem access to JSON files. DevDocs has no REST API; all six existing MCP servers either access files directly or wrap the web UI. The Docker image with all 100+ doc sets is ~3.5 GB compressed (99% documentation, 1% application). For gdev, the recommended approach is direct JSON file access from a TypeScript MCP server (following jiegec/devdocs-mcp-server's extraction model) with version-pinning from madhan-g-p/DevDocs-MCP. Local DevDocs provides genuine prompt injection reduction: content is scraped once from official upstream sources, sanitized (scripts/styles stripped by the filter pipeline), versioned, and auditable. Zeal/Dash use a different format (SQLite docsets) with no existing MCP integration.

### Kiwix Architecture, ZIM Libraries & Stack Overflow Alternatives
- **Status**: Complete
- **Report**: [kiwix-zim-research.md](kiwix-zim-research.md)
- **Summary**: ZIM file format is a mature compressed archive with embedded Xapian full-text search. `openzim-mcp` (57 stars, MIT, Python) is a production-quality MCP server that reads ZIM files directly via python-libzim without needing kiwix-serve. python-libzim is the best library for MCP development (pre-built wheels, full search). Full Stack Overflow ZIM is 74 GB (impractical), but curated smaller SE sites total ~4-5 GB. Three other MCP servers exist but are less mature.

### Prompt Injection Surface Analysis
- **Status**: Complete
- **Report**: [prompt-injection-research.md](prompt-injection-research.md)
- **Summary**: Web-fetched content presents a large, actively exploited prompt injection surface for AI coding assistants, with 66-84% attack success rates in auto-execution mode, 32% quarterly growth in observed attacks, and 3 CVEs demonstrating RCE in Copilot/Cursor (CVE-2025-53773, CVE-2025-59944, CVE-2025-54136). Local documentation corpora eliminate the dominant attack vectors (dynamic content injection, MITM, bot blocks, SEO poisoning, visual concealment via CSS/HTML tricks) while introducing smaller, more controllable residual risks centered on upstream supply chain integrity and content quality. Neither ZIM nor DevDocs use cryptographic signing -- integrity depends on download source trust and checksum verification (mitigated by Nix hash pinning). OWASP rates prompt injection as the #1 LLM threat; OpenAI says it is "unlikely to ever be fully solved"; Anthropic's best model-level defense still shows 1% residual attack success. The quantitative threat model strongly favors local-first documentation with web fallback under elevated controls.

### gdev Integration: Deployment, Packaging, Configuration, and Lifecycle
- **Status**: Complete
- **Report**: [gdev-integration-research.md](gdev-integration-research.md)
- **Summary**: Local documentation MCP servers integrate into the gdev implementation plan as additional tools within Phase 12's tool lifecycle system (proposed Units 12.10-12.14). The preferred deployment path uses devenv 2.0's native `claude.code.mcpServers` configuration to generate `.mcp.json`, with Nix-managed Python environments for openzim-mcp and man-mcp-server. Documentation data (ZIM files 4-5 GB, DevDocs JSON per-language) lives in `~/.local/share/gdev/docs/` with Nix hash-pinned downloads for supply chain integrity. Two community Nix flake projects (mcps.nix, mcp-servers-nix) validate the declarative MCP packaging pattern gdev should follow. The wizard auto-detects documentation needs from project files, shows per-option disk costs, defaults to lightweight options (man-pages, DevDocs for detected languages), and makes ZIM files opt-in due to size. Tool lifecycle follows the established `gdev enable/disable` pattern with section markers in shared files, lazy data download (not on enable), and explicit `gdev docs clean` for data removal. Updates use filename-based version comparison for ZIM files and integrate with `gdev outdated`/`gdev update`.

### Failover Architecture: Local-First with Web Fallback
- **Status**: Complete
- **Report**: [failover-architecture-research.md](failover-architecture-research.md)
- **Summary**: The multi-source routing/orchestration layer should use **skill-level routing** (a Claude Code SKILL.md with priority ordering instructions) rather than an MCP-level meta-server. The entire MCP proxy/gateway ecosystem (MetaMCP 2.3k stars, FastMCP, Envoy AI Gateway, combine-mcp, and 6+ others) implements only flat tool catalog aggregation -- none support priority-based routing or failover chains. Claude Code itself has no native priority routing between MCP servers; tool search presents all tools as a flat catalog. A custom meta-MCP server would require 1-2 weeks of development, introduce a single point of failure, add 300-400ms latency, and create a new prompt injection surface. In contrast, a skill-level approach requires only a SKILL.md file that instructs Claude to query local sources (DevDocs, OpenZIM, man pages, MCP-NixOS) in priority order before falling back to web search, with Claude itself serving as the CRAG-style relevance evaluator (replacing the T5 scoring model with contextual reasoning about result quality). The skill includes dynamic context injection via `!gdev docs status` to surface version mismatches and corpus coverage at query time. Five degradation scenarios are analyzed (stale docs, missing libraries, air-gapped environments, conflicting sources, MCP server crashes) with skill-level handling for each. A 4-phase evolution path allows incremental migration to MCP-level routing if the skill proves insufficient.

### Enterprise Azure Self-Hosting for Documentation Corpora
- **Status**: Complete
- **Report**: [azure-enterprise-hosting-research.md](azure-enterprise-hosting-research.md)
- **Summary**: For "too large for local disk" datasets (74 GB Stack Overflow ZIM, 3.5 GB DevDocs, ~100 GB total), the optimal enterprise hosting architecture is Azure Blob Storage with BlobFuse2 local caching. BlobFuse2 (already in nixpkgs as `blobfuse` v2.5.3) mounts Blob Storage as a local filesystem, making ZIM files transparently accessible to openzim-mcp with zero code changes. Its file cache mode downloads entire files to local SSD on first access, providing local-disk performance thereafter. Authentication uses DefaultAzureCredential, which picks up `az login` tokens automatically -- developers authenticate once and MCP servers access Azure storage transparently. A complete Terraform module design provisions storage accounts with Entra ID RBAC (shared keys disabled), private endpoints, and gdev profile integration. Monthly cost is ~$5-8 for the BlobFuse2 approach (20 developers), compared to ~$46-51 for an always-on kiwix-serve Container Apps service. The recommended hybrid approach keeps ~5 GB of curated core docs local (always available offline) while supplementing with Azure-hosted large corpora on demand.

### Multi-Cloud Terraform Abstraction for Documentation Hosting
- **Status**: Complete
- **Report**: [multi-cloud-terraform-research.md](multi-cloud-terraform-research.md)
- **Summary**: The practical approach for cloud-agnostic documentation hosting is per-provider Terraform modules generated by gdev (not a single cloud-agnostic module — HashiCorp and all major module ecosystems explicitly advise against cloud-agnostic modules). The S3 API is the de facto storage abstraction, covering AWS, GCS (via HMAC interop), MinIO, DigitalOcean Spaces, Hetzner, Oracle, Backblaze, Wasabi, and Cloudflare; Azure is the exception requiring BlobFuse2. Rclone mount with `--vfs-cache-mode full` is the recommended universal FUSE tool for ZIM random I/O across all providers (50+ backends, sparse file caching to local SSD). SSO/IAM follows the same cascading credential chain pattern on all clouds (env vars → CLI token → managed identity) — gdev detects the configured cloud CLI rather than abstracting auth. Storage cost is trivial at $0.52-5/month for 100 GB; hosted kiwix-serve adds $25-65/month. The concrete module structure has 5 provider directories (aws/, azure/, gcp/, s3-compatible/, local/) sharing common variable/output contracts, with an air-gapped option supporting MinIO, NFS, or direct disk. The MCP server configuration is identical regardless of provider — only the mount command changes.

## Open Questions

- Can sotoki build tag-filtered Stack Overflow subsets to reduce the 74 GB to something practical?
- ~~How to Nix-package openzim-mcp with its python-libzim native dependency?~~ **Answered**: Use `uv tool install` in devenv enterShell; PyPI wheels bundle native libzim, avoiding fragile nixpkgs libzim.
- ~~How to combine ZIM-based Q&A search with DevDocs API documentation in a unified interface?~~ **Answered**: Skill-level routing via SKILL.md priority ordering. Claude queries the appropriate source based on query type (API docs -> DevDocs, troubleshooting -> ZIM, system commands -> man pages). No unified interface needed; Claude's contextual reasoning handles source selection.
- ~~What is the ZIM file update cadence and can it be automated?~~ **Answered**: 2-3 month cadence for smaller SE sites. Automated via filename year-month comparison against download.kiwix.org directory listings. `gdev docs outdated`/`gdev docs update` commands handle the cycle.
- Should gdev adopt madhan-g-p/DevDocs-MCP directly or build a custom DevDocs MCP? Findings from DevDocs deep dive suggest building a lightweight TypeScript MCP that reads DevDocs JSON files directly (following jiegec model) with version-pinning logic from madhan. The data format is simple enough that a custom server is low effort.
- ~~How should the multi-source routing/orchestration layer work?~~ **Answered**: Skill-level routing via a Claude Code skill (`.claude/skills/lookup-docs/SKILL.md`) that encodes priority ordering, source tagging, and degradation rules. No meta-MCP server needed initially. See `failover-architecture-research.md`.
- No documentation MCP server sanitizes content for embedded prompt injection -- is this a tractable problem or an inherent MCP limitation?
- ~~Should gdev include MCP-NixOS despite its online dependency, given that NixOS APIs are first-party?~~ **Answered**: Yes. MCP-NixOS queries first-party NixOS infrastructure with low prompt injection risk. Auto-enabled when flake.nix or devenv.nix detected. Clearly labeled as online-dependent in CLAUDE.md.
- How many MCP servers is too many? Default config reaches 5-7 servers -- upper end of the 3-6 sweet spot identified in Phase 12 research. May need to drop Context7 if local docs prove sufficient.
- Should devenv.nix generation (Path A) or direct .mcp.json generation (Path B) be the primary code path? Recommendation is Path A when devenv is used, Path B as fallback.
- Will Claude reliably follow skill priority ordering under low effort settings, or does non-determinism require MCP-level enforcement?
- How does BlobFuse2 file cache handle ZIM file updates? (File invalidation when blob is replaced in Azure — needs testing)
- Should the BlobFuse2 mount be a systemd user service or managed by gdev directly? Systemd offers auto-mount on login; gdev offers more control.
- What is the actual first-access latency for a 74 GB ZIM file over a typical office connection? (74 GB @ 1 Gbps = ~10 min; @ 100 Mbps = ~100 min — may need background preload)

## Conclusions

### The case for local-first documentation is strong and quantifiable

Web-fetched content has a 66-84% prompt injection attack success rate in auto-execution mode, with 32% quarterly growth in observed attacks and 3 CVEs demonstrating RCE in production AI coding tools. OWASP ranks it the #1 LLM threat. Local documentation corpora eliminate the dominant attack vectors (dynamic injection, MITM, SEO poisoning, bot blocks) while introducing controllable residual risks mitigated by Nix hash pinning. The ContextCrush vulnerability in Context7 (55K stars, Feb 2026) demonstrates that even the most popular documentation MCP server is not immune. This is not a theoretical concern — it is an active, growing threat that local-first directly addresses.

### The building blocks already exist — gdev's job is curation and orchestration

- **openzim-mcp** (57 stars, MIT, Python) reads ZIM files directly via python-libzim with 21 tools including full-text Xapian search. No kiwix-serve needed.
- **DevDocs** stores docs as 3 simple JSON files per doc set. Direct file access from a TypeScript MCP server is straightforward.
- **BlobFuse2** (already in nixpkgs) makes Azure Blob Storage look like local files to openzim-mcp with zero code changes.
- **rclone mount** is the universal FUSE tool for all other clouds (50+ backends, sparse file caching for ZIM random I/O).
- **Skill-level routing** via a single SKILL.md file handles local-first → web fallback without any meta-MCP server infrastructure.

No new MCP protocol features, proxy servers, or custom routing infrastructure is required.

### Recommended architecture

**Local tier (always available, ~5 GB):** DevDocs JSON for detected project languages + man-mcp-server + MCP-NixOS (if NixOS detected). Installed by `gdev enable docs` with lazy download.

**Enterprise tier (on-demand, ~100 GB via cloud mount):** Full Stack Overflow ZIM + complete DevDocs + curated SE sites, hosted in any cloud via Terraform. BlobFuse2 (Azure) or rclone (everything else) mounts cloud storage as local paths — openzim-mcp and DevDocs MCP see local files. Developer authenticates once via `az login` / `aws sso login` / `gcloud auth login`; DefaultAzureCredential or equivalent picks up the token automatically.

**Fallback tier (web, elevated controls):** Context7 as labeled web fallback when local sources miss. Source tagging in all responses so the user knows where information came from.

**Routing:** A Claude Code skill (`.claude/skills/lookup-docs/SKILL.md`) encodes priority ordering: local DevDocs → local ZIM → man pages → MCP-NixOS → Context7 web fallback. Claude serves as the CRAG-style relevance evaluator. No meta-MCP server needed initially; 4-phase evolution path to MCP-level routing if skill proves insufficient.

### Integration with gdev

Doc MCP servers extend Phase 12 of the implementation plan as Units 12.10-12.14 (tool lifecycle, .mcp.json generation, wizard integration, profile mapping, update mechanism). Enterprise cloud hosting is a Terraform module generated per-provider from gdev profiles, with 5 provider templates (aws/, azure/, gcp/, s3-compatible/, local/) sharing common variable/output contracts. Total enterprise cost: $5-8/month for 20 developers (FUSE mount approach).

### Key risks and open items

1. **Skill routing reliability** — Will Claude consistently follow priority ordering under all effort levels? Non-determinism may require eventual migration to MCP-level enforcement.
2. **First-access latency for large ZIM** — 74 GB over a 100 Mbps connection takes ~100 minutes. Background preloading or a hosted kiwix-serve fallback may be needed.
3. **MCP server count** — Default config reaches 5-7 servers, at the upper end of the 3-6 sweet spot. May need to consolidate or drop Context7 if local sources prove sufficient.
4. **No content signing** — Neither ZIM nor DevDocs use cryptographic signatures. Integrity depends on download source trust + Nix hash pinning.
5. **Stack Overflow content quality** — 50% of code snippets are outdated (CISPA research). Source tagging ("from Stack Overflow ZIM, Nov 2023") is essential so users know when content may be stale.
