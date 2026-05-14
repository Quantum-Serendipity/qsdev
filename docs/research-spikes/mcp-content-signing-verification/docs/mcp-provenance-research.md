# MCP Provenance and Differential Trust: Detailed Findings

- **Research Date**: 2026-05-14
- **Question**: Can MCP server responses carry provenance metadata that an AI coding assistant could use for differential trust at query time?

## 1. MCP Protocol Capabilities for Metadata on Tool Responses

### Available Metadata Fields

The MCP specification (2025-11-25) provides three metadata mechanisms on tool call responses:

**a) `_meta` field (on CallToolResult and individual content items)**

The primary extensibility mechanism. Defined as `{ "additionalProperties": {}, "type": "object" }` — a fully open container that permits arbitrary key-value pairs. Present on:
- The `CallToolResult` envelope (result-level metadata)
- Individual content items: `TextContent`, `ImageContent`, `AudioContent`, `EmbeddedResource`

This is explicitly designed as "a vendor-namespaced extension point for implementation-specific metadata that travels with the payload without polluting the protocol schema." Anthropic already uses it for:
- `anthropic/maxResultSizeChars` — controls result persistence threshold in Claude Code
- `anthropic/alwaysLoad` — marks tools as always-loaded

**b) `annotations` field (on content items)**

Standard annotations with defined fields:
- `audience`: array of Role enum (controls who sees the content — "user", "assistant", or both)
- `priority`: number 0-1 (importance weighting)
- `lastModified`: ISO 8601 timestamp

These are closed — no custom annotation fields are specified.

**c) `structuredContent` field (on CallToolResult)**

Arbitrary JSON object with optional schema validation via `outputSchema`. Could contain provenance fields as part of a structured response, but this is designed for tool output data, not metadata about the output.

### Can Servers Include Arbitrary Metadata?

**Yes, via `_meta`.** The specification explicitly supports this. A server could include fields like:

```json
{
  "content": [
    {
      "type": "text",
      "text": "Documentation content here...",
      "_meta": {
        "gdev/source": "https://devdocs.io/typescript/...",
        "gdev/verificationStatus": "signed-valid",
        "gdev/contentHash": "sha256:abc123...",
        "gdev/lastVerified": "2026-05-14T10:00:00Z",
        "gdev/signingKey": "minisign:RWTxxxxxx"
      }
    }
  ],
  "_meta": {
    "gdev/serverVersion": "1.0.0",
    "gdev/verificationMode": "cached"
  }
}
```

This would be fully compliant with the MCP specification.

### Is There a Standard for Content Provenance in MCP?

**No.** No standard fields exist for content provenance, trust signals, or verification metadata. The specification focuses on:
- Server identity (OAuth, `.well-known` discovery)
- Tool capability descriptions (annotations on tool definitions)
- Content routing (audience, priority)

Content provenance within responses is not addressed at all. The 2026 MCP roadmap does not include any planned features for response-level provenance either.

### How Does Claude Code Handle _meta in Responses?

**Critical finding: The LLM does not see `_meta` fields.** Claude Code processes `_meta` in the harness layer, not at the model level:

- `anthropic/maxResultSizeChars` controls result persistence behavior in the harness
- `anthropic/alwaysLoad` controls tool loading behavior in the harness
- "Enforcement is in the harness, not the model — the LLM sees preserved bytes, not the annotation"

This means custom provenance metadata in `_meta` would be invisible to Claude unless:
1. Claude Code's harness was modified to extract and surface it (requires Anthropic changes)
2. The provenance data was embedded in the text content itself (visible to the model)

## 2. How Existing MCP Servers Handle Trust/Provenance

### Source Attribution in MCP Responses

No evidence found of any MCP server including structured source attribution in responses. Servers that serve documentation (if any exist) simply return the text content. Source attribution, when present, is informal — embedded in the text itself, not in metadata fields.

### Integrity/Verification Metadata

