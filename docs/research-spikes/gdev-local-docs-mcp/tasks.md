# Tasks: gdev Local Documentation MCP Servers

## Phase 1: Scoping & Initial Research

### Pending

### Active

### Completed (moved from Active)
- [x] **Failover architecture design** — How to implement local-first → web fallback in Claude Code: skill-level routing, MCP server chaining, detection of "no results" vs "partial results", graceful degradation patterns
  - Outcome: success
  - Completed: 2026-05-14
  - Notes: Deep analysis across 6 areas with 9 new sources. Surveyed entire MCP proxy/gateway ecosystem (MetaMCP, FastMCP, Envoy, combine-mcp, 6+ others) — none implement priority-based failover, only flat aggregation. Claude Code has no native routing between servers. Compared skill-level routing (SKILL.md with priority instructions) vs MCP-level routing (meta-MCP server) across 12 factors. Recommendation: skill-level routing — zero infrastructure, leverages Claude's contextual reasoning for result quality evaluation (replacing CRAG's T5 evaluator model), stronger security model with separate failure domains. Full architecture: component diagram, .mcp.json config, complete SKILL.md with dynamic context injection via `!gdev docs status`, CRAG-inspired three-state evaluation, source tagging, 5 degradation scenarios, 4-phase evolution path. Report: `failover-architecture-research.md`, 9 sources in `docs/`.

- [x] **Integration with gdev deployment model** — How this fits into the existing gdev-secure-devenv-bootstrap plan: which phase(s) it belongs in, .mcp.json generation, Nix packaging of DevDocs/Kiwix/ZIM services, profile-driven configuration
  - Outcome: success
  - Completed: 2026-05-14
  - Notes: Comprehensive research across 7 areas. Doc MCP servers extend Phase 12 (Units 12.10-12.14). Two generation paths: devenv.nix claude.code.mcpServers (preferred) or direct .mcp.json. Nix packaging via uv tool install for Python servers, Nix fetchurl with hash pinning for ZIM files. Two community Nix flake projects (mcps.nix, mcp-servers-nix) provide patterns. Wizard shows disk costs, auto-detects doc sets. Profile system encodes doc corpus per team. Tool lifecycle uses section markers in shared files, lazy download, explicit cleanup. Update mechanism via filename-based version comparison. Report: `gdev-integration-research.md`, 8 new sources in `docs/`.

- [x] **Enterprise Azure self-hosting for large corpora** — Terraform-deployed Azure infrastructure for hosting full Stack Overflow ZIM (74 GB), complete DevDocs (3.5 GB), and other large datasets centrally. MCP servers/libraries configured to query Azure-hosted data using SSO via Entra ID (Azure AD). Authentication flow via `az login` or browser-based device code flow.
  - Outcome: success
  - Completed: 2026-05-14
  - Notes: Azure Blob Storage ($1.80/mo for 100 GB Hot tier) + BlobFuse2 (already in nixpkgs v2.5.3) with file cache mode — ZIM files transparently accessible to openzim-mcp at local-disk speed. DefaultAzureCredential picks up `az login` tokens automatically. Hybrid architecture: ~5 GB curated core docs local + Azure-hosted large corpora on demand. ~$5-8/month for 20 devs. Complete Terraform module with Entra ID RBAC, private endpoints, gdev profile integration. Report: `azure-enterprise-hosting-research.md`, 10 sources in `docs/`.

