# Deep Dive: Sense (luuuc/sense) — Evaluation for gdev Integration

## Executive Summary

Sense is a local MCP server written in Go that provides structural codebase understanding to AI coding agents (Claude Code, Cursor, Codex CLI). It is **not a security tool** — despite its inclusion in a security tooling evaluation list, Sense solves a fundamentally different problem: reducing the token cost and improving the accuracy of AI agent codebase navigation through pre-built symbol graphs, semantic search, blast radius analysis, and convention detection.

**Recommendation: Configuration option (Phase 28 MCP registry entry, detect-and-offer policy).** Sense is a strong fit for gdev's AI agent tooling layer — it directly replaces semble (Unit 11.3) with a more capable, Go-native, zero-external-dependency alternative. It should NOT be a default (60MB binary, 100-200MB index per project, O'Saasy license restrictions). Concept borrowing from its hook architecture is also valuable for gdev's own hook system design.

---

## 1. What Sense Is (and Is Not)

### What It Solves
AI coding agents (Claude Code, Cursor, Codex CLI) spend significant tokens navigating codebases — grepping for symbols, reading files to understand call chains, manually tracing dependencies. Sense pre-indexes the codebase into a symbol graph stored in SQLite, then exposes four MCP tools that give agents structural answers directly:

- **sense_graph**: "Who calls this function? What does it inherit from? What tests cover it?" — answered from the pre-built edge graph, not by scanning files.
- **sense_search**: Hybrid semantic + keyword search over symbols and code, using a bundled ONNX embedding model. Falls back to text search when embeddings are unavailable.
- **sense_blast**: "If I change this symbol, what breaks?" — BFS traversal of reverse edges with confidence decay. Produces risk-tiered impact lists.
- **sense_conventions**: "What patterns does this project follow?" — detects 9 categories of conventions (naming, inheritance, architecture layers, framework idioms, key types) from the symbol graph.

### What It Is NOT
Sense is explicitly **not** a security analysis tool. It performs:
- No vulnerability scanning
- No dependency auditing
- No supply chain attack detection
- No secrets detection
- No SAST/DAST

The name "Sense" refers to codebase "sense-making" — structural comprehension — not security sensing.

### Claimed Performance Impact
Benchmarks across 7 real-world codebases (Flask, Gin, Axum, Discourse, Javalin, Next.js):
- Tool calls per task: 19 -> 10 (-47%)
- Tokens per task: 228K -> 156K (-32%)
- Cost per task: $0.42 -> $0.31 (-26%)
- Score per 100K tokens: 0.19 -> 0.30 (+64%)

Benchmark methodology: 6 scenarios, 4 evaluation steps per tool, scored on quality (55%), citation grounding (15%), efficiency (20%), keyword coverage (10%). The bench/ directory includes a self-tuning improvement loop, held-out frozen test cases, and LLM-as-judge evaluation — a relatively rigorous setup for an early-stage project.

---

## 2. Architecture & Mechanisms

### Indexing Pipeline
1. **Tree-sitter parsing** — 13 language grammars compiled in (Go bindings via `go-tree-sitter`). Extracts symbols (functions, classes, methods, modules) and relationships (calls, inherits, includes, composes).
2. **Edge resolution** — Cross-file reference resolution maps symbols to their callers/callees across the codebase. Framework-specific resolvers handle Rails concerns, Go interfaces, React hooks, etc.
3. **Embedding** — Bundled ONNX model (`yalue/onnxruntime_go`) generates vector embeddings for semantic search. Embedding is deferrable — can run lazily during MCP server startup rather than blocking scan.
4. **SQLite storage** — Everything goes into `.sense/index.db`. Tables: `sense_files` (path, language, indexed_at), `sense_symbols` (name, kind, file, line), `sense_edges` (source, target, kind). HNSW index (`coder/hnsw`) for approximate nearest-neighbor search on embeddings.
5. **Incremental updates** — `scan.RunIncremental()` re-indexes only changed files. The `--watch` flag uses `fsnotify` for continuous updates. The post-tool-use hook triggers incremental re-indexing after every Write/Edit by Claude Code.

### MCP Server
- Built on `mark3labs/mcp-go` SDK, stdio transport
- Handlers: `handleSearch`, `handleGraph`, `handleBlast`, `handleConventions`, `handleStatus`
- Maintains in-memory symbol cache with mutex-protected access
- Response compaction for large result sets (configurable thresholds)
- Freshness computation detects stale files since last scan

### Hook System (Claude Code Integration)
Sense implements 5 Claude Code lifecycle hooks:

