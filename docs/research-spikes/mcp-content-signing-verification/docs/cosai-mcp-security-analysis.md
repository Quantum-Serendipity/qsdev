# CoSAI OASIS: Model Context Protocol Security Analysis

- **Source**: https://github.com/cosai-oasis/ws4-secure-design-agentic-systems/blob/main/model-context-protocol-security.md
- **Retrieved**: 2026-05-14

## Trust Assumptions for Tool Responses

The document explicitly states that tool descriptions should be treated as untrusted: "descriptions of tool behavior such as annotations should be considered untrusted, unless obtained from a trusted server." This reflects a fundamental assumption that MCP servers cannot be implicitly trusted, particularly regarding metadata they provide about their own capabilities.

The threat model assumes adversaries can manipulate tool definitions post-deployment, making defensive validation essential rather than optional.

## Content Integrity and Verification Mechanisms

The document identifies "Missing Integrity/Verification Controls" (MCP-T6) as a critical gap. Recommended mitigations include:

- Cryptographic signatures for MCP server code and software bills of materials (SBOMs)
- Remote attestation to verify servers run expected code in trusted environments
- End-to-end cryptographic signatures proving authenticity of resources returned by servers
- Code signing verification for all MCP servers before installation
- TLS protection for all data in transit

## Content Provenance in Responses

The document emphasizes supply chain security but provides limited specifics on response provenance. It recommends organizations "obtain and verify the contents and cryptographic signatures prior to deployment" and maintain "policies restricting the approved sources and signing keys."

However, there's no detailed mechanism described for embedding provenance metadata within individual tool responses themselves.

## Server-to-Client Trust Model

The architecture assumes optional authentication between client and server, creating vulnerabilities. The document notes that MCP lacks native fine-grained authorization, enabling unauthorized data access and privilege escalation. Single-Tenant deployments explicitly require authentication to establish trust boundaries, but the protocol doesn't mandate this universally.

## Response Content Verification Recommendations

Key recommendations include:

- Treat "all AI-generated content as untrusted input" requiring identical validation as direct user input
- Deploy "prompt injection detection systems" analyzing patterns and structured formats
- Maintain "clear boundaries between instructions and data" through strict JSON schemas
- Apply "context-aware output encoding" appropriate to each execution context
- Use "parameterized queries" and input canonicalization for file paths

## Security Risks: Tool Response Content

**Resource Content Poisoning** (threat 4) directly targets response integrity: attackers embed malicious instructions in backend data sources that MCP servers retrieve, causing poisoned content to execute as commands when processed by LLMs. This represents persistent prompt injection through trusted data channels rather than direct user manipulation.

**Tool Poisoning** (threat 2) compromises tool metadata itself, while **Full Schema Poisoning** extends this to structural definitions, potentially injecting hidden parameters or altered return types affecting all subsequent invocations while appearing legitimate.

Human-in-the-loop approval, while recommended, faces "consent fatigue" where users reflexively approve prompts without careful review, undermining protective intent.
