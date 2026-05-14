# MCP Provenance and Differential Trust

## Research Question

Can MCP server responses carry provenance metadata (content source, verification status, hash) that Claude Code could use for differential trust at query time?

## MCP Protocol Capabilities for Metadata

The MCP specification (2025-11-25) provides a clear extensibility mechanism: the `_meta` field. This is an open-schema JSON object (`additionalProperties: {}`) present on both the `CallToolResult` envelope and on individual content items (`TextContent`, `ImageContent`, etc.). It is explicitly designed as a vendor-namespaced extension point — Anthropic already uses it for `anthropic/maxResultSizeChars` and `anthropic/alwaysLoad` in Claude Code.

A gdev MCP server could include provenance fields like `gdev/verificationStatus`, `gdev/contentHash`, `gdev/source`, and `gdev/lastVerified` in `_meta` without violating the specification. This is fully compliant.

The standard `annotations` field on content items provides `audience`, `priority`, and `lastModified` but is closed — no custom fields. The `structuredContent` field with `outputSchema` could carry provenance as part of a typed response, but this is designed for tool output data, not metadata about trust.

No standard exists for content provenance or trust signals in MCP. The 2026 MCP roadmap does not include response-level provenance. The focus is on server identity (`.well-known` discovery, OAuth) and enterprise features (SSO, audit trails).

## Whether Provenance Metadata Is Architecturally Feasible

**Yes, with a significant caveat.** The MCP server can easily carry provenance metadata in `_meta` and can verify content at query time. A cached verification approach (verify Minisign signatures at server startup, re-verify on file changes) adds negligible latency — a hash table lookup per query vs ~1ms for per-query SHA-256 computation.

The caveat: **Claude Code does not surface custom `_meta` fields to the model.** The harness processes only Anthropic-namespaced keys and discards everything else. "Enforcement is in the harness, not the model — the LLM sees preserved bytes, not the annotation." This means `_meta` provenance data would be invisible to Claude without Anthropic modifying Claude Code.

The alternative — embedding provenance in the text content itself (e.g., a `[Verified: sha256:abc123]` header) — makes it visible to the model but has problems: it pollutes content, the model has no systematic mechanism to use it for trust decisions, and an attacker who controls the content could inject fake provenance headers.

## Whether Claude Code Would Use It for Trust Decisions

**No, not in its current architecture.** Claude Code's trust model for MCP is binary at the server level: servers are either approved (trusted) or not. All responses from an approved server receive equal treatment. There is no:
- Per-response trust evaluation
- Metadata-based trust differentiation
- Content verification pipeline on received tool results
- Model-level instruction to weight responses by provenance

The CoSAI OASIS security analysis identifies "Missing Integrity/Verification Controls" (MCP-T6) as a critical gap and recommends "end-to-end cryptographic signatures proving authenticity of resources returned by servers." But this is a recommendation in a security analysis paper, not a feature in any implementation.

## Prior Art in AI Differential Trust

No AI coding assistant or MCP client implements differential trust for tool responses. The closest analogues are in RAG systems:

- **TruLens** and **RE-RAG** filter and rerank retrieved documents by relevance/quality before they reach the LLM
- **RAGChecker** performs claim-level entailment checking against source evidence
- **RAGAS** provides multi-dimensional scoring (faithfulness, relevancy, precision, recall)
- **Learn-to-Refuse** enables models to abstain when retrieval confidence is insufficient

The key architectural difference: RAG trust operates at the retrieval/reranking layer (pre-model), not by asking the model to make trust decisions based on metadata in the content. This is a pipeline-level mechanism, not a prompt-level one.

MCP-specific trust work (Stacklok Sigstore verification, Trustworthy MCP Registry paper) focuses entirely on server identity and supply chain integrity, not on response content provenance.

## Assessment: Is MCP-Level Provenance Worth Implementing?

**Verification at download time is sufficient; MCP-level provenance is a low-cost forward investment but not currently actionable.**

The practical architecture:

1. **Primary defense (now)**: Minisign signing at download/update time + Nix SRI hash pinning. Content on disk is verified. This is where the security value is.

2. **Runtime integrity (now)**: The MCP server verifies content hashes at startup and refuses to serve files that fail verification. This catches post-download tampering (local filesystem attacks, corrupted files) without any MCP protocol changes.

3. **Forward-looking metadata (low cost)**: Include `_meta` provenance fields in responses. Zero specification risk, negligible implementation cost. If the MCP ecosystem develops trust differentiation (CoSAI recommendations gain traction, Claude Code adds `_meta` processing for trust), gdev would be ready without changes.

4. **Pragmatic text signal (optional)**: A brief provenance line in documentation responses gives the model informational context about content origin. Not a security mechanism — treat it as source attribution, not trust enforcement.

What is NOT worth pursuing: building a custom trust differentiation system that requires Claude Code modifications, or designing a novel content attestation protocol on top of MCP. The ecosystem is not there, and the security gains over download-time verification are marginal for local-first documentation.

## Sources

- MCP Specification 2025-11-25, Tools: `docs/mcp-spec-tools-2025-11-25.md`
- MCP JSON Schema 2025-11-25: `docs/mcp-schema-json-2025-11-25.md`
- CoSAI OASIS MCP Security Analysis: `docs/cosai-mcp-security-analysis.md`
- 2026 MCP Roadmap: `docs/mcp-2026-roadmap.md`
- Claude Code MCP Documentation: `docs/claude-code-mcp-docs.md`
- Lakera MCP Trust Analysis: `docs/lakera-mcp-trust-analysis.md`
- Stacklok MCP Server Trust: `docs/stacklok-mcp-server-trust.md`
- RAG Trust Frameworks Survey: `docs/rag-trust-frameworks-survey.md`
