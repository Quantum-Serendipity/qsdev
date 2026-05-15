# mcph MCP Orchestrator — Evaluation for gdev Integration

## 1. Executive Summary

**mcph** (`@yawlabs/mcph`) is a TypeScript-based MCP server orchestrator by Yaw Labs that acts as a proxy between AI clients (Claude Code, Cursor, VS Code) and multiple upstream MCP servers. It provides cloud-managed server registry, intelligent routing, health monitoring, credential injection, and compliance grading. After thorough evaluation, mcph solves a **different problem** than gdev's Unit 3.5.1 McpServerDef registry and is **not a viable replacement or foundation** for it. mcph is a runtime orchestrator that manages live server connections at execution time; gdev's registry is a build-time code generator that produces static `.mcp.json` files during project initialization. The two operate at different layers and serve different purposes.

However, two components from the Yaw Labs ecosystem have genuine value for gdev: the **mcp-compliance test suite** (88 tests, MIT licensed) for validating MCP server configurations, and the **aws-mcp server** as a potential catalog entry.

**Verdict: Do not adopt mcph. Cherry-pick the compliance suite and evaluate aws-mcp as a catalog addition.**

---

## 2. mcph Architecture

### 2.1 What It Is

- **TypeScript CLI tool and MCP server proxy** (not a library, not a daemon)
- **npm package**: `@yawlabs/mcph` v0.47.5
- **Language**: TypeScript (~935K LoC including tests)
- **Dependencies**: `@modelcontextprotocol/sdk` (^1.29.0), `undici` (^7.8.0)
- **Runtime**: Node.js 18+
- **License**: **None declared** on the mcph repo itself (critical concern)
- **Created**: April 8, 2026 (5 weeks old)
- **Last commit**: May 14, 2026
- **Contributors**: 1 (jeffyaw, 137 commits)
- **Stars**: 1
- **Forks**: 0
- **Open issues**: 0

### 2.2 How It Works

mcph registers itself as a single MCP server entry in the AI client's configuration (`.mcp.json` or equivalent). At runtime, it:

1. **Polls mcp.hosting** every 60 seconds for the user's server inventory (config sync)
2. **Exposes 11 meta-tools** to the AI client (discover, activate, deactivate, dispatch, install, import, health, suggest, read_tool, exec, bundles)
3. **Spawns upstream servers on demand** via stdio (local) or connects via HTTP/SSE (remote)
4. **Proxies tool calls** from the AI client to the appropriate upstream server, namespacing tool names to prevent collisions
5. **Tracks health** per server (call counts, errors, latency)
6. **Ranks servers** via BM25 (optionally reranked with Voyage embeddings) for task-based dispatch

### 2.3 Configuration Model

Three-scope config precedence:
1. `<project>/.mcph/config.local.json` (machine-local, gitignored)
2. `<project>/.mcph/config.json` (team-shared, committed)
3. `~/.mcph/config.json` (personal default)

Config schema is minimal: `version`, `token`, `apiBase`, `servers` (allow-list), `blocked` (deny-list). The actual server definitions live on mcp.hosting's cloud backend, not in local config files.

### 2.4 Server Management

Servers are defined by: `id`, `name`, `namespace`, `type` (local/remote), `transport` (stdio/streamable-http/sse), `command`, `args`, `env`, `url`, `description`, `toolCache`, `complianceGrade`.

Credential injection happens at spawn time from encrypted cloud storage. Missing credential detection uses heuristic stderr pattern matching (regex for common "missing env var" patterns).

---

## 3. Feature Coverage Comparison

