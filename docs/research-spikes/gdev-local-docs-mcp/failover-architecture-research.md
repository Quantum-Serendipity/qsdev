# Failover Architecture Design: Local-First Documentation with Web Fallback

## Executive Summary

Designing the multi-source routing/orchestration layer for gdev's local documentation MCP servers requires choosing between two fundamentally different approaches: **skill-level routing** (a Claude Code skill that instructs the model to try local MCP tools first, then fall back to web) and **MCP-level routing** (a meta-MCP server that internally aggregates backends and handles failover). After analyzing the MCP proxy ecosystem (MetaMCP, FastMCP, Envoy AI Gateway, combine-mcp), Claude Code's tool dispatch model, CRAG-style multi-source retrieval patterns, and the specific constraints of gdev's security posture, this report recommends **skill-level routing** as the primary architecture, with optional MCP-level aggregation as a future optimization.

The key insight: Claude Code's tool search system already handles multi-server tool discovery efficiently. A well-crafted skill can direct the model's tool selection behavior with no additional infrastructure, while an MCP proxy would add a Python/Docker dependency, latency (300-400ms per tool list), a new failure mode, and a prompt injection surface (the proxy itself becomes trusted infrastructure that could be compromised).

---

## 1. MCP Server Chaining and Routing Patterns

### 1.1 How Claude Code Handles Multiple MCP Servers

Claude Code natively supports multiple simultaneous MCP servers with these key behaviors:

**Tool Discovery**: With Tool Search enabled (default since 2025), MCP tool schemas are deferred -- only tool names load at session start. When Claude needs a tool, it searches by capability description and loads the full schema on demand. This means adding more MCP servers has minimal context cost.

**Dispatch Model**: Claude Code does NOT route tool calls through any priority system. When multiple servers are connected, Claude sees all available tools as a flat catalog and chooses based on the tool name, description, and the current task context. There is no built-in "try server A first, then server B" mechanism.

**Precedence by Name**: MCP servers override by name with priority: local > project > user > plugin > claude.ai connectors. But this only controls *which definition wins* when the same server name appears at multiple scopes -- it does not create priority ordering between different servers.

**Per-Session Isolation**: Each Claude Code session spins up its own MCP server instances. If a server is stateful, multiple panes mean multiple instances.