| Hook | Purpose |
|---|---|
| `session-start` | Inject index stats + codebase summary into session context. Instructs agent to load Sense tools first. |
| `pre-tool-use` | Intercept grep/glob/bash/agent calls. When the agent is searching for a symbol, nudge it toward sense_search/sense_graph instead. |
| `post-tool-use` | After Write/Edit/NotebookEdit, trigger incremental re-index of the changed file within 4s timeout. |
| `pre-compact` | Before context compaction, inject top-5 hub symbols and graph stats so post-compaction context retains structural awareness. |
| `subagent-start` | Inject index stats and tool descriptions into sub-agent context so spawned agents know to use Sense. |

**Critical design decision**: All hooks return exit code 0 on any failure (missing index, malformed input, query error). A broken hook never blocks the user's workflow. This is the `silentRun` wrapper pattern — read stdin, open index, call handler, write response, swallow all errors.

### Setup System
`sense setup` auto-detects installed AI tools and generates configuration:
- **Claude Code**: `.mcp.json` entry, `.claude/settings.local.json` (hooks + permissions), markdown instructions, skill files
- **Cursor**: MCP config, cursor rules files
- **Codex CLI**: MCP config, agents documentation

Configuration is idempotent: JSON files are deep-merged, markdown uses marker comments, skill files are overwritten. This is notably similar to gdev's own generation approach.

---

## 3. Maturity Assessment

### Positive Signals
- **688 commits** on main branch — substantial development history
- **v0.84.3** release — implies many iteration cycles
- **Comprehensive test suite** — 16+ test files for mcpserver alone, plus e2e tests, benchmark tests, and internal tests across all packages
- **Well-structured Go codebase** — 27 internal packages, clean separation of concerns
- **Rigorous benchmarking** — dedicated bench/ directory with locked scoring weights, held-out test cases, LLM-as-judge pipeline
- **13 language support** with framework-specific handling (Rails, Django, FastAPI, React)
- **Platform support**: Linux (amd64/arm64), macOS (Apple Silicon/Intel)