| Feature | gdev Unit 3.5.1 (Our Design) | mcph |
|---------|------|------|
| **Server registry/catalog** | Go struct registry, compile-time, 14 hardcoded definitions with rich metadata | Cloud-hosted registry on mcp.hosting, unlimited entries, minimal metadata |
| **Server lifecycle** | Build-time: generates static `.mcp.json` during `qsdev init` | Runtime: spawns/kills servers on demand via meta-tools |
| **Config composition** | Generates `.mcp.json` as a file artifact | Replaces `.mcp.json` entirely -- mcph IS the only entry |
| **Tool budget tracking** | Explicit 40-tool ceiling with `CanEnable()` checks | Implicit via `MCPH_SERVER_CAP` (default 6 concurrent) and context cost estimates |
| **Auth/credential management** | SecretSpec integration (keyring, env, 1Password) | Cloud-encrypted credentials on mcp.hosting + heuristic stderr detection |
| **Transport selection** | Per-server in generated config (stdio vs URL) | Runtime negotiation (stdio, streamable-http, sse) |
| **Detection heuristics** | Project file scanning (`*.tf`, `services.mysql.enable`, etc.) | None -- user manually installs/imports servers |
| **Security tiering** | Three-tier model (Low/Medium/High) with policy enforcement | A-F compliance grading from test suite |
| **Offline operation** | Fully offline (static files) | Requires mcp.hosting connectivity (polls every 60s) |
| **Multi-client sync** | N/A (generates per-client config) | Core feature -- one config syncs to all clients/devices |
| **Health monitoring** | None (static config) | Per-server call counts, errors, latency |
| **Intelligent routing** | N/A (all enabled servers always active) | BM25 + optional semantic ranking |

### 3.1 Key Architectural Difference

**gdev operates at build time**: `qsdev init` scans a project, detects signals, runs a wizard, and generates static configuration files. The output is a `.mcp.json` that Claude Code reads directly. gdev is not running when Claude Code is working.

**mcph operates at runtime**: It IS the MCP server that Claude Code talks to. It interposes itself between the AI client and all upstream servers, routing calls dynamically. It must be running for any MCP server to work.

This is a fundamental difference. Adopting mcph would mean gdev no longer generates `.mcp.json` entries for individual servers -- instead, gdev would generate a single mcph entry and push server configurations to mcp.hosting (or a self-hosted instance). This is a wholesale architectural change, not a library swap.

---

## 4. Compliance Test Suite

The **mcp-compliance** package (`@yawlabs/mcp-compliance`, MIT licensed) is the most valuable artifact in the Yaw Labs ecosystem for gdev's purposes.

### 4.1 What It Tests

88 tests across 8 categories:
1. **Transport** -- HTTP-specific (CORS, TLS, session headers, rate limiting)
2. **Lifecycle** -- initialization, capability negotiation, shutdown
3. **Tools** -- tool listing, invocation, schema validation
4. **Resources** -- resource listing, reading, subscriptions
5. **Prompts** -- prompt listing, rendering
6. **Error handling** -- error codes, malformed input recovery
7. **Schema validation** -- JSON Schema compliance of tool inputs/outputs
8. **Security** -- various security posture checks

HTTP targets run all 85 transport-applicable tests; stdio targets run ~75.

### 4.2 Grading

- Weighted score: required tests 70%, optional 30%
- Letter grades A-F
- `--strict` mode for CI (exit 1 on required failure)
- `--min-grade` threshold enforcement

### 4.3 Relevance to gdev

**High value for Phase 17 (Test Infrastructure Framework)**. gdev could:
- Run compliance tests against each MCP server in the registry during CI
- Gate server additions on minimum compliance grade
- Generate compliance badges for the server catalog
- Use the programmatic API to integrate compliance checking into `qsdev mcp check`

**Limitation**: The compliance suite tests MCP protocol compliance, not the correctness of gdev's `.mcp.json` generation. It validates "does this server speak MCP correctly?" not "did qsdev configure this server correctly?"

### 4.4 Integration Path

```bash
# Test a server gdev would configure
npx @yawlabs/mcp-compliance test npx @hashicorp/terraform-mcp-server --strict --min-grade B

# JSON output for programmatic consumption
npx @yawlabs/mcp-compliance test npx @hashicorp/terraform-mcp-server --format json
```