**Automatic Reconnection**: HTTP/SSE servers get exponential backoff reconnection (5 attempts, 1s doubling). Stdio servers (which gdev's local docs would use) are NOT reconnected automatically.

**Key implication**: Claude Code provides no native routing or failover between MCP servers. If gdev wants "try local DevDocs first, then web search," that logic must live either in a skill's instructions or in a meta-MCP server.

### 1.2 The MCP Proxy/Gateway Ecosystem (2026)

A significant ecosystem of MCP aggregation tools has emerged:

| Project | Stars | Language | Approach | Failover Support |
|---------|-------|----------|----------|-----------------|
| **MetaMCP** | 2,300 | TypeScript | Docker-based aggregator with middleware, namespace grouping, rate limiting | No explicit failover |
| **FastMCP Proxy** | (part of FastMCP) | Python | `create_proxy()` with automatic namespace prefixing | No explicit failover |
| **IBM ContextForge** | 3,500 | Mixed | Federates MCP, A2A, REST/gRPC with plugins/guardrails | Plugin-based, potentially |
| **Envoy AI Gateway** | 1,500 | Go | Cloud-native with circuit breaking, load balancing | Circuit breaking only |
| **Docker MCP Gateway** | 1,300 | Mixed | Isolated containers with resource limits | Container isolation |
| **combine-mcp** | 30 | Go | Minimal stdio server merger | None |
| **mcp-proxy-server** | 198 | TypeScript | Tool aggregation with unified interface | None |
| **Lasso MCP Gateway** | 360 | Mixed | Security gateway with PII detection, guardrails | Policy-based |

**Critical finding**: None of these proxy/gateway projects implement priority-based routing or fallback logic. They all implement the **aggregation pattern** -- presenting multiple backends as a single flat tool catalog. The routing is purely by tool name namespace (e.g., `devdocs_search`, `zim_search`), not by priority or fallback chains.

This means building a meta-MCP server with CRAG-style fallback would be a custom development effort, not an off-the-shelf configuration.

### 1.3 MCP Protocol Limitations

The MCP protocol itself (JSON-RPC 2.0 over stdio/SSE/HTTP) has no concept of:
- Tool priority or ordering
- Fallback chains
- Result quality scoring
- "No results" signaling (beyond empty responses)
- Multi-server orchestration

Tool responses are simple: content (text/image), `isError` flag, and optional metadata. There is no standardized way for a tool to say "I found results but they might not be relevant" vs "I found nothing" vs "I found exactly what you need."

---

## 2. Skill-Level vs MCP-Level Routing: Tradeoff Analysis

### 2.1 Approach A: Skill-Level Routing

A Claude Code skill (`.claude/skills/lookup-docs/SKILL.md`) that instructs Claude on how to use the available documentation MCP tools in priority order.

**How it works:**

```yaml
---
name: lookup-docs
description: Look up documentation using local sources first, falling back to web search. Use when the user asks about API documentation, library usage, error messages, or needs code examples.
allowed-tools: >
  mcp__devdocs__search_docs
  mcp__devdocs__read_doc
  mcp__openzim__zim_query
  mcp__man__get_man_page
  mcp__man__search_man_pages
  mcp__nixos__nix
  WebSearch
  WebFetch
---

## Documentation Lookup Protocol

When looking up documentation, follow this priority order:

### 1. Local Sources First (preferred -- no network, no prompt injection risk)

**For API/library documentation:**
- Use `mcp__devdocs__search_docs` to search DevDocs for the relevant library
- If results found, use `mcp__devdocs__read_doc` to get the full content

**For Q&A / troubleshooting / error messages:**
- Use `mcp__openzim__zim_query` to search Stack Exchange ZIM archives

**For system commands / CLI tools:**
- Use `mcp__man__search_man_pages` then `mcp__man__get_man_page`

**For NixOS-specific queries:**
- Use `mcp__nixos__nix` for packages, options, Home Manager

### 2. Evaluate Local Results

If local sources returned results:
- Check if the results directly address the question
- If results are partial (related but not answering the specific question), note what was found and proceed to step 3

If local sources returned no results or only tangentially related content:
- Proceed to step 3

### 3. Web Fallback (only when local is insufficient)

**Important**: Web-sourced content carries higher prompt injection risk. Flag it as web-sourced.

- Use WebSearch to find relevant documentation
- Use WebFetch to retrieve specific documentation pages
- When presenting web-sourced results, prefix with: "[Web source -- verify independently]"

### Source Tagging

Always indicate the source of documentation in your response:
- "[DevDocs]" for local API documentation
- "[Stack Exchange]" for local Q&A content  
- "[man page]" for system documentation
- "[NixOS]" for Nix-specific content
- "[Web]" for web-fetched content
```

**Advantages:**

1. **Zero infrastructure**: No additional server to build, deploy, package, or maintain. Just a SKILL.md file deployed via gdev.
2. **Zero latency overhead**: No proxy layer adding 300-400ms per request.
3. **Leverages Claude's reasoning**: Claude can evaluate result quality contextually -- "this DevDocs result about Python 3.11's `match` statement answers the question about pattern matching" -- better than any programmatic threshold.
4. **Minimal prompt injection surface**: The skill itself is a local file reviewed by the team. No additional trusted infrastructure.
5. **Flexible degradation**: Claude can naturally handle partial results ("DevDocs had the function signature but not the edge case behavior, let me check Stack Exchange").
6. **Composable**: Works with any MCP server combination. Adding a new local source means adding one line to the skill.
7. **Survives server failures**: If an MCP server is down, Claude sees it as unavailable and moves to the next source without crashing.
8. **Context-aware**: Claude can skip irrelevant sources (no point checking man pages for a React question).

**Disadvantages:**

1. **Non-deterministic**: Claude interprets the instructions; behavior may vary between sessions or after context compaction.
2. **Token cost**: The skill content stays in context after invocation (but at ~500 tokens, this is modest).
3. **No guaranteed execution**: Claude may choose to skip local sources if it "thinks" it already knows the answer, especially under low effort settings.
4. **Hard to test**: No unit tests for "did Claude follow the priority order?"
5. **Compaction risk**: After auto-compaction, the skill content may be truncated (first 5,000 tokens preserved, but skill re-invocation budget is 25,000 tokens shared across all skills).

### 2.2 Approach B: MCP-Level Routing (Meta-MCP Server)

A custom MCP server that wraps DevDocs, OpenZIM, man pages, and web search behind a unified interface with internal routing logic.

**How it would work:**

```
Claude Code ──► gdev-docs-mcp (meta-server)
                    ├─► DevDocs JSON files (direct access)
                    ├─► OpenZIM/python-libzim (direct access)
                    ├─► man/apropos subprocess
                    └─► Web search API (fallback)
```

The meta-server would expose tools like:
- `search_docs(query, context?)` -- searches local sources in priority order, falls back to web
- `read_doc(source, path)` -- retrieves specific content from a named source
- `list_sources()` -- lists available documentation sources and their coverage

Internally, it would implement CRAG-style scoring:
1. Query DevDocs index -- if entries found with exact name match, return immediately
2. Query ZIM full-text search -- if Xapian returns results above a relevance threshold, return
3. If neither local source had adequate results, trigger web search
4. Return results with source metadata and confidence indicators

**Advantages:**

1. **Deterministic routing**: Code-level control over the fallback chain. Every query follows the same path.
2. **Testable**: Unit tests can verify routing logic, threshold behavior, and fallback triggering.
3. **Single tool surface**: Claude sees one `search_docs` tool instead of 4+ tools from separate servers. Simpler context, less chance of Claude choosing wrong.
4. **Result quality scoring**: Can implement Xapian relevance thresholds, DevDocs index match scoring, and CRAG-style three-state classification.
5. **Centralized telemetry**: One place to log which sources were queried, hit rates, fallback frequency.

**Disadvantages:**

1. **Significant build effort**: A custom Python MCP server wrapping python-libzim, DevDocs JSON parsing, man page subprocess calls, and a web search client. Estimated 1-2 weeks of development.
2. **New dependency chain**: python-libzim (native C++ via Cython), DevDocs JSON files, plus web search API client. Nix packaging complexity for native dependencies.
3. **Single point of failure**: If the meta-server crashes, ALL documentation access is lost. With separate servers, only the crashed one is unavailable.
4. **Latency addition**: The meta-server adds processing overhead on top of underlying source access.
5. **Prompt injection surface**: The meta-server becomes trusted infrastructure. A bug in its web search integration could introduce injection vectors that bypass the local-first security model.
6. **Reduced flexibility**: Hard-coded routing logic can't adapt to context ("for this React question, skip man pages"). Claude's contextual reasoning is lost.
7. **Maintenance burden**: Every upstream API change (DevDocs format, ZIM library version, web search API) requires meta-server updates.
8. **Opacity**: Claude can't explain "I checked DevDocs first, found partial results, then supplemented with Stack Exchange" because the routing happened inside the meta-server.

### 2.3 Comparison Matrix

| Factor | Skill-Level Routing | MCP-Level Routing |
|--------|-------------------|------------------|
| **Build effort** | ~1 hour (SKILL.md file) | ~1-2 weeks (custom MCP server) |
| **Infrastructure** | None (file in repo) | Python server + native deps |
| **Determinism** | Non-deterministic (Claude interprets) | Deterministic (code-level) |
| **Testability** | Integration tests only | Unit + integration tests |
| **Latency** | Zero overhead | +300-400ms proxy overhead |
| **Failure modes** | Graceful (individual servers) | Catastrophic (meta-server) |
| **Context cost** | ~500 tokens for skill | ~100 tokens for single tool |
| **Prompt injection surface** | Minimal (local SKILL.md) | Moderate (server code + web client) |
| **Adaptability** | High (Claude reasons about context) | Low (fixed routing logic) |
| **Observability** | Visible in conversation | Logged internally, opaque to user |
| **Nix packaging** | Trivial (text file) | Complex (python-libzim native) |
| **Maintenance** | Update text instructions | Update code for API changes |

### 2.4 Recommendation

**Skill-level routing is the right choice for gdev**, for these reasons:

1. **gdev's primary constraint is build effort and maintenance burden**, not routing determinism. A SKILL.md file ships in minutes; a custom MCP server takes weeks and creates ongoing maintenance.

2. **Claude Code's tool search already solves discovery**. The model can see all available tools and reason about which to use. The skill just provides the priority framework.

3. **The security model is stronger with separate servers**. Each MCP server is an independent process with its own failure domain. A meta-server creates a shared failure domain and a single point for prompt injection.

4. **Claude's contextual reasoning beats programmatic thresholds** for the ambiguous cases. When DevDocs returns a function signature but not usage examples, Claude can recognize this and supplement with Stack Exchange -- something a relevance threshold can't do.

5. **The skill approach is incrementally improvable**. Start with simple priority instructions, observe Claude's behavior, and refine. A code-level router requires getting the architecture right upfront.

The meta-MCP approach may become worthwhile later if: (a) the skill proves unreliable in practice, (b) telemetry requirements demand centralized logging, or (c) gdev needs to support non-Claude clients that lack skill-like instruction mechanisms.

---

## 3. "No Results" Detection

Detecting when local documentation doesn't have an answer is critical for the failover decision. Each local source has different "no results" signals.

### 3.1 DevDocs: Index-Based Detection

DevDocs search operates on `index.json`, which contains name/path/type triples for every documented entity.

**Complete miss**: Search against `index.json` entries returns zero matches. This is unambiguous -- the library or API is not in the local corpus.

**Partial match**: Search returns entries for the right library but not the specific function/method. For example, searching "useReducer" in a corpus that has React docs but an older version without hooks documentation.

**Detection strategy**:
- Empty result set → complete miss → fallback to web
- Results from wrong library (fuzzy match false positive) → treat as miss
- Results from right library but wrong section → partial hit → Claude evaluates contextually

**Implementation**: The DevDocs MCP server should return structured metadata with results: `{ results: [...], total_count: N, query: "...", searched_doc_sets: ["react~18", "typescript~5.3"] }`. An empty `results` array is the definitive "no results" signal.

### 3.2 ZIM/Kiwix: Xapian Relevance Scoring

ZIM files with embedded Xapian indexes provide BM25-scored full-text search results.

**Xapian scoring characteristics**:
- BM25 produces unbounded scores (not normalized 0-1 like CRAG's evaluator)
- Scores are relative within a query, not across queries -- a score of 15 on one query is not comparable to 15 on another
- Xapian provides `get_percent()` which normalizes the best match to 100% and scales others relative to it
- An empty result set means zero matching documents in the index

**Detection strategy**:
- Zero results → complete miss → fallback to web
- Results with very low `get_percent()` (e.g., top result < 20%) → likely irrelevant → treat as miss
- Results from wrong Stack Exchange site (e.g., cooking.stackexchange when looking for code) → metadata filtering needed
- Multiple results with high scores → strong local match

**Practical threshold**: The openzim-mcp server returns results with relevance metadata. A threshold of "top result's normalized percentage < 15-20%" is a reasonable heuristic for "no useful results," but this needs empirical tuning against real queries.

### 3.3 Man Pages: Binary Hit/Miss

Man pages are the simplest case:
- `apropos` search returns matching man page names, or nothing
- `man <page>` either returns content or exits with error
- No relevance scoring -- it's purely keyword matching on page names and descriptions

**Detection**: Empty `apropos` results → complete miss. This is unambiguous.

### 3.4 The Three-State Model (Adapted from CRAG)

CRAG's three-state classification maps well to the local docs context:

| State | Condition | Action |
|-------|-----------|--------|
| **CORRECT** | Local source returns results that directly address the query | Return local results, no web fallback |
| **AMBIGUOUS** | Local results are related but incomplete (right library, wrong specificity) | Return local results AND note that web sources may have more detail |
| **INCORRECT** | No local results, or results clearly unrelated | Fall back to web search |

**For skill-level routing**, the CORRECT/AMBIGUOUS/INCORRECT classification happens naturally in Claude's reasoning. The skill instructs Claude to evaluate whether local results "directly address the question" and to supplement or fall back accordingly.

**For MCP-level routing**, this would require implementing a scoring pipeline:
1. DevDocs index match: exact name match → CORRECT; partial match → AMBIGUOUS; no match → INCORRECT
2. Xapian score: top result > 50% → CORRECT; 15-50% → AMBIGUOUS; < 15% → INCORRECT
3. Man page: found → CORRECT; not found → INCORRECT (no ambiguous state)

### 3.5 Handling Partial Results

The most challenging case is partial results -- when local docs have some information but not enough. Examples:

- DevDocs has the function signature but the user needs edge case behavior
- Stack Exchange has a related question but for a different version/framework
- Man page exists but doesn't cover the specific flag combination

In skill-level routing, Claude handles this naturally: "DevDocs shows `Array.prototype.flat()` takes a depth argument, but doesn't show performance characteristics for large arrays. Let me check Stack Exchange for benchmarks."

In MCP-level routing, this requires the meta-server to either: (a) return partial results with a "partial" flag and let Claude decide about web fallback, or (b) always trigger web supplementation for partial results. Option (a) is better but adds complexity; option (b) undermines the local-first security model by triggering web fetches too aggressively.

---

## 4. Graceful Degradation Patterns

### 4.1 Stale/Outdated Local Documentation

**Scenario**: Local DevDocs has React 18 docs but the project uses React 19 with new features.

**Detection signals**:
- Version mismatch between local docs and project dependencies (detectable from `package.json` / `flake.nix`)
- Query about features not present in local index (e.g., React Server Components in React 18 docs)
- User explicitly mentions a version newer than local corpus

**Skill-level handling**: The skill can include version awareness:
```
If the user asks about a feature that doesn't appear in local DevDocs,
it may be from a newer version than the local corpus. Check the project's
dependency version and note if local docs may be outdated before falling
back to web search.
```

**Mitigation**: gdev's update mechanism should:
1. Track installed doc versions vs project dependency versions
2. Flag mismatches during `gdev status` or similar health checks
3. Offer `gdev docs update` to refresh specific doc sets
4. The skill can inject version info via `!` preprocessor: `` !`gdev docs versions` ``

### 4.2 Library Not in Local Corpus

**Scenario**: Developer asks about a library (e.g., `zod`) that wasn't included in the DevDocs profile.

**Detection**: DevDocs search returns zero results for the library slug.

**Handling sequence**:
1. Local DevDocs → no results for "zod"
2. Local ZIM (Stack Exchange) → may have Q&A about zod
3. If ZIM has useful Q&A → return with "[Stack Exchange]" tag
4. If neither has results → fall back to web with "[Web]" tag
5. Suggest: "Consider adding zod to your gdev documentation profile for future offline access"

**Skill-level instruction**:
```
If a library is not in the local DevDocs corpus, suggest that the user
add it to their gdev documentation profile: `gdev docs add <library>`
```

### 4.3 Air-Gapped Environment (Web Fallback Unavailable)

**Scenario**: Developer is on a plane, in a secure facility, or network is down. Web fallback is impossible.

**Detection**: WebSearch/WebFetch tools fail with network errors.

**Handling**:
1. Return whatever local results exist, even if partial
2. Clearly state: "Web sources are unavailable. Results are from local documentation only and may be incomplete."
3. Do NOT hallucinate to fill gaps -- explicitly acknowledge what's missing
4. Suggest checking web sources later for completeness

**Skill-level instruction**:
```
If web search fails due to network unavailability:
- Return local results with a note about network unavailability
- Do NOT guess or hallucinate information that local docs don't contain
- Suggest the user retry when network is available
```

This is actually the **strongest argument for local-first architecture**: in air-gapped environments, a web-only documentation system provides zero value, while local documentation provides substantial (if potentially stale) coverage.

### 4.4 Conflicting Information Between Sources

**Scenario**: DevDocs says function X takes 2 arguments; Stack Exchange answer shows it with 3 arguments.

**Handling priority**:
1. Official documentation (DevDocs) takes precedence over community Q&A (Stack Exchange)
2. Note the discrepancy and suggest checking version-specific docs
3. More recent source generally preferred for API changes

**Skill-level instruction**:
```
If local sources provide conflicting information:
- Official documentation (DevDocs, man pages) takes precedence over
  community Q&A (Stack Exchange)
- Note the discrepancy and which version each source references
- Recommend the user verify with the canonical source
```

### 4.5 MCP Server Process Crash

**Scenario**: openzim-mcp crashes due to a corrupted ZIM file or python-libzim segfault.

**Detection**: Claude Code marks the server as failed in `/mcp`. Stdio servers are NOT auto-reconnected.

**Handling**:
- The skill's priority order naturally skips unavailable servers
- Claude sees the tool as unavailable and moves to the next source
- Unlike a meta-MCP server, other sources remain fully functional

**Recovery**: User runs `/mcp` to see status, then `gdev docs health` to diagnose and restart the crashed server.

---

## 5. Prior Art in Multi-Source Search

### 5.1 CRAG (Corrective Retrieval-Augmented Generation)

CRAG (arXiv 2401.15884) is the closest academic prior art. Its three-state classification (CORRECT/AMBIGUOUS/INCORRECT) with web search fallback maps directly to the local-docs-with-web-fallback pattern.

**Key CRAG insights applicable to gdev:**
- The 0.7/0.3 threshold split is a useful starting framework but needs per-knowledge-base tuning
- The AMBIGUOUS state (merge local + web) is the most valuable innovation -- it handles partial results gracefully
- Even without web fallback, the scoring/filtering step alone dramatically improves context precision (+0.431 in experiments)
- CRAG uses per-document evaluation, not per-query -- each retrieved document gets its own relevance score

**Adaptation for gdev**: In skill-level routing, Claude IS the relevance evaluator. Instead of a T5 model scoring documents 0-1, Claude reads the results and makes a contextual judgment: "this DevDocs entry about `fetch()` answers the question about request headers" (CORRECT) or "this Stack Exchange answer is about `fetch` in a different context" (INCORRECT).

### 5.2 Search Federation (Metasearch)

Traditional metasearch engines (Searx/SearXNG, MetaGer, Dogpile) aggregate results from multiple search backends using result merging strategies:

- **Round-robin merging**: Alternate results from each source
- **Score normalization**: Convert disparate scoring systems to a common scale
- **Reciprocal rank fusion (RRF)**: Combine rankings across sources using `1/(k + rank)` formula
- **Cascade search**: Query sources in priority order, stop when sufficient results found

**Most relevant pattern for gdev**: Cascade search. This is exactly the local-first-then-web model. Metasearch systems implement this as:
1. Query Source A (timeout: 500ms)
2. If sufficient results (>= N with relevance > threshold) → return
3. Else query Source B (timeout: 500ms)
4. Merge results from both sources
5. Continue until sufficient results or all sources exhausted

In skill-level routing, Claude executes this cascade naturally through the priority instructions.

### 5.3 DNS-Style Fallback Chains

DNS resolution provides an instructive analogy:
1. Check local `/etc/hosts` (instant, fully trusted)
2. Check local resolver cache (fast, recently trusted)
3. Query upstream resolver (network required, less trusted)
4. Query root/authoritative servers (slowest, authoritative)

The parallel to gdev's documentation lookup:
1. Man pages (system-local, fully trusted, instant)
2. DevDocs (local files, trusted, fast)
3. ZIM/Stack Exchange (local files, lower trust than official docs, fast)
4. Web search (network required, lowest trust, slowest)

This priority ordering is exactly what the skill encodes.

### 5.4 Existing MCP Aggregation Projects

As surveyed in Section 1.2, no existing MCP aggregation project implements priority-based routing or failover. They all implement flat tool catalog aggregation. This confirms that gdev's failover requirement is genuinely novel in the MCP ecosystem and would require custom development for MCP-level implementation.

---

## 6. Concrete Architecture Recommendation

### 6.1 Recommended Architecture: Skill-Routed Multi-Server

```
┌─────────────────────────────────────────────────┐
│                  Claude Code                     │
│                                                  │
│  ┌──────────────────────────────────────────┐   │
│  │  lookup-docs skill (SKILL.md)            │   │
│  │  - Priority ordering instructions        │   │
│  │  - Source tagging rules                  │   │
│  │  - Degradation behavior                  │   │
│  │  - Dynamic context: !`gdev docs status`  │   │
│  └──────────────────────────────────────────┘   │
│                      │                           │
│          (Claude's tool selection)               │
│            ┌─────────┼──────────┐               │
│            ▼         ▼          ▼               │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐       │
│  │ DevDocs  │ │ OpenZIM  │ │ man-mcp  │       │
│  │ MCP      │ │ MCP      │ │ server   │       │
│  │ (stdio)  │ │ (stdio)  │ │ (stdio)  │       │
│  └────┬─────┘ └────┬─────┘ └────┬─────┘       │
│       │             │            │               │
│  JSON files    ZIM files    system man           │
│                                                  │
│  ┌──────────┐ ┌──────────────┐                  │
│  │ MCP-NixOS│ │ WebSearch /  │                  │
│  │ (online) │ │ WebFetch     │                  │
│  └──────────┘ │ (fallback)   │                  │
│               └──────────────┘                  │
└─────────────────────────────────────────────────┘
```

### 6.2 Component Details

**1. DevDocs MCP Server** (local, stdio)
- Custom TypeScript MCP server reading DevDocs JSON files directly
- Tools: `search_docs(query, doc_sets?)`, `read_doc(slug, path)`, `list_doc_sets()`
- Returns structured results with metadata: `{ source: "devdocs", doc_set: "react~18", entry_name: "useState", match_type: "exact|fuzzy|none" }`
- Nix-packaged with selected doc sets per gdev profile

**2. OpenZIM MCP Server** (local, stdio)
- `openzim-mcp` in simple mode (single `zim_query` tool)
- Pre-configured with curated ZIM files (Unix & Linux SE, Server Fault, DevOps SE, MDN)
- Nix-packaged with python-libzim native dependency

**3. man-mcp-server** (local, stdio)
- Existing `guyru/man-mcp-server` (Python, MIT)
- Zero configuration needed on NixOS (man pages already installed)

**4. MCP-NixOS** (online, stdio)
- Existing `utensils/mcp-nixos` (Python, MIT)
- Accepted online dependency for NixOS-specific queries (first-party APIs)

**5. WebSearch / WebFetch** (built-in, fallback)
- Claude Code's built-in web tools
- Used only when local sources are insufficient
- Subject to skill's source-tagging instructions

### 6.3 Configuration Format (.mcp.json)

gdev generates this configuration based on the team's documentation profile:

```json
{
  "mcpServers": {
    "devdocs": {
      "command": "${GDEV_LIBEXEC}/devdocs-mcp",
      "args": ["--docs-dir", "${GDEV_DATA}/devdocs"],
      "env": {
        "DEVDOCS_DOC_SETS": "${GDEV_DEVDOCS_SETS:-typescript~5.3,react~18,node~20}"
      }
    },
    "local-docs": {
      "command": "openzim-mcp",
      "env": {
        "OPENZIM_MCP_ZIM_DIR": "${GDEV_DATA}/zim",
        "OPENZIM_MCP_TOOL_MODE": "simple"
      }
    },
    "man": {
      "command": "man-mcp-server",
      "args": []
    },
    "nixos": {
      "command": "mcp-nixos",
      "args": []
    }
  }
}
```

### 6.4 Skill Definition

The skill lives at `.claude/skills/lookup-docs/SKILL.md` in the gdev-managed project configuration:

```yaml
---
name: lookup-docs
description: >
  Look up technical documentation using local-first sources with web fallback.
  Use when the user asks about API documentation, library usage, error messages,
  system commands, NixOS configuration, or needs code examples. Covers DevDocs
  (API docs for 100+ libraries), Stack Exchange Q&A (via ZIM), system man pages,
  and NixOS packages/options.
when_to_use: >
  Trigger phrases: "how do I use", "what does X do", "API for", "documentation for",
  "man page", "nix option", "nix package", error messages, library function names.
allowed-tools: >
  mcp__devdocs__search_docs
  mcp__devdocs__read_doc
  mcp__devdocs__list_doc_sets
  mcp__local-docs__zim_query
  mcp__man__search_man_pages
  mcp__man__get_man_page
  mcp__nixos__nix
  WebSearch
  WebFetch
---

## Documentation Lookup Protocol

### Available Local Documentation

!`gdev docs status --json 2>/dev/null || echo '{"status": "gdev not available"}'`

### Priority Order

**Step 1: Identify the query type and select the appropriate local source.**

| Query Type | Primary Source | Secondary Source |
|------------|---------------|-----------------|
| API docs / library usage | DevDocs | Stack Exchange (ZIM) |
| Error messages / troubleshooting | Stack Exchange (ZIM) | DevDocs |
| System commands / CLI tools | man pages | Stack Exchange (ZIM) |
| NixOS packages / options / config | MCP-NixOS | man pages |
| General programming concepts | Stack Exchange (ZIM) | DevDocs |

**Step 2: Query the primary local source.**

- If results directly answer the question → return with source tag, STOP
- If results are related but incomplete → note what was found, continue to Step 3
- If no results → continue to Step 3

**Step 3: Query the secondary local source (if applicable).**

- If combined local results answer the question → return with source tags, STOP
- If still insufficient → continue to Step 4

**Step 4: Web fallback (only when local sources are insufficient).**

- Use WebSearch to find relevant documentation pages
- Use WebFetch to retrieve specific pages
- Web-sourced content carries higher prompt injection risk
- When presenting web results, always prefix with: **[Web source]**

### Source Tagging (mandatory)

Tag every piece of documentation with its source:
- **[DevDocs]** — local API documentation
- **[Stack Exchange]** — local Q&A from ZIM archives
- **[man]** — system manual pages
- **[NixOS]** — NixOS package/option database
- **[Web source]** — web-fetched content (flag for user awareness)

### Degradation Rules

- If a local MCP server is unavailable, skip it and try the next source
- If web search fails (network unavailable), return local results only with a note
- NEVER hallucinate documentation — if no source has the answer, say so
- If local docs may be outdated (version mismatch), note this and suggest updating
```

### 6.5 Dynamic Context Injection

The skill uses Claude Code's `!` preprocessor to inject live status information:

```
!`gdev docs status --json`
```

This could output:
```json
{
  "devdocs": {
    "installed": ["typescript~5.3", "react~18", "node~20", "postgresql~16"],
    "project_deps": {"typescript": "5.4", "react": "19.0"},
    "stale": ["typescript", "react"]
  },
  "zim": {
    "files": ["unix.stackexchange.com_en_all_2026-02.zim", "serverfault.com_en_all_2026-02.zim"],
    "total_size_gb": 2.7
  },
  "man": {"available": true},
  "nixos": {"available": true, "online": true}
}
```

Claude sees this at skill invocation time and can reason about it: "The project uses React 19 but local DevDocs only has React 18 -- I should note this and check web sources for React 19-specific features."

### 6.6 Integration with gdev Deployment

gdev generates and deploys:

1. **`.mcp.json`** — MCP server configuration (per gdev profile)
2. **`.claude/skills/lookup-docs/SKILL.md`** — routing skill
3. **`.claude/rules/doc-sources.md`** — always-on rule reminding Claude about local doc availability
4. **Nix packages** — DevDocs MCP server, openzim-mcp, man-mcp-server, MCP-NixOS
5. **Data directory** — DevDocs JSON files and ZIM files under `~/.local/share/gdev/`

The `.claude/rules/doc-sources.md` rule (loaded every session, unlike the skill):
```markdown
# Documentation Sources

This project has local documentation available via MCP servers. When looking up
documentation, prefer the /lookup-docs skill which queries local sources first.
Local documentation is more reliable (no network needed, no bot blocks) and
carries lower prompt injection risk than web-fetched content.
```

### 6.7 Future Evolution Path

If skill-level routing proves insufficient, the architecture supports incremental migration:

1. **Phase 1 (current recommendation)**: Skill-level routing with separate MCP servers
2. **Phase 2 (if needed)**: Add telemetry to `gdev docs` CLI to track local hit rates, web fallback frequency, source usage patterns
3. **Phase 3 (if needed)**: Build a lightweight TypeScript MCP server that wraps DevDocs JSON + web fallback, replacing the DevDocs MCP + WebSearch combination with a single smarter tool
4. **Phase 4 (if needed)**: Full meta-MCP server with CRAG-style scoring, combining all local sources behind one interface

Each phase is independently valuable and backward-compatible.

---

## 7. Depth Checklist

- [x] **Underlying mechanism explained**: How Claude Code dispatches tool calls across multiple MCP servers, why no native routing exists, how skill instructions guide tool selection
- [x] **Key tradeoffs and limitations identified**: Skill-level (non-deterministic, compaction risk) vs MCP-level (build effort, single point of failure, prompt injection surface) with detailed comparison matrix
- [x] **Compared to alternatives**: CRAG, metasearch federation, DNS-style fallback, MCP proxy ecosystem (MetaMCP, FastMCP, Envoy, etc.)
- [x] **Failure modes and edge cases described**: Stale docs, missing libraries, air-gapped environments, conflicting sources, MCP server crashes, compaction behavior
- [x] **Concrete examples**: Full SKILL.md definition, .mcp.json configuration, query flow examples, degradation scenarios
- [x] **Standalone-readable**: Report covers the complete design space without requiring prior reading

---

## Sources

Sources saved to `docs/`:
- `mcp-gateway-proxy-patterns-chatforest.md` — MCP Gateway & Proxy Patterns (ChatForest)
- `claude-code-skills-documentation.md` — Claude Code Skills system documentation
- `claude-code-extend-features-overview.md` — Claude Code extension layer overview
- `claude-code-mcp-configuration.md` — Claude Code MCP configuration and scope hierarchy
- `metamcp-aggregator-github.md` — MetaMCP aggregator/orchestrator
- `mcp-router-apigene-blog.md` — MCP Router concepts (Apigene)
- `mcp-proxy-server-adamwattis-github.md` — MCP Proxy Server aggregation
- `crag-corrective-rag-architecture.md` — CRAG three-state classification architecture
- `fastmcp-proxy-provider.md` — FastMCP Proxy Provider aggregation

Additional sources from web search (not separately saved due to summary-level content):
- MCP protocol specification (JSON-RPC 2.0, no routing primitives)
- Xapian BM25 scoring and threshold configuration
- RAG multi-source retrieval patterns (2025-2026 landscape)
- Search federation / metasearch architectural patterns
- MCP error handling best practices (MCPcat, Stainless)
