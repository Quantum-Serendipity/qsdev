# Lakera: What the New MCP Specification Means to You, and Your Agents

- **Source**: https://www.lakera.ai/blog/what-the-new-mcp-specification-means-to-you-and-your-agents
- **Retrieved**: 2026-05-14

## How MCP Handles Trust Between Servers and Clients

Historically, MCP operated on what the article characterizes as "radical optimism." Agents blindly accepted whatever servers claimed about their capabilities through the `tools/list` call without verification. Agents would inherit "combined authority" based entirely on server assertions, creating significant vulnerability to misrepresentation.

## Trust Signals and Verification in Tool Responses

Early limitations: Tool descriptions were trusted implicitly, with no verification layer. "Malicious instructions could be hidden inside the tool descriptions themselves," such as innocuous-sounding tools containing hidden directives.

New specification improvements: Servers can now "publish a small identity document somewhere predictable," enabling agents to compare identity claims across time and detect when servers expand their declared capabilities unexpectedly.

## Content Integrity Verification

The article does not explicitly address cryptographic integrity verification or checksums for responses. It focuses on identity consistency rather than response-level validation.

## Provenance or Trust Status Communication

The new spec introduces Protected Resource Metadata, described as "a formal declaration of how a server expects you to authenticate, what permissions a tool really needs, and who is allowed to use what." This represents explicit provenance information about tool authorization requirements rather than implicit trust.

## Differential Trust Based on Metadata

The specification enables servers to maintain identity documents that agents can cross-reference, theoretically allowing differentiated trust decisions based on whether server behavior matches documented capabilities.