Could be integrated into `qsdev mcp check` as an optional deep validation: "Does the configured server actually work and speak compliant MCP?"

---

## 5. Open-Source MCP Servers Assessment

Yaw Labs publishes 10 MCP servers, all TypeScript, all MIT licensed (except mcph itself).

### 5.1 Servers Relevant to Consulting Workflows

| Server | Stars | Tools | Relevance to gdev | Assessment |
|--------|-------|-------|--------------------|------------|
| **aws-mcp** | 1 | 24 | **High** -- AWS is ubiquitous in consulting | SSO re-auth solves real pain point; 24 tools is large; Cloud Control API coverage is broad |
| **tailscale-mcp** | 22 | 89+4 | **Medium** -- relevant for teams using Tailscale | Very well-engineered (700+ tests, Zod validation, safety hints); tool count too high for gdev's 40-tool ceiling |
| **ssh-mcp** | 3 | ? | **Low-Medium** -- SSH is universal but niche for MCP | Diagnostics are useful; most devs don't need AI-mediated SSH |
| **postgres-mcp** | 1 | ? | **Low** -- gdev already has PostgreSQL MCP | Read-only-by-default is good design; no reason to switch |
| **caddy-mcp** | 5 | ? | **Low** -- Caddy is niche | Good engineering but limited consulting relevance |
| **npmjs-mcp** | 1 | ? | **Low-Medium** -- npm intelligence for AI | Package security analysis could complement Socket.dev |
| **fetch-mcp** | 1 | ? | **Low** -- generic HTTP fetch | SSRF protection is good; competitors exist |

### 5.2 aws-mcp Deep Dive

The most interesting server for gdev. Key differentiators from awslabs/mcp (the official AWS MCP):

- **SSO re-auth inside AI sessions**: Solves the "browser won't open from subprocess" problem that plagues every AWS CLI tool running inside an AI agent. Uses device-code flow.
- **Cloud Control API**: Generic CRUD across hundreds of resource types without per-service tool implementations
- **Dry-run diffing**: Shows what would change before applying
- **Multi-region parallel execution**: Useful for consulting teams managing multi-region deployments
- **JavaScript sandbox scripting**: Complex multi-step AWS workflows in one tool call