No MCP servers found that include integrity or verification metadata in responses. The ecosystem focus is on:
- **Server-level trust**: Verifying that the server binary/container comes from its claimed source (Stacklok's approach using Sigstore container attestation)
- **Transport-level integrity**: TLS for data in transit
- **Authentication**: OAuth 2.0 for server identity

Response-level integrity is an identified gap. The CoSAI OASIS security analysis flags "Missing Integrity/Verification Controls" (MCP-T6) as a critical gap and recommends "end-to-end cryptographic signatures proving authenticity of resources returned by servers" — but this is a recommendation, not an implementation.

### Documentation-Focused MCP Servers

The MCP ecosystem focuses on API integrations (GitHub, Jira, Sentry, databases), not documentation serving. No documentation-focused MCP server with provenance features was found.

## 3. Architectural Feasibility of a Verification-Aware MCP Server

### Query-Time Integrity Checking

**Feasible and straightforward.** A gdev documentation MCP server could:

1. **At startup**: Load the Minisign public key and hash manifest
2. **At query time**: Check content hash against the manifest before serving
3. **On verification failure**: Return an error or warning in the response

Implementation options:
- **Per-query hash verification**: Compute SHA-256 of content, compare against pinned hash. Cost: ~1ms for typical doc pages (<100KB). Negligible performance impact.
- **Cached verification**: Verify once at server startup or content load, cache the result. Eliminates per-query cost entirely. Re-verify on file modification (inotify/fswatch).
- **Signature verification at startup**: Verify Minisign signatures on the content files when the MCP server starts. Cache verification status. Cost: ~10ms per file for Minisign verification.

### Could Responses Include Provenance Fields?

**Yes, via two paths:**

**Path A: `_meta` fields (structured, invisible to model)**
```json
{
  "type": "text",
  "text": "...",
  "_meta": {
    "gdev/verificationStatus": "signed-valid",
    "gdev/contentHash": "sha256:abc123"
  }
}
```
Pro: Clean separation of metadata from content. Follows MCP convention.
Con: Claude does not see these fields. Only useful if Claude Code's harness processes them.

**Path B: Inline in text content (visible to model)**
```
[Source: TypeScript Handbook | Verified: sha256:abc123 | Status: signed-valid]

... actual documentation content ...
```
Pro: The model actually sees the provenance signals and could reason about trust.
Con: Pollutes content with metadata. Model might not reliably use it for trust decisions. Susceptible to the model ignoring or misinterpreting the signals. An attacker who controls the content could also inject fake provenance headers.

**Path C: `structuredContent` with schema**
Define an `outputSchema` that includes both content and provenance fields. The model receives structured data including verification metadata.
Pro: Schema-validated. Model sees all fields.
Con: More complex implementation. Still susceptible to model not reliably using the metadata for trust decisions.

### Would Claude Code Actually Use This for Trust Decisions?

**No, not without significant changes.** Current state:

1. **Harness level**: Claude Code only processes two `_meta` keys (`anthropic/maxResultSizeChars`, `anthropic/alwaysLoad`). Adding provenance processing would require Anthropic to modify Claude Code.
2. **Model level**: Claude would see inline provenance text but has no built-in mechanism for differential trust based on source metadata. It would need to be instructed via system prompt or CLAUDE.md to treat verification status as a trust signal. This is fragile — prompt-based trust is easily bypassed by content that contradicts the instructions.
3. **No framework support**: Neither Claude Code nor the MCP specification provides a trust classification framework for tool responses. All responses from approved MCP servers are treated equally.

### Performance Implications

| Approach | Per-Query Cost | Startup Cost | Memory |
|----------|---------------|--------------|--------|
| Per-query SHA-256 hash check | ~1ms per doc page | None | Hash manifest (~1KB per entry) |
| Cached verification status | ~0ms (hash table lookup) | ~10ms per file (Minisign) | Status cache (~100B per entry) |
| Startup-only verification | ~0ms | ~seconds (all files) | Full status cache |

Recommendation: **Cached verification at startup** is the practical choice. Verify all content at server start, re-verify on file changes. Per-query cost is a hash table lookup.

## 4. Prior Art in Differential Trust for AI Tool Use

### AI Frameworks with Trusted/Untrusted Distinction

**No AI framework found that implements differential trust for tool responses at the model level.** The MCP specification says "clients MUST consider tool annotations to be untrusted unless they come from trusted servers" — but this is about tool definitions, not response content, and the trust boundary is binary (trusted server vs untrusted server).

The Stacklok approach (Sigstore container attestation) provides binary server trust (verified/unverified) with three enforcement modes (disabled/warning/strict), but does not extend to per-response trust.

### RAG Systems with Source Quality Signals

RAG systems implement the closest analogues to differential trust:

- **TruLens**: Context precision filtering — evaluates how relevant retrieved fragments are
- **RE-RAG**: Two-stage reranking with cross-encoders for relevance scoring
- **RAGChecker**: Claim-level entailment checking against retrieved evidence
- **RAGAS**: Multi-dimensional scoring (faithfulness, relevancy, precision, recall)
- **Learn-to-Refuse**: Abstention when confidence is insufficient

Key insight: These systems evaluate trust at the **retrieval/reranking layer** (before the content reaches the LLM), not at the generation layer. The LLM receives pre-filtered, quality-scored content. This is architecturally different from asking the LLM itself to make trust decisions based on metadata.

### MCP-Specific Trust Mechanisms

- **CoSAI OASIS**: Recommends "end-to-end cryptographic signatures proving authenticity of resources returned by servers" — but this is a recommendation, not implemented.
- **Trustworthy MCP Registry paper** (MDPI Future Internet, 2025): Proposes cryptographic provenance for server identity and registry entries, but focused on server trust, not content/response trust.
- **2026 MCP Roadmap**: Lists "deeper security and authorization work" as "on the horizon" but provides no specifics about response-level provenance.

## 5. Assessment

### What Works Today

1. **`_meta` can carry provenance metadata** — Fully specification-compliant. A gdev MCP server could include verification status, content hashes, source URLs, and timestamps in `_meta` fields on every response.

2. **Query-time verification is cheap** — Cached hash verification adds negligible latency. Minisign signature verification at startup is fast.

3. **Inline provenance text reaches the model** — Embedding provenance in the text content is the only way to make the model aware of verification status without Claude Code modifications.

### What Doesn't Work Today

1. **Claude Code ignores custom `_meta` fields** — The harness only processes Anthropic-namespaced keys. Custom provenance metadata in `_meta` would be silently dropped.

2. **No model-level differential trust** — Claude has no mechanism to systematically weight tool responses differently based on provenance metadata. Prompt-based instructions to do this would be fragile and unreliable.

3. **No ecosystem precedent** — No MCP server, AI framework, or specification includes response-level provenance. Building this would be entirely novel, with no guarantee clients will use it.

### Recommendation

**Verification at download time (in gdev's Nix packaging) is sufficient for the foreseeable future.** MCP-level provenance metadata is architecturally feasible but currently useless because no client processes it for trust decisions.

The practical path is:
1. **Now**: Minisign signing at download/update time + Nix hash pinning = verified content on disk
2. **Now**: MCP server can verify content at startup (hash check) and refuse to serve tampered content = runtime integrity
3. **Future-proof**: Include `_meta` provenance fields in responses as a forward-looking investment. If Claude Code or the MCP ecosystem adds trust differentiation, gdev would be ready.
4. **Pragmatic text signal**: Include a brief provenance header in documentation responses (source, verification status) as visible-to-model metadata. Even without systematic trust processing, this gives the model context about where content came from.

Sources:
- MCP Specification 2025-11-25: https://modelcontextprotocol.io/specification/2025-11-25/server/tools
- MCP JSON Schema 2025-11-25: schema.json on GitHub
- CoSAI OASIS MCP Security: https://github.com/cosai-oasis/ws4-secure-design-agentic-systems/blob/main/model-context-protocol-security.md
- 2026 MCP Roadmap: https://blog.modelcontextprotocol.io/posts/2026-mcp-roadmap/
- Claude Code MCP Docs: https://code.claude.com/docs/en/mcp
- Lakera MCP Trust Analysis: https://www.lakera.ai/blog/what-the-new-mcp-specification-means-to-you-and-your-agents
- Stacklok MCP Server Trust: https://dev.to/stacklok/from-unknown-to-verified-solving-the-mcp-server-trust-problem-5967
- RAG Trust Frameworks Survey: https://arxiv.org/html/2601.05264v1