### Concerning Signals
- **4 stars, 1 fork** — minimal community adoption
- **Single author** (Luc B. Perussault-Diallo) — bus factor of 1
- **O'Saasy license** — not a standard OSI-approved license; SaaS competition restriction may create ambiguity for some enterprise adopters
- **60MB binary size** — large, includes bundled ONNX runtime and tree-sitter grammars
- **100-200MB index per project** — non-trivial disk footprint
- **No Windows native support** — WSL2 only (aligned with gdev's approach, so not a blocker)
- **Go 1.25.5** — requires a very recent Go toolchain
- **ONNX runtime dependency** — bundled via CGo bindings, potential cross-compilation complexity

### Maintenance Velocity
688 commits with v0.84.3 suggests active development as of May 2026. However, the extremely low star count versus the maturity of the codebase suggests this may be a recently open-sourced tool from a company/consultant's internal tooling, rather than a community-driven project. The author name matches the license copyright, confirming single-author provenance.

---

## 4. Integration Fit with gdev

### Option A: Configuration Option (RECOMMENDED)

**How it works**: Add Sense as a `detect-and-offer` entry in the Phase 28 MCP Server Registry. When gdev detects a codebase with supported languages, the wizard offers to enable Sense. If enabled, gdev generates:
1. `.mcp.json` entry for the Sense MCP server
2. Hook configurations in `.claude/settings.local.json` (5 hooks)
3. Permission entries for `sense` CLI commands
4. CLAUDE.md section describing available Sense tools

**Why this fits**:
- Aligns with gdev principle #12 (AI agent tools are opt-in enhancements)
- Aligns with gdev principle #13 (every tool is individually toggleable)
- Sense's `setup` command already generates exactly the files gdev generates — the integration is configuration generation, not code embedding
- The MCP registry (Phase 28) already has infrastructure for tool count ceiling enforcement (40 tools), security tier classification, and credential requirements
- Sense adds 5 MCP tools (graph, search, blast, conventions, status) — well within ceiling budget

**MCP Registry Entry**:
```go
McpServerEntry{
    Name:           "sense",
    Description:    "Structural codebase understanding for AI agents",
    ToolCount:      5,
    SecurityTier:   TierLow,    // read-only, local only, no network, no credentials
    ConfigPolicy:   DetectAndOffer,
    DetectionFunc:  detectSenseRelevance, // true if project has files in supported languages
    Prerequisites:  []string{"sense"},
    InstallHint:    "curl -fsSL https://luuuc.github.io/sense/install.sh | sh",
    BinarySize:     "~60 MB",
    IndexSize:      "100-200 MB per project",
}
```

**Interaction with existing tools**:
- **Replaces semble** (Unit 11.3): Sense covers semble's entire capability set (semantic code search) plus adds graph analysis, blast radius, and convention detection. Sense is Go-native (no Python dependency), zero-external-dependency, and has richer language support. If Sense is enabled, semble should be disabled automatically to avoid tool count waste.
- **Complements Version-Sentinel** (Unit 11.2): No overlap. Version-Sentinel guards dependency version changes; Sense navigates code structure.
- **Complements agent-postmortem** (Unit 11.1): No overlap. Agent-postmortem verifies task completion; Sense assists during the task.
- **No hook conflicts**: Sense uses `pre-tool-use` (nudging) and `post-tool-use` (re-indexing). gdev's security hooks use `PreToolUse` for package guardrails (install command blocking). Different trigger patterns — Sense intercepts grep/glob/bash for symbol-shaped queries, gdev's hooks intercept npm/pip/cargo install commands. They can coexist.

### Option B: Default (NOT RECOMMENDED)

**Why not**: 
- 60MB binary is a significant addition to a "single binary, zero prerequisites" tool
- 100-200MB index per project is non-trivial for developers who don't use AI agents
- O'Saasy license is not MIT/Apache — introduces legal review requirement for enterprise adopters
- The tool is single-author with 4 stars — too early to bet the default stack on it
- gdev principle #9 (single binary, zero prerequisites) conflicts with requiring a separate binary download

### Option C: Concept/Implementation Inspiration (ALSO RECOMMENDED, complementary to Option A)

Several Sense patterns are worth borrowing for gdev's own implementation:

1. **Silent hook failure pattern** (`silentRun` wrapper): Sense's design that hooks always return exit code 0, never blocking the host tool, is exactly the right pattern for gdev-generated hooks. gdev's PreToolUse hooks should adopt this: if the hook script fails (missing dependency, parse error, network timeout), it should allow the operation rather than blocking the developer. This is already implied in gdev's design but Sense provides a clean reference implementation.

2. **Post-tool-use incremental re-indexing**: The pattern of using `post-tool-use` hooks to keep an index fresh as the agent edits files is directly applicable to Version-Sentinel. When Claude Code edits a manifest file, a post-tool-use hook could trigger re-verification of the changed dependencies, rather than relying solely on pre-tool-use interception.

3. **Pre-compact context injection**: Sense's `pre-compact` hook that injects structural summaries before context window compaction is a pattern gdev could use to inject security posture summaries. Before compaction, inject: "This project has 6 security layers active, last scan found 0 vulnerabilities, 2 age-gated packages pending review." This keeps security awareness alive across long sessions.

4. **Detect-and-nudge vs hard-block**: Sense's `pre-tool-use` hook *nudges* (suggests alternatives) rather than *blocks* (denies the operation). This creates a two-tier intervention model: security hooks hard-block dangerous operations (gdev's current approach), while productivity hooks soft-nudge toward better alternatives (Sense's approach). gdev could formalize this distinction.

5. **Idempotent setup with deep-merge**: Sense's setup system that deep-merges JSON configs and uses marker comments for markdown is the same pattern gdev uses. The implementation in `internal/setup/` is a clean reference for gdev's own `addons/claudecode/` generation logic.

---

## 5. Tradeoffs & Limitations

### Strengths
- Zero external dependencies (no API keys, no network, no cloud services)
- Extremely fast queries (0.2ms p50, 3ms p95)
- Incremental indexing keeps index fresh automatically
- Comprehensive Claude Code integration (5 lifecycle hooks)
- Go-native — same language as gdev, could theoretically be vendored or forked
- Covers 13 languages across gdev's Tier 1-3 ecosystems

### Limitations
- **Not a security tool** — adds no security capability to gdev's defense-in-depth stack
- **Large binary** (60MB) — significant if gdev wanted to bundle it (which is not recommended)
- **Large index** (100-200MB) — disk cost per project
- **ONNX runtime dependency** — embedded via CGo, complicates cross-compilation for gdev's pure-Go distribution model (`CGO_ENABLED=0`)
- **Single author** — maintenance risk if the author abandons the project
- **O'Saasy license** — not OSI-approved; SaaS restriction may require legal review for enterprise consulting clients. The restriction prevents offering Sense "as a hosted, managed, or SaaS product" — this does not affect local CLI usage or gdev integration, but enterprise legal teams may flag it
- **Aggressive session-start hook** — instructs the agent: "Your FIRST tool call MUST be [ToolSearch] to load Sense tools" and "Use Sense MCP tools for ALL codebase understanding — do not use grep, glob, Read, Bash, or agents before loading Sense." This may conflict with gdev-generated CLAUDE.md instructions or other tools' hooks
- **No security scanning** — sense_blast computes structural impact, not vulnerability impact. "What breaks if I change this function?" is useful but different from "Is this dependency vulnerable?"

### Failure Modes
1. **Stale index**: If `sense scan` hasn't been run and the post-tool-use hook is not configured, the index becomes stale. Queries return outdated results. Mitigation: the session-start hook checks freshness and reports stale file counts.
2. **Missing index**: If `.sense/index.db` doesn't exist, all hooks silently return `{}`. The MCP tools return errors. Mitigation: `sense doctor` diagnoses this; `sense scan` recreates it.
3. **Hook conflict**: Sense's session-start hook demands to be the first tool loaded. If gdev's security hooks also demand priority, there's a conflict. Mitigation: gdev controls hook ordering in generated `settings.local.json`.
4. **Token overhead**: If both Sense MCP tools and gdev's security hooks are active, the tool descriptions consume context window space. With 5 Sense tools + gdev's security tools, this needs to stay under the 40-tool ceiling.
5. **Embedding model mismatch**: The bundled ONNX model produces embeddings of a specific dimension. If the model is updated between versions, the index must be rebuilt.

---

## 6. Comparison to Alternatives

### Sense vs Semble (already in gdev plan, Unit 11.3)

| Dimension | Sense | Semble |
|---|---|---|
| Language | Go (88%) | Python |
| Runtime dependency | None (bundled ONNX) | Python >=3.10 + uvx |
| Search | Hybrid semantic + keyword | Hybrid BM25 + semantic |
| Code understanding | Symbol graph + blast radius + conventions | Search only |
| MCP server | Yes (native) | Yes (via semble[mcp]) |
| CLI | Yes (full CLI) | Yes (limited) |
| Language support | 13 languages | Broad (tree-sitter based) |
| Index persistence | SQLite (.sense/) | In-memory + optional cache |
| Binary size | ~60 MB | N/A (Python package) |
| Stars | 4 | 798 |
| License | O'Saasy | MIT |
| Claude Code hooks | 5 lifecycle hooks | None |

**Verdict**: Sense is functionally superior (graph analysis, blast radius, conventions, hooks) but less proven (4 vs 798 stars) and uses a non-standard license. For gdev, Sense's Go-native design and zero-Python-dependency model is a significant advantage — gdev targets environments where Python may not be installed. If community adoption grows, Sense would fully replace semble in the gdev stack.

### Sense vs GitHub Copilot Workspace / Cursor Codebase Indexing

These are proprietary, cloud-backed alternatives that solve similar problems (codebase understanding for AI) but require network connectivity and vendor lock-in. Sense's local-only, zero-dependency model aligns better with gdev's air-gapped-friendly design philosophy.

### Sense vs ctags/LSP

Traditional code navigation tools (ctags, Language Server Protocol servers) provide symbol-level navigation but lack:
- Blast radius analysis
- Convention detection
- AI-agent-native MCP interface
- Semantic search

Sense is effectively a next-generation ctags purpose-built for AI agent workflows.

---

## 7. Sources

All raw sources saved to `docs/`:
- `docs/sense-github-repo-page.md` — Repository overview and metrics
- `docs/sense-readme-raw.md` — Full README content
- `docs/sense-main-go.md` — CLI entry point (complete source)
- `docs/sense-go-mod.md` — Dependency manifest (complete)
- `docs/sense-internal-directory-listing.md` — 27 internal packages
- `docs/sense-hook-go.md` — Hook dispatcher (complete source)
- `docs/sense-session-start-go.md` — Session start hook (complete source)
- `docs/sense-pre-tool-use-go.md` — Pre-tool-use nudging system
- `docs/sense-pre-compact-go.md` — Pre-compact context injection
- `docs/sense-post-tool-use-go.md` — Incremental re-indexing hook
- `docs/sense-subagent-start-go.md` — Sub-agent context injection
- `docs/sense-mcpserver-overview.md` — MCP server architecture
- `docs/sense-setup-go.md` — Auto-configuration system
- `docs/sense-blast-engine-go.md` — Blast radius BFS algorithm
- `docs/sense-conventions-go.md` — Convention detection system
- `docs/sense-license.md` — O'Saasy license terms
- `docs/sense-blast-directory.md` — Blast package file listing
- `docs/sense-scan-directory.md` — Scan package file listing
- `docs/sense-bench-directory.md` — Benchmark infrastructure
- `docs/sense-extract-directory.md` — Language extractor modules