**Concern**: Requires Node.js 22+ (higher than mcph's 18+ requirement). 24 tools is 60% of gdev's 40-tool budget -- too many for a single server in the default configuration.

### 5.3 Quality Assessment

All Yaw Labs servers share common traits:
- TypeScript with Zod validation
- Safety hints (readOnlyHint, destructiveHint) per MCP spec
- Consistent error handling patterns
- Active development (all updated May 2026)

Negative signals:
- Very low star counts (1-22) across all repos
- Single contributor across the entire org
- No evidence of production usage beyond Yaw Labs themselves
- 5 weeks old (entire org created April 2026)

---

## 6. Embeddability Assessment

### 6.1 Can gdev use mcph as a library?

**No.** mcph is a monolithic CLI application, not a library. Key reasons:

1. **No exported API surface**: The package exports only a CLI entry point (`./dist/index.js`). There are no library exports.
2. **Cloud dependency**: Core functionality requires a mcp.hosting account and API token. The `fetchConfig()` function calls `https://mcp.hosting` and cannot be redirected without env var overrides.
3. **Language mismatch**: gdev is Go; mcph is TypeScript. Even if mcph exported a library API, gdev couldn't consume it directly.
4. **Runtime model mismatch**: mcph is a long-running proxy process; gdev is a one-shot CLI that generates files and exits.

### 6.2 Could gdev wrap mcph?

Theoretically, gdev could:
1. Install mcph as a single `.mcp.json` entry
2. Push server definitions to mcp.hosting via API
3. Let mcph handle runtime orchestration

This would require:
- Every gdev user to have a mcp.hosting account (free tier: 3 servers only)
- Internet connectivity during development (mcph polls every 60s)
- Trusting a 5-week-old startup's cloud service with credential management
- Replacing gdev's entire `.mcp.json` generation pipeline

**This is not practical.** It trades a simple, offline, self-contained code generator for a cloud-dependent runtime proxy from a pre-seed startup.

### 6.3 Self-Hosted Option

mcp.hosting can be self-hosted ($15/seat/month Team plan) via Docker Compose or Helm. This addresses the cloud dependency concern but introduces:
- Operational overhead (PostgreSQL, Redis, Caddy, AWS SES for auth)
- Per-seat licensing cost
- Dependency on proprietary closed-source backend images (GHCR pull token required)

For a consulting company standardizing developer tooling, self-hosted mcp.hosting is worth monitoring as the product matures, but not viable today given the maturity level.

---

## 7. Maturity Assessment

| Signal | Assessment |
|--------|------------|
| **Age** | 5 weeks (created April 8, 2026) |
| **Version** | v0.47.5 (rapid iteration, pre-1.0) |
| **Contributors** | 1 person (jeffyaw) |
| **Stars** | 1 |
| **Forks** | 0 |
| **Open issues** | 0 |
| **License** | **None declared** on mcph (MIT on other repos) |
| **Test coverage** | Extensive (~450KB of test files, vitest) |
| **CI** | GitHub Actions for CI and releases |
| **Releases** | 47+ releases in 5 weeks (aggressive shipping) |
| **Documentation** | Thorough (26KB README, roadmap, contributing guide) |
| **Security** | SECURITY.md present; compliance-aware routing |

**Assessment: Experimental/early-stage.** The engineering quality is high for a solo developer project -- extensive tests, good documentation, thoughtful architecture. But the single-contributor bus factor, absent license on the core package, zero community adoption, and 5-week history make this unsuitable as a production dependency.

The 47 releases in 5 weeks signal rapid iteration that could include breaking changes. No stability guarantees exist.

---

## 8. Comparison to Unit 3.5.1 McpServerDef Registry

### 8.1 What mcph Does That Our Design Doesn't

1. **Runtime health monitoring**: mcph tracks per-server call counts, errors, and latency. Our design generates static config with no runtime awareness.
2. **Intelligent routing**: BM25 + semantic ranking to load only relevant servers. Our design loads all enabled servers statically.
3. **Dynamic server management**: Load/unload servers mid-session. Our design requires re-running `qsdev init` or `qsdev enable/disable`.
4. **Multi-device sync**: One config propagates to all clients. Our design is per-project.
5. **Compliance grading**: A-F grades from 88-test suite. Our design has no automated compliance checking.
6. **Context cost estimation**: Token cost per server surfaced in discovery. Our design tracks tool counts but not token costs.

### 8.2 What Our Design Does That mcph Doesn't

1. **Project detection heuristics**: Auto-detect databases, IaC, SDKs from project files. mcph has zero detection capability.
2. **Three-tier auto-configuration policy**: AutoDetect / DetectAndOffer / OptionalCatalog. mcph is all-manual.
3. **Hard tool budget ceiling**: Explicit 40-tool limit with `CanEnable()` enforcement. mcph has a soft concurrent server cap (default 6) but no tool-count ceiling.
4. **Security tiering with policy enforcement**: Low/Medium/High tiers drive automation level. mcph has compliance grades but no policy enforcement beyond minimum grade filtering.
5. **SecretSpec integration**: Credentials flow through keyring, env, or 1Password. mcph uses cloud-encrypted storage (vendor lock-in).
6. **Offline operation**: Static file generation works without internet. mcph requires connectivity.
7. **Go integration**: Native Go struct registry integrates directly with gdev's codebase. mcph is TypeScript.
8. **Wizard-driven setup**: Interactive prompts during `qsdev init` for detect-and-offer servers. mcph relies on the AI client itself for server management via meta-tools.

### 8.3 Net Assessment

The designs are complementary layers, not competitors:
- **gdev** is a build-time project configurator that generates optimal MCP configs for a specific project
- **mcph** is a runtime orchestrator that dynamically manages servers during AI sessions

If mcph matures significantly (1.0+, community adoption, proper licensing, offline mode), it could potentially serve as the runtime layer beneath gdev's build-time configuration. gdev would generate mcph config files instead of `.mcp.json` files. But that future is speculative and at least 6-12 months away.

---

## 9. Recommendations

### 9.1 Do Not Adopt

- **Do not adopt mcph as a dependency or foundation for Unit 3.5.1.** The architectural mismatch (runtime proxy vs. build-time generator), cloud dependency, language mismatch, maturity level, and absent license make this impractical.
- **Do not recommend mcph to gdev users.** The product is too early and the vendor lock-in to mcp.hosting's cloud is a concern for consulting teams handling client data.

### 9.2 Cherry-Pick: Compliance Suite

- **Adopt `@yawlabs/mcp-compliance` (MIT licensed) for gdev's test infrastructure.** Use it in Phase 17 (Test Infrastructure Framework) to validate MCP server configurations.
- Integration point: `qsdev mcp check --compliance` runs the compliance suite against each enabled server and reports grades.
- CI integration: GitHub Action `YawLabs/mcp-compliance@v0` for testing MCP servers in gdev's own CI pipeline.
- This is low-risk: MIT licensed, runs as a standalone CLI, no cloud dependency.

### 9.3 Evaluate: aws-mcp as Catalog Entry

- **Evaluate `@yawlabs/aws-mcp` as an alternative to the official `awslabs/mcp` server** for gdev's optional catalog.
- Key advantage: SSO re-auth via device-code flow solves a real pain point for consulting engineers using AWS SSO.
- Key concern: 24 tools exceeds half the budget; would need profile-based filtering to be practical.
- Requires separate spike or sub-task within Phase 12.8 implementation.

### 9.4 Monitor the Ecosystem

- Watch for mcph to reach 1.0, declare a license, and gain community adoption.
- Watch for mcp.hosting self-hosted to mature and drop in cost.
- The Yaw Labs engineering quality is high; if the company survives, these tools could become significant.
- Re-evaluate in 6 months (November 2026).

---

## 10. Sources

| Source | Local Path |
|--------|------------|
| mcph README | `docs/mcph-readme-full.md` |
| mcph Roadmap | `docs/mcph-roadmap.md` |
| mcph package.json | `docs/mcph-package-json.md` |
| mcph types.ts | `docs/mcph-types-ts.md` |
| mcph meta-tools | `docs/mcph-meta-tools-summary.md` |
| mcph config schema | `docs/mcph-config-schema-v1.md` |
| mcph upstream architecture | `docs/mcph-upstream-architecture.md` |
| mcph credential handling | `docs/mcph-credentials-handling.md` |
| mcp-compliance README | `docs/yawlabs-mcp-compliance-readme.md` |
| aws-mcp README | `docs/yawlabs-aws-mcp-readme.md` |
| tailscale-mcp README | `docs/yawlabs-tailscale-mcp-readme.md` |
| mcp-hosting-deploy README | `docs/mcp-hosting-deploy-readme.md` |
| YawLabs org inventory | `docs/yawlabs-github-org-repos.md` |
| Yaw Labs overview article | `docs/yaw-labs-terminal-context-ammunition.md` |
| gdev MCP expansion design | `mcp-expansion-design.md` |

---

## Depth Checklist

- [x] Underlying mechanism explained (proxy architecture, meta-tools, config polling, transport management)
- [x] Key tradeoffs and limitations identified (cloud dependency, language mismatch, single contributor, no license)
- [x] Compared to alternative (gdev's Unit 3.5.1 design -- detailed feature-by-feature comparison)
- [x] Failure modes and edge cases described (offline operation impossible, free tier 3-server cap, credential detection heuristics)
- [x] Concrete examples found (full source tree analyzed, config schema, meta-tool definitions, deployment options)
- [x] Report is standalone-readable (sufficient for architectural decision without reading source repos)