- [x] **Multi-cloud Terraform abstraction for docs hosting** — Extend the Azure-specific Terraform deployment to a cloud-agnostic module supporting AWS, GCP, and other Terraform-supported providers. Abstract storage (S3/GCS/Blob), IAM/SSO (AWS IAM Identity Center, GCP Workforce Identity, Entra ID), mount patterns (s3fs/gcsfuse/BlobFuse2), and compute (ECS/Cloud Run/ACI). Goal: gdev can deploy centralized doc hosting on any cloud the adopting org uses.
  - Outcome: success
  - Completed: 2026-05-14
  - Notes: Deep research across 9 areas with 13 sources. Key findings: (a) Per-provider Terraform modules with common interface contract, not cloud-agnostic modules — matches HashiCorp official guidance and all major module ecosystems (CloudPosse, Gruntwork). (b) S3 API covers ~90% of providers; Azure is the exception needing BlobFuse2. (c) rclone mount with --vfs-cache-mode full is the universal FUSE tool for ZIM random I/O (50+ backends, sparse file caching). (d) All clouds use identical credential chain cascade — gdev detects configured CLI. (e) Storage costs trivial ($0.52-5/mo for 100 GB); hosted kiwix-serve adds $25-65/mo. (f) Concrete module structure with 5 provider directories (aws/azure/gcp/s3-compatible/local/) sharing common variable/output contracts. (g) Air-gapped option via MinIO, NFS, or direct disk — same MCP config regardless of provider. Report: `multi-cloud-terraform-research.md`, 13 sources in `docs/`.

### Completed
- [x] **Prompt injection surface analysis** — Quantify the prompt injection risk of web fetch vs local documentation: what vectors exist in web-fetched docs, how local sources eliminate or reduce them, residual risks in local corpora
  - Outcome: success
  - Completed: 2026-05-14
  - Notes: Deep analysis across 6 areas with 13 sources. Web fetch shows 66-84% attack success rates, 32% quarterly growth in observed attacks, 3 CVEs demonstrating RCE in Copilot/Cursor. Local-first eliminates dynamic injection, MITM, bot blocks, SEO poisoning. Residual risks (upstream supply chain, content quality) are lower-likelihood and mitigated by Nix hash pinning and source tagging. OWASP #1 threat; OpenAI says "unlikely to ever be fully solved"; Anthropic's best defense still 1% residual. Quantitative threat model strongly favors local-first with web fallback. Report: `prompt-injection-research.md`, 13 sources in `docs/`.

- [x] **DevDocs architecture & self-hosting** — How DevDocs works (data format, indexing, API), self-hosting options, existing MCP servers for DevDocs, what documentation it covers, update mechanisms
  - Outcome: success
  - Completed: 2026-05-14
  - Notes: Data format is three JSON files per doc set (index.json, db.json, meta.json). No REST API exists. Docker image with all docs is ~3.5 GB. Six MCP servers surveyed; jiegec/devdocs-mcp-server (extract from Docker, no server needed) and madhan-g-p/DevDocs-MCP (version-pinning from package.json) are strongest. Recommended: direct JSON file access from TypeScript MCP server, no DevDocs web server. Report: `devdocs-research.md`, 19 sources saved to `docs/`.

- [x] **Existing documentation MCP servers landscape** — Survey of existing MCP servers that serve documentation (Context7, devdocs-mcp, etc.), their capabilities, limitations, security properties
  - Outcome: success
  - Completed: 2026-05-14
  - Notes: Surveyed 16+ servers across 6 registries. Context7 (55K stars) suffered ContextCrush prompt injection (Feb 2026) and quiet 83% rate limit cut — validates local-first approach. Recommended stack: OpenZIM MCP + madhan-g-p/DevDocs-MCP (offline SQLite+JSON) + man-mcp-server + MCP-NixOS, with Context7 as labeled fallback only. Key gap: no multi-source routing/orchestration layer exists. Report: `mcp-landscape-research.md`, 20 sources saved to `docs/`.

- [x] **Kiwix architecture, Stack Overflow alternatives & ZIM direct access** — How Kiwix works (ZIM format, search API), self-hosting Stack Overflow dumps, alternative Q&A corpus options, existing MCP servers. ZIM parsing libraries across languages for direct file access without self-hosting.
  - Outcome: success
  - Completed: 2026-05-14
  - Notes: `openzim-mcp` (57 stars, MIT, v2.0.0a12) reads ZIM files directly via python-libzim — no kiwix-serve needed. 21 tools including full-text search. python-libzim is best library (pre-built wheels). Full SO ZIM is 74 GB (impractical), but curated SE sites ~4-5 GB are practical. Report: `kiwix-zim-research.md`, 20 sources saved to `docs/`.
