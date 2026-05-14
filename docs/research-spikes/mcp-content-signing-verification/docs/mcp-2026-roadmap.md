# 2026 MCP Roadmap

- **Source**: https://blog.modelcontextprotocol.io/posts/2026-mcp-roadmap/
- **Retrieved**: 2026-05-14

## Trust, Provenance, or Content Verification

Not addressed in the roadmap. While security work appears on the horizon, no specific features for trust or provenance verification of content/responses are mentioned.

## Planned Changes to Tool Response Formats

The roadmap includes experimentation with "streamed and reference-based result types" listed under the "On the Horizon" section, though implementation details remain undefined.

## Metadata Extensibility

The Transport Evolution priority includes developing "a standard metadata format, that can be served via `.well-known`" to enable server capability discovery without live connections. This addresses horizontal scalability and discoverability needs. This is server-level metadata, not response-level metadata.

## Registry and Discovery Features

The roadmap identifies a gap: "there's no standard way for a registry or crawler to learn what a server does without connecting to it." The solution involves implementing the `.well-known` metadata format.

## Security-Related Items

- Primary focus: SSO-integrated authentication and audit trails under Enterprise Readiness
- On the Horizon: Deeper security and authorization work, plus two active proposals — DPoP and Workload Identity Federation SEPs
- Note: No specific timeline provided; these fall outside the top four priority areas

Overall timeline: The roadmap emphasizes working group-driven development rather than release dates, reflecting the project's maturity phase.
